package bootstrap

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"

	"go-boilerplate/app/features/posts"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/middleware"
	// scaffold:feature-imports
)

func RegisterRoutes(e *echo.Echo, c *Container) {
	e.GET("/health", c.HealthHandler.Check)

	signupLimiter := middleware.RateLimit(rate.Limit(5.0/60.0), 5)

	v1 := e.Group("/api/v1")
	usersFeature.RegisterRoutes(v1.Group("/users"), c.UsersHandler, c.TokenMaker, signupLimiter, c.HashFn)
	posts.RegisterRoutes(v1.Group("/posts"), c.PostsHandler)
	// scaffold:feature-routes
}
