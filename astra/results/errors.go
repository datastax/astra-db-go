// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package results

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNoDocuments error = errors.New("no documents found")

// ErrTooManyDocumentsToCount is returned when a Count command exceeds the upper bounds.
var ErrTooManyDocumentsToCount error = errors.New("too many documents")

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

// DataAPIErrors is a slice of errors that implements error interface.
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

	// Result contains the partial results of the operation
	Result *InsertManyResult
}

// InsertedCount returns the number of documents that were successfully inserted before the error occurred.
func (e *InsertManyError) InsertedCount() int {
	if e.Result == nil {
		return 0
	}
	return e.Result.InsertedCount()
}

// RawIDs returns the raw inserted IDs as a slice of any.
func (e *InsertManyError) RawIDs() ([]any, error) {
	if e.Result == nil {
		return nil, nil
	}
	return e.Result.RawIDs()
}

// DecodeIDs unmarshalls the inserted IDs into v.
// v should be a pointer to a slice of the appropriate ID type.
func (e *InsertManyError) DecodeIDs(v any) error {
	if e.Result == nil {
		return nil
	}
	return e.Result.DecodeIDs(v)
}

// Error implements the error interface for InsertManyError.
func (e *InsertManyError) Error() string {
	count := e.InsertedCount()
	if len(e.Errors) == 0 {
		return fmt.Sprintf("insertMany failed after inserting %d documents", count)
	}

	// Summarize the first error and add a count of others if they exist
	msg := e.Errors[0].Error()
	if len(e.Errors) > 1 {
		msg = fmt.Sprintf("%s (+ %d more errors)", msg, len(e.Errors)-1)
	}

	return fmt.Sprintf("insertMany failed after inserting %d documents: %s", count, msg)
}

// Unwrap allows errors.As and errors.Is to work with the underlying errors.
func (e *InsertManyError) Unwrap() error {
	return e.Errors
}
