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
	"fmt"
	"strings"
)

// EscapeFieldNames escapes field names which may contain '.'s and '&'s
// for use in Data API queries. Accepts one or more path segments (string or int)
// and returns a dot-separated escaped field path.
//
// Example:
//
//	EscapeFieldNames("websites", "www.datastax.com", "visits")
//	// Output: "websites.www&.datastax&.com.visits"
//
// # Important Usage Notes
//
// This should NOT be used for insertion operations. It is only for use in areas
// where a field path is required; not just a field name (e.g. filters, projections,
// updates, etc.)
//
// Segments should be of type string or int. Other types will be converted to strings
// so you will be OK as long as you implement stringer.
func EscapeFieldNames(segments ...any) string {
	if len(segments) == 0 {
		return ""
	}

	parts := make([]string, 0, len(segments))
	for _, seg := range segments {
		parts = append(parts, escapeSegment(seg))
	}
	return strings.Join(parts, ".")
}

func escapeSegment(segment any) string {
	switch v := segment.(type) {
	case string:
		return strings.NewReplacer("&", "&&", ".", "&.").Replace(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	default:
		// This should never happen. But this is just a fallback in case it does. We
		// could also panic here but you could make the argument that that goes against
		// the go proverb "don't panic".
		return strings.NewReplacer("&", "&&", ".", "&.").Replace(fmt.Sprintf("%v", v))
	}
}

// UnescapeFieldPath splits a field path into its individual segments, accounting
// for escaped characters.
//
// Returns an error if the path contains invalid escape sequences or malformed structure
// (e.g., leading/trailing dots, consecutive dots, or a trailing '&').
func UnescapeFieldPath(path string) ([]string, error) {
	// Fast path for empty string
	if path == "" {
		return []string{}, nil
	}
	// Early return for simple cases with no special characters
	if !strings.ContainsAny(path, "&.") {
		return []string{path}, nil
	}

	if strings.HasPrefix(path, ".") {
		return nil, fmt.Errorf("path starts with unescaped '.'")
	}

	// Try to guess the length of our segments slice to avoid allocations. Performance
	// doesn't *really* matter here. But since this is free we might as well.
	segments := make([]string, 0, strings.Count(path, ".")+1)

	// The current segment we are building.
	var segment strings.Builder
	escaped := false
	for i, ch := range path {
		if escaped { // We are currently in an escape sequence
			switch ch {
			case '&', '.': // Valid escaped characters
				segment.WriteRune(ch)
			default:
				return nil, fmt.Errorf("invalid escape sequence '&%c' at byte %d", ch, i-1)
			}
			escaped = false
		} else if ch == '&' { // Start of an escape sequence
			escaped = true
		} else if ch == '.' { // Segment separator
			if i == len(path)-1 {
				return nil, fmt.Errorf("path ends with an unescaped '.'")
			} else if segment.Len() == 0 {
				return nil, fmt.Errorf("empty path segment at byte %d", i)
			}
			// Append this segment and reset our builder
			segments = append(segments, segment.String())
			segment.Reset()
		} else {
			// We are not escaped and this is a normal character. Add it to current segment.
			segment.WriteRune(ch)
		}
	}
	// If we ended while still in an escape sequence, that means trailing '&'.
	if escaped {
		return nil, fmt.Errorf("path ends with an unescaped '&'")
	}
	segments = append(segments, segment.String())
	return segments, nil
}
