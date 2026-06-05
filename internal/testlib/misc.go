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

// Package testutil provides utilities for testing. As more patterns
// emerge, we can add more helpers.
package testlib

import (
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/datastax/astra-db-go/astra/serdes"
)

var whitespace = regexp.MustCompile(`\s+`)

// cleanString removes all whitespace from s.
func cleanString(s string) string {
	return whitespace.ReplaceAllString(s, "")
}

// AssertJSONEqual marshals each arg to JSON and compares it to expected,
// ignoring whitespace differences. Reason: we have a lot of tests that compare
// fluent APIs to raw structs and also compare JSON payloads to expected JSON.
//
// Example usage:
//
//	tests := []struct {
//		expectedJSON string
//		fluent       *update.CollectionUpdateBuilder
//		raw          update.U
//	}{
//		{
//			`{"$set":{"number_of_pages": 423, "rating": 4.5}}`,
//			update.Coll().Set("number_of_pages", 423).Set("rating", 4.5),
//			update.U{"$set": update.U{"number_of_pages": 423, "rating": 4.5}},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.expectedJSON, func(t *testing.T) {
//			testlib.AssertJSONEqual(t, tt.expectedJSON, tt.fluent, tt.raw)
//		})
//	}
func AssertJSONEqual(t *testing.T, expected string, args ...any) {
	t.Helper() // marks this as a helper so failures point to the call site
	for _, arg := range args {
		got, err := serdes.Serialize(arg, serdes.TargetNone, serdes.SortMapKeys)

		if err != nil {
			t.Fatalf("failed to marshal argument: %v", err)
		}
		if cleanString(string(got)) != cleanString(expected) {
			t.Errorf("\nGOT:\n%s\n\nWANT:\n%s\n\nRAW ARG:\n%v", got, expected, arg)
			return
		}
	}
}

// A struct for defining test cases for AssertJSONEqual, which can be used in a
// table-driven test. Meant to be used in tandem with [RunJSONTestCases].
//
// Example usage:
//
//	tests := []testlib.JSONTestCase{{
//		Name:     "ascending",
//		Expected: `{"rating":1}`,
//		SerArgs: []any{
//			sort.Asc("rating"),  // fluent
//			sort.S{"rating": 1}, // raw map
//		},
//	}, {
//		Name:     "descending",
//		Expected: `{"title":-1}`,
//		SerArgs: []any{
//			sort.Desc("title"),  // fluent
//			sort.S{"title": -1}, // raw map
//		},
//	}}
//	// Run the tests
//	testlib.RunJSONTestCases(t, tests)
type JSONTestCase struct {
	Name     string
	Expected string
	Args     []any
}

// RunJSONTestCases runs a series of JSON equality test cases defined by JSONTestCase.
func RunJSONTestCases(t *testing.T, cases []JSONTestCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			AssertJSONEqual(t, tc.Expected, tc.Args...)
		})
	}
}

type HasFatal interface {
	Helper()
	Fatalf(format string, args ...any)
}

func AwaitAll[T any, R any](t HasFatal, xs []T, fn func(T) (R, error)) []R {
	t.Helper()

	var wg sync.WaitGroup
	var errAtomic atomic.Value
	results := make([]R, len(xs))

	for i, x := range xs {
		wg.Add(1)
		go func(index int, item T) {
			defer wg.Done()

			res, err := fn(item)
			if err != nil {
				errAtomic.Store(err)
				return
			}
			results[index] = res
		}(i, x)
	}

	wg.Wait()

	if err := errAtomic.Load(); err != nil {
		t.Fatalf("AwaitAll: %v", err.(error))
	}

	return results
}

func FailIf(t HasFatal, pred bool, msg string, args ...any) {
	t.Helper()
	if pred {
		t.Fatalf(msg, args...)
	}
}

func FailIfErr(t HasFatal, err error, msg string, args ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", fmt.Sprintf(msg, args...), err)
	}
}

func PanicIf(pred bool, msg string, args ...any) {
	if pred {
		panic(fmt.Sprintf(msg, args...))
	}
}

func PanicIfErr(err error, msg string, args ...any) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", fmt.Sprintf(msg, args...), err))
	}
}

// CleanString removes all whitespace characters from a string.
func CleanString(s string) string {
	// Use a regular expression to replace all whitespace characters (including spaces, tabs, newlines)
	// with an empty string.
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}
