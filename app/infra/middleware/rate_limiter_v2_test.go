package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"linkHub/app/infra/middleware"
)

func TestRateLimitV2_UnderLimit_Passes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware.RateLimitV2(5.0/60.0, 5)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitV2_OverLimit_Returns429(t *testing.T) {
	e := echo.New()
	// burst=0, refillRate=0 → bucket always empty
	handler := middleware.RateLimitV2(0, 0)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler(c)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimitV2_BurstExhausted_Returns429(t *testing.T) {
	e := echo.New()
	// burst=2, no refill (rate=0) → first 2 pass, 3rd blocked
	mw := middleware.RateLimitV2(0, 2)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	makeCtx := func() echo.Context {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	c1 := makeCtx()
	assert.NoError(t, handler(c1))
	assert.Equal(t, http.StatusOK, c1.Response().Status)

	c2 := makeCtx()
	assert.NoError(t, handler(c2))
	assert.Equal(t, http.StatusOK, c2.Response().Status)

	c3 := makeCtx()
	_ = handler(c3)
	assert.Equal(t, http.StatusTooManyRequests, c3.Response().Status)
}

func TestRateLimitV2_DifferentIPs_IndependentBuckets(t *testing.T) {
	e := echo.New()
	// burst=1 per IP
	mw := middleware.RateLimitV2(0, 1)

	makeHandler := func(ip string) (echo.Context, func() error) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
		return c, func() error { return h(c) }
	}

	c1, h1 := makeHandler("192.168.1.1")
	c2, h2 := makeHandler("192.168.1.2")

	_ = h1()
	assert.Equal(t, http.StatusOK, c1.Response().Status)

	_ = h2()
	assert.Equal(t, http.StatusOK, c2.Response().Status)
}
