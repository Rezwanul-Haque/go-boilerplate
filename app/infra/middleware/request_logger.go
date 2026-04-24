package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/ports"
)

func RequestLogger(logger ports.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			logger.Info("request",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"duration", time.Since(start).String(),
				"ip", c.RealIP(),
			)
			return err
		}
	}
}
