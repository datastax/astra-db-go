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

package sort_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/datastax/astra-db-go/internal/testutils"
	"github.com/datastax/astra-db-go/sort"
)

// Compile-time interface checks
var _ sort.Sortable = sort.Sort{}
var _ sort.Sortable = sort.S{}
var _ sort.Sortable = sort.Clauses{}

// TestAscDescJSON verifies ascending/descending field sort JSON parity with
// the Data API curl examples:
//
//	curl ... --data '{ "find": { "sort": {"rating": 1} } }'
//	curl ... --data '{ "find": { "sort": {"title": -1} } }'
func TestAscDescJSON(t *testing.T) {
	tests := []testutils.JSONTestCase{{
		Name:     "ascending",
		Expected: `{"rating":1}`,
		Args: []any{
			sort.Asc("rating"),  // fluent
			sort.S{"rating": 1}, // raw map
		},
	}, {
		Name:     "descending",
		Expected: `{"title":-1}`,
		Args: []any{
			sort.Desc("title"),  // fluent
			sort.S{"title": -1}, // raw map
		},
	}}
	testutils.RunJSONTestCases(t, tests)
}

// TestMultiFieldOrderPreservation verifies that multi-field sorts preserve
// insertion order, which is critical because JSON object key order matters
// for the Data API sort specification.
func TestMultiFieldOrderPreservation(t *testing.T) {
	tests := []testutils.JSONTestCase{{
		Name:     "test from github discussion",
		Expected: `{"field1":1,"field2":-1}`,
		Args: []any{
			sort.Clauses{sort.S{"field1": 1}, sort.S{"field2": -1}},
			sort.By("field1", 1).By("field2", -1),
			sort.By("field1", sort.Ascending).By("field2", sort.Descending),
			sort.Asc("field1").Desc("field2"),
		},
	}, {
		Name:     "multi-field sort preserves order",
		Expected: `{"title":1,"rating":-1}`,
		Args: []any{
			sort.Asc("title").Desc("rating"),
			sort.Clauses{sort.S{"title": 1}, sort.S{"rating": -1}}, // also test raw slice/map
		},
	}, {
		Name:     "funky sort",
		Expected: `{"title":1,"rating":-1}`,
		Args: []any{
			sort.Asc("title").Desc("rating"),
			sort.Clauses{sort.S{"title": 1}, sort.S{"rating": -1}}, // also test raw slice/map
		},
	}}
	testutils.RunJSONTestCases(t, tests)
}

// TestVectorSortJSON verifies vector sort JSON parity with the docs:
//
//	curl ... --data '{ "find": { "sort": {"$vector": [0.1, 0.2, 0.3]} } }'
func TestVectorSortJSON(t *testing.T) {
	s := sort.Vector([]float32{0.1, 0.2, 0.3})
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	expected := `{"$vector":[0.1,0.2,0.3]}`
	if string(got) != expected {
		t.Errorf("got %s, want %s", string(got), expected)
	}
}

// TestVectorizeSortJSON verifies vectorize sort JSON parity with the docs:
//
//	curl ... --data '{ "find": { "sort": {"$vectorize": "search text"} } }'
func TestVectorizeSortJSON(t *testing.T) {
	s := sort.Vectorize("search text")
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	expected := `{"$vectorize":"search text"}`
	if string(got) != expected {
		t.Errorf("got %s, want %s", string(got), expected)
	}
}

// TestLexicalSortJSON verifies lexical sort JSON.
func TestLexicalSortJSON(t *testing.T) {
	s := sort.Lexical("find books about space")
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	expected := `{"$lexical":"find books about space"}`
	if string(got) != expected {
		t.Errorf("got %s, want %s", string(got), expected)
	}
}

// TestRawMapSortJSON verifies that S (raw map) produces valid JSON.
func TestRawMapSortJSON(t *testing.T) {
	s := sort.S{"$vector": []float32{0.1, 0.2, 0.3}}
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	expected := `{"$vector":[0.1,0.2,0.3]}`
	if string(got) != expected {
		t.Errorf("got %s, want %s", string(got), expected)
	}
}

// TestEmptySortMarshal verifies that an empty Sort marshals to null.
func TestEmptySortMarshal(t *testing.T) {
	s := sort.Sort{}
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	if string(got) != "null" {
		t.Errorf("got %s, want null", string(got))
	}
}

func dataAPIVectorTests() []testutils.JSONTestCase {
	return []testutils.JSONTestCase{{
		// https://docs.datastax.com/en/astra-db-serverless/api-reference/sort-rows.html#example-sorting-against-a-search-vector
		Name:     "example from docs",
		Expected: `{"$binary":"PaPXCr8euFI+x64U"}`,
		Args: []any{
			sort.DataAPIVector{Values: []float32{0.08, -0.62, 0.39}},
		},
	}, {
		// https://github.com/datastax/astra-db-csharp/issues/67
		Name:     "example from .NET client issue",
		Expected: `{"$binary":"PczMzb5MzM0+mZma"}`,
		Args: []any{
			sort.DataAPIVector{Values: []float32{0.10000000149011612, -0.20000000298023224, 0.30000001192092896}},
		},
	}, {
		Name:     "empty vector",
		Expected: `{"$binary":""}`,
		Args: []any{
			sort.DataAPIVector{Values: []float32{}},
		},
	}}
}

func TestDataAPIVectorMarshalJSON(t *testing.T) {
	tests := dataAPIVectorTests()
	testutils.RunJSONTestCases(t, tests)
}

func TestDataAPIVectorUnmarshalJSON(t *testing.T) {
	// Re-using this test case struct even though the names are SLIGHTLY off in this context.
	tests := dataAPIVectorTests()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var v sort.DataAPIVector
			err := json.Unmarshal([]byte(tt.Expected), &v)
			if err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}
			if !slices.Equal(v.Values, tt.Args[0].(sort.DataAPIVector).Values) {
				t.Errorf("got %v, want %v", v.Values, tt.Args[0].(sort.DataAPIVector).Values)
			}
		})
	}
	testutils.RunJSONTestCases(t, tests)
}
