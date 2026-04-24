package bootstrap

import (
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

func RegisterRoutes(e *echo.Echo, usersHandler *usersFeature.Handler, tokenMaker token.Maker) {
	signupLimiter := middleware.RateLimit(rate.Limit(5.0/60.0), 5)

	v1 := e.Group("/api/v1")
	usersGroup := v1.Group("/users")
	usersFeature.RegisterRoutes(usersGroup, usersHandler, tokenMaker, signupLimiter)
}
