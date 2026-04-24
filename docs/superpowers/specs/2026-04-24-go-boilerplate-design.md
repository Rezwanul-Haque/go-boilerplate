# Go Boilerplate — Design Spec
**Date:** 2026-04-24  
**Status:** Approved

---

## Overview

Production-ready Go REST API boilerplate using Echo framework, Clean Architecture + Vertical Slice Architecture, and SOLID principles. Ships with a fully implemented `users` feature covering auth lifecycle endpoints.

---

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.26 |
| Module | `go-boilerplate` |
| HTTP Framework | Echo v4 |
| Database | PostgreSQL via pgx v5 |
| Migrations | golang-migrate |
| Auth | JWT — stateless, access + refresh tokens |
| Logging | zerolog (structured) |
| Config | godotenv |
| Rate Limiting | golang.org/x/time/rate (IP-based, signup only) |
| Containerization | Docker multi-stage + docker-compose |

---

## Architecture

**Pattern:** Clean Architecture + Vertical Slice Hybrid

Each feature is a self-contained vertical slice owning its HTTP handler, business logic, and domain contracts (interfaces). Infrastructure implementations are injected at bootstrap — features never import infra packages directly.

**Dependency rule:** Dependencies flow inward only.
```
handler → service interface → repository interface
                                      ↑
               infra/database/users/pg_repository.go implements it
```

**SOLID alignment:**
- **S** — each file has one responsibility (handler, service, repo, dto, errors separate)
- **O** — new features add new slices, never modify existing ones
- **L** — service/repository impls are substitutable via interfaces
- **I** — small, focused interfaces per feature (not god interfaces)
- **D** — features depend on abstractions (interfaces), not concrete infra

---

## Folder Structure

```
go-boilerplate/
├── cmd/
│   └── main.go                      # wires all dependencies, starts server
│
├── app/
│   ├── bootstrap/
│   │   ├── app.go                   # Echo instance, global middleware chain
│   │   └── routes.go                # registers all feature route groups
│   │
│   ├── features/
│   │   └── users/
│   │       ├── handler.go           # Echo HTTP handlers
│   │       ├── service.go           # Service interface + implementation
│   │       ├── repository.go        # Repository interface only
│   │       ├── model.go             # User domain model
│   │       ├── dto.go               # Request / Response structs + validation
│   │       ├── errors.go            # Feature-scoped sentinel errors
│   │       └── routes.go            # Mounts routes onto echo.Group
│   │
│   ├── infra/
│   │   ├── database/
│   │   │   ├── postgres.go          # pgx pool setup, implements ports.DB
│   │   │   ├── migrations/
│   │   │   │   └── 001_create_users.sql
│   │   │   └── users/
│   │   │       └── pg_repository.go # implements users.Repository
│   │   ├── logger/
│   │   │   └── zerolog.go           # implements ports.Logger
│   │   ├── middleware/
│   │   │   ├── auth.go              # JWT validation middleware
│   │   │   ├── rate_limiter.go      # IP-based rate limiter (signup only)
│   │   │   └── request_logger.go    # structured per-request logging
│   │   └── notification/
│   │       └── mock_notifier.go     # implements ports.Notifier, logs to stdout
│   │
│   └── shared/
│       ├── ports/
│       │   ├── db.go                # DB / Querier / Tx interfaces
│       │   ├── logger.go            # Logger interface
│       │   └── notifier.go          # Notifier interface
│       ├── config/
│       │   └── config.go            # env-based config struct
│       ├── response/
│       │   └── response.go          # unified JSON response helpers
│       ├── apperror/
│       │   └── apperror.go          # typed app errors → HTTP status mapping
│       └── token/
│           └── jwt.go               # sign / parse / validate JWT
│
├── docker/
│   └── Dockerfile                   # multi-stage: builder + minimal runtime
├── docker-compose.yml               # api + postgres services
├── Makefile                         # make dev, make build, make migrate-up/down
├── .env.example
└── go.mod
```

---

## Users Feature — Endpoints

| Method | Path | Auth | Rate Limited |
|---|---|---|---|
| POST | `/api/v1/users/signup` | None | Yes — 5 req/min per IP |
| POST | `/api/v1/users/login` | None | No |
| POST | `/api/v1/users/forgot-password` | None | No |
| POST | `/api/v1/users/reset-password` | None | No |
| PUT | `/api/v1/users/change-password` | JWT required | No |
| POST | `/api/v1/users/refresh-token` | None | No |

---

## Users Feature — Contracts

### `users.UserRepository` + `users.PasswordResetRepository` (`features/users/repository.go`)
```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByEmail(ctx context.Context, email string) (*User, error)
    FindByID(ctx context.Context, id uuid.UUID) (*User, error)
    UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
}

type PasswordResetRepository interface {
    SaveResetToken(ctx context.Context, id uuid.UUID, token string, expiresAt time.Time) error
    FindByResetToken(ctx context.Context, token string) (*User, error)
    ClearResetToken(ctx context.Context, id uuid.UUID) error
}
```

`pg_repository.go` implements both interfaces. Service struct depends only on what it needs — `ForgotPassword` and `ResetPassword` consume `PasswordResetRepository`; all other operations consume `UserRepository`.

### `users.Service` interface (`features/users/service.go`)
```go
type Service interface {
    Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error)
    Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
    ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
    ResetPassword(ctx context.Context, req ResetPasswordRequest) error
    ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
    RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error)
}

// service implementation depends on split repositories
type service struct {
    repo      UserRepository
    resetRepo PasswordResetRepository
    notifier  ports.Notifier
    token     token.Maker
    logger    ports.Logger
}
```

---

## Infrastructure Ports (`shared/ports/`)

```go
// db.go
type DB interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// logger.go
type Logger interface {
    Info(msg string, fields ...any)
    Error(msg string, err error, fields ...any)
    Debug(msg string, fields ...any)
    Warn(msg string, fields ...any)
}

// notifier.go
type Notifier interface {
    SendPasswordReset(ctx context.Context, email, token string) error
}
```

---

## Auth Flow

**Signup:** hash password (bcrypt) → store user → return access + refresh JWT  
**Login:** verify password → return access + refresh JWT  
**Forgot password:** generate reset token → store with 1h expiry → call Notifier  
**Reset password:** validate token + expiry → hash new password → clear token  
**Change password:** validate current password → hash new password → update  
**Refresh token:** validate refresh JWT → issue new access token

**JWT:** access token TTL = 15min, refresh token TTL = 7 days. Both signed with HS256. Claims include `user_id`, `email`, `type` (access|refresh).

---

## Rate Limiting

Middleware applied at route level to `POST /api/v1/users/signup` only.  
Strategy: token bucket per IP (`golang.org/x/time/rate`).  
Limit: 5 requests/min. Returns `429 Too Many Requests` on breach.

---

## Database Schema

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    reset_token VARCHAR(255),
    reset_token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Docker / Makefile

**docker-compose services:** `api` (port 8080) + `postgres` (port 5432)  
**Dockerfile:** multi-stage — `golang:1.26-alpine` builder → `alpine` runtime, non-root user

**Makefile targets:**

| Target | Action |
|---|---|
| `make dev` | Run via docker-compose |
| `make build` | Build binary |
| `make migrate-up` | Run migrations |
| `make migrate-down` | Rollback migrations |
| `make test` | Run tests |
| `make lint` | Run golangci-lint |

---

## Error Handling

`apperror.AppError` wraps business errors with HTTP status codes. Handler maps `AppError` → JSON response. Unknown errors → 500. All errors logged with zerolog including request ID.

---

## Out of Scope

- Real email/SMS delivery (mock notifier only)
- Redis / token blacklisting
- Role-based access control
- Metrics / tracing
