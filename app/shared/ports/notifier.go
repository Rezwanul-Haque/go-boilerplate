package ports

import "context"

type Notifier interface {
	SendPasswordReset(ctx context.Context, email, token string) error
}
