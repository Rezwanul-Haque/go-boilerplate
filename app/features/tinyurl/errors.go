package tinyurl

import (
	"net/http"

	"go-boilerplate/app/shared/apperror"
)

var (
	ErrTinyurlNotFound = apperror.New(http.StatusNotFound, "tiny url not found")
	ErrTinyurlConflict = apperror.New(http.StatusConflict, "tiny url already exists")
	ErrTinyurlExpired  = apperror.New(http.StatusConflict, "tiny url has expired")
)
