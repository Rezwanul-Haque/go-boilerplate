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

func NewClient(redisAddr, redisPassword string, queueDB int) ports.QueueClient {
	opt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       queueDB,
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
