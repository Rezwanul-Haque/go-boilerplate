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
