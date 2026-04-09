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

// Package sort provides type-safe sort specifications for Astra DB queries.
//
// Sort specifications control result ordering for find, update, and delete
// operations. They support ascending/descending field sorts, vector similarity
// search, vectorize text search, and lexical text search.
//
// Use the fluent builder for type-safe, order-preserving sorts:
//
//	sort.Asc("rating").Desc("title")
//	sort.Vector([]float32{0.1, 0.2, 0.3})
//	sort.Vectorize("find books about space")
//
// Use [S] as a raw map escape hatch when you need full control:
//
//	sort.S{"$vector": myVec}
package sort

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Ascending is the sort order value for ascending (1).
const Ascending = 1

// Descending is the sort order value for descending (-1).
const Descending = -1

// Sortable is implemented by types that can be used as sort specifications.
// It is satisfied by [Sort] (the fluent builder) and [S] (raw map).
type Sortable interface {
	json.Marshaler
	isSort()
}

// S represents a SINGLE sort specification as a raw map.
// Use this when you need full control over the sort JSON, or when working
// with sort specifications that don't fit the fluent builder.
//
// Example:
//
//	sort.S{"$vector": []float32{0.1, 0.2, 0.3}}
//	sort.S{"rating": 1}
//
// Since maps do not preserve order, S should only be used for single-clause sorts:
//
//	// Don't do this:
//	sort.S{"title": -1, "rating": 1}
//
// Instead, use the fluent builder or [Clauses] for multi-field sorts:
type S map[string]any

func (S) isSort() {}

// MarshalJSON marshals the raw sort map to JSON.
func (s S) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any(s))
}

// Clauses represents multiple sort specifications. Use this when you need
// full control over the sort JSON, or when working with sort
// specifications that don't fit the fluent builder.
//
// Example:
//
//	sort.Clauses{sort.S{"title": -1}, sort.S{"rating": 1}}
type Clauses []S

func (Clauses) isSort() {}

func (s Clauses) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	for _, clause := range s {
		for k, v := range clause {
			if !first {
				buf.WriteByte(',')
			}
			key, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			val, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			buf.Write(key)
			buf.WriteByte(':')
			buf.Write(val)
			first = false
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// clause is a single key-value pair in a sort specification.
type clause struct {
	field string
	value any
}

// Sort is an ordered sort specification built with fluent constructors.
// Unlike maps, Sort preserves the insertion order of clauses, which matters
// for multi-field sorts.
//
// Create a Sort with package-level constructors:
//
//	sort.Asc("rating")                         // ascending field sort
//	sort.Desc("title")                         // descending field sort
//	sort.Vector([]float32{0.1, 0.2, 0.3})     // vector similarity search
//	sort.By("my_vec", myVec)          // custom sort with raw value
//	sort.Vectorize("find books about space")   // vectorize text search
//	sort.Lexical("search query")               // lexical text search
//
// Chain additional clauses:
//
//	sort.Asc("rating").Desc("title")
type Sort struct {
	clauses []clause
}

func (Sort) isSort() {}

// Asc creates a Sort with an ascending clause on the given field.
func Asc(field string) Sort {
	return Sort{clauses: []clause{{field: field, value: Ascending}}}
}

// Desc creates a Sort with a descending clause on the given field.
func Desc(field string) Sort {
	return Sort{clauses: []clause{{field: field, value: Descending}}}
}

// By creates a Sort with a single key-value clause.
func By(field string, value any) Sort {
	return Sort{clauses: []clause{{field: field, value: value}}}
}

// Vector creates a Sort for vector similarity search.
// The vector parameter should be a slice of float32 or float64 values.
func Vector(v any) Sort {
	return Sort{clauses: []clause{{field: "$vector", value: v}}}
}

// Vectorize creates a Sort for vectorize text search.
func Vectorize(text string) Sort {
	return Sort{clauses: []clause{{field: "$vectorize", value: text}}}
}

// Lexical creates a Sort for lexical text search.
func Lexical(text string) Sort {
	return Sort{clauses: []clause{{field: "$lexical", value: text}}}
}

// Asc appends an ascending clause and returns the extended Sort.
func (s Sort) Asc(field string) Sort {
	s.clauses = append(s.clauses, clause{field: field, value: Ascending})
	return s
}

// Desc appends a descending clause and returns the extended Sort.
func (s Sort) Desc(field string) Sort {
	s.clauses = append(s.clauses, clause{field: field, value: Descending})
	return s
}

// By appends a key-value clause and returns the extended Sort.
func (s Sort) By(field string, value any) Sort {
	s.clauses = append(s.clauses, clause{field: field, value: value})
	return s
}

// MarshalJSON writes the sort as a JSON object, preserving clause order.
// An empty Sort marshals to null.
func (s Sort) MarshalJSON() ([]byte, error) {
	if len(s.clauses) == 0 {
		return []byte("null"), nil
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, c := range s.clauses {
		if i > 0 {
			buf.WriteByte(',')
		}
		// Write key
		key, err := json.Marshal(c.field)
		if err != nil {
			return nil, fmt.Errorf("sort: marshal key %q: %w", c.field, err)
		}
		buf.Write(key)
		buf.WriteByte(':')

		// Write value — use strconv for int to avoid float representation
		switch v := c.value.(type) {
		case int:
			buf.WriteString(strconv.Itoa(v))
		default:
			val, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("sort: marshal value for %q: %w", c.field, err)
			}
			buf.Write(val)
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
