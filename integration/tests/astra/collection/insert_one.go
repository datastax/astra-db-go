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
	"github.com/datastax/astra-db-go/integration/harness"
	"github.com/datastax/astra-db-go/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("collection.insert-one")

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
			return insertOneDecodeAny(res), nil
		})

		t.NoDiff(ids, got)
	})

	s.Run("should insert an untyped document with datatypes", func(t *harness.T) {
		doc := astra.NewDocument{
			"oid":     datatypes.NewObjectId(),
			"u4":      datatypes.NewUUIDv4(),
			"u7":      datatypes.NewUUIDv7(),
			"date":    time.Now(),
			"$vector": datatypes.NewVector([]float32{1, 2, 3, 4, 5}),
			"nested_doc": astra.NewDocument{
				"u4": datatypes.NewUUIDv4(),
			},
			"nested_map": map[string]any{
				"u4": datatypes.NewUUIDv4(),
			},
		}

		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		if reflect.TypeOf(insertOneDecodeAny(res)) != reflect.TypeFor[string]() {
			t.Fatalf("expected InsertOne result to be of type string, got %T", res)
		}
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

		var id datatypes.UUID
		err = res.DecodeID(&id)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		t.NoDiff(id, doc.ID)

		var got harness.EverythingDoc
		err = t.Collection.FindOne(t.Ctx, filter.Eq("_id", doc.ID), options.CollectionFindOne().SetProjection(map[string]any{"*": 1})).Decode(&got)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)
		t.NoDiff(doc, got)
	})

	s.Run("should fail to insert a row into a collection", func(t *harness.T) {
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewRow{})
		testlib.FailIf(t, err == nil, "expected InsertOne to fail when inserting a row into a collection")
	})
}

func insertOneDecodeAny(res *results.InsertOneResult) any {
	var id any
	if err := res.DecodeID(&id); err != nil {
		panic("failed to decode inserted ID: " + err.Error())
	}
	return id
}
