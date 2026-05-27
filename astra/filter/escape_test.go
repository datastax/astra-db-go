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

package filter_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/datastax/astra-db-go/astra/filter"
)

func TestEscapeFieldNames(t *testing.T) {
	tests := []struct {
		name     string
		segments []any
		expected string
	}{
		// These test cases were taken from the TS unit tests here:
		// https://github.com/datastax/astra-db-ts/blob/master/tests/unit/lib/field-escaping.test.ts#L84
		{"empty input", nil, ""},
		{"single segment no special chars", []any{"foo"}, "foo"},
		{"multiple segments no special chars", []any{"a", "b", "c"}, "a.b.c"},
		{"segment with ampersand", []any{"a&b"}, "a&&b"},
		{"segment with dot", []any{"a.b"}, "a&.b"},
		{"mixed special chars from TS tests", []any{"a&", "b..", 0, "c&d"}, "a&&.b&.&..0.c&&d"},
		{"numeric int segment", []any{42}, "42"},
		{"int64 segment", []any{int64(99)}, "99"},
		{"segment with only ampersand", []any{"&"}, "&&"},
		{"segment with only dot", []any{"."}, "&."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.EscapeFieldNames(tt.segments...)
			if result != tt.expected {
				t.Errorf("EscapeFieldNames(%v) = %q, want %q", tt.segments, result, tt.expected)
			}
		})
	}
}

func TestUnescapeFieldPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
		wantErr  bool
		name     string
	}{
		{"", nil, false, "empty string"},
		{"a.a", []string{"a", "a"}, false, "simple two segments"},
		{"a&.", []string{"a."}, false, "escaped dot at end"},
		{"a&.a", []string{"a.a"}, false, "escaped dot mid-segment"},
		{"a&.a&&&.a", []string{"a.a&.a"}, false, "mixed escapes"},
		{"&&.&.", []string{"&", "."}, false, "escaped ampersand and dot as separate segments"},
		{"&&", []string{"&"}, false, "escaped ampersand only"},
		{"&.", []string{"."}, false, "escaped dot only"},
		{"a.b.c", []string{"a", "b", "c"}, false, "three simple segments"},
		{".a", nil, true, "starts with dot"},
		{"a.", nil, true, "ends with dot"},
		{"a..b", nil, true, "consecutive dots"},
		{"a&", nil, true, "trailing ampersand"},
		{"a&b", nil, true, "invalid escape sequence"},
		{".", nil, true, "just a dot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filter.UnescapeFieldPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("UnescapeFieldPath(%q) expected error, got nil", tt.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("UnescapeFieldPath(%q) unexpected error: %v", tt.path, err)
			}
			if !slices.Equal(result, tt.expected) {
				t.Errorf("UnescapeFieldPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	tests := []struct {
		segments        []any
		expectedEscaped string
		name            string
	}{
		{[]any{"a", "b", "c"}, "a.b.c", "simple"},
		{[]any{"a.b", "c.d.e"}, "a&.b.c&.d&.e", "with dots"},
		{[]any{"a&b", "c&&d"}, "a&&b.c&&&&d", "with ampersands"},
		{[]any{"a&", "b..", "c&d"}, "a&&.b&.&..c&&d", "mixed"},
		{[]any{"foo", 0, "bar"}, "foo.0.bar", "numeric"},
		{[]any{"hello"}, "hello", "single"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := filter.EscapeFieldNames(tt.segments...)
			if escaped != tt.expectedEscaped {
				t.Errorf("EscapeFieldNames(%v) = %q, want %q", tt.segments, escaped, tt.expectedEscaped)
			}
			result, err := filter.UnescapeFieldPath(escaped)
			if err != nil {
				t.Fatalf("UnescapeFieldPath(%q) unexpected error: %v", escaped, err)
			}

			// Convert original segments to strings for comparison
			expected := make([]string, len(tt.segments))
			for i, seg := range tt.segments {
				expected[i] = fmt.Sprintf("%v", seg)
			}
			if !slices.Equal(result, expected) {
				t.Errorf("round-trip failed: EscapeFieldNames(%v) = %q, UnescapeFieldPath = %v, want %v",
					tt.segments, escaped, result, expected)
			}
		})
	}
}

func ExampleEscapeFieldNames() {
	fmt.Println(filter.EscapeFieldNames("websites", "www.datastax.com", "visits"))
	fmt.Println(filter.EscapeFieldNames("shows", "tom&jerry", "episodes", 3, "views"))
	// Output:
	// websites.www&.datastax&.com.visits
	// shows.tom&&jerry.episodes.3.views
}
