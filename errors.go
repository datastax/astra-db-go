package astradb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

// DataAPIError represents errors as they are returned from the API.
// Example error:
//
//	{
//		"message":"Document already exists with the given _id",
//		"errorCode":"DOCUMENT_ALREADY_EXISTS",
//		"id":"4055f085-68d8-4c2d-8d91-90a0722b5fef",
//		"title":"Document already exists with the given _id",
//		"family":"REQUEST",
//		"scope":"DOCUMENT"
//	}
type DataAPIError struct {
	Message        string `json:"message"`
	ErrorCode      string `json:"errorCode"`
	ExceptionClass string `json:"exceptionClass"`
	Family         string `json:"family"`
	Scope          string `json:"scope"`
	Title          string `json:"title"`
	ID             string `json:"id"`
}

// Slice of errors that implements error interface
type DataAPIErrors []DataAPIError

// Error implements the error interface for a pointer to the slice
func (e DataAPIErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	// We convert our struct errors to the error interface
	errs := make([]error, len(e))
	for i := range e {
		// Take the address because DataAPIError.Error()
		// is defined on the pointer (*DataAPIError)
		errs[i] = &e[i]
	}
	return errors.Join(errs...).Error()
}

// Unwrap allows errors.As to look inside the slice
func (e DataAPIErrors) Unwrap() []error {
	if e == nil {
		return nil
	}
	errs := make([]error, len(e))
	for i := range e {
		errs[i] = &e[i]
	}
	return errs
}

// Error implements the [error] interface.
//
// [error]: https://pkg.go.dev/builtin#error
func (a *DataAPIError) Error() string {
	if a == nil {
		return "<nil> DataAPIError"
	}
	// The things we care about are: Family, Scope, ErrorCode.
	msg := a.Message
	if msg == "" {
		msg = "unknown data api error"
	}

	// We only include fields that are actually populated
	var meta []string
	if a.ErrorCode != "" {
		meta = append(meta, fmt.Sprintf("code: %s", a.ErrorCode))
	}
	if a.Family != "" {
		meta = append(meta, fmt.Sprintf("family: %s", a.Family))
	}
	if a.Scope != "" {
		meta = append(meta, fmt.Sprintf("scope: %s", a.Scope))
	}

	if len(meta) > 0 {
		return fmt.Sprintf("%s (%s)", msg, strings.Join(meta, ", "))
	}

	return msg
}

// InsertManyError represents an error that occurred during an insertMany operation.
// It contains the partial results (successfully inserted IDs) along with the underlying errors.
type InsertManyError struct {
	// Errors contains all DataAPIErrors that occurred during the operation
	Errors DataAPIErrors

	// InsertedIds contains the IDs of documents that were successfully inserted
	// before the error occurred
	InsertedIds []any
}

// InsertedCount returns the number of documents that were successfully inserted before the error occurred.
func (e *InsertManyError) InsertedCount() int {
	return len(e.InsertedIds)
}

// Error implements the error interface for InsertManyError.
func (e *InsertManyError) Error() string {
	return fmt.Sprintf("insertMany failed after inserting %d documents: %v", e.InsertedCount(), e.Errors)
}

// Unwrap allows errors.As and errors.Is to work with the underlying errors.
func (e *InsertManyError) Unwrap() error {
	return e.Errors
}
