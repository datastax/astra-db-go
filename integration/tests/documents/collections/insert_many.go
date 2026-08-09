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
	"sort"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("insert-many")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should insertMany documents", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"name": "Inis Mona"},
			{"name": "Helvetios"},
			{"name": "Epona"},
		}
		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		var ids []datatypes.UUID
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)
		testlib.FailIf(t, len(ids) != len(docs), "expected %d IDs, got %d", len(docs), len(ids))

		for _, id := range ids {
			testlib.FailIf(t, id.String() == "", "expected non-empty UUID string")
		}
	})

	s.Run("should insertMany many documents", func(t *harness.T) {
		docs := make([]astra.NewDocument, 1000)
		for i := range docs {
			docs[i] = astra.NewDocument{"name": fmt.Sprintf("Player %d", i)}
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		var ids []datatypes.UUID
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)
		testlib.FailIf(t, len(ids) != len(docs), "expected %d IDs, got %d", len(docs), len(ids))
	})

	s.Run("should insertMany 0 documents", func(t *harness.T) {
		res, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != 0, "expected 0 inserted, got %d", res.InsertedCount())

		var ids []datatypes.UUID
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)
		testlib.FailIf(t, len(ids) != 0, "expected 0 IDs, got %d", len(ids))
	})

	s.Run("should insertMany 0 documents ordered", func(t *harness.T) {
		res, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{}, options.CollectionInsertMany().SetOrdered(true))
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != 0, "expected 0 inserted, got %d", res.InsertedCount())

		var ids []datatypes.UUID
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)
		testlib.FailIf(t, len(ids) != 0, "expected 0 IDs, got %d", len(ids))
	})

	s.Run("should insertMany documents with ids", func(t *harness.T) {
		id1 := datatypes.NewUUIDv7().String()
		id2 := datatypes.NewUUIDv7().String()
		id3 := datatypes.NewUUIDv7().String()

		docs := []astra.NewDocument{
			{"name": "Inis Mona", "_id": id1},
			{"name": "Helvetios", "_id": id2},
			{"name": "Epona", "_id": id3},
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		var ids []string
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)

		expectedIds := []string{id1, id2, id3}
		sort.Strings(ids)
		sort.Strings(expectedIds)
		t.NoDiff(expectedIds, ids)
	})

	s.Run("should insertMany documents with UUIDs", func(t *harness.T) {
		id1 := datatypes.NewUUIDv7()
		id2 := datatypes.NewUUIDv7()
		id3 := datatypes.NewUUIDv7()

		docs := []astra.NewDocument{
			{"name": "Inis Mona", "_id": id1},
			{"name": "Helvetios", "_id": id2},
			{"name": "Epona", "_id": id3},
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		var ids []datatypes.UUID
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)

		expectedIds := []string{id1.String(), id2.String(), id3.String()}
		gotIds := make([]string, len(ids))
		for i, id := range ids {
			gotIds[i] = id.String()
		}

		sort.Strings(expectedIds)
		sort.Strings(gotIds)
		t.NoDiff(expectedIds, gotIds)
	})

	s.Run("should insertMany documents with ObjectIds", func(t *harness.T) {
		id1 := datatypes.NewObjectId()
		id2 := datatypes.NewObjectId()
		id3 := datatypes.NewObjectId()

		docs := []astra.NewDocument{
			{"name": "Inis Mona", "_id": id1},
			{"name": "Helvetios", "_id": id2},
			{"name": "Epona", "_id": id3},
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		var ids []datatypes.ObjectId
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)

		expectedIds := []string{id1.String(), id2.String(), id3.String()}
		gotIds := make([]string, len(ids))
		for i, id := range ids {
			gotIds[i] = id.String()
		}

		sort.Strings(expectedIds)
		sort.Strings(gotIds)
		t.NoDiff(expectedIds, gotIds)
	})

	s.Run("should insertMany documents with a mix of ids", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"name": "Inis Mona", "_id": datatypes.NewObjectId()},
			{"name": "Helvetios", "_id": datatypes.NewUUIDv4()},
			{"name": "Epona"},
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		rawIds, err := res.RawIDs()
		testlib.FailIfErr(t, err, "RawIDs failed: %v", err)
		testlib.FailIf(t, len(rawIds) != len(docs), "expected %d IDs, got %d", len(docs), len(rawIds))
	})

	s.Run("should insertMany with vectors", func(t *harness.T) {
		docs := []astra.NewDocument{
			{"name": "a", "key": t.Key(0), "$vector": []float32{1, 1, 1, 1, 1}},
			{"name": "b", "key": t.Key(0)},
			{"name": "c", "key": t.Key(0), "$vector": []float32{1, 1, 1, 1, 1}},
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
		testlib.FailIf(t, res == nil, "expected result")

		var found astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("name", "a"), filter.Eq("key", t.Key(0))),
			options.CollectionFindOne().SetProjection(map[string]any{"$vector": 1})).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		vector := found.MustGet("$vector").(datatypes.Vector)
		floats, err := vector.AsFloatArray()
		testlib.FailIfErr(t, err, "AsFloatArray failed: %v", err)
		t.NoDiff([]float32{1, 1, 1, 1, 1}, floats)

		var found2 astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.And(filter.Eq("name", "b"), filter.Eq("key", t.Key(0)))).Decode(&found2)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		_, hasVector := found2.Get("$vector")
		testlib.FailIf(t, hasVector, "expected no $vector field for document b")
	})

	s.Run("should insertMany documents ordered", func(t *harness.T) {
		id1 := datatypes.NewUUIDv7().String()
		id2 := datatypes.NewUUIDv7().String()
		id3 := datatypes.NewUUIDv7().String()

		docs := []astra.NewDocument{
			{"name": "Inis Mona", "_id": id1},
			{"name": "Helvetios", "_id": id2},
			{"name": "Epona", "_id": id3},
		}

		res, err := t.Collection.InsertMany(t.Ctx, docs, options.CollectionInsertMany().SetOrdered(true))
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		testlib.FailIf(t, res.InsertedCount() != len(docs), "expected %d inserted, got %d", len(docs), res.InsertedCount())

		var ids []string
		err = res.DecodeIDs(&ids)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)

		expectedIds := []string{id1, id2, id3}
		t.NoDiff(expectedIds, ids)
	})

	s.Run("should error out when one of the docs in insertMany is invalid with ordered true", func(t *harness.T) {
		docs := make([]astra.NewDocument, 20)
		for i := range docs {
			docs[i] = astra.NewDocument{"_id": t.Key(i)}
		}
		docs[10] = docs[9] // duplicate

		_, err := t.Collection.InsertMany(t.Ctx, docs, options.CollectionInsertMany().SetOrdered(true))
		testlib.FailIf(t, err == nil, "expected InsertMany to fail")

		var insertErr *results.InsertManyError
		testlib.ErrMustBe[*results.InsertManyError](t, err, "expected InsertManyError")
		if !errors.As(err, &insertErr) {
			t.Fatalf("expected InsertManyError, got %T", err)
		}

		testlib.FailIf(t, len(insertErr.Errors) != 1, "expected 1 error, got %d", len(insertErr.Errors))
		testlib.FailIf(t, insertErr.Errors[0].ErrorCode != "DOCUMENT_ALREADY_EXISTS", "expected DOCUMENT_ALREADY_EXISTS, got %s", insertErr.Errors[0].ErrorCode)

		var insertedIds []string
		err = insertErr.DecodeIDs(&insertedIds)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)

		expectedIds := make([]string, 10)
		for i := 0; i < 10; i++ {
			expectedIds[i] = t.Key(i)
		}

		sort.Strings(insertedIds)
		sort.Strings(expectedIds)
		t.NoDiff(expectedIds, insertedIds)
	})

	s.Run("should error out when one of the docs in insertMany is invalid with ordered false", func(t *harness.T) {
		docs := make([]astra.NewDocument, 20)
		for i := range docs {
			docs[i] = astra.NewDocument{"_id": t.Key(i)}
		}
		docs[10] = docs[9] // duplicate

		_, err := t.Collection.InsertMany(t.Ctx, docs, options.CollectionInsertMany().SetOrdered(false))
		testlib.FailIf(t, err == nil, "expected InsertMany to fail")

		var insertErr *results.InsertManyError
		testlib.ErrMustBe[*results.InsertManyError](t, err, "expected InsertManyError")
		if !errors.As(err, &insertErr) {
			t.Fatalf("expected InsertManyError, got %T", err)
		}

		testlib.FailIf(t, len(insertErr.Errors) != 1, "expected 1 error, got %d", len(insertErr.Errors))
		testlib.FailIf(t, insertErr.Errors[0].ErrorCode != "DOCUMENT_ALREADY_EXISTS", "expected DOCUMENT_ALREADY_EXISTS, got %s", insertErr.Errors[0].ErrorCode)

		var insertedIds []string
		err = insertErr.DecodeIDs(&insertedIds)
		testlib.FailIfErr(t, err, "DecodeIDs failed: %v", err)

		// With ordered=false, all docs except the duplicate should be inserted (19 total)
		expectedIds := make([]string, 0, 19)
		for i := 0; i < 10; i++ {
			expectedIds = append(expectedIds, t.Key(i))
		}
		for i := 11; i < 20; i++ {
			expectedIds = append(expectedIds, t.Key(i))
		}

		sort.Strings(insertedIds)
		sort.Strings(expectedIds)
		t.NoDiff(expectedIds, insertedIds)
	})

	// Note: The TS tests have "fails fast on hard errors" tests that use a failing client.
	// These would require special test harness setup in Go and are not migrated here.

	// Note: The TS test has a timeout test that's very specific to the TS implementation
	// with chunkSize and timeout parameters. This would need adaptation for Go's timeout handling.
}
