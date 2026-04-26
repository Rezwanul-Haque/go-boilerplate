package tinyurl

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, item *Tinyurl) error
	List(ctx context.Context, limit, offset int) ([]*Tinyurl, error)
	FindByShortCode(ctx context.Context, shortcode string) (*Tinyurl, error)
	IncrementClickCount(ctx context.Context, shortcode string) error
	FindLatestShortCode(ctx context.Context) (string, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
