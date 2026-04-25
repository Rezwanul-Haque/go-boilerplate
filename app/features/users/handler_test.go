package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/app/features/users"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

// --- mock Service ---

type mockService struct {
	signupResp  *users.AuthResponse
	signupErr   error
	loginResp   *users.AuthResponse
	loginErr    error
	forgotErr   error
	resetErr    error
	changeErr   error
	refreshResp *users.AuthResponse
	refreshErr  error
}

func (m *mockService) Signup(_ context.Context, _ users.SignupRequest) (*users.AuthResponse, error) {
	return m.signupResp, m.signupErr
}
func (m *mockService) Login(_ context.Context, _ users.LoginRequest) (*users.AuthResponse, error) {
	return m.loginResp, m.loginErr
}
func (m *mockService) ForgotPassword(_ context.Context, _ users.ForgotPasswordRequest) error {
	return m.forgotErr
}
func (m *mockService) ResetPassword(_ context.Context, _ users.ResetPasswordRequest) error {
	return m.resetErr
}
func (m *mockService) ChangePassword(_ context.Context, _ uuid.UUID, _ users.ChangePasswordRequest) error {
	return m.changeErr
}
func (m *mockService) RefreshToken(_ context.Context, _ users.RefreshTokenRequest) (*users.AuthResponse, error) {
	return m.refreshResp, m.refreshErr
}
func (m *mockService) ListUsers(_ context.Context, _, _ int) (*users.OffsetPageResponse, error) {
	return &users.OffsetPageResponse{}, nil
}
func (m *mockService) ListUsersCursor(_ context.Context, _ string, _ int) (*users.CursorPageResponse, error) {
	return &users.CursorPageResponse{}, nil
}

// --- test helpers ---

type testValidator struct{ v *validator.Validate }

func (tv *testValidator) Validate(i interface{}) error { return tv.v.Struct(i) }

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	return e
}

func postJSON(e *echo.Echo, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// --- tests ---

func TestSignupHandler_Success_Returns201(t *testing.T) {
	svc := &mockService{
		signupResp: &users.AuthResponse{
			AccessToken:  "access",
			RefreshToken: "refresh",
			User:         users.UserResponse{ID: uuid.New().String(), Email: "new@example.com"},
		},
	}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/signup", map[string]string{"email": "new@example.com", "password": "password123"})

	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.True(t, resp.Success)
}

func TestSignupHandler_EmailExists_Returns409(t *testing.T) {
	svc := &mockService{signupErr: users.ErrEmailAlreadyExists}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/signup", map[string]string{"email": "dup@example.com", "password": "password123"})

	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSignupHandler_InvalidBody_Returns400(t *testing.T) {
	svc := &mockService{}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/signup", map[string]string{"email": "notanemail", "password": "short"})

	require.NoError(t, h.Signup(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginHandler_Success_Returns200(t *testing.T) {
	svc := &mockService{
		loginResp: &users.AuthResponse{AccessToken: "tok", RefreshToken: "ref"},
	}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/login", map[string]string{"email": "user@example.com", "password": "password123"})

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLoginHandler_InvalidCredentials_Returns401(t *testing.T) {
	svc := &mockService{loginErr: users.ErrInvalidCredentials}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/login", map[string]string{"email": "user@example.com", "password": "wrongpass"})

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestForgotPasswordHandler_Returns200(t *testing.T) {
	svc := &mockService{}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/forgot-password", map[string]string{"email": "user@example.com"})

	require.NoError(t, h.ForgotPassword(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestChangePasswordHandler_NoClaims_Returns401(t *testing.T) {
	svc := &mockService{changeErr: errors.New("should not reach")}
	h := users.NewHandler(svc)
	e := newTestEcho()

	b, _ := json.Marshal(map[string]string{"current_password": "old", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPut, "/users/change-password", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.ChangePassword(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefreshTokenHandler_InvalidToken_Returns401(t *testing.T) {
	svc := &mockService{refreshErr: users.ErrInvalidRefreshToken}
	h := users.NewHandler(svc)
	e := newTestEcho()
	c, rec := postJSON(e, "/users/refresh-token", map[string]string{"refresh_token": "badtoken"})

	require.NoError(t, h.RefreshToken(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// Ensure token package is used (for Claims)
var _ = (*token.Claims)(nil)
