package utils

import (
	"fmt"
	"reflect"
)

// I know 'utils' isn't a very idiomatic or descriptive name, but sometimes a thing is just what it says on the tin

func RequireSlicePtr(v any) (reflect.Value, reflect.Value, error) {
	ptr := reflect.ValueOf(v)
	if ptr.Kind() != reflect.Ptr {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("expected pointer to slice, got %s", ptr.Kind())
	}

	slice := ptr.Elem()
	if slice.Kind() == reflect.Interface {
		slice = slice.Elem()
	}

	if slice.Kind() != reflect.Slice {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("expected pointer to slice, got pointer to %s", slice.Kind())
	}

	return ptr, slice, nil
}

func NonNilMap[M ~map[K]V, K comparable, V any](m M) M {
	if m == nil {
		return make(M)
	}
	return m
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
