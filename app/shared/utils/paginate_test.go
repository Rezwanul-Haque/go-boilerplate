package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go-boilerplate/app/shared/utils"
)

func TestPagination_Normalize_Defaults(t *testing.T) {
	p := utils.Pagination{}
	p.Normalize()
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.Limit)
}

func TestPagination_Normalize_CapsLimit(t *testing.T) {
	p := utils.Pagination{Page: 1, Limit: 999}
	p.Normalize()
	assert.Equal(t, 100, p.Limit)
}

func TestPagination_Offset(t *testing.T) {
	p := utils.Pagination{Page: 3, Limit: 20}
	assert.Equal(t, 40, p.Offset())
}
