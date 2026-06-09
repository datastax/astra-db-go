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
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/results"
)

func TestDataAPIErrorAllMeta(t *testing.T) {
	e := &results.DataAPIError{
		Message:   "Document already exists",
		ErrorCode: "DOCUMENT_ALREADY_EXISTS",
		Family:    "REQUEST",
		Scope:     "DOCUMENT",
	}
	expected := "Document already exists (code: DOCUMENT_ALREADY_EXISTS, family: REQUEST, scope: DOCUMENT)"
	if got := e.Error(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDataAPIErrorOnlyMessage(t *testing.T) {
	e := &results.DataAPIError{Message: "something failed"}
	if got := e.Error(); got != "something failed" {
		t.Errorf("expected %q, got %q", "something failed", got)
	}
}

func TestDataAPIErrorEmptyMessage(t *testing.T) {
	e := &results.DataAPIError{}
	if got := e.Error(); got != "unknown data api error" {
		t.Errorf("expected %q, got %q", "unknown data api error", got)
	}
}

func TestDataAPIErrorNil(t *testing.T) {
	var e *results.DataAPIError
	if got := e.Error(); got != "<nil> DataAPIError" {
		t.Errorf("expected %q, got %q", "<nil> DataAPIError", got)
	}
}

func TestDataAPIErrorsJoin(t *testing.T) {
	errs := results.DataAPIErrors{
		{Message: "first", ErrorCode: "ERR1"},
		{Message: "second", ErrorCode: "ERR2"},
	}
	got := errs.Error()
	// errors.Join separates with newline
	expected := "first (code: ERR1)\nsecond (code: ERR2)"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDataAPIErrorsEmpty(t *testing.T) {
	errs := results.DataAPIErrors{}
	if got := errs.Error(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestDataAPIErrorsUnwrap(t *testing.T) {
	errs := results.DataAPIErrors{
		{Message: "a"},
		{Message: "b"},
	}
	unwrapped := errs.Unwrap()
	if len(unwrapped) != 2 {
		t.Fatalf("expected 2 unwrapped errors, got %d", len(unwrapped))
	}
	// Verify each unwrapped error is a *DataAPIError
	var dae *results.DataAPIError
	for i, ue := range unwrapped {
		if !errors.As(ue, &dae) {
			t.Errorf("unwrapped[%d] is not a *DataAPIError", i)
		}
	}
}

func TestDataAPIErrorsUnwrapNil(t *testing.T) {
	var errs results.DataAPIErrors
	if got := errs.Unwrap(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestEnsureNonEmptySliceValid(t *testing.T) {
	err := ensureNonEmptySlice([]int{1, 2, 3})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestEnsureNonEmptySliceEmpty(t *testing.T) {
	err := ensureNonEmptySlice([]int{})
	if !errors.Is(err, ErrEmptySlice) {
		t.Errorf("expected ErrEmptySlice, got %v", err)
	}
}

func TestEnsureNonEmptySliceNotSlice(t *testing.T) {
	err := ensureNonEmptySlice("not a slice")
	if !errors.Is(err, ErrNotSlice) {
		t.Errorf("expected ErrNotSlice, got %v", err)
	}
}
