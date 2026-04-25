package users

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
	"go-boilerplate/app/shared/utils"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Signup godoc
// @Summary     Sign up
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body SignupRequest true "Signup request"
// @Success     201 {object} response.Response{data=AuthResponse}
// @Failure     400 {object} response.Response
// @Failure     409 {object} response.Response
// @Router      /api/v1/users/signup [post]
func (h *Handler) Signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	resp, err := h.svc.Signup(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, resp)
}

// Login godoc
// @Summary     Login
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body LoginRequest true "Login request"
// @Success     200 {object} response.Response{data=AuthResponse}
// @Failure     400 {object} response.Response
// @Failure     401 {object} response.Response
// @Router      /api/v1/users/login [post]
func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	resp, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

// ForgotPassword godoc
// @Summary     Forgot password
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body ForgotPasswordRequest true "Forgot password request"
// @Success     200 {object} response.Response
// @Failure     400 {object} response.Response
// @Router      /api/v1/users/forgot-password [post]
func (h *Handler) ForgotPassword(c echo.Context) error {
	var req ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if err := h.svc.ForgotPassword(c.Request().Context(), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword godoc
// @Summary     Reset password
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body ResetPasswordRequest true "Reset password request"
// @Success     200 {object} response.Response
// @Failure     400 {object} response.Response
// @Failure     401 {object} response.Response
// @Router      /api/v1/users/reset-password [post]
func (h *Handler) ResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if err := h.svc.ResetPassword(c.Request().Context(), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "password reset successful"})
}

// ChangePassword godoc
// @Summary     Change password
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body ChangePasswordRequest true "Change password request"
// @Success     200 {object} response.Response
// @Failure     400 {object} response.Response
// @Failure     401 {object} response.Response
// @Router      /api/v1/users/change-password [put]
func (h *Handler) ChangePassword(c echo.Context) error {
	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	claims, ok := c.Get("claims").(*token.Claims)
	if !ok {
		return response.Error(c, apperror.ErrUnauthorized)
	}

	if err := h.svc.ChangePassword(c.Request().Context(), claims.UserID, req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, map[string]string{"message": "password changed successfully"})
}

// ListUsers godoc
// @Summary     List users (offset pagination)
// @Tags        users
// @Produce     json
// @Param       page  query int false "Page number" default(1)
// @Param       limit query int false "Items per page" default(20)
// @Success     200 {object} response.Response{data=OffsetPageResponse}
// @Failure     400 {object} response.Response
// @Router      /api/v1/users [get]
func (h *Handler) ListUsers(c echo.Context) error {
	var p utils.Pagination
	if err := c.Bind(&p); err != nil {
		return response.Error(c, err)
	}
	p.Normalize()

	resp, err := h.svc.ListUsers(c.Request().Context(), p.Page, p.Limit)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

// ListUsersCursor godoc
// @Summary     List users (cursor pagination)
// @Tags        users
// @Produce     json
// @Param       cursor query string false "Cursor (base64-encoded RFC3339Nano timestamp)"
// @Param       limit  query int    false "Items per page" default(20)
// @Success     200 {object} response.Response{data=CursorPageResponse}
// @Failure     400 {object} response.Response
// @Router      /api/v1/users/cursor [get]
func (h *Handler) ListUsersCursor(c echo.Context) error {
	limit := 20
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			limit = parsed
		}
	}

	resp, err := h.svc.ListUsersCursor(c.Request().Context(), c.QueryParam("cursor"), limit)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

// RefreshToken godoc
// @Summary     Refresh access token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body RefreshTokenRequest true "Refresh token request"
// @Success     200 {object} response.Response{data=AuthResponse}
// @Failure     400 {object} response.Response
// @Failure     401 {object} response.Response
// @Router      /api/v1/users/refresh-token [post]
func (h *Handler) RefreshToken(c echo.Context) error {
	var req RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	resp, err := h.svc.RefreshToken(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}
