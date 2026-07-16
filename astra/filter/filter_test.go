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

	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

// cleanString removes all whitespace characters from a string.
func cleanString(s string) string {
	// Use a regular expression to replace all whitespace characters (including spaces, tabs, newlines)
	// with an empty string.
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}

// This is the example we are testing:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/filter-operator-collections.html#combine-operators-and-or
var TestCombineOperatorsAndOrExpected = cleanString(`{
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
}`)

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
	got, err := serdes.Serialize(filters, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got), TestCombineOperatorsAndOrExpected)
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
	got, err := serdes.Serialize(filters, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got), TestCombineOperatorsAndOrExpected)
}

func TestEqDefault(t *testing.T) {
	filters := filter.F{"num_pages": 300}
	composedFilters := filter.Eq("num_pages", 300)

	raw, err := serdes.Serialize(filters, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(raw), `{"num_pages":300}`)

	composed, err := serdes.Serialize(composedFilters, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(composed), `{"num_pages":300}`)
}

func TestOrSingleChild(t *testing.T) {
	f := filter.Or(filter.Eq("x", 1))
	got, err := serdes.Serialize(f, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got), `{"$or":[{"x":1}]}`)
}

func TestNotSingleChild(t *testing.T) {
	f := filter.Coll.Not(filter.Eq("x", 1))
	got, err := serdes.Serialize(f, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got), `{"$not":{"x":1}}`)
}

func TestEmptyFilterMarshal(t *testing.T) {
	f := filter.Filter{}
	got, err := serdes.Serialize(f, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got), `null`)
}

// Docs example for lexical match operator:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-one.html#use-lexicographical-matching-to-find-a-document
func TestLexicalMatch(t *testing.T) {
	cf := filter.Coll.LexicalMatch("tree hill")
	gotC, err := serdes.Serialize(cf, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(gotC), `{"$lexical":{"$match":"tree hill"}}`)

	ct := filter.Table.LexicalMatch("field", "tree hill")
	gotT, err := serdes.Serialize(ct, serdes.TargetTable)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(gotT), `{"field":{"$match":"tree hill"}}`)
}

func TestTableKeysValues(t *testing.T) {
	f1 := filter.Table.Keys(filter.In("metadata", "Language", "Edition"))
	got1, err := serdes.Serialize(f1, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got1), `{"metadata":{"$keys":{"$in":["Language","Edition"]}}}`)

	f2 := filter.Table.Values(filter.All("metadata", "Language", "Edition"))
	got2, err := serdes.Serialize(f2, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize filter")
	testlib.NoDiff(t, string(got2), `{"metadata":{"$values":{"$all":["Language","Edition"]}}}`)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("filter.Table.Values(filter.Eq(...)) did not panic")
		}
	}()

	_ = filter.Table.Values(filter.Eq("metadata", "Language"))
}
