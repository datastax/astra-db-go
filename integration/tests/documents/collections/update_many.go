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
	"fmt"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/update"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("update-many")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should updateMany documents with ids", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"_id": fmt.Sprintf("%s1", t.Key(0)), "age": 1, "key": t.Key(0)},
			{"_id": fmt.Sprintf("%s2", t.Key(0)), "age": 2, "key": t.Key(0)},
			{"_id": fmt.Sprintf("%s3", t.Key(0)), "age": 3, "key": t.Key(0)},
		}
		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected InsertedCount to be %d, got %d", len(docs), res.InsertedCount())

		idToUpdateAndCheck := docs[0]["_id"]
		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("_id", idToUpdateAndCheck),
			update.Coll().Set("name", "aether_realm").Unset("age"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)

		testlib.FailIf(t, updateManyResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateManyResp.UpsertedId)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("name", "aether_realm"), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		testlib.FailIf(t, doc.MustGet("_id") != idToUpdateAndCheck, "expected _id to be %v, got %v", idToUpdateAndCheck, doc.MustGet("_id"))
		testlib.FailIf(t, doc.MustGet("name") != "aether_realm", "expected name to be 'aether_realm', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age field to be removed")
	})

	s.Run("should update when updateMany is invoked with updates for records <= 20", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{"key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "soad"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)

		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 20, "expected ModifiedCount to be 20, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateManyResp.UpsertedId)
	})

	s.Run("should update when updateMany is invoked with updates for records > 20", func(t *harness.T) {
		docList := make([]astra.NewDocument, 101)
		for i := 0; i < 101; i++ {
			docList[i] = astra.NewDocument{"key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Set("name", "soad"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)

		testlib.FailIf(t, updateManyResp.MatchedCount != 101, "expected MatchedCount to be 101, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 101, "expected ModifiedCount to be 101, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateManyResp.UpsertedId)
	})

	s.Run("should upsert with upsert flag set to false/not set when not found", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{"key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", "dream_theater"),
			update.Coll().Set("age", 10),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)

		testlib.FailIf(t, updateManyResp.MatchedCount != 0, "expected MatchedCount to be 0, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateManyResp.UpsertedId)
	})

	s.Run("should upsert with upsert flag set to true when not found", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{"key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.F{"key": "fire"},
			update.Coll().Set("age", 10),
			options.CollectionUpdateMany().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)

		testlib.FailIf(t, updateManyResp.MatchedCount != 0, "expected MatchedCount to be 0, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 1, "expected UpsertedCount to be 1, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId == nil, "expected UpsertedId to be set")
	})

	s.Run("should increment number when $inc is used", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			age := i
			if i == 5 {
				age = 5
			} else if i == 8 {
				age = 8
			}
			docList[i] = astra.NewDocument{
				"_id": fmt.Sprintf("%s%d", t.Key(0), i),
				"age": age,
				"key": t.Key(0),
			}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Inc("age", 1),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)
		testlib.FailIf(t, updateOneResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateOneResp.UpsertedCount)
		testlib.FailIf(t, updateOneResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateOneResp.UpsertedId)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("age").(float64) != 6, "expected age to be 6, got %v", doc.MustGet("age"))

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Inc("age", 1),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 20, "expected ModifiedCount to be 20, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateManyResp.UpsertedId)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 20, "expected 20 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			docID := d.MustGet("_id").(string)
			docIdNum := 0
			fmt.Sscanf(docID[len(t.Key(0)):], "%d", &docIdNum)

			expectedAge := float64(docIdNum + 1)
			if docIdNum == 5 {
				expectedAge = 7
			} else if docIdNum == 8 {
				expectedAge = 9
			}

			testlib.FailIf(t, d.MustGet("age").(float64) != expectedAge, "expected age to be %v for doc %s, got %v", expectedAge, docID, d.MustGet("age"))
		}
	})

	s.Run("should increment decimal when $inc is used", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			age := float64(i) + 0.5
			if i == 5 {
				age = 5.5
			} else if i == 8 {
				age = 8.5
			}
			docList[i] = astra.NewDocument{
				"_id": fmt.Sprintf("%s%d", t.Key(0), i),
				"age": age,
				"key": t.Key(0),
			}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Inc("age", 1),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("age").(float64) != 6.5, "expected age to be 6.5, got %v", doc.MustGet("age"))

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Inc("age", 1),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 20, "expected ModifiedCount to be 20, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 20, "expected 20 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			docID := d.MustGet("_id").(string)
			docIdNum := 0
			fmt.Sscanf(docID[len(t.Key(0)):], "%d", &docIdNum)

			expectedAge := float64(docIdNum) + 0.5 + 1
			if docIdNum == 5 {
				expectedAge = 7.5
			} else if docIdNum == 8 {
				expectedAge = 9.5
			}

			testlib.FailIf(t, d.MustGet("age").(float64) != expectedAge, "expected age to be %v for doc %s, got %v", expectedAge, docID, d.MustGet("age"))
		}
	})

	s.Run("should rename a field when $rename is used in update and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{
				"_id": fmt.Sprintf("%s%d", t.Key(0), i),
				"key": t.Key(0),
			}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Rename("key", "name"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("name", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("name") != t.Key(0), "expected name to be %v, got %v", t.Key(0), doc.MustGet("name"))
		_, hasKey := doc.Get("key")
		testlib.FailIf(t, hasKey, "expected key field to be removed")

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Rename("key", "name"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 19, "expected MatchedCount to be 19, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 19, "expected ModifiedCount to be 19, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("name", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 20, "expected 20 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			testlib.FailIf(t, d.MustGet("name") != t.Key(0), "expected name to be %v, got %v", t.Key(0), d.MustGet("name"))
			_, hasKey := d.Get("key")
			testlib.FailIf(t, hasKey, "expected key field to be removed")
		}
	})

	s.Run("should rename a sub doc field when $rename is used in update and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{
				"_id":    fmt.Sprintf("%s%d", t.Key(0), i),
				"key":    t.Key(0),
				"nested": map[string]any{"key": t.Key(0)},
			}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Rename("nested.key", "nested.name"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		nested := doc.MustGet("nested").(map[string]any)
		testlib.FailIf(t, nested["name"] != t.Key(0), "expected nested.name to be %v, got %v", t.Key(0), nested["name"])
		_, hasKey := nested["key"]
		testlib.FailIf(t, hasKey, "expected nested.key field to be removed")

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Rename("nested.key", "nested.name"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 19, "expected ModifiedCount to be 19, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 20, "expected 20 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			nested := d.MustGet("nested").(map[string]any)
			testlib.FailIf(t, nested["name"] != t.Key(0), "expected nested.name to be %v, got %v", t.Key(0), nested["name"])
			_, hasKey := nested["key"]
			testlib.FailIf(t, hasKey, "expected nested.key field to be removed")
		}
	})

	s.Run("should set date to current date in the fields inside $currentDate in update and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().CurrentDate("date"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("date") == nil, "expected date field to be set")

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().CurrentDate("date"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 20, "expected ModifiedCount to be 20, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 20, "expected 20 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			testlib.FailIf(t, d.MustGet("date") == nil, "expected date field to be set")
		}
	})

	s.Run("should set fields under $setOnInsert when upsert is true in updateOne", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "key": t.Key(0), "name": "idk"}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.F{"_id": fmt.Sprintf("%s5", t.Key(0)), "key": t.Key(0)},
			update.Coll().Set("name", "rammstein").SetOnInsert("age", 20),
			options.CollectionUpdateOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)
		testlib.FailIf(t, updateOneResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateOneResp.UpsertedCount)
		testlib.FailIf(t, updateOneResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateOneResp.UpsertedId)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("name") != "rammstein", "expected name to be 'rammstein', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age field to not be set")

		updateOneResp1, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.F{"_id": fmt.Sprintf("%s21", t.Key(0)), "key": t.Key(0)},
			update.Coll().Set("name", "rammstein").SetOnInsert("age", 20),
			options.CollectionUpdateOne().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp1.MatchedCount != 0, "expected MatchedCount to be 0, got %d", updateOneResp1.MatchedCount)
		testlib.FailIf(t, updateOneResp1.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateOneResp1.ModifiedCount)
		testlib.FailIf(t, updateOneResp1.UpsertedCount != 1, "expected UpsertedCount to be 1, got %d", updateOneResp1.UpsertedCount)
		testlib.FailIf(t, updateOneResp1.UpsertedId != fmt.Sprintf("%s21", t.Key(0)), "expected UpsertedId to be %s21, got %v", t.Key(0), updateOneResp1.UpsertedId)

		updatedDoc1 := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s21", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc1 astra.Document
		err = updatedDoc1.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("name") != "rammstein", "expected name to be 'rammstein', got %v", doc1.MustGet("name"))
		testlib.FailIf(t, doc1.MustGet("age").(float64) != 20, "expected age to be 20, got %v", doc1.MustGet("age"))
	})

	s.Run("should set fields under $setOnInsert when upsert is true in updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "key": t.Key(0), "name": "idk"}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.F{"_id": fmt.Sprintf("%s5", t.Key(0)), "key": t.Key(0)},
			update.Coll().Set("name", "rammstein").SetOnInsert("age", 20),
			options.CollectionUpdateMany().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateManyResp.ModifiedCount)
		testlib.FailIf(t, updateManyResp.UpsertedCount != 0, "expected UpsertedCount to be 0, got %d", updateManyResp.UpsertedCount)
		testlib.FailIf(t, updateManyResp.UpsertedId != nil, "expected UpsertedId to be nil, got %v", updateManyResp.UpsertedId)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s5", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("name") != "rammstein", "expected name to be 'rammstein', got %v", doc.MustGet("name"))
		_, hasAge := doc.Get("age")
		testlib.FailIf(t, hasAge, "expected age field to not be set")

		updateManyResp1, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.F{"_id": fmt.Sprintf("%s21", t.Key(0)), "key": t.Key(0)},
			update.Coll().Set("name", "rammstein").SetOnInsert("age", 20),
			options.CollectionUpdateMany().SetUpsert(true),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp1.MatchedCount != 0, "expected MatchedCount to be 0, got %d", updateManyResp1.MatchedCount)
		testlib.FailIf(t, updateManyResp1.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp1.ModifiedCount)
		testlib.FailIf(t, updateManyResp1.UpsertedCount != 1, "expected UpsertedCount to be 1, got %d", updateManyResp1.UpsertedCount)
		testlib.FailIf(t, updateManyResp1.UpsertedId != fmt.Sprintf("%s21", t.Key(0)), "expected UpsertedId to be %s21, got %v", t.Key(0), updateManyResp1.UpsertedId)

		updatedDoc1 := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s21", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc1 astra.Document
		err = updatedDoc1.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("name") != "rammstein", "expected name to be 'rammstein', got %v", doc1.MustGet("name"))
		testlib.FailIf(t, doc1.MustGet("age").(float64) != 20, "expected age to be 20, got %v", doc1.MustGet("age"))
	})

	s.Run("should set a field value to new value when the new value is < existing value with $min in updateOne and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			age := 50
			if i == 4 {
				age = 10
			}
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "age": age, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Min("age", 5),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("age").(float64) != 5, "expected age to be 5, got %v", doc.MustGet("age"))

		updateOneResp1, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Min("age", 15),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp1.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp1.MatchedCount)
		testlib.FailIf(t, updateOneResp1.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateOneResp1.ModifiedCount)

		updatedDoc1 := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc1 astra.Document
		err = updatedDoc1.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("age").(float64) != 5, "expected age to be 5, got %v", doc1.MustGet("age"))

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Min("age", 15),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 19, "expected ModifiedCount to be 19, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		for _, d := range allDocs {
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, d.MustGet("age").(float64) != 5, "expected age to be 5, got %v", d.MustGet("age"))
			} else {
				testlib.FailIf(t, d.MustGet("age").(float64) != 15, "expected age to be 15, got %v", d.MustGet("age"))
			}
		}

		updateManyResp1, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Min("age", 50),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp1.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp1.MatchedCount)
		testlib.FailIf(t, updateManyResp1.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp1.ModifiedCount)

		cursor = t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		allDocs = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		for _, d := range allDocs {
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, d.MustGet("age").(float64) != 5, "expected age to be 5, got %v", d.MustGet("age"))
			} else {
				testlib.FailIf(t, d.MustGet("age").(float64) != 15, "expected age to be 15, got %v", d.MustGet("age"))
			}
		}
	})

	s.Run("should set a field value to new value when the new value is > existing value with $max in updateOne and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 20)
		for i := 0; i < 20; i++ {
			age := 800
			if i == 4 {
				age = 900
			}
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "age": age, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Max("age", 950),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("age").(float64) != 950, "expected age to be 950, got %v", doc.MustGet("age"))

		updateOneResp1, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Max("age", 15),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp1.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp1.MatchedCount)
		testlib.FailIf(t, updateOneResp1.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateOneResp1.ModifiedCount)

		updatedDoc1 := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc1 astra.Document
		err = updatedDoc1.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("age").(float64) != 950, "expected age to be 950, got %v", doc1.MustGet("age"))

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Max("age", 900),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 19, "expected ModifiedCount to be 19, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		for _, d := range allDocs {
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, d.MustGet("age").(float64) != 950, "expected age to be 950, got %v", d.MustGet("age"))
			} else {
				testlib.FailIf(t, d.MustGet("age").(float64) != 900, "expected age to be 900, got %v", d.MustGet("age"))
			}
		}

		updateManyResp1, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Max("age", 50),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp1.MatchedCount != 20, "expected MatchedCount to be 20, got %d", updateManyResp1.MatchedCount)
		testlib.FailIf(t, updateManyResp1.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp1.ModifiedCount)

		cursor = t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(20))
		allDocs = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		for _, d := range allDocs {
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, d.MustGet("age").(float64) != 950, "expected age to be 950, got %v", d.MustGet("age"))
			} else {
				testlib.FailIf(t, d.MustGet("age").(float64) != 900, "expected age to be 900, got %v", d.MustGet("age"))
			}
		}
	})

	s.Run("should multiply a value by number provided for each field in the $mul in updateOne and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "age": 50, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Mul("age", 1.07),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc.MustGet("age").(float64) != 53.5, "expected age to be 53.5, got %v", doc.MustGet("age"))

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.And(filter.In("_id", fmt.Sprintf("%s0", t.Key(0)), fmt.Sprintf("%s1", t.Key(0)), fmt.Sprintf("%s2", t.Key(0)), fmt.Sprintf("%s3", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Mul("age", 1.07),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 4, "expected MatchedCount to be 4, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 4, "expected ModifiedCount to be 4, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		for _, d := range allDocs {
			testlib.FailIf(t, d.MustGet("age").(float64) != 53.5, "expected age to be 53.5, got %v", d.MustGet("age"))
		}
	})

	s.Run("should push an element to an array when an item is added using $push", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "arr": []string{"tag1", "tag2"}, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Push("arr", "tag3"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr := doc.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
		testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Push("arr", "tag3"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 5, "expected ModifiedCount to be 5, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
				testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
				testlib.FailIf(t, arr[3] != "tag3", "expected arr[3] to be 'tag3', got %v", arr[3])
			} else {
				testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
				testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
			}
		}
	})

	s.Run("should push an element to an array when each item in $each is added using $push with $position", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "arr": []string{"tag1", "tag2"}, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().PushEachPosition("arr", 1, "tag3", "tag4"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr := doc.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
		testlib.FailIf(t, arr[1] != "tag3", "expected arr[1] to be 'tag3', got %v", arr[1])
		testlib.FailIf(t, arr[2] != "tag4", "expected arr[2] to be 'tag4', got %v", arr[2])

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().PushEachPosition("arr", 1, "tag3", "tag4"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 5, "expected ModifiedCount to be 5, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, len(arr) != 6, "expected arr length to be 6, got %d", len(arr))
				testlib.FailIf(t, arr[1] != "tag3", "expected arr[1] to be 'tag3', got %v", arr[1])
				testlib.FailIf(t, arr[2] != "tag4", "expected arr[2] to be 'tag4', got %v", arr[2])
				testlib.FailIf(t, arr[3] != "tag3", "expected arr[3] to be 'tag3', got %v", arr[3])
				testlib.FailIf(t, arr[4] != "tag4", "expected arr[4] to be 'tag4', got %v", arr[4])
			} else {
				testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
				testlib.FailIf(t, arr[1] != "tag3", "expected arr[1] to be 'tag3', got %v", arr[1])
				testlib.FailIf(t, arr[2] != "tag4", "expected arr[2] to be 'tag4', got %v", arr[2])
			}
		}
	})

	s.Run("should push an element to an array skipping duplicates when an item is added using $addToSet", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "arr": []string{"tag1", "tag2"}, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().AddToSet("arr", "tag3"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr := doc.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
		testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])

		updateOneResp2, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().AddToSet("arr", "tag3"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp2.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp2.MatchedCount)
		testlib.FailIf(t, updateOneResp2.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateOneResp2.ModifiedCount)

		updatedDoc2 := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc2 astra.Document
		err = updatedDoc2.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr2 := doc2.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr2) != 3, "expected arr length to be 3, got %d", len(arr2))
		testlib.FailIf(t, arr2[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr2[2])

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().AddToSet("arr", "tag3"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 4, "expected ModifiedCount to be 4, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
			testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
		}

		updateManyResp2, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().AddToSet("arr", "tag3"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp2.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp2.MatchedCount)
		testlib.FailIf(t, updateManyResp2.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp2.ModifiedCount)

		cursor = t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		allDocs = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
			testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
		}
	})

	s.Run("should push an element to an array skipping duplicates when an item is added using $addToSet with $each", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "arr": []string{"tag1", "tag2"}, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().AddToSetEach("arr", "tag3", "tag4"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr := doc.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
		testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
		testlib.FailIf(t, arr[3] != "tag4", "expected arr[3] to be 'tag4', got %v", arr[3])

		updateOneResp2, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().AddToSetEach("arr", "tag3", "tag4"),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp2.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp2.MatchedCount)
		testlib.FailIf(t, updateOneResp2.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateOneResp2.ModifiedCount)

		updatedDoc2 := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc2 astra.Document
		err = updatedDoc2.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr2 := doc2.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr2) != 4, "expected arr length to be 4, got %d", len(arr2))
		testlib.FailIf(t, arr2[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr2[2])
		testlib.FailIf(t, arr2[3] != "tag4", "expected arr[3] to be 'tag4', got %v", arr2[3])

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().AddToSetEach("arr", "tag3", "tag4"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 4, "expected ModifiedCount to be 4, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
			testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
			testlib.FailIf(t, arr[3] != "tag4", "expected arr[3] to be 'tag4', got %v", arr[3])
		}

		updateManyResp2, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().AddToSetEach("arr", "tag3", "tag4"),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp2.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp2.MatchedCount)
		testlib.FailIf(t, updateManyResp2.ModifiedCount != 0, "expected ModifiedCount to be 0, got %d", updateManyResp2.ModifiedCount)

		cursor = t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		allDocs = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
			testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
			testlib.FailIf(t, arr[3] != "tag4", "expected arr[3] to be 'tag4', got %v", arr[3])
		}
	})

	s.Run("should remove last 1 item from array when $pop is passed with 1 in updateOne and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "arr": []string{"tag1", "tag2", "tag3", "tag4", "tag5"}, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Pop("arr", 1),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr := doc.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
		testlib.FailIf(t, arr[0] != "tag1", "expected arr[0] to be 'tag1', got %v", arr[0])
		testlib.FailIf(t, arr[1] != "tag2", "expected arr[1] to be 'tag2', got %v", arr[1])
		testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
		testlib.FailIf(t, arr[3] != "tag4", "expected arr[3] to be 'tag4', got %v", arr[3])

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Pop("arr", 1),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 5, "expected ModifiedCount to be 5, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
				testlib.FailIf(t, arr[0] != "tag1", "expected arr[0] to be 'tag1', got %v", arr[0])
				testlib.FailIf(t, arr[1] != "tag2", "expected arr[1] to be 'tag2', got %v", arr[1])
				testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
			} else {
				testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
				testlib.FailIf(t, arr[0] != "tag1", "expected arr[0] to be 'tag1', got %v", arr[0])
				testlib.FailIf(t, arr[1] != "tag2", "expected arr[1] to be 'tag2', got %v", arr[1])
				testlib.FailIf(t, arr[2] != "tag3", "expected arr[2] to be 'tag3', got %v", arr[2])
				testlib.FailIf(t, arr[3] != "tag4", "expected arr[3] to be 'tag4', got %v", arr[3])
			}
		}
	})

	s.Run("should remove first 1 item from array when $pop is passed with -1 in updateOne and updateMany", func(t *harness.T) {
		docList := make([]astra.NewDocument, 5)
		for i := 0; i < 5; i++ {
			docList[i] = astra.NewDocument{"_id": fmt.Sprintf("%s%d", t.Key(0), i), "arr": []string{"tag1", "tag2", "tag3", "tag4", "tag5"}, "key": t.Key(0)}
		}
		res, err := t.Collection.InsertMany(t.Ctx, docList)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res.InsertedCount() != len(docList), "expected InsertedCount to be %d, got %d", len(docList), res.InsertedCount())

		updateOneResp, err := t.Collection.UpdateOne(
			t.Ctx,
			filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))),
			update.Coll().Pop("arr", -1),
		)
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)
		testlib.FailIf(t, updateOneResp.MatchedCount != 1, "expected MatchedCount to be 1, got %d", updateOneResp.MatchedCount)
		testlib.FailIf(t, updateOneResp.ModifiedCount != 1, "expected ModifiedCount to be 1, got %d", updateOneResp.ModifiedCount)

		updatedDoc := t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("_id", fmt.Sprintf("%s4", t.Key(0))), filter.Eq("key", t.Key(0))))
		var doc astra.Document
		err = updatedDoc.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		arr := doc.MustGet("arr").([]any)
		testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
		testlib.FailIf(t, arr[0] != "tag2", "expected arr[0] to be 'tag2', got %v", arr[0])
		testlib.FailIf(t, arr[1] != "tag3", "expected arr[1] to be 'tag3', got %v", arr[1])
		testlib.FailIf(t, arr[2] != "tag4", "expected arr[2] to be 'tag4', got %v", arr[2])
		testlib.FailIf(t, arr[3] != "tag5", "expected arr[3] to be 'tag5', got %v", arr[3])

		updateManyResp, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.Coll().Pop("arr", -1),
		)
		testlib.FailIfErr(t, err, "UpdateMany failed: %v", err)
		testlib.FailIf(t, updateManyResp.MatchedCount != 5, "expected MatchedCount to be 5, got %d", updateManyResp.MatchedCount)
		testlib.FailIf(t, updateManyResp.ModifiedCount != 5, "expected ModifiedCount to be 5, got %d", updateManyResp.ModifiedCount)

		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetLimit(5))
		var allDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			allDocs = append(allDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(allDocs) != 5, "expected 5 documents, got %d", len(allDocs))

		for _, d := range allDocs {
			arr := d.MustGet("arr").([]any)
			if d.MustGet("_id") == fmt.Sprintf("%s4", t.Key(0)) {
				testlib.FailIf(t, len(arr) != 3, "expected arr length to be 3, got %d", len(arr))
				testlib.FailIf(t, arr[0] != "tag3", "expected arr[0] to be 'tag3', got %v", arr[0])
				testlib.FailIf(t, arr[1] != "tag4", "expected arr[1] to be 'tag4', got %v", arr[1])
				testlib.FailIf(t, arr[2] != "tag5", "expected arr[2] to be 'tag5', got %v", arr[2])
			} else {
				testlib.FailIf(t, len(arr) != 4, "expected arr length to be 4, got %d", len(arr))
				testlib.FailIf(t, arr[0] != "tag2", "expected arr[0] to be 'tag2', got %v", arr[0])
				testlib.FailIf(t, arr[1] != "tag3", "expected arr[1] to be 'tag3', got %v", arr[1])
				testlib.FailIf(t, arr[2] != "tag4", "expected arr[2] to be 'tag4', got %v", arr[2])
				testlib.FailIf(t, arr[3] != "tag5", "expected arr[3] to be 'tag5', got %v", arr[3])
			}
		}
	})

	s.Run("fails gracefully on 2XX exceptions", func(t *harness.T) {
		// Use invalid update operator to trigger DataAPIError
		_, err := t.Collection.UpdateMany(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			update.U{"$invalid": 1},
		)
		testlib.FailIf(t, err == nil, "expected error for invalid update operator")

		var dataAPIErr *results.DataAPIError
		testlib.FailIf(t, !errors.As(err, &dataAPIErr), "expected DataAPIError, got %T", err)
		testlib.FailIf(t, dataAPIErr.ErrorCode != "UNSUPPORTED_UPDATE_OPERATION", "expected UNSUPPORTED_UPDATE_OPERATION error code, got %s", dataAPIErr.ErrorCode)
	})
}
