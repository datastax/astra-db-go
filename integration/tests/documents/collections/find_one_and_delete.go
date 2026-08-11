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
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("find-one-and-delete")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should findOneAndDelete", func(t *harness.T) {
		// Insert a document
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "car"})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// FindOneAndDelete
		result := t.Collection.FindOneAndDelete(t.Ctx, filter.Eq("_id", docID))
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != docID, "expected _id to be %v, got %v", docID, doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("name").(string) != "car", "expected name to be 'car', got %v", doc.MustGet("name"))
	})

	s.Run("should findOneAndDelete with a projection", func(t *harness.T) {
		// Insert documents
		docs := []astra.NewDocument{
			{"name": "a", "age": 42, "key": t.Key(0)},
			{"name": "aa", "age": 42, "key": t.Key(0)},
			{"name": "aaa", "age": 42, "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndDelete with projection
		result := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
			options.CollectionFindOneAndDelete().SetProjection(map[string]any{"name": 1}),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("name").(string) != "a", "expected name to be 'a', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age to be excluded from projection")
	})

	s.Run("should findOneAndDelete with sort", func(t *harness.T) {
		// Insert documents
		docs := []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
			{"name": "c", "key": t.Key(0)},
			{"name": "b", "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndDelete with ascending sort
		result1 := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().SetSort(sort.Asc("name")),
		)
		testlib.FailIfErr(t, result1.Err(), "FindOneAndDelete failed: %v", result1.Err())

		var doc1 astra.Document
		err = result1.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("name").(string) != "a", "expected name to be 'a', got %v", doc1.MustGet("name"))

		// FindOneAndDelete with descending sort
		result2 := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().SetSort(sort.Desc("name")),
		)
		testlib.FailIfErr(t, result2.Err(), "FindOneAndDelete failed: %v", result2.Err())

		var doc2 astra.Document
		err = result2.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc2.MustGet("name").(string) != "c", "expected name to be 'c', got %v", doc2.MustGet("name"))
	})

	s.Run("should not return metadata when includeResultMetadata is false", func(t *harness.T) {
		// Insert document
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "a", "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// FindOneAndDelete
		result := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		// Verify document has expected fields
		testlib.FailIf(t, doc.MustGet("name").(string) != "a", "expected name to be 'a'")
		testlib.FailIf(t, doc.MustGet("key").(string) != t.Key(0), "expected key to match")
		_, hasID := doc.Get("_id")
		testlib.FailIf(t, !hasID, "expected _id to be present")
	})

	s.Run("should not return metadata by default", func(t *harness.T) {
		// Insert document
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": "b", "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// FindOneAndDelete
		result := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.And(filter.Eq("name", "b"), filter.Eq("key", t.Key(0))),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		// Verify document has expected fields
		testlib.FailIf(t, doc.MustGet("name").(string) != "b", "expected name to be 'b'")
		testlib.FailIf(t, doc.MustGet("key").(string) != t.Key(0), "expected key to match")
		_, hasID := doc.Get("_id")
		testlib.FailIf(t, !hasID, "expected _id to be present")
	})

	s.Run("should findOneAndDelete with $vector sort", func(t *harness.T) {
		// Insert documents with vectors
		docs := []astra.NewDocument{
			{"name": "a", "$vector": datatypes.NewVector([]float32{1.0, 1.0, 0.9, 1.0, 1.0}), "key": t.Key(0)},
			{"name": "c", "$vector": datatypes.NewVector([]float32{-0.4, -0.2, -0.3, -0.4, -0.1}), "key": t.Key(0)},
			{"name": "b", "$vector": datatypes.NewVector([]float32{-0.1, -0.2, -0.3, -0.4, -0.5}), "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOneAndDelete with vector sort
		result := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().SetSort(sort.Vector([]float32{1, 1, 1, 1, 1})),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("name").(string) != "a", "expected name to be 'a', got %v", doc.MustGet("name"))
	})

	s.Run("should return null if no document is found", func(t *harness.T) {
		// FindOneAndDelete with non-matching filter
		result := t.Collection.FindOneAndDelete(t.Ctx, filter.Eq("key", t.Key(0)))
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err := result.Decode(&doc)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got %v", err)
	})

	s.Run("should work with datatypes", func(t *harness.T) {
		// Create documents with various datatypes
		uuid := datatypes.NewUUID()
		oid := datatypes.NewObjectId()
		now := time.Now().Truncate(time.Millisecond)
		vec := datatypes.NewVector([]float32{0.1, 0.2, 0.3, 0.4, 0.5})

		docs := []astra.NewDocument{
			{"_id": "a" + t.Key(0), "name": "a", "key": t.Key(0), "uuid": uuid},
			{"_id": "b" + t.Key(0), "name": "b", "key": t.Key(0), "oid": oid},
			{"_id": "c" + t.Key(0), "name": "c", "key": t.Key(0), "date": now},
			{"_id": "d" + t.Key(0), "name": "d", "key": t.Key(0), "$vector": vec},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Delete and verify first document (with uuid)
		result := t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().
				SetSort(sort.Asc("name")).
				SetProjection(map[string]any{"*": 1}),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())
		var doc1 astra.Document
		err = result.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("_id").(string) != "a"+t.Key(0), "expected _id to match")
		testlib.FailIf(t, doc1.MustGet("name").(string) != "a", "expected name to be 'a'")
		testlib.FailIf(t, doc1.MustGet("uuid") != uuid, "expected uuid to match")

		// Delete and verify second document (with oid)
		result = t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().
				SetSort(sort.Asc("name")).
				SetProjection(map[string]any{"*": 1}),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())
		var doc2 astra.Document
		err = result.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc2.MustGet("_id").(string) != "b"+t.Key(0), "expected _id to match")
		testlib.FailIf(t, doc2.MustGet("name").(string) != "b", "expected name to be 'b'")
		testlib.FailIf(t, doc2.MustGet("oid") != oid, "expected oid to match")

		// Delete and verify third document (with date)
		result = t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().
				SetSort(sort.Asc("name")).
				SetProjection(map[string]any{"*": 1}),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())
		var doc3 astra.Document
		err = result.Decode(&doc3)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc3.MustGet("_id").(string) != "c"+t.Key(0), "expected _id to match")
		testlib.FailIf(t, doc3.MustGet("name").(string) != "c", "expected name to be 'c'")
		testlib.FailIf(t, doc3.MustGet("date").(time.Time) != now, "expected date to match")

		// Delete and verify fourth document (with vector)
		result = t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().
				SetSort(sort.Asc("name")).
				SetProjection(map[string]any{"*": 1}),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())
		var doc4 astra.Document
		err = result.Decode(&doc4)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc4.MustGet("_id").(string) != "d"+t.Key(0), "expected _id to match")
		testlib.FailIf(t, doc4.MustGet("name").(string) != "d", "expected name to be 'd'")
		gotVec := doc4.MustGet("$vector").(datatypes.Vector)
		gotFloats, err := gotVec.AsFloatArray()
		testlib.FailIfErr(t, err, "AsFloatArray failed: %v", err)
		expectedFloats, err := vec.AsFloatArray()
		testlib.FailIfErr(t, err, "AsFloatArray failed: %v", err)
		t.NoDiff(expectedFloats, gotFloats)

		// Verify no more documents
		result = t.Collection.FindOneAndDelete(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOneAndDelete().
				SetSort(sort.Asc("name")).
				SetProjection(map[string]any{"*": 1}),
		)
		testlib.FailIfErr(t, result.Err(), "FindOneAndDelete failed: %v", result.Err())

		var doc astra.Document
		err = result.Decode(&doc)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got %v", err)
	})
}
