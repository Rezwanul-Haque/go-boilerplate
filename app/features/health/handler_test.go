package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/features/health"
)

type mockPinger struct{ err error }

func (m *mockPinger) PingContext(_ context.Context) error { return m.err }

type mockCachePinger struct{ err error }

func (m *mockCachePinger) Ping(_ context.Context) error { return m.err }

func newCtx() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestHealthCheck_DBOk_Returns200(t *testing.T) {
	h := health.NewHandler(&mockPinger{}, &mockCachePinger{})
	c, rec := newCtx()

	require.NoError(t, h.Check(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
	assert.Contains(t, rec.Body.String(), `"database":"ok"`)
	assert.Contains(t, rec.Body.String(), `"cache":"ok"`)
}

func TestHealthCheck_DBDown_Returns503(t *testing.T) {
	h := health.NewHandler(&mockPinger{err: errors.New("connection refused")}, &mockCachePinger{})
	c, rec := newCtx()

	require.NoError(t, h.Check(c))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"degraded"`)
	assert.Contains(t, rec.Body.String(), `"database":"error"`)
}

func TestHealthCheck_CacheDown_Returns503(t *testing.T) {
	h := health.NewHandler(&mockPinger{}, &mockCachePinger{err: errors.New("redis down")})
	c, rec := newCtx()

	require.NoError(t, h.Check(c))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"degraded"`)
	assert.Contains(t, rec.Body.String(), `"cache":"error"`)
}

func TestHealthCheck_CacheNil_Shows_Disabled(t *testing.T) {
	h := health.NewHandler(&mockPinger{}, nil)
	c, rec := newCtx()

	require.NoError(t, h.Check(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"cache":"disabled"`)
}
