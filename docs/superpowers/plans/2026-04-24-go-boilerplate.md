# Go Boilerplate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready Go REST API boilerplate with Echo, Clean Architecture + Vertical Slice, SOLID principles, and a complete users auth feature (signup, login, forgot/reset/change password, refresh token).

**Architecture:** Clean Architecture + Vertical Slice Hybrid. Features own contracts (interfaces); infra owns implementations. All dependencies injected at `cmd/main.go`. Features never import infra packages. Dependency flow: `handler → service interface → repository interface ← pg_repository`.

**Tech Stack:** Go 1.26, Echo v4, pgx v5 (via stdlib driver), golang-jwt/jwt v5, zerolog, godotenv, golang.org/x/time/rate, bcrypt, golang-migrate v4, go-playground/validator v10, testify v1.9

**Test strategy:**
- **Unit tests** — all business logic, handlers, middleware tested with mocks (no DB, no network). Run with `make test-unit`.
- **Integration tests** — `pg_repository` + full HTTP round-trip tested against real Postgres. Guarded by `//go:build integration` tag. Run with `make test-integration` (requires Docker Compose running).

---

## File Map

| File | Responsibility |
|---|---|
| `cmd/main.go` | Wire all dependencies, start server, graceful shutdown |
| `app/bootstrap/app.go` | Echo instance, global middleware, validator |
| `app/bootstrap/routes.go` | Register all feature route groups |
| `app/features/users/model.go` | User domain struct |
| `app/features/users/errors.go` | Feature-scoped sentinel errors |
| `app/features/users/repository.go` | `UserRepository` + `PasswordResetRepository` interfaces |
| `app/features/users/dto.go` | Request/Response structs with validate tags |
| `app/features/users/service.go` | `Service` interface + implementation |
| `app/features/users/handler.go` | Echo HTTP handlers |
| `app/features/users/routes.go` | Mount routes onto echo.Group |
| `app/infra/database/postgres.go` | pgx pool setup via stdlib |
| `app/infra/database/migrations/001_create_users.sql` | Users table DDL |
| `app/infra/database/users/pg_repository.go` | Implements `UserRepository` + `PasswordResetRepository` |
| `app/infra/logger/zerolog.go` | Implements `ports.Logger` via zerolog |
| `app/infra/middleware/auth.go` | JWT Bearer validation middleware |
| `app/infra/middleware/rate_limiter.go` | IP-based token bucket rate limiter |
| `app/infra/middleware/request_logger.go` | Structured per-request logging middleware |
| `app/infra/notification/mock_notifier.go` | Implements `ports.Notifier`, logs to stdout |
| `app/shared/ports/db.go` | `DB` interface (database/sql compatible) |
| `app/shared/ports/logger.go` | `Logger` interface |
| `app/shared/ports/notifier.go` | `Notifier` interface |
| `app/shared/apperror/apperror.go` | Typed errors with HTTP status codes |
| `app/shared/response/response.go` | Unified JSON response helpers |
| `app/shared/config/config.go` | Env-based config struct |
| `app/shared/token/jwt.go` | JWT sign/parse/verify, `Maker` interface |
| `docker/Dockerfile` | Multi-stage build |
| `docker-compose.yml` | api + postgres services |
| `Makefile` | dev, build, migrate, test, lint |

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `.env.example`

- [ ] **Step 1: Initialize module**

```bash
cd /Users/rezwanul-haque/me/workspaces/personal/go-boilerplate
go mod init go-boilerplate
```

Expected: `go.mod` created with `module go-boilerplate` and `go 1.26`

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/labstack/echo/v4@v4.12.0
go get github.com/jackc/pgx/v5@v5.6.0
go get github.com/golang-jwt/jwt/v5@v5.2.1
go get github.com/rs/zerolog@v1.33.0
go get github.com/joho/godotenv@v1.5.1
go get golang.org/x/time@v0.5.0
go get golang.org/x/crypto@v0.24.0
go get github.com/google/uuid@v1.6.0
go get github.com/golang-migrate/migrate/v4@v4.17.1
go get github.com/golang-migrate/migrate/v4/database/postgres@v4.17.1
go get github.com/go-playground/validator/v10@v10.22.0
go get github.com/stretchr/testify@v1.9.0
go mod tidy
```

Expected: `go.sum` created, no errors

- [ ] **Step 3: Create `.env.example`**

```
APP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=go_boilerplate
DB_SSL_MODE=disable
JWT_SECRET=change-me-in-production-min-32-chars
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7
LOG_LEVEL=info
```

- [ ] **Step 4: Create all directories**

```bash
mkdir -p app/bootstrap \
  app/features/users \
  app/infra/database/migrations \
  app/infra/database/users \
  app/infra/logger \
  app/infra/middleware \
  app/infra/notification \
  app/shared/ports \
  app/shared/apperror \
  app/shared/response \
  app/shared/config \
  app/shared/token \
  cmd \
  docker
```

- [ ] **Step 5: Commit**

```bash
git init
git add go.mod go.sum .env.example
git commit -m "chore: initialize go module with dependencies"
```

---

## Task 2: Shared Ports

**Files:**
- Create: `app/shared/ports/db.go`
- Create: `app/shared/ports/logger.go`
- Create: `app/shared/ports/notifier.go`

- [ ] **Step 1: Write `app/shared/ports/db.go`**

```go
package ports

import (
	"context"
	"database/sql"
)

type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
```

- [ ] **Step 2: Write `app/shared/ports/logger.go`**

```go
package ports

type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, err error, fields ...any)
	Debug(msg string, fields ...any)
	Warn(msg string, fields ...any)
}
```

- [ ] **Step 3: Write `app/shared/ports/notifier.go`**

```go
package ports

import "context"

type Notifier interface {
	SendPasswordReset(ctx context.Context, email, token string) error
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./app/shared/ports/...
```

Expected: no output, exit 0

- [ ] **Step 5: Commit**

```bash
git add app/shared/ports/
git commit -m "feat: add shared infrastructure port interfaces"
```

---

## Task 3: Shared AppError

**Files:**
- Create: `app/shared/apperror/apperror.go`
- Create: `app/shared/apperror/apperror_test.go`

- [ ] **Step 1: Write failing test `app/shared/apperror/apperror_test.go`**

```go
package apperror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-boilerplate/app/shared/apperror"
)

func TestAppError_Error_ReturnsMessage(t *testing.T) {
	err := apperror.New(http.StatusBadRequest, "bad request")
	assert.Equal(t, "bad request", err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := apperror.Wrap(http.StatusInternalServerError, "wrapped", inner)
	assert.Equal(t, inner, errors.Unwrap(err))
}

func TestIsAppError_WithAppError(t *testing.T) {
	err := apperror.New(http.StatusNotFound, "not found")
	appErr, ok := apperror.IsAppError(err)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, appErr.Code)
}

func TestIsAppError_WithStdError(t *testing.T) {
	err := errors.New("plain error")
	_, ok := apperror.IsAppError(err)
	assert.False(t, ok)
}

func TestIsAppError_Wrapped(t *testing.T) {
	inner := apperror.New(http.StatusConflict, "conflict")
	wrapped := fmt.Errorf("wrapping: %w", inner)
	appErr, ok := apperror.IsAppError(wrapped)
	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/shared/apperror/... -v
```

Expected: `FAIL` — package does not exist yet

- [ ] **Step 3: Write `app/shared/apperror/apperror.go`**

```go
package apperror

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

var (
	ErrNotFound     = New(http.StatusNotFound, "resource not found")
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized")
	ErrForbidden    = New(http.StatusForbidden, "forbidden")
	ErrBadRequest   = New(http.StatusBadRequest, "bad request")
	ErrInternal     = New(http.StatusInternalServerError, "internal server error")
	ErrConflict     = New(http.StatusConflict, "resource already exists")
)
```

- [ ] **Step 4: Fix test import — add `"fmt"` to test file imports**

Update `app/shared/apperror/apperror_test.go` imports:
```go
import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-boilerplate/app/shared/apperror"
)
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./app/shared/apperror/... -v
```

Expected:
```
--- PASS: TestAppError_Error_ReturnsMessage
--- PASS: TestAppError_Unwrap
--- PASS: TestIsAppError_WithAppError
--- PASS: TestIsAppError_WithStdError
--- PASS: TestIsAppError_Wrapped
PASS
```

- [ ] **Step 6: Commit**

```bash
git add app/shared/apperror/
git commit -m "feat: add apperror typed error with HTTP status mapping"
```

---

## Task 4: Shared Response Helpers

**Files:**
- Create: `app/shared/response/response.go`
- Create: `app/shared/response/response_test.go`

- [ ] **Step 1: Write failing test `app/shared/response/response_test.go`**

```go
package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"
)

func newCtx(method, path string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestOK_Returns200WithData(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	err := response.OK(c, map[string]string{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.True(t, body.Success)
}

func TestCreated_Returns201WithData(t *testing.T) {
	c, rec := newCtx(http.MethodPost, "/")
	err := response.Created(c, map[string]string{"id": "123"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestError_AppError_ReturnsCorrectStatus(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	appErr := apperror.New(http.StatusNotFound, "not found")
	err := response.Error(c, appErr)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var body response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.False(t, body.Success)
	assert.Equal(t, "not found", body.Error)
}

func TestError_UnknownError_Returns500(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	err := response.Error(c, errors.New("some internal error"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/shared/response/... -v
```

Expected: `FAIL` — package does not exist yet

- [ ] **Step 3: Write `app/shared/response/response.go`**

```go
package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/apperror"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func OK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

func Error(c echo.Context, err error) error {
	if appErr, ok := apperror.IsAppError(err); ok {
		return c.JSON(appErr.Code, Response{Success: false, Error: appErr.Message})
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   fmt.Sprintf("validation failed: %s %s", ve[0].Field(), ve[0].Tag()),
		})
	}

	if he, ok := err.(*echo.HTTPError); ok {
		return c.JSON(he.Code, Response{Success: false, Error: fmt.Sprintf("%v", he.Message)})
	}

	return c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "internal server error"})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/shared/response/... -v
```

Expected:
```
--- PASS: TestOK_Returns200WithData
--- PASS: TestCreated_Returns201WithData
--- PASS: TestError_AppError_ReturnsCorrectStatus
--- PASS: TestError_UnknownError_Returns500
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/shared/response/
git commit -m "feat: add unified JSON response helpers"
```

---

## Task 5: Shared Config

**Files:**
- Create: `app/shared/config/config.go`

- [ ] **Step 1: Write `app/shared/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	JWTSecret  string
	AccessTTL  int
	RefreshTTL int
	LogLevel   string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		AppPort:    getEnv("APP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "go_boilerplate"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),
		JWTSecret:  getEnv("JWT_SECRET", "changeme"),
		AccessTTL:  getEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
		RefreshTTL: getEnvInt("JWT_REFRESH_TTL_DAYS", 7),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
	}, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./app/shared/config/...
```

Expected: no output, exit 0

- [ ] **Step 3: Commit**

```bash
git add app/shared/config/
git commit -m "feat: add env-based config with DSN helper"
```

---

## Task 6: Shared JWT Token Maker

**Files:**
- Create: `app/shared/token/jwt.go`
- Create: `app/shared/token/jwt_test.go`

- [ ] **Step 1: Write failing test `app/shared/token/jwt_test.go`**

```go
package token_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-boilerplate/app/shared/token"
)

const testSecret = "supersecretkey1234567890abcdefghij"

func TestCreateAndVerifyToken_Access(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	userID := uuid.New()
	email := "test@example.com"

	tok, err := maker.CreateToken(userID, email, token.AccessToken, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := maker.VerifyToken(tok)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, token.AccessToken, claims.Type)
}

func TestCreateAndVerifyToken_Refresh(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	userID := uuid.New()

	tok, err := maker.CreateToken(userID, "r@example.com", token.RefreshToken, 7*24*time.Hour)
	require.NoError(t, err)

	claims, err := maker.VerifyToken(tok)
	require.NoError(t, err)
	assert.Equal(t, token.RefreshToken, claims.Type)
}

func TestVerifyToken_Expired(t *testing.T) {
	maker := token.NewJWTMaker(testSecret)
	tok, err := maker.CreateToken(uuid.New(), "e@example.com", token.AccessToken, -time.Minute)
	require.NoError(t, err)

	_, err = maker.VerifyToken(tok)
	assert.Error(t, err)
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	maker1 := token.NewJWTMaker(testSecret)
	maker2 := token.NewJWTMaker("differentsecret1234567890abcdefghij")

	tok, err := maker1.CreateToken(uuid.New(), "w@example.com", token.AccessToken, 15*time.Minute)
	require.NoError(t, err)

	_, err = maker2.VerifyToken(tok)
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/shared/token/... -v
```

Expected: `FAIL` — package does not exist yet

- [ ] **Step 3: Write `app/shared/token/jwt.go`**

```go
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type Maker interface {
	CreateToken(userID uuid.UUID, email string, tokenType TokenType, ttl time.Duration) (string, error)
	VerifyToken(tokenStr string) (*Claims, error)
}

type jwtMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) Maker {
	return &jwtMaker{secretKey: secretKey}
}

func (m *jwtMaker) CreateToken(userID uuid.UUID, email string, tokenType TokenType, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(m.secretKey))
}

func (m *jwtMaker) VerifyToken(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.secretKey), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/shared/token/... -v
```

Expected:
```
--- PASS: TestCreateAndVerifyToken_Access
--- PASS: TestCreateAndVerifyToken_Refresh
--- PASS: TestVerifyToken_Expired
--- PASS: TestVerifyToken_WrongSecret
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/shared/token/
git commit -m "feat: add JWT token maker with access/refresh token support"
```

---

## Task 7: Infra Logger

**Files:**
- Create: `app/infra/logger/zerolog.go`

- [ ] **Step 1: Write `app/infra/logger/zerolog.go`**

```go
package logger

import (
	"os"

	"github.com/rs/zerolog"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/ports"
)

type zerologLogger struct {
	log zerolog.Logger
}

func New(cfg *config.Config) ports.Logger {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &zerologLogger{log: log}
}

func (l *zerologLogger) Info(msg string, fields ...any) {
	l.log.Info().Fields(fields).Msg(msg)
}

func (l *zerologLogger) Error(msg string, err error, fields ...any) {
	l.log.Error().Err(err).Fields(fields).Msg(msg)
}

func (l *zerologLogger) Debug(msg string, fields ...any) {
	l.log.Debug().Fields(fields).Msg(msg)
}

func (l *zerologLogger) Warn(msg string, fields ...any) {
	l.log.Warn().Fields(fields).Msg(msg)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./app/infra/logger/...
```

Expected: no output, exit 0

- [ ] **Step 3: Commit**

```bash
git add app/infra/logger/
git commit -m "feat: add zerolog structured logger implementing ports.Logger"
```

---

## Task 8: Infra Database + Migration

**Files:**
- Create: `app/infra/database/postgres.go`
- Create: `app/infra/database/migrations/001_create_users.sql`

- [ ] **Step 1: Write `app/infra/database/postgres.go`**

```go
package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go-boilerplate/app/shared/config"
)

func NewPostgresDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
```

- [ ] **Step 2: Write `app/infra/database/migrations/000001_create_users.up.sql`**

golang-migrate uses two files per version: `.up.sql` and `.down.sql`.

```sql
CREATE TABLE IF NOT EXISTS users (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  VARCHAR(255) UNIQUE NOT NULL,
    password_hash          VARCHAR(255) NOT NULL,
    reset_token            VARCHAR(255),
    reset_token_expires_at TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_reset_token ON users(reset_token);
```

- [ ] **Step 2b: Write `app/infra/database/migrations/000001_create_users.down.sql`**

```sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./app/infra/database/...
```

Expected: no output, exit 0

- [ ] **Step 4: Commit**

```bash
git add app/infra/database/
git commit -m "feat: add postgres connection pool and users migration"
```

---

## Task 9: Users Feature — Model, Errors, Repository Interfaces

**Files:**
- Create: `app/features/users/model.go`
- Create: `app/features/users/errors.go`
- Create: `app/features/users/repository.go`

- [ ] **Step 1: Write `app/features/users/model.go`**

```go
package users

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID  `json:"id"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	ResetToken          *string    `json:"-"`
	ResetTokenExpiresAt *time.Time `json:"-"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
```

- [ ] **Step 2: Write `app/features/users/errors.go`**

```go
package users

import (
	"net/http"

	"go-boilerplate/app/shared/apperror"
)

var (
	ErrEmailAlreadyExists  = apperror.New(http.StatusConflict, "email already exists")
	ErrInvalidCredentials  = apperror.New(http.StatusUnauthorized, "invalid email or password")
	ErrUserNotFound        = apperror.New(http.StatusNotFound, "user not found")
	ErrInvalidResetToken   = apperror.New(http.StatusBadRequest, "invalid or expired reset token")
	ErrInvalidRefreshToken = apperror.New(http.StatusUnauthorized, "invalid refresh token")
	ErrWrongPassword       = apperror.New(http.StatusUnauthorized, "current password is incorrect")
)
```

- [ ] **Step 3: Write `app/features/users/repository.go`**

```go
package users

import (
	"context"
	"time"

	"github.com/google/uuid"
)

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

- [ ] **Step 4: Verify compilation**

```bash
go build ./app/features/users/...
```

Expected: no output, exit 0

- [ ] **Step 5: Commit**

```bash
git add app/features/users/model.go app/features/users/errors.go app/features/users/repository.go
git commit -m "feat: add users domain model, errors, and repository interfaces"
```

---

## Task 10: Users DTOs

**Files:**
- Create: `app/features/users/dto.go`

- [ ] **Step 1: Write `app/features/users/dto.go`**

```go
package users

type SignupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./app/features/users/...
```

Expected: no output, exit 0

- [ ] **Step 3: Commit**

```bash
git add app/features/users/dto.go
git commit -m "feat: add users request/response DTOs with validation tags"
```

---

## Task 11: Users Service (TDD)

**Files:**
- Create: `app/features/users/service.go`
- Create: `app/features/users/service_test.go`

- [ ] **Step 1: Write failing tests `app/features/users/service_test.go`**

```go
package users_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/app/features/users"
	"go-boilerplate/app/shared/token"
)

// --- mock UserRepository ---

type mockUserRepo struct {
	byEmail   map[string]*users.User
	byID      map[uuid.UUID]*users.User
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*users.User),
		byID:    make(map[uuid.UUID]*users.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, u *users.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*users.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*users.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if u, ok := m.byID[id]; ok {
		u.PasswordHash = hash
	}
	return nil
}

// --- mock PasswordResetRepository ---

type mockResetRepo struct {
	tokens  map[string]*users.User
	saveErr error
}

func newMockResetRepo() *mockResetRepo {
	return &mockResetRepo{tokens: make(map[string]*users.User)}
}

func (m *mockResetRepo) SaveResetToken(_ context.Context, id uuid.UUID, tok string, expiresAt time.Time) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	exp := expiresAt
	m.tokens[tok] = &users.User{ID: id, ResetTokenExpiresAt: &exp}
	return nil
}

func (m *mockResetRepo) FindByResetToken(_ context.Context, tok string) (*users.User, error) {
	u, ok := m.tokens[tok]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockResetRepo) ClearResetToken(_ context.Context, id uuid.UUID) error {
	for k, u := range m.tokens {
		if u.ID == id {
			delete(m.tokens, k)
		}
	}
	return nil
}

// --- mock Notifier ---

type mockNotifier struct {
	called bool
	email  string
}

func (m *mockNotifier) SendPasswordReset(_ context.Context, email, _ string) error {
	m.called = true
	m.email = email
	return nil
}

// --- helpers ---

const testJWTSecret = "supersecretkey1234567890abcdefghij"

func newTestService(userRepo *mockUserRepo, resetRepo *mockResetRepo, notifier *mockNotifier) users.Service {
	maker := token.NewJWTMaker(testJWTSecret)
	return users.NewService(userRepo, resetRepo, notifier, maker)
}

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

// --- tests ---

func TestSignup_Success(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	resp, err := svc.Signup(context.Background(), users.SignupRequest{
		Email: "new@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "new@example.com", resp.User.Email)
}

func TestSignup_EmailAlreadyExists(t *testing.T) {
	repo := newMockUserRepo()
	repo.byEmail["exists@example.com"] = &users.User{ID: uuid.New(), Email: "exists@example.com"}
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	_, err := svc.Signup(context.Background(), users.SignupRequest{
		Email: "exists@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, users.ErrEmailAlreadyExists)
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{ID: id, Email: "login@example.com", PasswordHash: hashedPassword(t, "password123")}
	repo.byEmail[u.Email] = u
	repo.byID[id] = u
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	resp, err := svc.Login(context.Background(), users.LoginRequest{
		Email: "login@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{ID: id, Email: "login@example.com", PasswordHash: hashedPassword(t, "password123")}
	repo.byEmail[u.Email] = u
	repo.byID[id] = u
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	_, err := svc.Login(context.Background(), users.LoginRequest{
		Email: "login@example.com", Password: "wrongpassword",
	})

	assert.ErrorIs(t, err, users.ErrInvalidCredentials)
}

func TestLogin_EmailNotFound(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	_, err := svc.Login(context.Background(), users.LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, users.ErrInvalidCredentials)
}

func TestForgotPassword_EmailNotFound_ReturnsNil(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	err := svc.ForgotPassword(context.Background(), users.ForgotPasswordRequest{
		Email: "nobody@example.com",
	})

	assert.NoError(t, err)
}

func TestForgotPassword_Success_CallsNotifier(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{ID: id, Email: "forgot@example.com"}
	repo.byEmail[u.Email] = u
	repo.byID[id] = u
	notifier := &mockNotifier{}
	svc := newTestService(repo, newMockResetRepo(), notifier)

	err := svc.ForgotPassword(context.Background(), users.ForgotPasswordRequest{
		Email: "forgot@example.com",
	})

	require.NoError(t, err)
	assert.True(t, notifier.called)
	assert.Equal(t, "forgot@example.com", notifier.email)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	err := svc.ResetPassword(context.Background(), users.ResetPasswordRequest{
		Token: "badtoken", Password: "newpassword",
	})

	assert.ErrorIs(t, err, users.ErrInvalidResetToken)
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	resetRepo := newMockResetRepo()
	id := uuid.New()
	expired := time.Now().Add(-time.Hour)
	resetRepo.tokens["expiredtoken"] = &users.User{ID: id, ResetTokenExpiresAt: &expired}
	svc := newTestService(newMockUserRepo(), resetRepo, &mockNotifier{})

	err := svc.ResetPassword(context.Background(), users.ResetPasswordRequest{
		Token: "expiredtoken", Password: "newpassword",
	})

	assert.ErrorIs(t, err, users.ErrInvalidResetToken)
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	repo := newMockUserRepo()
	id := uuid.New()
	u := &users.User{ID: id, Email: "change@example.com", PasswordHash: hashedPassword(t, "currentpass")}
	repo.byID[id] = u
	svc := newTestService(repo, newMockResetRepo(), &mockNotifier{})

	err := svc.ChangePassword(context.Background(), id, users.ChangePasswordRequest{
		CurrentPassword: "wrongpass", NewPassword: "newpassword",
	})

	assert.ErrorIs(t, err, users.ErrWrongPassword)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})

	_, err := svc.RefreshToken(context.Background(), users.RefreshTokenRequest{
		RefreshToken: "notavalidtoken",
	})

	assert.ErrorIs(t, err, users.ErrInvalidRefreshToken)
}

func TestRefreshToken_AccessTokenUsedAsRefresh(t *testing.T) {
	maker := token.NewJWTMaker(testJWTSecret)
	accessTok, err := maker.CreateToken(uuid.New(), "r@example.com", token.AccessToken, time.Minute)
	require.NoError(t, err)

	svc := newTestService(newMockUserRepo(), newMockResetRepo(), &mockNotifier{})
	_, err = svc.RefreshToken(context.Background(), users.RefreshTokenRequest{
		RefreshToken: accessTok,
	})

	assert.ErrorIs(t, err, users.ErrInvalidRefreshToken)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/features/users/... -v -run TestSignup
```

Expected: `FAIL` — `NewService` and `Service` not defined yet

- [ ] **Step 3: Write `app/features/users/service.go`**

```go
package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/app/shared/ports"
	"go-boilerplate/app/shared/token"
)

type Service interface {
	Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error)
}

type service struct {
	repo      UserRepository
	resetRepo PasswordResetRepository
	notifier  ports.Notifier
	token     token.Maker
}

func NewService(repo UserRepository, resetRepo PasswordResetRepository, notifier ports.Notifier, tokenMaker token.Maker) Service {
	return &service{
		repo:      repo,
		resetRepo: resetRepo,
		notifier:  notifier,
		token:     tokenMaker,
	}
}

func (s *service) Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error) {
	existing, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildAuthResponse(user)
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.buildAuthResponse(user)
}

func (s *service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	resetToken, err := generateSecureToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Hour)
	if err := s.resetRepo.SaveResetToken(ctx, user.ID, resetToken, expiresAt); err != nil {
		return err
	}

	return s.notifier.SendPasswordReset(ctx, user.Email, resetToken)
}

func (s *service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	user, err := s.resetRepo.FindByResetToken(ctx, req.Token)
	if err != nil {
		return ErrInvalidResetToken
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return ErrInvalidResetToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(ctx, user.ID, string(hash)); err != nil {
		return err
	}

	return s.resetRepo.ClearResetToken(ctx, user.ID)
}

func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return ErrWrongPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

func (s *service) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error) {
	claims, err := s.token.VerifyToken(req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if claims.Type != token.RefreshToken {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return s.buildAuthResponse(user)
}

func (s *service) buildAuthResponse(user *User) (*AuthResponse, error) {
	accessTok, err := s.token.CreateToken(user.ID, user.Email, token.AccessToken, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshTok, err := s.token.CreateToken(user.ID, user.Email, token.RefreshToken, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		User:         UserResponse{ID: user.ID.String(), Email: user.Email},
	}, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/features/users/... -v -run Test
```

Expected:
```
--- PASS: TestSignup_Success
--- PASS: TestSignup_EmailAlreadyExists
--- PASS: TestLogin_Success
--- PASS: TestLogin_WrongPassword
--- PASS: TestLogin_EmailNotFound
--- PASS: TestForgotPassword_EmailNotFound_ReturnsNil
--- PASS: TestForgotPassword_Success_CallsNotifier
--- PASS: TestResetPassword_InvalidToken
--- PASS: TestResetPassword_ExpiredToken
--- PASS: TestChangePassword_WrongCurrentPassword
--- PASS: TestRefreshToken_InvalidToken
--- PASS: TestRefreshToken_AccessTokenUsedAsRefresh
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/features/users/service.go app/features/users/service_test.go
git commit -m "feat: add users service with full auth lifecycle (TDD)"
```

---

## Task 12: Infra pg_repository

**Files:**
- Create: `app/infra/database/users/pg_repository.go`

- [ ] **Step 1: Write `app/infra/database/users/pg_repository.go`**

```go
package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	usersFeature "go-boilerplate/app/features/users"
)

type Repository interface {
	usersFeature.UserRepository
	usersFeature.PasswordResetRepository
}

type pgRepository struct {
	db *sql.DB
}

func NewPgRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, user *usersFeature.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *pgRepository) FindByEmail(ctx context.Context, email string) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, reset_token, reset_token_expires_at, created_at, updated_at
		FROM users WHERE email = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, email))
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, reset_token, reset_token_expires_at, created_at, updated_at
		FROM users WHERE id = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, id))
}

func (r *pgRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	const q = `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, q, hashedPassword, time.Now(), id)
	return err
}

func (r *pgRepository) SaveResetToken(ctx context.Context, id uuid.UUID, tok string, expiresAt time.Time) error {
	const q = `UPDATE users SET reset_token = $1, reset_token_expires_at = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, q, tok, expiresAt, time.Now(), id)
	return err
}

func (r *pgRepository) FindByResetToken(ctx context.Context, tok string) (*usersFeature.User, error) {
	const q = `
		SELECT id, email, password_hash, reset_token, reset_token_expires_at, created_at, updated_at
		FROM users WHERE reset_token = $1`
	return r.scan(r.db.QueryRowContext(ctx, q, tok))
}

func (r *pgRepository) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE users SET reset_token = NULL, reset_token_expires_at = NULL, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, time.Now(), id)
	return err
}

func (r *pgRepository) scan(row *sql.Row) (*usersFeature.User, error) {
	u := &usersFeature.User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash,
		&u.ResetToken, &u.ResetTokenExpiresAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, usersFeature.ErrUserNotFound
	}
	return u, err
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./app/infra/database/users/...
```

Expected: no output, exit 0

- [ ] **Step 3: Commit**

```bash
git add app/infra/database/users/
git commit -m "feat: add postgres implementation of users repositories"
```

---

## Task 13: Infra Mock Notifier

**Files:**
- Create: `app/infra/notification/mock_notifier.go`

- [ ] **Step 1: Write `app/infra/notification/mock_notifier.go`**

```go
package notification

import (
	"context"
	"fmt"

	"go-boilerplate/app/shared/ports"
)

type MockNotifier struct{}

func NewMockNotifier() ports.Notifier {
	return &MockNotifier{}
}

func (m *MockNotifier) SendPasswordReset(_ context.Context, email, tok string) error {
	fmt.Printf("[MockNotifier] password reset token for %s: %s\n", email, tok)
	return nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./app/infra/notification/...
```

Expected: no output, exit 0

- [ ] **Step 3: Commit**

```bash
git add app/infra/notification/
git commit -m "feat: add mock notifier that logs reset tokens to stdout"
```

---

## Task 14: Infra Middleware — Auth (TDD)

**Files:**
- Create: `app/infra/middleware/auth.go`
- Create: `app/infra/middleware/auth_test.go`

- [ ] **Step 1: Write failing test `app/infra/middleware/auth_test.go`**

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"
)

const authTestSecret = "supersecretkey1234567890abcdefghij"

func newEchoCtx(method, path, authHeader string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestAuthMiddleware_NoHeader_Returns401(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	c, rec := newEchoCtx(http.MethodGet, "/", "")

	handler := middleware.Auth(maker)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	c, rec := newEchoCtx(http.MethodGet, "/", "Bearer notavalidtoken")

	handler := middleware.Auth(maker)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidToken_PassesThrough(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	tok, err := maker.CreateToken(uuid.New(), "auth@example.com", token.AccessToken, time.Minute)
	require.NoError(t, err)

	c, rec := newEchoCtx(http.MethodGet, "/", "Bearer "+tok)

	handler := middleware.Auth(maker)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_ValidToken_SetsClaims(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	userID := uuid.New()
	tok, err := maker.CreateToken(userID, "claims@example.com", token.AccessToken, time.Minute)
	require.NoError(t, err)

	c, _ := newEchoCtx(http.MethodGet, "/", "Bearer "+tok)

	var capturedClaims *token.Claims
	handler := middleware.Auth(maker)(func(c echo.Context) error {
		capturedClaims = c.Get("claims").(*token.Claims)
		return c.String(http.StatusOK, "ok")
	})

	require.NoError(t, handler(c))
	require.NotNil(t, capturedClaims)
	assert.Equal(t, userID, capturedClaims.UserID)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/infra/middleware/... -v -run TestAuth
```

Expected: `FAIL` — `middleware.Auth` not defined yet

- [ ] **Step 3: Write `app/infra/middleware/auth.go`**

```go
package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

func Auth(maker token.Maker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Error(c, apperror.ErrUnauthorized)
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return response.Error(c, apperror.ErrUnauthorized)
			}

			claims, err := maker.VerifyToken(parts[1])
			if err != nil {
				return response.Error(c, apperror.ErrUnauthorized)
			}

			c.Set("claims", claims)
			return next(c)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/infra/middleware/... -v -run TestAuth
```

Expected:
```
--- PASS: TestAuthMiddleware_NoHeader_Returns401
--- PASS: TestAuthMiddleware_InvalidToken_Returns401
--- PASS: TestAuthMiddleware_ValidToken_PassesThrough
--- PASS: TestAuthMiddleware_ValidToken_SetsClaims
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/infra/middleware/auth.go app/infra/middleware/auth_test.go
git commit -m "feat: add JWT auth middleware (TDD)"
```

---

## Task 15: Infra Middleware — Rate Limiter (TDD)

**Files:**
- Create: `app/infra/middleware/rate_limiter.go`
- Create: `app/infra/middleware/rate_limiter_test.go`

- [ ] **Step 1: Write failing test `app/infra/middleware/rate_limiter_test.go`**

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"go-boilerplate/app/infra/middleware"
)

func TestRateLimit_UnderLimit_Passes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/signup", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// burst of 5 — first request must pass
	handler := middleware.RateLimit(rate.Limit(5.0/60.0), 5)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_OverLimit_Returns429(t *testing.T) {
	e := echo.New()
	// limit=0 means no tokens ever available
	rateLimitMiddleware := middleware.RateLimit(rate.Limit(0), 0)

	handler := rateLimitMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/signup", nil)
	req.RemoteAddr = "2.3.4.5:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler(c)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/infra/middleware/... -v -run TestRateLimit
```

Expected: `FAIL` — `middleware.RateLimit` not defined yet

- [ ] **Step 3: Write `app/infra/middleware/rate_limiter.go`**

```go
package middleware

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type ipLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPLimiterStore(r rate.Limit, b int) *ipLimiterStore {
	return &ipLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (s *ipLimiterStore) get(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if l, ok := s.limiters[ip]; ok {
		return l
	}
	l := rate.NewLimiter(s.r, s.b)
	s.limiters[ip] = l
	return l
}

// RateLimit returns a per-IP token bucket middleware.
// r is requests per second; b is the burst size.
// For 5 req/min use: RateLimit(rate.Limit(5.0/60.0), 5)
func RateLimit(r rate.Limit, b int) echo.MiddlewareFunc {
	store := newIPLimiterStore(r, b)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !store.get(ip).Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "too many requests, please try again later",
				})
			}
			return next(c)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/infra/middleware/... -v -run TestRateLimit
```

Expected:
```
--- PASS: TestRateLimit_UnderLimit_Passes
--- PASS: TestRateLimit_OverLimit_Returns429
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/infra/middleware/rate_limiter.go app/infra/middleware/rate_limiter_test.go
git commit -m "feat: add IP-based rate limiter middleware (TDD)"
```

---

## Task 16: Infra Middleware — Request Logger

**Files:**
- Create: `app/infra/middleware/request_logger.go`

- [ ] **Step 1: Write `app/infra/middleware/request_logger.go`**

```go
package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/ports"
)

func RequestLogger(logger ports.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			logger.Info("request",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"duration", time.Since(start).String(),
				"ip", c.RealIP(),
			)
			return err
		}
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./app/infra/middleware/...
```

Expected: no output, exit 0

- [ ] **Step 3: Commit**

```bash
git add app/infra/middleware/request_logger.go
git commit -m "feat: add structured request logger middleware"
```

---

## Task 17: Users Handler (TDD)

**Files:**
- Create: `app/features/users/handler.go`
- Create: `app/features/users/handler_test.go`

- [ ] **Step 1: Write failing test `app/features/users/handler_test.go`**

```go
package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/features/users"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

// --- mock Service ---

type mockService struct {
	signupResp      *users.AuthResponse
	signupErr       error
	loginResp       *users.AuthResponse
	loginErr        error
	forgotErr       error
	resetErr        error
	changeErr       error
	refreshResp     *users.AuthResponse
	refreshErr      error
}

func (m *mockService) Signup(_ context.Context, _ users.SignupRequest) (*users.AuthResponse, error) {
	return m.signupResp, m.signupErr
}
func (m *mockService) Login(_ context.Context, _ users.LoginRequest) (*users.AuthResponse, error) {
	return m.loginResp, m.loginErr
}
func (m *mockService) ForgotPassword(_ context.Context, _ users.ForgotPasswordRequest) error {
	return m.forgotErr
}
func (m *mockService) ResetPassword(_ context.Context, _ users.ResetPasswordRequest) error {
	return m.resetErr
}
func (m *mockService) ChangePassword(_ context.Context, _ uuid.UUID, _ users.ChangePasswordRequest) error {
	return m.changeErr
}
func (m *mockService) RefreshToken(_ context.Context, _ users.RefreshTokenRequest) (*users.AuthResponse, error) {
	return m.refreshResp, m.refreshErr
}

// --- test setup ---

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	return e
}

type testValidator struct{ v *validator.Validate }

func (tv *testValidator) Validate(i interface{}) error { return tv.v.Struct(i) }

func postJSON(e *echo.Echo, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// --- tests ---

func TestSignupHandler_Success_Returns201(t *testing.T) {
	svc := &mockService{
		signupResp: &users.AuthResponse{
			AccessToken:  "access",
			RefreshToken: "refresh",
			User:         users.UserResponse{ID: uuid.New().String(), Email: "new@example.com"},
		},
	}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/signup", map[string]string{"email": "new@example.com", "password": "password123"})

	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.True(t, resp.Success)
}

func TestSignupHandler_EmailExists_Returns409(t *testing.T) {
	svc := &mockService{signupErr: users.ErrEmailAlreadyExists}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/signup", map[string]string{"email": "dup@example.com", "password": "password123"})

	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSignupHandler_InvalidBody_Returns400(t *testing.T) {
	svc := &mockService{}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/signup", map[string]string{"email": "notanemail", "password": "short"})

	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginHandler_Success_Returns200(t *testing.T) {
	svc := &mockService{
		loginResp: &users.AuthResponse{AccessToken: "tok", RefreshToken: "ref"},
	}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/login", map[string]string{"email": "user@example.com", "password": "password123"})

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLoginHandler_InvalidCredentials_Returns401(t *testing.T) {
	svc := &mockService{loginErr: users.ErrInvalidCredentials}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/login", map[string]string{"email": "user@example.com", "password": "wrongpass"})

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestForgotPasswordHandler_Returns200(t *testing.T) {
	svc := &mockService{}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/forgot-password", map[string]string{"email": "user@example.com"})

	require.NoError(t, h.ForgotPassword(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestChangePasswordHandler_NoClaims_Returns500(t *testing.T) {
	svc := &mockService{changeErr: errors.New("should not reach")}
	h := users.NewHandler(svc)
	e := newTestEcho()

	b, _ := json.Marshal(map[string]string{"current_password": "old", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPut, "/users/change-password", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no claims set in context → should return 401
	require.NoError(t, h.ChangePassword(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./app/features/users/... -v -run TestSignupHandler
```

Expected: `FAIL` — `users.NewHandler` not defined yet

- [ ] **Step 3: Write `app/features/users/handler.go`**

```go
package users

import (
	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	resp, err := h.svc.Signup(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, resp)
}

func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	resp, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

func (h *Handler) ForgotPassword(c echo.Context) error {
	var req ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if err := h.svc.ForgotPassword(c.Request().Context(), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

func (h *Handler) ResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if err := h.svc.ResetPassword(c.Request().Context(), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "password reset successful"})
}

func (h *Handler) ChangePassword(c echo.Context) error {
	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	claims, ok := c.Get("claims").(*token.Claims)
	if !ok {
		return response.Error(c, apperror.ErrUnauthorized)
	}

	if err := h.svc.ChangePassword(c.Request().Context(), claims.UserID, req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "password changed successfully"})
}

func (h *Handler) RefreshToken(c echo.Context) error {
	var req RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	resp, err := h.svc.RefreshToken(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}
```

- [ ] **Step 4: Run all handler tests to verify they pass**

```bash
go test ./app/features/users/... -v -run TestSignupHandler -run TestLoginHandler -run TestForgotPassword -run TestChangePassword
```

Expected:
```
--- PASS: TestSignupHandler_Success_Returns201
--- PASS: TestSignupHandler_EmailExists_Returns409
--- PASS: TestSignupHandler_InvalidBody_Returns400
--- PASS: TestLoginHandler_Success_Returns200
--- PASS: TestLoginHandler_InvalidCredentials_Returns401
--- PASS: TestForgotPasswordHandler_Returns200
--- PASS: TestChangePasswordHandler_NoClaims_Returns500
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/features/users/handler.go app/features/users/handler_test.go
git commit -m "feat: add users HTTP handler (TDD)"
```

---

## Task 18: Users Routes + Bootstrap

**Files:**
- Create: `app/features/users/routes.go`
- Create: `app/bootstrap/app.go`
- Create: `app/bootstrap/routes.go`

- [ ] **Step 1: Write `app/features/users/routes.go`**

```go
package users

import (
	"github.com/labstack/echo/v4"
	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"
)

func RegisterRoutes(g *echo.Group, h *Handler, tokenMaker token.Maker, signupLimiter echo.MiddlewareFunc) {
	g.POST("/signup", h.Signup, signupLimiter)
	g.POST("/login", h.Login)
	g.POST("/forgot-password", h.ForgotPassword)
	g.POST("/reset-password", h.ResetPassword)
	g.POST("/refresh-token", h.RefreshToken)
	g.PUT("/change-password", h.ChangePassword, middleware.Auth(tokenMaker))
}
```

- [ ] **Step 2: Write `app/bootstrap/app.go`**

```go
package bootstrap

import (
	"github.com/go-playground/validator/v10"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/echo/v4"
	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/ports"
	"go-boilerplate/app/shared/token"
)

type customValidator struct {
	v *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.v.Struct(i)
}

func NewEcho(logger ports.Logger, tokenMaker token.Maker) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = &customValidator{v: validator.New()}

	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORS())
	e.Use(middleware.RequestLogger(logger))

	return e
}
```

- [ ] **Step 3: Write `app/bootstrap/routes.go`**

```go
package bootstrap

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"
)

func RegisterRoutes(e *echo.Echo, usersHandler *usersFeature.Handler, tokenMaker token.Maker) {
	// 5 requests per minute per IP, burst of 5
	signupLimiter := middleware.RateLimit(rate.Limit(5.0/60.0), 5)

	v1 := e.Group("/api/v1")
	usersGroup := v1.Group("/users")
	usersFeature.RegisterRoutes(usersGroup, usersHandler, tokenMaker, signupLimiter)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./app/bootstrap/... ./app/features/users/...
```

Expected: no output, exit 0

- [ ] **Step 5: Commit**

```bash
git add app/features/users/routes.go app/bootstrap/
git commit -m "feat: add users routes and Echo bootstrap"
```

---

## Task 19: Wiring + Docker + Makefile

**Files:**
- Create: `cmd/main.go`
- Create: `docker/Dockerfile`
- Create: `docker-compose.yml`
- Create: `Makefile`

- [ ] **Step 1: Write `cmd/main.go`**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"go-boilerplate/app/bootstrap"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/database"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/logger"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/token"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg)

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("failed to connect to database", err)
		os.Exit(1)
	}
	defer db.Close()

	runMigrations(db)

	tokenMaker := token.NewJWTMaker(cfg.JWTSecret)
	notifier := notification.NewMockNotifier()

	usersRepo := dbUsers.NewPgRepository(db)
	usersSvc := usersFeature.NewService(usersRepo, usersRepo, notifier, tokenMaker)
	usersHandler := usersFeature.NewHandler(usersSvc)

	e := bootstrap.NewEcho(log, tokenMaker)
	bootstrap.RegisterRoutes(e, usersHandler, tokenMaker)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.AppPort)
		log.Info("server starting", "addr", addr)
		if err := e.Start(addr); err != nil {
			log.Info("server stopped")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", err)
	}
}

func runMigrations(db *sql.DB) {
	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate driver error: %v\n", err)
		return
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://app/infra/database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate init error: %v\n", err)
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "migrate up error: %v\n", err)
	}
}
```

- [ ] **Step 2: Write `docker/Dockerfile`**

```dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/main.go

FROM alpine:3.19

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/app/infra/database/migrations ./app/infra/database/migrations

USER appuser

EXPOSE 8080

CMD ["./server"]
```

- [ ] **Step 3: Write `docker-compose.yml`**

```yaml
version: '3.8'

services:
  api:
    build:
      context: .
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
    environment:
      APP_PORT: "8080"
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: go_boilerplate
      DB_SSL_MODE: disable
      JWT_SECRET: change-me-in-production-min-32-chars
      JWT_ACCESS_TTL_MINUTES: "15"
      JWT_REFRESH_TTL_DAYS: "7"
      LOG_LEVEL: info
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: go_boilerplate
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

- [ ] **Step 4: Write `Makefile`**

```makefile
.PHONY: dev build migrate-up migrate-down test lint tidy

DB_URL=postgres://postgres:postgres@localhost:5432/go_boilerplate?sslmode=disable

dev:
	docker-compose up --build

dev-down:
	docker-compose down

build:
	go build -o bin/server ./cmd/main.go

migrate-up:
	migrate -path app/infra/database/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path app/infra/database/migrations -database "$(DB_URL)" down

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
```

- [ ] **Step 5: Verify full project builds**

```bash
go build ./...
```

Expected: no output, exit 0

- [ ] **Step 6: Run all tests**

```bash
go test ./... -v -count=1
```

Expected: all tests pass, no failures

- [ ] **Step 7: Commit**

```bash
git add cmd/main.go docker/Dockerfile docker-compose.yml Makefile
git commit -m "feat: wire all dependencies, add Docker and Makefile"
```

---

## Task 20: Integration Tests — pg_repository

**Files:**
- Create: `app/infra/database/users/pg_repository_integration_test.go`

These tests run against a real Postgres instance. Guarded by `//go:build integration`.  
Requires `docker-compose up -d postgres` before running.

- [ ] **Step 1: Write `app/infra/database/users/pg_repository_integration_test.go`**

```go
//go:build integration

package users_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	usersFeature "go-boilerplate/app/features/users"
	dbUsers "go-boilerplate/app/infra/database/users"
)

func integrationDSN() string {
	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	return "host=localhost port=5432 user=postgres password=postgres dbname=go_boilerplate sslmode=disable"
}

type PgRepositorySuite struct {
	suite.Suite
	db   *sql.DB
	repo dbUsers.Repository
}

func (s *PgRepositorySuite) SetupSuite() {
	db, err := sql.Open("pgx", integrationDSN())
	require.NoError(s.T(), err)
	require.NoError(s.T(), db.PingContext(context.Background()))
	s.db = db
	s.repo = dbUsers.NewPgRepository(db)
}

func (s *PgRepositorySuite) TearDownSuite() {
	s.db.Close()
}

func (s *PgRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(context.Background(), "DELETE FROM users")
	require.NoError(s.T(), err)
}

func (s *PgRepositorySuite) TestCreate_AndFindByEmail() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "create@example.com",
		PasswordHash: "hashedpassword",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := s.repo.Create(ctx, user)
	require.NoError(s.T(), err)

	found, err := s.repo.FindByEmail(ctx, "create@example.com")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "create@example.com", found.Email)
	assert.Equal(s.T(), "hashedpassword", found.PasswordHash)
}

func (s *PgRepositorySuite) TestCreate_DuplicateEmail_ReturnsError() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "dup@example.com",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(s.T(), s.repo.Create(ctx, user))
	err := s.repo.Create(ctx, &usersFeature.User{
		Email:        "dup@example.com",
		PasswordHash: "hash2",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	assert.Error(s.T(), err)
}

func (s *PgRepositorySuite) TestFindByEmail_NotFound() {
	_, err := s.repo.FindByEmail(context.Background(), "nobody@example.com")
	assert.ErrorIs(s.T(), err, usersFeature.ErrUserNotFound)
}

func (s *PgRepositorySuite) TestFindByID_NotFound() {
	_, err := s.repo.FindByID(context.Background(), [16]byte{})
	assert.ErrorIs(s.T(), err, usersFeature.ErrUserNotFound)
}

func (s *PgRepositorySuite) TestUpdatePassword() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "update@example.com",
		PasswordHash: "oldhash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(s.T(), s.repo.Create(ctx, user))

	found, err := s.repo.FindByEmail(ctx, "update@example.com")
	require.NoError(s.T(), err)

	require.NoError(s.T(), s.repo.UpdatePassword(ctx, found.ID, "newhash"))

	updated, err := s.repo.FindByID(ctx, found.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "newhash", updated.PasswordHash)
}

func (s *PgRepositorySuite) TestResetToken_SaveFindClear() {
	ctx := context.Background()
	user := &usersFeature.User{
		Email:        "reset@example.com",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(s.T(), s.repo.Create(ctx, user))

	found, err := s.repo.FindByEmail(ctx, "reset@example.com")
	require.NoError(s.T(), err)

	expiresAt := time.Now().Add(time.Hour)
	require.NoError(s.T(), s.repo.SaveResetToken(ctx, found.ID, "myresettoken", expiresAt))

	byToken, err := s.repo.FindByResetToken(ctx, "myresettoken")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), found.ID, byToken.ID)
	assert.NotNil(s.T(), byToken.ResetTokenExpiresAt)

	require.NoError(s.T(), s.repo.ClearResetToken(ctx, found.ID))

	_, err = s.repo.FindByResetToken(ctx, "myresettoken")
	assert.ErrorIs(s.T(), err, usersFeature.ErrUserNotFound)
}

func TestPgRepositorySuite(t *testing.T) {
	suite.Run(t, new(PgRepositorySuite))
}
```

- [ ] **Step 2: Run integration tests (requires Postgres running)**

```bash
docker-compose up -d postgres
sleep 3
go test ./app/infra/database/users/... -v -tags integration
```

Expected:
```
--- PASS: TestPgRepositorySuite/TestCreate_AndFindByEmail
--- PASS: TestPgRepositorySuite/TestCreate_DuplicateEmail_ReturnsError
--- PASS: TestPgRepositorySuite/TestFindByEmail_NotFound
--- PASS: TestPgRepositorySuite/TestFindByID_NotFound
--- PASS: TestPgRepositorySuite/TestUpdatePassword
--- PASS: TestPgRepositorySuite/TestResetToken_SaveFindClear
PASS
```

- [ ] **Step 3: Add integration test target to Makefile**

Update `Makefile` — replace the existing file with:

```makefile
.PHONY: dev dev-down build migrate-up migrate-down test test-unit test-integration lint tidy

DB_URL=postgres://postgres:postgres@localhost:5432/go_boilerplate?sslmode=disable

dev:
	docker-compose up --build

dev-down:
	docker-compose down

build:
	go build -o bin/server ./cmd/main.go

migrate-up:
	migrate -path app/infra/database/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path app/infra/database/migrations -database "$(DB_URL)" down

test: test-unit

test-unit:
	go test ./... -v -count=1

test-integration:
	docker-compose up -d postgres
	sleep 3
	go test ./... -v -count=1 -tags integration

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
```

- [ ] **Step 4: Commit**

```bash
git add app/infra/database/users/pg_repository_integration_test.go Makefile
git commit -m "test: add pg_repository integration tests and update Makefile targets"
```

---

## Task 21: Integration Tests — HTTP Round-Trip

**Files:**
- Create: `app/features/users/handler_integration_test.go`

Full HTTP stack test: real Echo + real service + real DB. Guarded by `//go:build integration`.

- [ ] **Step 1: Write `app/features/users/handler_integration_test.go`**

```go
//go:build integration

package users_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/features/users"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=go_boilerplate sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, db.PingContext(context.Background()))
	t.Cleanup(func() {
		db.ExecContext(context.Background(), "DELETE FROM users")
		db.Close()
	})
	return db
}

type integrationValidator struct{ v *validator.Validate }

func (iv *integrationValidator) Validate(i interface{}) error { return iv.v.Struct(i) }

func newIntegrationStack(t *testing.T) (*echo.Echo, *users.Handler) {
	t.Helper()
	db := integrationDB(t)

	repo := dbUsers.NewPgRepository(db)
	maker := token.NewJWTMaker("supersecretkey1234567890abcdefghij")
	svc := users.NewService(repo, repo, notification.NewMockNotifier(), maker)
	h := users.NewHandler(svc)

	e := echo.New()
	e.Validator = &integrationValidator{v: validator.New()}
	return e, h
}

func postJSONIntegration(e *echo.Echo, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestIntegration_Signup_Login_ChangePassword(t *testing.T) {
	e, h := newIntegrationStack(t)

	// 1. Signup
	c, rec := postJSONIntegration(e, "/signup", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	require.NoError(t, h.Signup(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	var signupResp response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&signupResp))
	assert.True(t, signupResp.Success)

	// extract tokens from signup
	dataBytes, _ := json.Marshal(signupResp.Data)
	var authResp users.AuthResponse
	require.NoError(t, json.Unmarshal(dataBytes, &authResp))
	assert.NotEmpty(t, authResp.AccessToken)
	assert.NotEmpty(t, authResp.RefreshToken)

	// 2. Login with same credentials
	c, rec = postJSONIntegration(e, "/login", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// 3. Login with wrong password → 401
	c, rec = postJSONIntegration(e, "/login", map[string]string{
		"email": "flow@example.com", "password": "wrongpassword",
	})
	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 4. Duplicate signup → 409
	c, rec = postJSONIntegration(e, "/signup", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusConflict, rec.Code)

	// 5. Refresh token
	c, rec = postJSONIntegration(e, "/refresh-token", map[string]string{
		"refresh_token": authResp.RefreshToken,
	})
	require.NoError(t, h.RefreshToken(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIntegration_ForgotPassword_ResetPassword(t *testing.T) {
	e, h := newIntegrationStack(t)
	ctx := context.Background()
	db := integrationDB(t)
	repo := dbUsers.NewPgRepository(db)

	// Signup first
	c, rec := postJSONIntegration(e, "/signup", map[string]string{
		"email": "forgot@example.com", "password": "password123",
	})
	require.NoError(t, h.Signup(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	// ForgotPassword
	c, rec = postJSONIntegration(e, "/forgot-password", map[string]string{
		"email": "forgot@example.com",
	})
	require.NoError(t, h.ForgotPassword(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Fetch reset token directly from DB
	u, err := repo.FindByEmail(ctx, "forgot@example.com")
	require.NoError(t, err)
	require.NotNil(t, u.ResetToken)

	// ResetPassword with valid token
	c, rec = postJSONIntegration(e, "/reset-password", map[string]string{
		"token": *u.ResetToken, "password": "newpassword123",
	})
	require.NoError(t, h.ResetPassword(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Login with new password
	c, rec = postJSONIntegration(e, "/login", map[string]string{
		"email": "forgot@example.com", "password": "newpassword123",
	})
	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}
```

- [ ] **Step 2: Run integration tests**

```bash
go test ./app/features/users/... -v -count=1 -tags integration -run TestIntegration
```

Expected:
```
--- PASS: TestIntegration_Signup_Login_ChangePassword
--- PASS: TestIntegration_ForgotPassword_ResetPassword
PASS
```

- [ ] **Step 3: Commit**

```bash
git add app/features/users/handler_integration_test.go
git commit -m "test: add HTTP round-trip integration tests for users feature"
```

---

## Final Smoke Test

- [ ] Copy `.env.example` to `.env` and run `make dev`

```bash
cp .env.example .env
make dev
```

Expected: postgres starts healthy, api connects, migrations run, server listens on `:8080`

- [ ] **Test signup endpoint**

```bash
curl -X POST http://localhost:8080/api/v1/users/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

Expected:
```json
{"success":true,"data":{"access_token":"...","refresh_token":"...","user":{"id":"...","email":"test@example.com"}}}
```

- [ ] **Test rate limiting — send 6 signup requests rapidly**

```bash
for i in {1..6}; do
  curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/api/v1/users/signup \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"ratelimit${i}@example.com\",\"password\":\"password123\"}"
done
```

Expected: first 5 return `201` or `409`, 6th returns `429`
