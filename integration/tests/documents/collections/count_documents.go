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
	"errors"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("count-documents")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	docs := make([]astra.NewDocument, 20)
	for i := 0; i < 20; i++ {
		docs[i] = astra.NewDocument{
			"_id":  i,
			"name": "Bloodywood",
			"age":  i,
		}
	}

	s.Before(func(t *harness.T) {
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		empty := make([]astra.NewDocument, 1001)
		for i := 0; i < 1001; i++ {
			empty[i] = astra.NewDocument{}
		}
		_, err = t.Collection_.InsertMany(t.Ctx, empty, options.CollectionInsertMany().SetOrdered(true).SetChunkSize(100))
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
	})

	s.Run("should return a single doc for an _id filter", func(t *harness.T) {
		count, _ := t.Collection.CountDocuments(t.Ctx, filter.Eq("_id", 0), 1000)
		testlib.FailIf(t, count != 1, "expected count to be 1, got %d", count)
	})

	s.Run("should return count of all documents with no filter", func(t *harness.T) {
		count, _ := t.Collection.CountDocuments(t.Ctx, filter.F{}, 1000)
		testlib.FailIf(t, count != 20, "expected count to be 20, got %d", count)
	})

	s.Run("should return count of documents with filter", func(t *harness.T) {
		count, _ := t.Collection.CountDocuments(t.Ctx, filter.And(filter.Eq("name", "Bloodywood"), filter.Lt("age", 10)), 1000)
		testlib.FailIf(t, count != 10, "expected count to be 10, got %d", count)
	})

	s.Run("should return 0 if no filter matches", func(t *harness.T) {
		count, _ := t.Collection.CountDocuments(t.Ctx, filter.Gt("age", 30), 1000)
		testlib.FailIf(t, count != 0, "expected count to be 0, got %d", count)
	})

	s.Run("should throw an error when # docs over limit", func(t *harness.T) {
		count, err := t.Collection.CountDocuments(t.Ctx, filter.F{}, 1)
		testlib.FailIf(t, err == nil, "expected error when count exceeds limit")
		testlib.FailIf(t, !errors.Is(err, results.ErrTooManyDocumentsToCount), "expected ErrTooManyDocumentsToCount, got %v", err)
		testlib.FailIf(t, count != 1, "expected count to be 1 (the limit), got %d", count)
	})

	s.Run("should throw an error when moreData is returned", func(t *harness.T) {
		count, err := t.Collection_.CountDocuments(t.Ctx, filter.F{}, 2000)
		testlib.FailIf(t, err == nil, "expected error when server returns moreData")
		testlib.FailIf(t, !errors.Is(err, results.ErrTooManyDocumentsToCount), "expected ErrTooManyDocumentsToCount, got %v", err)
		testlib.FailIf(t, count != 1000, "expected count to be 1000 (server limit), got %d", count)
	})
}
