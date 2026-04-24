//go:build integration

package users_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/features/users"
	dbUsers "go-boilerplate/app/infra/database/users"
	"go-boilerplate/app/infra/notification"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=go_boilerplate sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, db.PingContext(context.Background()))
	t.Cleanup(func() {
		db.ExecContext(context.Background(), "DELETE FROM users")
		db.Close()
	})
	return db
}

type integrationValidator struct{ v *validator.Validate }

func (iv *integrationValidator) Validate(i interface{}) error { return iv.v.Struct(i) }

func newIntegrationStack(t *testing.T) (*echo.Echo, *users.Handler) {
	t.Helper()
	db := integrationDB(t)

	repo := dbUsers.NewPgRepository(db)
	maker := token.NewJWTMaker("supersecretkey1234567890abcdefghij")
	svc := users.NewService(repo, repo, notification.NewMockNotifier(), maker)
	h := users.NewHandler(svc)

	e := echo.New()
	e.Validator = &integrationValidator{v: validator.New()}
	return e, h
}

func postJSONIntegration(e *echo.Echo, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestIntegration_Signup_Login_Refresh(t *testing.T) {
	e, h := newIntegrationStack(t)

	// 1. Signup
	c, rec := postJSONIntegration(e, "/signup", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	require.NoError(t, h.Signup(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	var signupResp response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&signupResp))
	assert.True(t, signupResp.Success)

	dataBytes, _ := json.Marshal(signupResp.Data)
	var authResp users.AuthResponse
	require.NoError(t, json.Unmarshal(dataBytes, &authResp))
	assert.NotEmpty(t, authResp.AccessToken)
	assert.NotEmpty(t, authResp.RefreshToken)

	// 2. Login with correct credentials
	c, rec = postJSONIntegration(e, "/login", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// 3. Login with wrong password
	c, rec = postJSONIntegration(e, "/login", map[string]string{
		"email": "flow@example.com", "password": "wrongpassword",
	})
	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 4. Duplicate signup
	c, rec = postJSONIntegration(e, "/signup", map[string]string{
		"email": "flow@example.com", "password": "password123",
	})
	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusConflict, rec.Code)

	// 5. Refresh token
	c, rec = postJSONIntegration(e, "/refresh-token", map[string]string{
		"refresh_token": authResp.RefreshToken,
	})
	require.NoError(t, h.RefreshToken(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIntegration_ForgotPassword_ResetPassword(t *testing.T) {
	e, h := newIntegrationStack(t)
	db := integrationDB(t)
	repo := dbUsers.NewPgRepository(db)

	// Signup
	c, rec := postJSONIntegration(e, "/signup", map[string]string{
		"email": "forgot@example.com", "password": "password123",
	})
	require.NoError(t, h.Signup(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	// ForgotPassword
	c, rec = postJSONIntegration(e, "/forgot-password", map[string]string{
		"email": "forgot@example.com",
	})
	require.NoError(t, h.ForgotPassword(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Fetch reset token directly from DB
	u, err := repo.FindByEmail(context.Background(), "forgot@example.com")
	require.NoError(t, err)
	require.NotNil(t, u.ResetToken)

	// ResetPassword with valid token
	c, rec = postJSONIntegration(e, "/reset-password", map[string]string{
		"token": *u.ResetToken, "password": "newpassword123",
	})
	require.NoError(t, h.ResetPassword(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Login with new password
	c, rec = postJSONIntegration(e, "/login", map[string]string{
		"email": "forgot@example.com", "password": "newpassword123",
	})
	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}
