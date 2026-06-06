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

package collection_test

import (
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/integration/harness"
	"github.com/datastax/astra-db-go/internal/testlib"
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
				Big:      *big.NewInt(123),
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
			testlib.ErrMustBe[*serdes.UnsupportedValueError](t, err, "expected %T insertion to fail", ty)
			return nil, nil
		})
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
