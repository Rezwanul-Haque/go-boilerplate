package tinyurl

import "time"

type CreateTinyurlRequest struct {
	OriginalURL string `json:"original_url" validate:"required,url"`
}

type TinyurlResponse struct {
	ID          string    `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	ClickCount  int64     `json:"click_count"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}
