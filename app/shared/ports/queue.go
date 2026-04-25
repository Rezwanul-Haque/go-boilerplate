package ports

import "context"

type QueueClient interface {
	EnqueueSendEmail(ctx context.Context, to, subject, body string) error
	EnqueueExampleTask(ctx context.Context, userID, message string) error
	Close() error
}
