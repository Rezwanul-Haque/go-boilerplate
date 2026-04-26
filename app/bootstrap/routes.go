package bootstrap

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/time/rate"

	"go-boilerplate/app/features/posts"
	tinyurlFeature "go-boilerplate/app/features/tinyurl"
	usersFeature "go-boilerplate/app/features/users"
	"go-boilerplate/app/infra/middleware"
	_ "go-boilerplate/docs/swagger"
	// scaffold:feature-imports
)

func RegisterRoutes(e *echo.Echo, c *Container) {
	e.GET("/docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
	})
	e.GET("/docs/*", echoSwagger.WrapHandler)
	e.GET("/health", c.HealthHandler.Check)
	e.GET("/:shortCode", c.TinyurlHandler.Redirect)

	signupLimiter := middleware.RateLimit(rate.Limit(5.0/60.0), 5)

	v1 := e.Group("/api/v1")
	usersFeature.RegisterRoutes(v1.Group("/users"), c.UsersHandler, c.TokenMaker, signupLimiter, c.HashFn)
	posts.RegisterRoutes(v1.Group("/posts"), c.PostsHandler)
	tinyurlFeature.RegisterRoutes(v1.Group("/tinyurl"), c.TinyurlHandler)
	// scaffold:feature-routes
}
