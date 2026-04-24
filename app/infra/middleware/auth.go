package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"
	"go-boilerplate/app/shared/token"
)

func Auth(maker token.Maker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Error(c, apperror.ErrUnauthorized)
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return response.Error(c, apperror.ErrUnauthorized)
			}

			claims, err := maker.VerifyToken(parts[1])
			if err != nil {
				return response.Error(c, apperror.ErrUnauthorized)
			}

			c.Set("claims", claims)
			return next(c)
		}
	}
}
