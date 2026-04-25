package bootstrap

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/time/rate"

	_ "go-boilerplate/docs/swagger"
	"go-boilerplate/app/features/posts"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/middleware"
	// scaffold:feature-imports
)

func RegisterRoutes(e *echo.Echo, c *Container) {
	e.GET("/docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
	})
	e.GET("/docs/*", echoSwagger.WrapHandler)
	e.GET("/health", c.HealthHandler.Check)

	signupLimiter := middleware.RateLimit(rate.Limit(5.0/60.0), 5)

	v1 := e.Group("/api/v1")
	usersFeature.RegisterRoutes(v1.Group("/users"), c.UsersHandler, c.TokenMaker, signupLimiter, c.HashFn)
	posts.RegisterRoutes(v1.Group("/posts"), c.PostsHandler)
	// scaffold:feature-routes
}
