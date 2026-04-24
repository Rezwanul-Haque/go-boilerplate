package notification

import (
	"context"
	"fmt"

	"go-boilerplate/app/shared/ports"
)

type MockNotifier struct{}

func NewMockNotifier() ports.Notifier {
	return &MockNotifier{}
}

func (m *MockNotifier) SendPasswordReset(_ context.Context, email, tok string) error {
	fmt.Printf("[MockNotifier] password reset token for %s: %s\n", email, tok)
	return nil
}
