# Asynq Queue System Design

**Date:** 2026-04-25  
**Status:** Approved

## Overview

Add async task queue to go-boilerplate using `hibiken/asynq` (already in go.sum). Redis already present — used as queue backend. Worker runs embedded in same process as HTTP server. Asynqmon dashboard added to docker-compose with basic auth.

## Goals

- Demonstrate queue pattern with two task types: email notification + generic example
- Follow existing clean arch conventions (ports/interfaces, DI container, infra layer)
- Minimal friction to add new task types later

## Directory Structure

```
app/infra/queue/
├── client.go              # QueueClient struct wrapping asynq.Client
├── server.go              # Worker server, handler registration, Start/Stop
├── tasks/
│   ├── email.go           # TypeSendEmail constant + EmailPayload struct
│   └── example.go         # TypeExampleTask constant + ExamplePayload struct
└── handlers/
    ├── email.go           # ProcessEmailTask — delegates to ports.Notifier
    └── example.go         # ProcessExampleTask — logs payload, simulates work
```

## Components

### Port: `ports.QueueClient`

New interface in `app/shared/ports/`:

```go
type QueueClient interface {
    EnqueueSendEmail(ctx context.Context, payload tasks.EmailPayload) error
    EnqueueExampleTask(ctx context.Context, payload tasks.ExamplePayload) error
    Close() error
}
```

Services depend on this interface, not concrete asynq type.

### `app/infra/queue/client.go`

- Wraps `asynq.Client`
- `EnqueueSendEmail` / `EnqueueExampleTask` marshal payload to JSON, call `client.Enqueue`
- Default queue: `default`, max retry: 3
- Implements `ports.QueueClient`

### `app/infra/queue/server.go`

- Wraps `asynq.Server` with config from `app/shared/config`
- `NewServer(redisAddr, redisPassword string, concurrency int)` constructor
- `RegisterHandlers(notifier ports.Notifier)` — wires task types to handler funcs
- `Start()` / `Stop()` — called from `cmd/main.go`

### `app/infra/queue/tasks/email.go`

```go
const TypeSendEmail = "email:send"

type EmailPayload struct {
    To      string
    Subject string
    Body    string
}
```

### `app/infra/queue/tasks/example.go`

```go
const TypeExampleTask = "example:task"

type ExamplePayload struct {
    UserID  string
    Message string
}
```

### `app/infra/queue/handlers/email.go`

- Unmarshals `EmailPayload` from `asynq.Task`
- Calls `ports.Notifier.Send(ctx, payload.To, payload.Subject, payload.Body)`
- Returns error on failure (asynq retries on non-nil error)

### `app/infra/queue/handlers/example.go`

- Unmarshals `ExamplePayload`
- Logs payload fields via zerolog
- Simulates work with no side effects

## Bootstrap / DI

`app/bootstrap/container.go`:
- Add `QueueClient ports.QueueClient` field to container struct
- Initialize `queue.NewClient(cfg.RedisAddr, cfg.RedisPassword)` in `NewContainer`

`cmd/main.go`:
- After HTTP server setup, initialize `queue.NewServer(...)`
- Call `server.RegisterHandlers(container.Notifier)`
- Start worker in goroutine: `go queueServer.Start()`
- On SIGTERM: call `queueServer.Stop()` before process exit

## Config

No new env vars for queue client (reuses `REDIS_ADDR`, `REDIS_PASSWORD`).

New optional env var:
- `QUEUE_CONCURRENCY` — worker concurrency (default: 10)

## Docker Compose

Add `asynqmon` service:

```yaml
asynqmon:
  image: hibiken/asynqmon
  ports:
    - "8081:8080"
  environment:
    - REDIS_ADDR=redis:6379
    - PORT=8080
    - BASIC_AUTH_USERNAME=${ASYNQMON_USERNAME:-admin}
    - BASIC_AUTH_PASSWORD=${ASYNQMON_PASSWORD:-admin}
  depends_on:
    redis:
      condition: service_healthy
```

Dashboard accessible at `http://localhost:8081`. Credentials via `.env`:
```
ASYNQMON_USERNAME=admin
ASYNQMON_PASSWORD=admin
```

## Data Flow

```
HTTP Handler
    → Service.SomeAction()
    → QueueClient.EnqueueSendEmail(ctx, EmailPayload{...})
    → asynq.Client.Enqueue() → Redis

asynq.Server (goroutine)
    → polls Redis
    → handlers.ProcessEmailTask()
    → ports.Notifier.Send()
```

## Error Handling

- Handler returns `error` → asynq retries (max 3, exponential backoff)
- Exhausted retries → task moves to Dead queue (visible in Asynqmon)
- Client enqueue errors logged at call site, not swallowed

## Testing

- Handler unit tests: construct `asynq.Task` with JSON payload, call handler func directly
- No mock of asynq internals needed — handlers are plain functions

## Out of Scope

- Scheduled/cron tasks
- Multiple named queues with priorities
- Task deduplication
- Separate worker binary
