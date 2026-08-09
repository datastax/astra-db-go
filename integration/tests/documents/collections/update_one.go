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
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/astra/update"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("update-one")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should updateOne document by id", func(t *harness.T) {
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"age": 3, "key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		updateRes, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.Eq("_id", docID),
			update.Coll().Set("name", "ruoska").Unset("age"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		testlib.FailIf(t, updateRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateRes.ModifiedCount)
		testlib.FailIf(t, updateRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateRes.MatchedCount)
		testlib.FailIf(t, updateRes.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateRes.UpsertedId)
		testlib.FailIf(t, updateRes.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateRes.UpsertedCount)

		found := t.Collection.FindOne(t.Ctx, filter.Eq("key", t.Key(0)))

		var doc astra.Document
		err = found.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != docID, "expected _id to be %v, got %v", docID, doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("name") != "ruoska", "expected name to be 'ruoska', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age field to be removed")
	})

	s.Run("should updateOne document by col", func(t *harness.T) {
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		updateRes, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("age", 3),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		testlib.FailIf(t, updateRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateRes.ModifiedCount)
		testlib.FailIf(t, updateRes.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateRes.MatchedCount)
		testlib.FailIf(t, updateRes.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateRes.UpsertedId)
		testlib.FailIf(t, updateRes.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateRes.UpsertedCount)

		found := t.Collection.FindOne(t.Ctx, filter.Eq("key", t.Key(0)))

		var doc astra.Document
		err = found.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != docID, "expected _id to be %v, got %v", docID, doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("age").(float64) != 3, "expected age to be 3, got %v", doc.MustGet("age"))
	})

	s.Run("should updateOne with sort", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
			{"name": "c", "key": t.Key(0)},
			{"name": "b", "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		_, err = t.Collection.UpdateOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "aa"),
			options.CollectionUpdateOne().SetSort(sort.Asc("name")),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetSort(sort.Asc("name")).SetLimit(20))
		var results []astra.Document
		for cursor.Next(t.Ctx) {
			var doc astra.Document
			err := cursor.Decode(&doc)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			results = append(results, doc)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		testlib.FailIf(t, len(results) != 3, "expected 3 documents, got %d", len(results))
		testlib.FailIf(t, results[0].MustGet("name") != "aa", "expected first name to be 'aa', got %v", results[0].MustGet("name"))
		testlib.FailIf(t, results[1].MustGet("name") != "b", "expected second name to be 'b', got %v", results[1].MustGet("name"))
		testlib.FailIf(t, results[2].MustGet("name") != "c", "expected third name to be 'c', got %v", results[2].MustGet("name"))
	})

	s.Run("should updateOne document by col with sort", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"age": 2, "user": "a", "key": t.Key(0)},
			{"age": 0, "user": "a", "key": t.Key(0)},
			{"age": 1, "user": "a", "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		updateRes, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("user", "a"), filter.Eq("key", t.Key(0))),
			update.Coll().Set("age", 10),
			options.CollectionUpdateOne().SetSort(sort.Asc("age")),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		testlib.FailIf(t, updateRes.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateRes.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetSort(sort.Desc("age")).SetLimit(20))
		var results []astra.Document
		for cursor.Next(t.Ctx) {
			var doc astra.Document
			err := cursor.Decode(&doc)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			results = append(results, doc)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		testlib.FailIf(t, len(results) != 3, "expected 3 documents, got %d", len(results))
		testlib.FailIf(t, results[0].MustGet("age").(float64) != 10, "expected first age to be 10, got %v", results[0].MustGet("age"))
	})

	s.Run("should upsert a doc with upsert flag true in updateOne call", func(t *harness.T) {
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var docID any
		err = res.DecodeID(&docID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		objectId := datatypes.NewObjectId()

		updateRes, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.F{"age": 12, "key": t.Key(0)},
			update.Coll().Set("oid", objectId),
			options.CollectionUpdateOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		testlib.FailIf(t, updateRes.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateRes.ModifiedCount)
		testlib.FailIf(t, updateRes.MatchedCount != 0, "expected MatchedCount to be 0, got %d", updateRes.MatchedCount)
		testlib.FailIf(t, updateRes.UpsertedId == nil, "expected UpsertedId to be set")
		testlib.FailIf(t, updateRes.UpsertedCount != 1, "expected UpsertedCount to be 1, got %d", updateRes.UpsertedCount)

		// Verify the upserted document
		found := t.Collection.FindOne(t.Ctx, filter.Eq("age", 12))

		var doc astra.Document
		err = found.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") == nil, "expected _id to be set")
		testlib.FailIf(t, doc.MustGet("_id") == docID, "expected _id to be different from original, got same")
		testlib.FailIf(t, doc.MustGet("age").(float64) != 12, "expected age to be 12, got %v", doc.MustGet("age"))

		oidVal := doc.MustGet("oid").(datatypes.ObjectId)
		testlib.FailIf(t, oidVal != objectId, "expected oid to match, got %v", oidVal)
	})

	s.Run("should not overwrite user-specified _id in $setOnInsert", func(t *harness.T) {
		updateRes, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.F{"key": t.Key(0)},
			update.Coll().SetOnInsert("_id", "foo"),
			options.CollectionUpdateOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		testlib.FailIf(t, updateRes.UpsertedId != "foo", "expected UpsertedId to be 'foo', got %v", updateRes.UpsertedId)
	})

	s.Run("(LONG) should deserialize the upsertedId property", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "update_one_upsert_id_test", options.CreateCollection().SetDefaultIdType(
			options.CollectionIdTypeObjectId,
		))
		testlib.FailIfErr(t, err, "CreateCollection failed: %v", err)

		updateRes, err := coll.UpdateOne(
			t.Ctx,
			filter.F{"key": t.Key(0)},
			update.Coll().Set("name", "hi"),
			options.CollectionUpdateOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		testlib.FailIf(t, updateRes.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateRes.ModifiedCount)
		testlib.FailIf(t, updateRes.MatchedCount != 0, "expected MatchedCount to be 0, got %d", updateRes.MatchedCount)
		testlib.FailIf(t, updateRes.UpsertedCount != 1, "expected UpsertedCount to be 1, got %d", updateRes.UpsertedCount)

		_, ok := updateRes.UpsertedId.(datatypes.ObjectId)
		testlib.FailIf(t, !ok, "expected UpsertedId to be ObjectID, got %T", updateRes.UpsertedId)
	})
}
