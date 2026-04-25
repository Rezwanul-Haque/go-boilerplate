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
