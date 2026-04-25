package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/shared/model"
	"go-boilerplate/app/shared/ports"
)

type resetTokenRepo struct {
	cache ports.Cache
}

func NewResetTokenRepo(cache ports.Cache) usersFeature.PasswordResetRepository {
	return &resetTokenRepo{cache: cache}
}

func tokenKey(tok string) string      { return "reset_token:" + tok }
func userKey(id uuid.UUID) string     { return "reset_user:" + id.String() }

func (r *resetTokenRepo) SaveResetToken(ctx context.Context, id uuid.UUID, tok string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if err := r.cache.Set(ctx, tokenKey(tok), id.String(), ttl); err != nil {
		return err
	}
	return r.cache.Set(ctx, userKey(id), tok, ttl)
}

func (r *resetTokenRepo) FindByResetToken(ctx context.Context, tok string) (*usersFeature.User, error) {
	val, err := r.cache.Get(ctx, tokenKey(tok))
	if err != nil {
		return nil, fmt.Errorf("reset token not found or expired")
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return nil, fmt.Errorf("invalid reset token data")
	}
	return &usersFeature.User{Base: model.Base{ID: id}}, nil
}

func (r *resetTokenRepo) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	tok, err := r.cache.Get(ctx, userKey(id))
	if err != nil {
		return nil // already cleared or expired — no-op
	}
	_ = r.cache.Delete(ctx, tokenKey(tok))
	_ = r.cache.Delete(ctx, userKey(id))
	return nil
}
