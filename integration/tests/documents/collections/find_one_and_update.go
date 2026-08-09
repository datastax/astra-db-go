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
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/astra/update"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("find-one-and-update")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should findOneAndUpdate", func(t *harness.T) {
		// Insert a document
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "old", "age": 0})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// FindOneAndUpdate with returnDocument after
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("_id", docID),
			update.Coll().Set("name", "new").Unset("age"),
			options.CollectionFindOneAndUpdate().
				SetReturnDocument(options.ReturnDocumentAfter),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != docID, "expected _id to be %v, got %v", docID, doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("name").(string) != "new", "expected name to be 'new', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age to be unset, but it exists")
	})

	s.Run("should findOneAndUpdate with returnDocument before", func(t *harness.T) {
		// Insert a document
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "old", "age": 0})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// FindOneAndUpdate with returnDocument before
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("_id", docID),
			update.Coll().Set("name", "new").Unset("age"),
			options.CollectionFindOneAndUpdate().
				SetReturnDocument(options.ReturnDocumentBefore),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != docID, "expected _id to be %v, got %v", docID, doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("name").(string) != "old", "expected name to be 'old', got %v", doc.MustGet("name"))
		testlib.FailIf(t, doc.MustGet("age").(float64) != 0, "expected age to be 0, got %v", doc.MustGet("age"))
	})

	s.Run("should findOneAndUpdate with upsert true", func(t *harness.T) {
		// FindOneAndUpdate with upsert (document doesn't exist)
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.F{"_id": t.Key(0)},
			update.Coll().Set("name", "new").Unset("age"),
			options.CollectionFindOneAndUpdate().
				SetReturnDocument(options.ReturnDocumentAfter).
				SetUpsert(true),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err := result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != t.Key(0), "expected _id to be %v, got %v", t.Key(0), doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("name").(string) != "new", "expected name to be 'new', got %v", doc.MustGet("name"))
		_, hasAddress := doc.Get("address")
		testlib.FailIf(t, hasAddress, "expected address to not exist, but it does")
	})

	s.Run("should findOneAndUpdate with upsert true and returnDocument before", func(t *harness.T) {
		// FindOneAndUpdate with upsert and returnDocument before (document doesn't exist)
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.F{"_id": t.Key(0)},
			update.Coll().Set("name", "new").Unset("age"),
			options.CollectionFindOneAndUpdate().
				SetReturnDocument(options.ReturnDocumentBefore).
				SetUpsert(true),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err := result.Decode(&doc)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got %v", err)
	})

	s.Run("should findOneAndUpdate without any updates to apply", func(t *harness.T) {
		// Insert a document
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndUpdate with no actual changes
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "a"),
			options.CollectionFindOneAndUpdate().
				SetSort(sort.Asc("name")),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("name").(string) != "a", "expected name to be 'a', got %v", doc.MustGet("name"))
	})

	s.Run("should findOneAndUpdate with a projection", func(t *harness.T) {
		// Insert documents
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "age": 42, "key": t.Key(0)},
			{"name": "aa", "age": 42, "key": t.Key(0)},
			{"name": "aaa", "age": 42, "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndUpdate with projection
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
			update.Coll().Set("name", "b"),
			options.CollectionFindOneAndUpdate().
				SetProjection(map[string]any{"name": 1}).
				SetReturnDocument(options.ReturnDocumentAfter),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("name").(string) != "b", "expected name to be 'b', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age to not be in projection, but it exists")
	})

	s.Run("should findOneAndUpdate with sort", func(t *harness.T) {
		// Insert documents
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
			{"name": "c", "key": t.Key(0)},
			{"name": "b", "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndUpdate with ascending sort
		result1 := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "aaa"),
			options.CollectionFindOneAndUpdate().
				SetSort(sort.Asc("name")),
		)
		testlib.FailIfErr(t, result1.Err(), "FindOneAndUpdate failed: %v", result1.Err())

		var doc1 astra.Document
		err = result1.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc1.MustGet("name").(string) != "a", "expected name to be 'a', got %v", doc1.MustGet("name"))

		// FindOneAndUpdate with descending sort
		result2 := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "ccc"),
			options.CollectionFindOneAndUpdate().
				SetSort(sort.Desc("name")),
		)
		testlib.FailIfErr(t, result2.Err(), "FindOneAndUpdate failed: %v", result2.Err())

		var doc2 astra.Document
		err = result2.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc2.MustGet("name").(string) != "c", "expected name to be 'c', got %v", doc2.MustGet("name"))
	})

	s.Run("should not return metadata when includeResultMetadata is false", func(t *harness.T) {
		// Insert a document
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "a", "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// FindOneAndUpdate
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
			update.Coll().Set("name", "b"),
			options.CollectionFindOneAndUpdate().
				SetReturnDocument(options.ReturnDocumentAfter),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		// Verify document has expected fields
		testlib.FailIf(t, doc.MustGet("name").(string) != "b", "expected name to be 'b', got %v", doc.MustGet("name"))
		testlib.FailIf(t, doc.MustGet("key") != t.Key(0), "expected key to be %v, got %v", t.Key(0), doc.MustGet("key"))
		_, hasID := doc.Get("_id")
		testlib.FailIf(t, !hasID, "expected _id to exist")
	})

	s.Run("should not return metadata by default", func(t *harness.T) {
		// Insert a document
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "a", "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// FindOneAndUpdate
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
			update.Coll().Set("name", "b"),
			options.CollectionFindOneAndUpdate().
				SetReturnDocument(options.ReturnDocumentAfter),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		// Verify document has expected fields
		testlib.FailIf(t, doc.MustGet("name").(string) != "b", "expected name to be 'b', got %v", doc.MustGet("name"))
		testlib.FailIf(t, doc.MustGet("key") != t.Key(0), "expected key to be %v, got %v", t.Key(0), doc.MustGet("key"))
		_, hasID := doc.Get("_id")
		testlib.FailIf(t, !hasID, "expected _id to exist")
	})

	s.Run("should findOneAndUpdate with $vector sort", func(t *harness.T) {
		// Insert documents with vectors
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "$vector": datatypes.NewVector([]float32{1.0, 1.0, 1.0, 1.0, 1.0}), "key": t.Key(0)},
			{"name": "c", "$vector": datatypes.NewVector([]float32{-0.1, -0.2, -0.3, -0.4, -0.5}), "key": t.Key(0)},
			{"name": "b", "$vector": datatypes.NewVector([]float32{-0.1, -0.2, -0.3, -0.4, -0.5}), "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndUpdate with vector sort
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "aaa"),
			options.CollectionFindOneAndUpdate().
				SetSort(sort.Vector([]float32{1, 1, 1, 1, 1})),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("name").(string) != "a", "expected name to be 'a', got %v", doc.MustGet("name"))
	})

	s.Run("should return null if no document is found", func(t *harness.T) {
		// FindOneAndUpdate with no matching document
		result := t.Collection.FindOneAndUpdate(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("car", "bus"),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndUpdate failed: %v", result.Err())

		var doc astra.Document
		err := result.Decode(&doc)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got %v", err)
	})

	s.Run("should work with datatypes", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"_id": t.Key(0), "name": "a", "key": t.Key(0), "uuid": datatypes.NewUUID()},
			{"_id": t.Key(1), "name": "b", "key": t.Key(0), "oid": datatypes.NewObjectId()},
			{"_id": t.Key(2), "name": "c", "key": t.Key(0), "date": time.Now().Truncate(time.Millisecond)},
			{"_id": t.Key(3), "name": "d", "key": t.Key(0), "$vector": datatypes.NewVector([]float32{0.1, 0.2, 0.3, 0.4, 0.5})},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		expected := append(docs, nil)

		for i := range docs {
			res := t.Collection.FindOneAndUpdate(
				t.Ctx,
				filter.And(filter.Eq("key", t.Key(0)), filter.Ne("touched", 1)),
				update.Coll().Set("touched", 1),
				options.CollectionFindOneAndUpdate().
					SetSort(sort.Asc("name")).
					SetProjection(map[string]any{"$vector": 1, "touched": 0}),
			)

			var got astra.Document
			err = res.Decode(&got)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			testlib.NoDiff(t, expected[i].ToMap(), got.ToMap())
		}
	})
}
