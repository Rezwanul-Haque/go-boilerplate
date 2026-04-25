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
