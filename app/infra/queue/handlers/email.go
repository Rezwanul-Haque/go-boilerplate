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
