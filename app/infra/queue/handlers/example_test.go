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
