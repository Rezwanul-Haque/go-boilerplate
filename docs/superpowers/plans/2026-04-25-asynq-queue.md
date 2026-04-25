# Asynq Queue System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add async task queue using asynq backed by existing Redis, with email and example task types, wired into the DI container, and Asynqmon dashboard in docker-compose.

**Architecture:** Worker runs embedded in same process as HTTP server, started as a goroutine alongside Echo. Queue infra lives in `app/infra/queue/` following existing infra layer pattern. `ports.QueueClient` interface decouples services from asynq internals.

**Tech Stack:** `github.com/hibiken/asynq v0.26.0` (already in go.mod as indirect), Redis (existing `redis:7-alpine`), `hibiken/asynqmon` Docker image for dashboard.

---

## File Map

| Action | File | Responsibility |
|--------|------|---------------|
| Modify | `app/shared/config/config.go` | Add `QueueConcurrency` env var |
| Create | `app/shared/ports/queue.go` | `QueueClient` interface |
| Create | `app/infra/queue/tasks/email.go` | `TypeSendEmail` constant + `EmailPayload` struct |
| Create | `app/infra/queue/tasks/example.go` | `TypeExampleTask` constant + `ExamplePayload` struct |
| Create | `app/infra/queue/handlers/email.go` | `EmailHandler.Process` — delegates to `ports.Notifier` |
| Create | `app/infra/queue/handlers/email_test.go` | Unit tests for email handler |
| Create | `app/infra/queue/handlers/example.go` | `ProcessExampleTask` — logs payload |
| Create | `app/infra/queue/handlers/example_test.go` | Unit tests for example handler |
| Create | `app/infra/queue/client.go` | `Client` struct implementing `ports.QueueClient` |
| Create | `app/infra/queue/server.go` | `Server` struct wrapping `asynq.Server` |
| Modify | `app/bootstrap/container.go` | Add `QueueClient` field + init |
| Modify | `cmd/main.go` | Start/stop queue server alongside HTTP server |
| Modify | `docker-compose.yml` | Add `asynqmon` service on port 8081 |

---

### Task 1: Add `QueueConcurrency` to config

**Files:**
- Modify: `app/shared/config/config.go`

- [ ] **Step 1: Add field to `Config` struct**

In `app/shared/config/config.go`, add `QueueConcurrency int` to the struct after `HTTPClientTimeout`:

```go
type Config struct {
	AppPort       string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	JWTSecret     string
	AccessTTL     int
	RefreshTTL    int
	LogLevel          string
	RunMigrations     bool
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	HTTPClientTimeout int
	QueueConcurrency  int
}
```

- [ ] **Step 2: Load from env in `Load()`**

Add to the return struct inside `Load()` after `HTTPClientTimeout`:

```go
QueueConcurrency: getEnvInt("QUEUE_CONCURRENCY", 10),
```

- [ ] **Step 3: Verify compiles**

```bash
go build ./app/shared/config/...
```

Expected: no output (success)

- [ ] **Step 4: Commit**

```bash
git add app/shared/config/config.go
git commit -m "feat(config): add QUEUE_CONCURRENCY env var"
```

---

### Task 2: Add `QueueClient` port interface

**Files:**
- Create: `app/shared/ports/queue.go`

- [ ] **Step 1: Create port file**

Create `app/shared/ports/queue.go`:

```go
package ports

import "context"

type QueueClient interface {
	EnqueueSendEmail(ctx context.Context, to, subject, body string) error
	EnqueueExampleTask(ctx context.Context, userID, message string) error
	Close() error
}
```

- [ ] **Step 2: Verify compiles**

```bash
go build ./app/shared/ports/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add app/shared/ports/queue.go
git commit -m "feat(ports): add QueueClient interface"
```

---

### Task 3: Define task type constants and payload structs

**Files:**
- Create: `app/infra/queue/tasks/email.go`
- Create: `app/infra/queue/tasks/example.go`

- [ ] **Step 1: Create email task definition**

Create `app/infra/queue/tasks/email.go`:

```go
package tasks

const TypeSendEmail = "email:send"

type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}
```

- [ ] **Step 2: Create example task definition**

Create `app/infra/queue/tasks/example.go`:

```go
package tasks

const TypeExampleTask = "example:task"

type ExamplePayload struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}
```

- [ ] **Step 3: Verify compiles**

```bash
go build ./app/infra/queue/tasks/...
```

Expected: no output

- [ ] **Step 4: Commit**

```bash
git add app/infra/queue/tasks/
git commit -m "feat(queue): add task type constants and payload structs"
```

---

### Task 4: Implement email handler (TDD)

**Files:**
- Create: `app/infra/queue/handlers/email_test.go`
- Create: `app/infra/queue/handlers/email.go`

- [ ] **Step 1: Write failing test**

Create `app/infra/queue/handlers/email_test.go`:

```go
package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/infra/queue/handlers"
	"go-boilerplate/app/infra/queue/tasks"
)

type mockNotifier struct {
	capturedEmail string
	capturedToken string
}

func (m *mockNotifier) SendPasswordReset(_ context.Context, email, token string) error {
	m.capturedEmail = email
	m.capturedToken = token
	return nil
}

func TestProcessEmailTask(t *testing.T) {
	payload := tasks.EmailPayload{
		To:      "user@example.com",
		Subject: "Test Subject",
		Body:    "Test body content",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(tasks.TypeSendEmail, data)
	notifier := &mockNotifier{}
	h := handlers.NewEmailHandler(notifier)

	err = h.Process(context.Background(), task)

	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", notifier.capturedEmail)
	assert.Equal(t, "Test body content", notifier.capturedToken)
}

func TestProcessEmailTask_InvalidPayload(t *testing.T) {
	task := asynq.NewTask(tasks.TypeSendEmail, []byte("not-json"))
	notifier := &mockNotifier{}
	h := handlers.NewEmailHandler(notifier)

	err := h.Process(context.Background(), task)

	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test — expect failure**

```bash
go test ./app/infra/queue/handlers/... -v -run TestProcessEmailTask
```

Expected: FAIL — `handlers` package does not exist yet

- [ ] **Step 3: Implement email handler**

Create `app/infra/queue/handlers/email.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"go-boilerplate/app/infra/queue/tasks"
	"go-boilerplate/app/shared/ports"
)

type EmailHandler struct {
	notifier ports.Notifier
}

func NewEmailHandler(notifier ports.Notifier) *EmailHandler {
	return &EmailHandler{notifier: notifier}
}

func (h *EmailHandler) Process(ctx context.Context, t *asynq.Task) error {
	var p tasks.EmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal email payload: %w", err)
	}
	return h.notifier.SendPasswordReset(ctx, p.To, p.Body)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./app/infra/queue/handlers/... -v -run TestProcessEmailTask
```

Expected:
```
--- PASS: TestProcessEmailTask (0.00s)
--- PASS: TestProcessEmailTask_InvalidPayload (0.00s)
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/infra/queue/handlers/email.go app/infra/queue/handlers/email_test.go
git commit -m "feat(queue): add email task handler"
```

---

### Task 5: Implement example handler (TDD)

**Files:**
- Create: `app/infra/queue/handlers/example_test.go`
- Create: `app/infra/queue/handlers/example.go`

- [ ] **Step 1: Write failing test**

Create `app/infra/queue/handlers/example_test.go`:

```go
package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/infra/queue/handlers"
	"go-boilerplate/app/infra/queue/tasks"
)

func TestProcessExampleTask(t *testing.T) {
	payload := tasks.ExamplePayload{
		UserID:  "user-123",
		Message: "hello from queue",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(tasks.TypeExampleTask, data)

	err = handlers.ProcessExampleTask(context.Background(), task)

	assert.NoError(t, err)
}

func TestProcessExampleTask_InvalidPayload(t *testing.T) {
	task := asynq.NewTask(tasks.TypeExampleTask, []byte("not-json"))

	err := handlers.ProcessExampleTask(context.Background(), task)

	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test — expect failure**

```bash
go test ./app/infra/queue/handlers/... -v -run TestProcessExampleTask
```

Expected: FAIL — `handlers.ProcessExampleTask` not defined

- [ ] **Step 3: Implement example handler**

Create `app/infra/queue/handlers/example.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"go-boilerplate/app/infra/queue/tasks"
)

func ProcessExampleTask(ctx context.Context, t *asynq.Task) error {
	var p tasks.ExamplePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal example payload: %w", err)
	}
	log.Ctx(ctx).Info().Str("user_id", p.UserID).Str("message", p.Message).Msg("processed example task")
	return nil
}
```

- [ ] **Step 4: Run all handler tests — expect pass**

```bash
go test ./app/infra/queue/handlers/... -v
```

Expected:
```
--- PASS: TestProcessEmailTask (0.00s)
--- PASS: TestProcessEmailTask_InvalidPayload (0.00s)
--- PASS: TestProcessExampleTask (0.00s)
--- PASS: TestProcessExampleTask_InvalidPayload (0.00s)
PASS
```

- [ ] **Step 5: Commit**

```bash
git add app/infra/queue/handlers/example.go app/infra/queue/handlers/example_test.go
git commit -m "feat(queue): add example task handler"
```

---

### Task 6: Implement queue client

**Files:**
- Create: `app/infra/queue/client.go`

- [ ] **Step 1: Create client**

Create `app/infra/queue/client.go`:

```go
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"go-boilerplate/app/infra/queue/tasks"
	"go-boilerplate/app/shared/ports"
)

const maxRetry = 3

type Client struct {
	client *asynq.Client
}

func NewClient(redisAddr, redisPassword string) ports.QueueClient {
	opt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	}
	return &Client{client: asynq.NewClient(opt)}
}

func (c *Client) EnqueueSendEmail(ctx context.Context, to, subject, body string) error {
	p := tasks.EmailPayload{To: to, Subject: subject, Body: body}
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}
	_, err = c.client.EnqueueContext(ctx, asynq.NewTask(tasks.TypeSendEmail, data), asynq.MaxRetry(maxRetry))
	return err
}

func (c *Client) EnqueueExampleTask(ctx context.Context, userID, message string) error {
	p := tasks.ExamplePayload{UserID: userID, Message: message}
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal example payload: %w", err)
	}
	_, err = c.client.EnqueueContext(ctx, asynq.NewTask(tasks.TypeExampleTask, data), asynq.MaxRetry(maxRetry))
	return err
}

func (c *Client) Close() error {
	return c.client.Close()
}
```

- [ ] **Step 2: Verify compiles**

```bash
go build ./app/infra/queue/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add app/infra/queue/client.go
git commit -m "feat(queue): add asynq queue client"
```

---

### Task 7: Implement queue server

**Files:**
- Create: `app/infra/queue/server.go`

- [ ] **Step 1: Create server**

Create `app/infra/queue/server.go`:

```go
package queue

import (
	"github.com/hibiken/asynq"

	"go-boilerplate/app/infra/queue/handlers"
	"go-boilerplate/app/infra/queue/tasks"
	"go-boilerplate/app/shared/ports"
)

type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

func NewServer(redisAddr, redisPassword string, concurrency int) *Server {
	opt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	}
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
	})
	return &Server{
		srv: srv,
		mux: asynq.NewServeMux(),
	}
}

func (s *Server) RegisterHandlers(notifier ports.Notifier) {
	emailHandler := handlers.NewEmailHandler(notifier)
	s.mux.HandleFunc(tasks.TypeSendEmail, emailHandler.Process)
	s.mux.HandleFunc(tasks.TypeExampleTask, handlers.ProcessExampleTask)
}

func (s *Server) Start() error {
	return s.srv.Run(s.mux)
}

func (s *Server) Stop() {
	s.srv.Shutdown()
}
```

- [ ] **Step 2: Verify compiles**

```bash
go build ./app/infra/queue/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add app/infra/queue/server.go
git commit -m "feat(queue): add asynq worker server"
```

---

### Task 8: Wire queue client into bootstrap container

**Files:**
- Modify: `app/bootstrap/container.go`

- [ ] **Step 1: Add `QueueClient` field, import, and init**

Replace the full contents of `app/bootstrap/container.go`:

```go
package bootstrap

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"go-boilerplate/app/features/health"
	"go-boilerplate/app/features/posts"
	usersFeature "go-boilerplate/app/features/users"
	cacheInfra "go-boilerplate/app/infra/cache"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/httpclient"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/infra/queue"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/ports"
	"go-boilerplate/app/shared/token"
	// scaffold:container-imports
)

type Container struct {
	TokenMaker    token.Maker
	HashFn        func(ctx context.Context, userID uuid.UUID) (string, error)
	Cache         ports.Cache
	HTTPClient    ports.HTTPClient
	QueueClient   ports.QueueClient
	HealthHandler *health.Handler
	UsersHandler  *usersFeature.Handler
	PostsHandler  *posts.Handler
	// scaffold:container-fields
}

func NewContainer(db *sql.DB, cfg *config.Config, log ports.Logger, redisCache ports.Cache) *Container {
	tokenMaker := token.NewJWTMaker(cfg.JWTSecret)
	notifier := notification.NewMockNotifier()
	resetRepo := cacheInfra.NewResetTokenRepo(redisCache)

	usersRepo := dbUsers.NewPgRepository(db)
	usersSvc := usersFeature.NewService(usersRepo, resetRepo, notifier, tokenMaker)

	hashFn := func(ctx context.Context, userID uuid.UUID) (string, error) {
		user, err := usersRepo.FindByID(ctx, userID)
		if err != nil {
			return "", err
		}
		return user.PasswordHash, nil
	}

	queueClient := queue.NewClient(cfg.RedisAddr, cfg.RedisPassword)

	// scaffold:container-wire
	return &Container{
		TokenMaker:    tokenMaker,
		HashFn:        hashFn,
		Cache:         redisCache,
		HTTPClient:    httpclient.New(cfg),
		QueueClient:   queueClient,
		HealthHandler: health.NewHandler(db, redisCache),
		UsersHandler:  usersFeature.NewHandler(usersSvc),
		PostsHandler:  posts.NewHandler(posts.NewService(httpclient.New(cfg), redisCache)),
		// scaffold:container-init
	}
}
```

- [ ] **Step 2: Verify compiles**

```bash
go build ./app/bootstrap/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add app/bootstrap/container.go
git commit -m "feat(bootstrap): wire QueueClient into DI container"
```

---

### Task 9: Start and stop queue server in main.go

**Files:**
- Modify: `cmd/main.go`

- [ ] **Step 1: Replace main.go contents**

Replace the full contents of `cmd/main.go`:

```go
// @title          Go Boilerplate API
// @version        1.0
// @description    Production-ready Go REST API boilerplate with JWT auth, Redis cache, and pagination.
// @host           localhost:8080
// @BasePath       /
// @securityDefinitions.apikey BearerAuth
// @in             header
// @name           Authorization
// @description    Enter: Bearer <token>

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
	"go-boilerplate/app/infra/cache"
	"go-boilerplate/app/infra/database"
	"go-boilerplate/app/infra/logger"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/infra/queue"
	"go-boilerplate/app/shared/config"
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

	redisCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		log.Error("failed to connect to redis", err)
		os.Exit(1)
	}

	if cfg.RunMigrations {
		runMigrations(db)
	}

	c := bootstrap.NewContainer(db, cfg, log, redisCache)
	e := bootstrap.NewEcho(log)
	bootstrap.RegisterRoutes(e, c)

	notifier := notification.NewMockNotifier()
	queueServer := queue.NewServer(cfg.RedisAddr, cfg.RedisPassword, cfg.QueueConcurrency)
	queueServer.RegisterHandlers(notifier)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Info("server starting", "addr", addr)

	go func() {
		if err := e.Start(addr); err != nil {
			log.Info("server stopped")
		}
	}()

	go func() {
		if err := queueServer.Start(); err != nil {
			log.Error("queue worker stopped", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	queueServer.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", err)
	}

	if err := c.QueueClient.Close(); err != nil {
		log.Error("queue client close error", err)
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

- [ ] **Step 2: Full build check**

```bash
go build ./...
```

Expected: no output

- [ ] **Step 3: Run `go mod tidy` to promote asynq from indirect to direct**

```bash
go mod tidy
```

Expected: `go.mod` updated — `github.com/hibiken/asynq v0.26.0` moves from indirect `require` block to direct `require` block (no longer has `// indirect` comment)

- [ ] **Step 4: Commit**

```bash
git add cmd/main.go go.mod go.sum
git commit -m "feat(main): start asynq worker server alongside HTTP server"
```

---

### Task 10: Add asynqmon dashboard to docker-compose

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Add asynqmon service**

Add the `asynqmon` service block to `docker-compose.yml` between the `redis` service and the `volumes:` section:

```yaml
  asynqmon:
    image: hibiken/asynqmon
    ports:
      - "8081:8080"
    environment:
      REDIS_ADDR: redis:6379
      PORT: "8080"
      BASIC_AUTH_USERNAME: ${ASYNQMON_USERNAME:-admin}
      BASIC_AUTH_PASSWORD: ${ASYNQMON_PASSWORD:-admin}
    depends_on:
      redis:
        condition: service_healthy
    restart: unless-stopped
```

Full `docker-compose.yml` after edit:

```yaml
services:
  api:
    build:
      context: .
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      REDIS_ADDR: redis:6379
      RUN_MIGRATIONS: "true"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: go_boilerplate
    ports:
      - "5434:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  asynqmon:
    image: hibiken/asynqmon
    ports:
      - "8081:8080"
    environment:
      REDIS_ADDR: redis:6379
      PORT: "8080"
      BASIC_AUTH_USERNAME: ${ASYNQMON_USERNAME:-admin}
      BASIC_AUTH_PASSWORD: ${ASYNQMON_PASSWORD:-admin}
    depends_on:
      redis:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

- [ ] **Step 2: Validate docker-compose config**

```bash
docker compose config --quiet
```

Expected: no output (valid)

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(docker): add asynqmon dashboard on port 8081"
```

---

### Task 11: Smoke test

- [ ] **Step 1: Run all tests**

```bash
go test ./...
```

Expected: all pass including new handler tests

- [ ] **Step 2: Start Redis and Asynqmon**

```bash
docker compose up -d redis asynqmon
```

Expected: both containers healthy

- [ ] **Step 3: Verify dashboard**

Open `http://localhost:8081` — login with `admin` / `admin`.  
Expected: Asynqmon dashboard loads, shows empty queues.

- [ ] **Step 4: Start API to confirm worker connects**

```bash
go run ./cmd/main.go
```

Expected log lines (no errors):
```
{"level":"info","msg":"server starting","addr":":8080"}
```

Worker connects to Redis — no `dial tcp` errors in output.

- [ ] **Step 5: Enqueue a task and verify in dashboard**

In a separate terminal, run this one-shot Go program to enqueue an example task:

```go
// save as /tmp/enqueue_test.go and run with: go run /tmp/enqueue_test.go
package main

import (
	"context"
	"fmt"
	"log"

	"go-boilerplate/app/infra/queue"
)

func main() {
	client := queue.NewClient("localhost:6379", "")
	defer client.Close()

	err := client.EnqueueExampleTask(context.Background(), "user-001", "smoke test message")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("task enqueued — check http://localhost:8081")
}
```

Run:
```bash
cd /Users/rezwanul-haque/me/workspaces/personal/go-boilerplate && go run /tmp/enqueue_test.go
```

Expected:
- Output: `task enqueued — check http://localhost:8081`
- API terminal logs: `"msg":"processed example task","user_id":"user-001","message":"smoke test message"`
- Asynqmon dashboard: task appears in Processed count on the default queue
