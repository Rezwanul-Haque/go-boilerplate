# Queue Usage Guide

## Architecture

`ports.QueueClient` is a fat interface used only for wiring in `bootstrap/container.go` and closing in `main.go`.

Features **never** depend on `ports.QueueClient` directly. Each feature defines its own narrow interface — Go duck
typing means `*queue.Client` satisfies it automatically.

```
ports.QueueClient     ← Container holds this (consistency with repo pattern)
       ↓
queue.Client          ← concrete, implements ports.QueueClient
       ↓
feature.someQueue     ← narrow interface defined in feature, satisfied by queue.Client
```

---

## Adding a new task type

**1. Define payload and type constant** in `app/infra/queue/tasks/`:

```go
// app/infra/queue/tasks/welcome.go
package tasks

const TypeSendWelcome = "email:welcome"

type WelcomePayload struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}
```

**2. Implement the handler** in `app/infra/queue/handlers/`:

```go
// app/infra/queue/handlers/welcome.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"

    "go-boilerplate/app/infra/queue/tasks"
)

func ProcessWelcomeTask(ctx context.Context, t *asynq.Task) error {
    var p tasks.WelcomePayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil {
        return fmt.Errorf("unmarshal welcome payload: %w", err)
    }
    log.Ctx(ctx).Info().Str("user_id", p.UserID).Str("email", p.Email).Msg("sending welcome email")
    // call real email service here
    return nil
}
```

**3. Add enqueue method to `app/infra/queue/client.go`**:

```go
func (c *Client) EnqueueSendWelcome(ctx context.Context, userID, email string) error {
    p := tasks.WelcomePayload{UserID: userID, Email: email}
    data, err := json.Marshal(p)
    if err != nil {
        return fmt.Errorf("marshal welcome payload: %w", err)
    }
    _, err = c.client.EnqueueContext(ctx, asynq.NewTask(tasks.TypeSendWelcome, data), asynq.MaxRetry(maxRetry))
    return err
}
```

**4. Add method to `ports.QueueClient`** in `app/shared/ports/queue.go`:

```go
type QueueClient interface {
    EnqueueSendEmail(ctx context.Context, to, subject, body string) error
    EnqueueExampleTask(ctx context.Context, userID, message string) error
    EnqueueSendWelcome(ctx context.Context, userID, email string) error // add
    Close() error
}
```

**5. Register handler in `app/infra/queue/server.go`**:

```go
func (s *Server) RegisterHandlers(notifier ports.Notifier) {
    emailHandler := handlers.NewEmailHandler(notifier)
    s.mux.HandleFunc(tasks.TypeSendEmail, emailHandler.Process)
    s.mux.HandleFunc(tasks.TypeExampleTask, handlers.ProcessExampleTask)
    s.mux.HandleFunc(tasks.TypeSendWelcome, handlers.ProcessWelcomeTask) // add
}
```

---

## Using the queue in a feature (ISP pattern)

Features define a **narrow local interface** — only the methods they actually need.

**1. Define narrow interface inside the feature package**

```go
// app/features/users/service.go

// welcomeQueue is the only queue contract this feature cares about.
// queue.Client satisfies this automatically via Go duck typing.
type welcomeQueue interface {
    EnqueueSendWelcome(ctx context.Context, userID, email string) error
}

type service struct {
    repo      UserRepository
    resetRepo PasswordResetRepository
    notifier  ports.Notifier
    token     token.Maker
    queue     welcomeQueue // narrow interface, not ports.QueueClient
}

func NewService(
    repo UserRepository,
    resetRepo PasswordResetRepository,
    notifier ports.Notifier,
    tokenMaker token.Maker,
    queue welcomeQueue,
) Service {
    return &service{
        repo:      repo,
        resetRepo: resetRepo,
        notifier:  notifier,
        token:     tokenMaker,
        queue:     queue,
    }
}
```

**2. Use in a service method**

```go
func (s *service) Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error) {
    // ... existing signup logic ...

    _ = s.queue.EnqueueSendWelcome(ctx, user.ID.String(), user.Email)

    return authResponse, nil
}
```

**3. Wire in `app/bootstrap/container.go`**

`container.QueueClient` is `ports.QueueClient` — it satisfies `welcomeQueue` because `queue.Client` implements both.

```go
// container.QueueClient passed directly — satisfies welcomeQueue via duck typing
usersSvc := usersFeature.NewService(usersRepo, resetRepo, notifier, tokenMaker, container.QueueClient)
```

Wait — `container.QueueClient` is `ports.QueueClient` (interface), not `*queue.Client` (concrete).
Go interfaces are satisfied structurally: `ports.QueueClient` value holds a `*queue.Client` underneath, which has `EnqueueSendWelcome`. Pass it directly and it works.

---

## Rules

- **Define interfaces where consumed** — each feature owns its queue interface, not `ports/`
- **Handlers must be idempotent** — asynq retries on non-nil error (max 3). Same task may run more than once.
- **Keep payloads small** — store IDs, not full objects. Fetch fresh data inside the handler.
- **Don't swallow enqueue errors** — log at call site; don't let queue failure break HTTP response unless critical.
- **Dead tasks** visible in Asynqmon at `http://localhost:8081` (login: `admin` / `admin`, or set `ASYNQMON_USERNAME` / `ASYNQMON_PASSWORD` in `.env`).
- Queue uses **Redis DB 1** (separate from cache on DB 0). Override with `QUEUE_DB` env var.
