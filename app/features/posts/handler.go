package posts

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetPost(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return response.Error(c, apperror.New(http.StatusBadRequest, "invalid post id"))
	}

	result, err := h.svc.GetPost(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return response.Error(c, apperror.New(http.StatusNotFound, "post not found"))
		}
		return response.Error(c, err)
	}

	if result.Cached {
		c.Response().Header().Set("X-Cache", "HIT")
	} else {
		c.Response().Header().Set("X-Cache", "MISS")
	}

	return response.OK(c, result.Post)
}
