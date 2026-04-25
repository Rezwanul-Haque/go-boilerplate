package httpclient

import (
	"net/http"
	"time"

	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/ports"
)

func New(cfg *config.Config) ports.HTTPClient {
	return &http.Client{
		Timeout: time.Duration(cfg.HTTPClientTimeout) * time.Second,
	}
}
