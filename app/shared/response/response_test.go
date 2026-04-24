package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/app/shared/apperror"
	"go-boilerplate/app/shared/response"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCtx(method, path string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestOK_Returns200WithData(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	err := response.OK(c, map[string]string{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.True(t, body.Success)
}

func TestCreated_Returns201WithData(t *testing.T) {
	c, rec := newCtx(http.MethodPost, "/")
	err := response.Created(c, map[string]string{"id": "123"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestError_AppError_ReturnsCorrectStatus(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	appErr := apperror.New(http.StatusNotFound, "not found")
	err := response.Error(c, appErr)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var body response.Response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.False(t, body.Success)
	assert.Equal(t, "not found", body.Error)
}

func TestError_UnknownError_Returns500(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	err := response.Error(c, errors.New("some internal error"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
