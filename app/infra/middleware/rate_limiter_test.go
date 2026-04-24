package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"go-boilerplate/app/infra/middleware"
)

func TestRateLimit_UnderLimit_Passes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/signup", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware.RateLimit(rate.Limit(5.0/60.0), 5)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_OverLimit_Returns429(t *testing.T) {
	e := echo.New()
	rateLimitMiddleware := middleware.RateLimit(rate.Limit(0), 0)

	handler := rateLimitMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/signup", nil)
	req.RemoteAddr = "2.3.4.5:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler(c)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
