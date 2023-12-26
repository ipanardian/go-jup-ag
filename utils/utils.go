package utils

import (
	"fmt"
	"github.com/google/go-querystring/query"
	"net/url"
)

// Pointer is a helper to make a pointer to the given value.
func Pointer[T any](v T) *T {
	return &v
}

func StructToUrlValues(v interface{}) (url.Values, error) {
	if v == nil {
		return nil, fmt.Errorf("struct is nil")
	}

	if _, ok := v.(url.Values); ok {
		return v.(url.Values), nil
	}

	uv, err := query.Values(v)
	if err != nil {
		return nil, fmt.Errorf("failed to convert struct to url values: %w", err)
	}

	return uv, nil
}
