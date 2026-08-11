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

package collections

import (
	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("delete-many")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Before(func(t *harness.T) {
		docs := make([]astra.NewDocument, 50)
		for i := 0; i < 50; i++ {
			docs[i] = astra.NewDocument{"age": i}
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed in Before: %v", err)

		empty := make([]astra.NewDocument, 100)
		for i := 0; i < 100; i++ {
			empty[i] = astra.NewDocument{}
		}
		_, err = t.Collection_.InsertMany(t.Ctx, empty)
		testlib.FailIfErr(t, err, "InsertMany failed in Before for collection_: %v", err)
	})

	s.Run("should deleteMany when match is <= 20", func(t *harness.T) {
		deleteRes, _ := t.Collection.DeleteMany(t.Ctx, filter.Lt("age", 10))
		testlib.FailIf(t, deleteRes.DeletedCount != 10, "expected DeletedCount to be 10, got %d", deleteRes.DeletedCount)
	})

	s.Run("should deleteMany when match is > 20", func(t *harness.T) {
		deleteRes, _ := t.Collection.DeleteMany(t.Ctx, filter.Gte("age", 10))
		testlib.FailIf(t, deleteRes.DeletedCount != 40, "expected DeletedCount to be 40, got %d", deleteRes.DeletedCount)
	})

	s.Run("should delete all documents given an empty filter", func(t *harness.T) {
		deleteRes, err := t.Collection_.DeleteMany(t.Ctx, filter.F{})
		testlib.FailIfErr(t, err, "DeleteMany with empty filter failed: %v", err)
		testlib.FailIf(t, deleteRes.DeletedCount != -1, "expected DeletedCount to be -1 for empty filter, got %d", deleteRes.DeletedCount)

		count, err := t.Collection_.CountDocuments(t.Ctx, filter.F{}, 1000)
		testlib.FailIfErr(t, err, "CountDocuments failed: %v", err)
		testlib.FailIf(t, count != 0, "expected 0 documents after deleteMany with empty filter, got %d", count)
	})

	s.Run("fails gracefully on 2XX exceptions", func(t *harness.T) {
		_, err := t.Collection.DeleteMany(t.Ctx, filter.F{"$invalid": 3})
		testlib.FailIf(t, err == nil, "expected error for invalid filter")
		testlib.ErrMustBe[*results.DataAPIError](t, err, "expected DataAPIError for invalid filter")
	})
}
