package bootstrap

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"

	"go-boilerplate/app/features/health"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/middleware"
	"go-boilerplate/app/shared/token"
	// scaffold:feature-imports
)

func RegisterRoutes(
	e *echo.Echo,
	healthHandler *health.Handler,
	usersHandler *usersFeature.Handler,
	tokenMaker token.Maker,
	hashFn func(ctx context.Context, userID uuid.UUID) (string, error),
	// scaffold:feature-params
) {
	e.GET("/health", healthHandler.Check)

	signupLimiter := middleware.RateLimit(rate.Limit(5.0/60.0), 5)

	v1 := e.Group("/api/v1")
	usersGroup := v1.Group("/users")
	usersFeature.RegisterRoutes(usersGroup, usersHandler, tokenMaker, signupLimiter, hashFn)
	// scaffold:feature-routes
}
