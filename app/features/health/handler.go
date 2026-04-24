package health

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

func NewHandler(db Pinger) *Handler {
	return &Handler{db: db}
}

type status struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

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

	code := http.StatusOK
	if overall != "ok" {
		code = http.StatusServiceUnavailable
	}

	return c.JSON(code, status{
		Status:    overall,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	})
}
