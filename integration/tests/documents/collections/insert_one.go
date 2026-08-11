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
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("insert-one")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should insert an untyped document with IDs of all kinds", func(t *harness.T) {
		ids := []any{
			"hi",
			nil,
			3.14,
			datatypes.NewObjectId(),
			datatypes.NewUUIDv4(),
			datatypes.NewUUIDv7(),
		}

		got := testlib.AwaitAll(t, ids, func(id any) (any, error) {
			res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"_id": id})
			if err != nil {
				return nil, fmt.Errorf("InsertOne failed for id %v: %v", id, err)
			}
			return insertOneDecode[any](t, res), nil
		})

		t.NoDiff(ids, got)
	})

	s.Run("should insert an untyped document with datatypes", func(t *harness.T) {
		lm := datatypes.NewLinkedMap[string, any]()
		lm.Set("set", datatypes.NewSet(1, 2, 3))

		doc := astra.NewDocument{
			"key":     t.Key(0),
			"oid":     datatypes.NewObjectId(),
			"u4":      datatypes.NewUUIDv4(),
			"u7":      map[string]any{"$uuid": datatypes.NewUUIDv7().String()},
			"date":    time.Now(),
			"$vector": []float32{1, 2, 3, 4, 5},
			"nested_doc": astra.NewDocument{
				"oid": datatypes.NewObjectId(),
			},
			"nested_map": map[string]any{
				"u6": datatypes.NewUUIDv6(),
			},
			"nested_linked_map": lm,
		}

		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		insertOneDecode[string](t, res)

		var got astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.Eq("key", t.Key(0)), options.CollectionFindOne().
			SetProjection(map[string]any{"_id": 0, "$vector": true})).
			Decode(&got)

		// can't really check the whole doc for equality due to the insertion using some non-default types
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)
		t.NoDiff([]any{1.0, 2.0, 3.0}, got.MustGet("nested_linked_map", "set"))
	})

	s.Run("should insert a typed document with datatypes", func(t *harness.T) {
		doc := harness.EverythingDoc{
			ID:     datatypes.NewUUIDv7(),
			Vector: datatypes.NewVector([]float32{1, 2, 3, 4, 5}),
			Nested: harness.EverythingDocInner{
				UUID:     datatypes.NewUUIDv4(),
				ObjectId: datatypes.NewObjectId(),
				Date:     time.Now().Truncate(time.Millisecond),
				Big:      big.NewInt(123),
			},
		}

		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		id := insertOneDecode[datatypes.UUID](t, res)
		t.NoDiff(id, doc.ID)

		var got harness.EverythingDoc
		err = t.Collection.FindOne(t.Ctx, filter.Eq("_id", doc.ID), options.CollectionFindOne().
			SetProjection(map[string]any{"*": 1})).
			Decode(&got)

		testlib.FailIfErr(t, err, "FindOne failed: %v", err)
		t.NoDiff(doc, got)
	})

	s.Run("should fail to insert a row into a collection", func(t *harness.T) {
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewRow{})
		testlib.FailIf(t, err == nil, "expected InsertOne to fail when inserting a row into a collection")
	})

	s.Run("should fail to insert table-only datatypes into a collection", func(t *harness.T) {
		types := []any{
			datatypes.DateOnlyNow(),
			datatypes.TimeOnlyNow(),
			datatypes.MustParseDuration("P1W"),
		}

		testlib.AwaitAll(t, types, func(ty any) (any, error) {
			_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"val": ty})
			testlib.ErrMustBe[*serdes.EncodeError](t, err, "expected %T insertion to fail", ty)
			return nil, nil
		})
	})

	s.Run("should insertOne document with a non-_id UUID", func(t *harness.T) {
		id := datatypes.NewUUIDv4()
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"foreignId": id})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		testlib.FailIf(t, res == nil, "expected result")

		var found astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.Eq("foreignId", id)).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		foreignId := found.MustGet("foreignId").(datatypes.UUID)
		testlib.FailIf(t, foreignId.String() != id.String(), "expected foreignId %s, got %s", id, foreignId)
	})

	s.Run("should insertOne document with a non-_id ObjectId", func(t *harness.T) {
		id := datatypes.NewObjectId()
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"foreignId": id})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		testlib.FailIf(t, res == nil, "expected result")

		var found astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.Eq("foreignId", id)).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		foreignId := found.MustGet("foreignId").(datatypes.ObjectId)
		testlib.FailIf(t, foreignId.String() != id.String(), "expected foreignId %s, got %s", id, foreignId)
	})

	s.Run("should insertOne with a $date", func(t *harness.T) {
		timestamp := time.Now().Truncate(time.Millisecond)
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{
			"name": t.Key(0),
			"date": map[string]any{"$date": timestamp.UnixMilli()},
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		testlib.FailIf(t, res == nil, "expected result")

		var found astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.Eq("name", t.Key(0))).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		date := found.MustGet("date").(time.Time)
		testlib.FailIf(t, !date.Equal(timestamp), "expected date %v, got %v", timestamp, date)
	})

	s.Run("should fail insert of doc over size 1 MB", func(t *harness.T) {
		bigDoc := make([]byte, 1024*1024)
		for i := range bigDoc {
			bigDoc[i] = 'a'
		}
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": string(bigDoc)})
		testlib.FailIf(t, err == nil, "expected InsertOne to fail for doc over 1 MB")
	})

	s.Run("should fail if the number of levels in the doc is > 16", func(t *harness.T) {
		doc := astra.NewDocument{"l1": map[string]any{"l2": map[string]any{"l3": map[string]any{"l4": map[string]any{"l5": map[string]any{"l6": map[string]any{"l7": map[string]any{"l8": map[string]any{"l9": map[string]any{"l10": map[string]any{"l11": map[string]any{"l12": map[string]any{"l13": map[string]any{"l14": map[string]any{"l15": map[string]any{"l16": map[string]any{"l17": "l17value"}}}}}}}}}}}}}}}}}
		_, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.ErrMustBe[*results.DataAPIError](t, err, "expected DataAPIError")
	})

	s.Run("should fail if the field length is > 1000", func(t *harness.T) {
		fieldName := make([]byte, 1001)
		for i := range fieldName {
			fieldName[i] = 'a'
		}
		doc := astra.NewDocument{string(fieldName): "value"}
		_, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.ErrMustBe[*results.DataAPIError](t, err, "expected DataAPIError")
	})

	s.Run("should fail if the string field value is > 8000", func(t *harness.T) {
		longValue := make([]byte, 8001)
		for i := range longValue {
			longValue[i] = 'a'
		}
		doc := astra.NewDocument{"name": string(longValue)}
		_, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.ErrMustBe[*results.DataAPIError](t, err, "expected DataAPIError")
	})

	s.Run("should fail if an array field size is > 1000", func(t *harness.T) {
		tags := make([]string, 1001)
		for i := range tags {
			tags[i] = "tag"
		}
		doc := astra.NewDocument{"tags": tags}
		_, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.ErrMustBe[*results.DataAPIError](t, err, "expected DataAPIError")
	})

	s.Run("should fail if a doc contains more than 1000 properties", func(t *harness.T) {
		doc := astra.NewDocument{}
		for i := 1; i <= 1001; i++ {
			doc[fmt.Sprintf("prop%d", i)] = fmt.Sprintf("prop%dvalue", i)
		}
		_, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.ErrMustBe[*results.DataAPIError](t, err, "expected DataAPIError")
	})
}

func insertOneDecode[T any](t *harness.T, res *results.InsertOneResult) T {
	t.Helper()
	var id T
	if err := res.DecodeID(&id); err != nil {
		t.Fatalf("failed to decode inserted ID as %s: %v", reflect.TypeFor[T]().String(), err)
	}
	return id
}
