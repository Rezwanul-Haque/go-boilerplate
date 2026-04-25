package cache

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	usersFeature "go-boilerplate/app/features/users"
)

type noopResetTokenRepo struct{}

func NewNoopResetTokenRepo() usersFeature.PasswordResetRepository {
	return &noopResetTokenRepo{}
}

func (n *noopResetTokenRepo) SaveResetToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return errors.New("password reset unavailable: cache not configured")
}

func (n *noopResetTokenRepo) FindByResetToken(_ context.Context, _ string) (*usersFeature.User, error) {
	return nil, errors.New("password reset unavailable: cache not configured")
}

func (n *noopResetTokenRepo) ClearResetToken(_ context.Context, _ uuid.UUID) error {
	return nil
}
