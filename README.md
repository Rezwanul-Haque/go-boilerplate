# Go Boilerplate

Production-ready REST API boilerplate using Echo, Clean Architecture + Vertical Slice, and SOLID principles.

---

## Stack

| | |
|---|---|
| Language | Go 1.26 |
| HTTP | Echo v4 |
| Database | PostgreSQL (pgx v5 via stdlib) |
| Auth | JWT — stateless access + refresh tokens |
| Migrations | golang-migrate |
| Logging | zerolog (structured) |
| Config | godotenv |
| Rate Limiting | golang.org/x/time/rate (token bucket, per IP) |
| Container | Docker + docker-compose |

---

## Architecture

**Clean Architecture + Vertical Slice Hybrid.**

Each feature owns its HTTP handler, business logic, and domain contracts (interfaces). Infrastructure implementations are injected at bootstrap — features never import infra packages directly.

```
handler → service interface → repository interface
                                      ↑
               infra/database/{feature}/pg_repository.go implements it
```

```
go-boilerplate/
├── cmd/
│   ├── main.go                         # entry point — wires all deps
│   └── scaffold/
│       └── main.go                     # feature generator CLI
│
├── app/
│   ├── bootstrap/
│   │   ├── app.go                      # Echo instance, global middleware
│   │   └── routes.go                   # registers all feature route groups
│   │
│   ├── features/
│   │   └── users/
│   │       ├── model.go                # domain model
│   │       ├── errors.go               # feature-scoped sentinel errors
│   │       ├── repository.go           # repository interfaces (no DB imports)
│   │       ├── dto.go                  # request / response structs
│   │       ├── service.go              # service interface + implementation
│   │       ├── handler.go              # Echo HTTP handlers
│   │       └── routes.go               # mounts routes onto echo.Group
│   │
│   ├── infra/
│   │   ├── database/
│   │   │   ├── postgres.go             # pgx pool setup
│   │   │   ├── migrations/             # SQL migration files
│   │   │   └── users/
│   │   │       └── pg_repository.go    # implements users.Repository
│   │   ├── logger/
│   │   │   └── zerolog.go              # implements ports.Logger
│   │   ├── middleware/
│   │   │   ├── auth.go                 # JWT Bearer validation
│   │   │   ├── rate_limiter.go         # IP-based token bucket
│   │   │   └── request_logger.go       # structured per-request logging
│   │   └── notification/
│   │       └── mock_notifier.go        # implements ports.Notifier (stdout)
│   │
│   └── shared/
│       ├── ports/                      # infrastructure interfaces (DB, Logger, Notifier)
│       ├── apperror/                   # typed errors with HTTP status codes
│       ├── response/                   # unified JSON response helpers
│       ├── config/                     # env-based config
│       └── token/                      # JWT sign / parse / verify
│
├── docker/Dockerfile                   # multi-stage Alpine build
├── docker-compose.yml
└── Makefile
```

---

## Getting Started

### 1. Clone and configure

```bash
cp .env.example .env
# Edit .env — set JWT_SECRET to a secure random string in production
```

### 2. Run with Docker

```bash
make dev
```

Starts Postgres + API on `http://localhost:8080`. Migrations run automatically on startup.

### 3. Run locally (requires Postgres running)

```bash
make migrate-up
go run ./cmd/main.go
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | Server port |
| `DB_HOST` | `localhost` | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_USER` | `postgres` | Postgres user |
| `DB_PASSWORD` | `postgres` | Postgres password |
| `DB_NAME` | `go_boilerplate` | Database name |
| `DB_SSL_MODE` | `disable` | SSL mode |
| `JWT_SECRET` | `changeme` | HS256 signing secret — **change in production** |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Access token TTL |
| `JWT_REFRESH_TTL_DAYS` | `7` | Refresh token TTL |
| `LOG_LEVEL` | `info` | zerolog level (debug/info/warn/error) |

---

## Users API

| Method | Path | Auth | Rate Limit |
|---|---|---|---|
| `POST` | `/api/v1/users/signup` | — | 5 req/min per IP |
| `POST` | `/api/v1/users/login` | — | — |
| `POST` | `/api/v1/users/forgot-password` | — | — |
| `POST` | `/api/v1/users/reset-password` | — | — |
| `POST` | `/api/v1/users/refresh-token` | — | — |
| `PUT` | `/api/v1/users/change-password` | Bearer JWT | — |

### Example: signup

```bash
curl -X POST http://localhost:8080/api/v1/users/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@example.com","password":"password123"}'
```

```json
{
  "success": true,
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<jwt>",
    "user": { "id": "...", "email": "dev@example.com" }
  }
}
```

### Example: change password (authenticated)

```bash
curl -X PUT http://localhost:8080/api/v1/users/change-password \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"password123","new_password":"newpassword456"}'
```

---

## Creating a New Feature

```bash
make feature name=orders
```

Generates a complete feature skeleton:

```
app/features/orders/
├── model.go
├── errors.go
├── repository.go      ← interface only, no DB imports
├── dto.go
├── service.go
├── handler.go
└── routes.go

app/infra/database/orders/
└── pg_repository.go   ← postgres implementation
```

Then follow the printed next steps to wire it into `cmd/main.go` and `app/bootstrap/routes.go`.

---

## Makefile Targets

| Target | Description |
|---|---|
| `make dev` | Start API + Postgres via docker-compose |
| `make dev-down` | Stop docker-compose services |
| `make build` | Build binary to `bin/server` |
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Rollback last migration |
| `make test` | Run unit tests |
| `make test-unit` | Run unit tests (no DB required) |
| `make test-integration` | Run integration tests (starts Postgres) |
| `make lint` | Run golangci-lint |
| `make tidy` | Run go mod tidy |
| `make feature name=<name>` | Scaffold a new feature |

---

## Testing

**Unit tests** — no database, no network. Use mocks for all external dependencies.

```bash
make test-unit
```

**Integration tests** — require real Postgres. Guarded by `//go:build integration`.

```bash
make test-integration
```

Set `TEST_DB_DSN` env var to override the default test database connection.

---

## Adding a Feature — Step by Step

1. **Generate skeleton**
   ```bash
   make feature name=products
   ```

2. **Add domain fields** to `app/features/products/model.go`

3. **Add DTOs** to `app/features/products/dto.go`

4. **Add migration** at `app/infra/database/migrations/000002_create_products.up.sql`

5. **Implement pg_repository** in `app/infra/database/products/pg_repository.go`

6. **Wire in `cmd/main.go`**
   ```go
   productsRepo    := productsdb.NewPgRepository(db)
   productsSvc     := products.NewService(productsRepo)
   productsHandler := products.NewHandler(productsSvc)
   ```

7. **Register routes in `app/bootstrap/routes.go`**
   ```go
   products.RegisterRoutes(v1.Group("/products"), productsHandler)
   ```
