package health

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	_ "go-boilerplate/app/shared/response" // swagger type resolution
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

type CachePinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db    Pinger
	cache CachePinger
}

func NewHandler(db Pinger, cache CachePinger) *Handler {
	return &Handler{db: db, cache: cache}
}

type Status struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

// Check godoc
// @Summary     Health check
// @Tags        health
// @Produce     json
// @Success     200 {object} response.Response{data=health.Status}
// @Failure     503 {object} response.Response
// @Router      /health [get]
func (h *Handler) Check(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	overall := "ok"

	if err := h.db.PingContext(ctx); err != nil {
		checks["database"] = "error"
		overall = "degraded"
	} else {
		checks["database"] = "ok"
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			checks["cache"] = "error"
			overall = "degraded"
		} else {
			checks["cache"] = "ok"
		}
	} else {
		checks["cache"] = "disabled"
	}

	code := http.StatusOK
	if overall != "ok" {
		code = http.StatusServiceUnavailable
	}

	return c.JSON(code, Status{
		Status:    overall,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	})
}
