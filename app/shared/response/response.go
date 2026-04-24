package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/apperror"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func OK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

func Error(c echo.Context, err error) error {
	if appErr, ok := apperror.IsAppError(err); ok {
		return c.JSON(appErr.Code, Response{Success: false, Error: appErr.Message})
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   fmt.Sprintf("validation failed: %s %s", ve[0].Field(), ve[0].Tag()),
		})
	}

	if he, ok := err.(*echo.HTTPError); ok {
		return c.JSON(he.Code, Response{Success: false, Error: fmt.Sprintf("%v", he.Message)})
	}

	return c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "internal server error"})
}
