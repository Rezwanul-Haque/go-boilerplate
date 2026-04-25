package users

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	List(ctx context.Context, limit, offset int) ([]*User, int64, error)
	ListAfterCursor(ctx context.Context, cursor time.Time, limit int) ([]*User, error)
}

type PasswordResetRepository interface {
	SaveResetToken(ctx context.Context, id uuid.UUID, token string, expiresAt time.Time) error
	FindByResetToken(ctx context.Context, token string) (*User, error)
	ClearResetToken(ctx context.Context, id uuid.UUID) error
}
