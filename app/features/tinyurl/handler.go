package tinyurl

import (
	"go-boilerplate/app/shared/response"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c echo.Context) error {
	var req CreateTinyurlRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	resp, err := h.svc.Create(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, resp)
}

func (h *Handler) List(c echo.Context) error {
	limit := 20
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	resp, err := h.svc.List(c.Request().Context(), limit, offset)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

func (h *Handler) Redirect(c echo.Context) error {
	shortCode := c.Param("shortCode")
	originalURL, err := h.svc.Redirect(c.Request().Context(), shortCode)
	if err != nil {
		return response.Error(c, err)
	}
	return c.Redirect(http.StatusFound, originalURL)
}
