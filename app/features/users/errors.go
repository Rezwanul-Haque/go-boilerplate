package users

import (
	"net/http"

	"go-boilerplate/app/shared/apperror"
)

var (
	ErrEmailAlreadyExists  = apperror.New(http.StatusConflict, "email already exists")
	ErrInvalidCredentials  = apperror.New(http.StatusUnauthorized, "invalid email or password")
	ErrUserNotFound        = apperror.New(http.StatusNotFound, "user not found")
	ErrInvalidResetToken   = apperror.New(http.StatusBadRequest, "invalid or expired reset token")
	ErrInvalidRefreshToken = apperror.New(http.StatusUnauthorized, "invalid refresh token")
	ErrWrongPassword       = apperror.New(http.StatusUnauthorized, "current password is incorrect")
)
