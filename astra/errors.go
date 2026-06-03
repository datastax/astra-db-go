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
