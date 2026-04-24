package users

import (
	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(g *echo.Group, h *Handler, tokenMaker token.Maker, signupLimiter echo.MiddlewareFunc) {
	g.POST("/signup", h.Signup, signupLimiter)
	g.POST("/login", h.Login)
	g.POST("/forgot-password", h.ForgotPassword)
	g.POST("/reset-password", h.ResetPassword)
	g.POST("/refresh-token", h.RefreshToken)
	g.PUT("/change-password", h.ChangePassword, middleware.Auth(tokenMaker))
}
