package astra

import (
	"errors"
	"reflect"
)

// ErrNotFound is returned when a command returns "not found".
var ErrNotFound error = errors.New("not found")

// ErrNil is returned when an argument is nil.
var ErrNil error = errors.New("must be non-nil")

// ErrNotSlice is returned when an argument should be a slice.
var ErrNotSlice error = errors.New("must be slice")

// ErrEmptySlice is returned when an argument must be a non-empty slice.
var ErrEmptySlice error = errors.New("must be non-empty")

// ErrCmdNilDb is returned when a command tries to execute with a nil db
var ErrCmdNilDb error = errors.New("command cannot execute with nil Db")

// ensureNonEmptySlice returns an error if v is anything other than a non-empty slice.
func ensureNonEmptySlice(v any) error {
	rval := reflect.ValueOf(v)
	if rval.Kind() != reflect.Slice {
		return ErrNotSlice
	}
	if rval.Len() == 0 {
		return ErrEmptySlice
	}
	// Non-empty slice
	return nil
}
