package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"
)

const authTestSecret = "supersecretkey1234567890abcdefghij"
const authTestSalt = "$2a$10$examplepasswordhashfortesting"

func newEchoCtx(method, path, authHeader string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func fixedHashFn(hash string) func(ctx context.Context, userID uuid.UUID) (string, error) {
	return func(_ context.Context, _ uuid.UUID) (string, error) {
		return hash, nil
	}
}

func TestAuthMiddleware_NoHeader_Returns401(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	c, rec := newEchoCtx(http.MethodGet, "/", "")

	handler := middleware.Auth(maker, fixedHashFn(authTestSalt))(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	c, rec := newEchoCtx(http.MethodGet, "/", "Bearer notavalidtoken")

	handler := middleware.Auth(maker, fixedHashFn(authTestSalt))(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidToken_PassesThrough(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	tok, err := maker.CreateToken(uuid.New(), "auth@example.com", token.AccessToken, time.Minute, authTestSalt)
	require.NoError(t, err)

	c, rec := newEchoCtx(http.MethodGet, "/", "Bearer "+tok)

	handler := middleware.Auth(maker, fixedHashFn(authTestSalt))(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_ValidToken_SetsClaims(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	userID := uuid.New()
	tok, err := maker.CreateToken(userID, "claims@example.com", token.AccessToken, time.Minute, authTestSalt)
	require.NoError(t, err)

	c, _ := newEchoCtx(http.MethodGet, "/", "Bearer "+tok)

	var capturedClaims *token.Claims
	handler := middleware.Auth(maker, fixedHashFn(authTestSalt))(func(c echo.Context) error {
		capturedClaims = c.Get("claims").(*token.Claims)
		return c.String(http.StatusOK, "ok")
	})

	require.NoError(t, handler(c))
	require.NotNil(t, capturedClaims)
	assert.Equal(t, userID, capturedClaims.UserID)
}

func TestAuthMiddleware_WrongSalt_Returns401(t *testing.T) {
	maker := token.NewJWTMaker(authTestSecret)
	tok, err := maker.CreateToken(uuid.New(), "salt@example.com", token.AccessToken, time.Minute, authTestSalt)
	require.NoError(t, err)

	c, rec := newEchoCtx(http.MethodGet, "/", "Bearer "+tok)

	handler := middleware.Auth(maker, fixedHashFn("differenthash"))(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
