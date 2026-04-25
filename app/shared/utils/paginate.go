package utils

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type Pagination struct {
	Page   int `query:"page"`
	Limit  int `query:"limit"`
}

func (p *Pagination) Normalize() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.Limit < 1 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}
