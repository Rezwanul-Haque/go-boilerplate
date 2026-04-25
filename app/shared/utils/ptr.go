package utils

// Ptr returns a pointer to v. Useful for optional struct fields and test fixtures.
func Ptr[T any](v T) *T {
	return &v
}
