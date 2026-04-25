package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go-boilerplate/app/shared/utils"
)

func TestPtr_ReturnsPointerToValue(t *testing.T) {
	s := utils.Ptr("hello")
	assert.Equal(t, "hello", *s)

	n := utils.Ptr(42)
	assert.Equal(t, 42, *n)
}
