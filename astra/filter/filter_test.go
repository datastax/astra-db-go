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
	"regexp"
	"testing"

	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/serdes"
)

// cleanString removes all whitespace characters from a string.
func cleanString(s string) string {
	// Use a regular expression to replace all whitespace characters (including spaces, tabs, newlines)
	// with an empty string.
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}

func notExpected[T any](t *testing.T, expected T, got T) {
	t.Errorf("\nExpected: %v\nGot: %v", expected, got)
}

// This is the example we are testing:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/filter-operator-collections.html#combine-operators-and-or
const TestCombineOperatorsAndOrExpected = `{
    "$and": [
    	{
    		"$or": [
    			{ "is_checked_out": false },
    			{ "number_of_pages": { "$lt": 300 } }
    		]
    	},
    	{
    		"$or": [
    			{ "genres": { "$in": [ "Fantasy", "Romance" ] }},
    			{ "publication_year": { "$gte": 2002 } }
    		]
    	}
    ]
}`

func TestCombineOperatorsAndOrF(t *testing.T) {
	filters := filter.F{
		"$and": filter.A{
			filter.F{"$or": filter.A{
				filter.F{"is_checked_out": false},
				filter.F{"number_of_pages": filter.F{"$lt": 300}},
			}},
			filter.F{"$or": filter.A{
				filter.F{"genres": filter.F{"$in": filter.A{"Fantasy", "Romance"}}},
				filter.F{"publication_year": filter.F{"$gte": 2002}},
			}},
		},
	}
	got, err := serdes.Serialize(filters, serdes.TargetCollection, 0)
	if err != nil {
		t.Error(err)
	}
	// When comparing, ignore whitespace.
	if cleanString(string(got)) != cleanString(TestCombineOperatorsAndOrExpected) {
		notExpected(t, TestCombineOperatorsAndOrExpected, string(got))
	}
}

func TestCombineOperatorsAndOrStructured(t *testing.T) {
	filters := filter.And(
		filter.Or(
			filter.Eq("is_checked_out", false),
			filter.Lt("number_of_pages", 300),
		),
		filter.Or(
			filter.In("genres", "Fantasy", "Romance"),
			filter.Gte("publication_year", 2002),
		),
	)
	got, err := serdes.Serialize(filters, serdes.TargetCollection, 0)
	if err != nil {
		t.Error(err)
	}
	// When comparing, ignore whitespace.
	if cleanString(string(got)) != cleanString(TestCombineOperatorsAndOrExpected) {
		notExpected(t, TestCombineOperatorsAndOrExpected, string(got))
	}
}

func TestEqDefault(t *testing.T) {
	composedFilters := filter.Eq("num_pages", 300)
	filters := filter.F{"num_pages": 300}
	got, err := serdes.Serialize(filters, serdes.TargetCollection, 0)
	if err != nil {
		t.Error(err)
	}
	expected := `{"num_pages":300}`
	if string(got) != expected {
		notExpected(t, expected, string(got))
	}
	composed, err := serdes.Serialize(composedFilters, serdes.TargetCollection, 0)
	if err != nil {
		t.Error(err)
	}
	if string(composed) != expected {
		notExpected(t, expected, string(composed))
	}
}

func TestOrSingleChild(t *testing.T) {
	f := filter.Or(filter.Eq("x", 1))
	got, err := serdes.Serialize(f, serdes.TargetCollection, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"$or":[{"x":1}]}`
	if string(got) != expected {
		notExpected(t, expected, string(got))
	}
}

func TestEmptyFilterMarshal(t *testing.T) {
	f := filter.Filter{}
	got, err := serdes.Serialize(f, serdes.TargetCollection, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "null" {
		notExpected(t, "null", string(got))
	}
}

// Docs example for lexical match operator:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-one.html#use-lexicographical-matching-to-find-a-document
func TestLexicalMatch(t *testing.T) {
	f := filter.LexicalMatch("tree hill")
	got, err := serdes.Serialize(f, serdes.TargetCollection, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"$lexical":{"$match":"tree hill"}}`
	if string(got) != expected {
		notExpected(t, expected, string(got))
	}
}
