package tinyurl

import (
	"go-boilerplate/app/shared/model"
	"time"
)

type Tinyurl struct {
	model.Base
	ShortCode   string    `db:"short_code"`
	OriginalURL string    `db:"original_url"`
	ClickCount  int64     `db:"click_count"`
	ExpiresAt   time.Time `db:"expires_at"`
}
