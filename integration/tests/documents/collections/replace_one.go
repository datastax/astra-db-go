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
	"github.com/datastax/astra-db-go/v2/astra/cursors"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("replace-one")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should replaceOne", func(t *harness.T) {
		// Insert a document
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "deep_purple", "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Replace the document
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.And(filter.Eq("_id", docID), filter.Eq("key", t.Key(0))),
			astra.NewDocument{"name": "shallow_yellow_green"},
		)
		testlib.FailIfErr(t, err, "ReplaceOne failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", replaceRes.ModifiedCount)
	})

	s.Run("should replaceOne with same doc", func(t *harness.T) {
		// Insert a document
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "halestorm", "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Replace with same content
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.And(filter.Eq("_id", docID), filter.Eq("key", t.Key(0))),
			astra.NewDocument{"name": "halestorm", "key": t.Key(0)},
		)
		testlib.FailIfErr(t, err, "ReplaceOne failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", replaceRes.ModifiedCount)
	})

	s.Run("should replaceOne with multiple matches", func(t *harness.T) {
		// Insert multiple documents with same key
		docs := []astra.NewDocument{
			{"key": t.Key(0)},
			{"key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Replace one (should only replace one document)
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			astra.NewDocument{"key": "ignea"},
		)
		testlib.FailIfErr(t, err, "ReplaceOne failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", replaceRes.ModifiedCount)
	})

	s.Run("should replaceOne with upsert true if match", func(t *harness.T) {
		// Insert a document
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"_id": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// Replace with upsert=true (should match and replace)
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.Eq("_id", t.Key(0)),
			astra.NewDocument{"key": t.Key(0)},
			options.CollectionReplaceOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "ReplaceOne failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", replaceRes.ModifiedCount)
		testlib.FailIf(t, replaceRes.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", replaceRes.UpsertedCount)
		testlib.FailIf(t, replaceRes.UpsertedId != nil, "expected UpsertedId to be nil, got %v", replaceRes.UpsertedId)
	})

	s.Run("should replaceOne with upsert true if no match", func(t *harness.T) {
		// Insert a document (but not with the ID we'll search for)
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// Replace with upsert=true (should insert new document)
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.Eq("_id", t.Key(0)),
			astra.NewDocument{"key": t.Key(0)},
			options.CollectionReplaceOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "ReplaceOne failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 0, "expected MatchedCount to be 0, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", replaceRes.ModifiedCount)
		testlib.FailIf(t, replaceRes.UpsertedCount != 1, "expected UpsertedCount to be 1, got %d", replaceRes.UpsertedCount)
		testlib.FailIf(t, replaceRes.UpsertedId != t.Key(0), "expected UpsertedId to be %v, got %v", t.Key(0), replaceRes.UpsertedId)
	})

	s.Run("should replaceOne with an empty doc", func(t *harness.T) {
		// Insert a document
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Replace with empty document
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
			astra.NewDocument{},
		)
		testlib.FailIfErr(t, err, "ReplaceOne failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", replaceRes.ModifiedCount)
	})

	s.Run("should replaceOne with sort", func(t *harness.T) {
		// Insert multiple documents
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
			{"name": "c", "key": t.Key(0)},
			{"name": "b", "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Replace with sort ascending (should replace 'a')
		res1, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			astra.NewDocument{"name": "aaa", "key": t.Key(0)},
			options.CollectionReplaceOne().SetSort(sort.Asc("name")),
		)
		testlib.FailIfErr(t, err, "ReplaceOne with sort asc failed: %v", err)
		testlib.FailIf(t, res1.MatchedCount != 1, "expected MatchedCount to be 1, got %d", res1.MatchedCount)
		testlib.FailIf(t, res1.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", res1.ModifiedCount)

		// Replace with sort descending (should replace 'c')
		res2, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			astra.NewDocument{"name": "ccc", "key": t.Key(0)},
			options.CollectionReplaceOne().SetSort(sort.Desc("name")),
		)
		testlib.FailIfErr(t, err, "ReplaceOne with sort desc failed: %v", err)
		testlib.FailIf(t, res2.MatchedCount != 1, "expected MatchedCount to be 1, got %d", res2.MatchedCount)
		testlib.FailIf(t, res2.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", res2.ModifiedCount)

		all, err := cursors.DecodeAll[map[string]any](t.Ctx, t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Asc("name")).SetProjection(map[string]any{"_id": 0, "name": 1}),
		))

		t.NoDiff([]map[string]any{
			{"name": "aaa"},
			{"name": "b"},
			{"name": "ccc"},
		}, all)
	})

	s.Run("should replaceOne with $vector sort", func(t *harness.T) {
		// Insert documents with vectors
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "$vector": datatypes.NewVector([]float32{1.0, 1.0, 1.0, 1.0, 1.0}), "key": t.Key(0)},
			{"name": "c", "$vector": datatypes.NewVector([]float32{-0.1, -0.2, -0.3, -0.4, -0.5}), "key": t.Key(0)},
			{"name": "b", "$vector": datatypes.NewVector([]float32{-0.1, -0.2, -0.3, -0.4, -0.5}), "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Replace with vector sort (should replace the document closest to [1,1,1,1,1])
		replaceRes, err := t.Collection.ReplaceOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			astra.NewDocument{"name": "aaa"},
			options.CollectionReplaceOne().SetSort(sort.Vector([]float32{1, 1, 1, 1, 1})),
		)
		testlib.FailIfErr(t, err, "ReplaceOne with vector sort failed: %v", err)

		testlib.FailIf(t, replaceRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", replaceRes.MatchedCount)
		testlib.FailIf(t, replaceRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", replaceRes.ModifiedCount)
	})
}
