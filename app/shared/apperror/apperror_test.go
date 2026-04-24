package apperror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"go-boilerplate/app/shared/apperror"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error_ReturnsMessage(t *testing.T) {
	err := apperror.New(http.StatusBadRequest, "bad request")
	assert.Equal(t, "bad request", err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := apperror.Wrap(http.StatusInternalServerError, "wrapped", inner)
	assert.Equal(t, inner, errors.Unwrap(err))
}

func TestIsAppError_WithAppError(t *testing.T) {
	err := apperror.New(http.StatusNotFound, "not found")
	appErr, ok := apperror.IsAppError(err)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, appErr.Code)
}

func TestIsAppError_WithStdError(t *testing.T) {
	err := errors.New("plain error")
	_, ok := apperror.IsAppError(err)
	assert.False(t, ok)
}

func TestIsAppError_Wrapped(t *testing.T) {
	inner := apperror.New(http.StatusConflict, "conflict")
	wrapped := fmt.Errorf("wrapping: %w", inner)
	appErr, ok := apperror.IsAppError(wrapped)
	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Code)
}
