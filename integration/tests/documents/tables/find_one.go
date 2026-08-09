// Copyright DataStax, Inc.
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

package tables

import (
	"errors"
	"math/big"
	"net"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("find-one")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should find one partial row", func(t *harness.T) {
		uuid := datatypes.NewUUID()

		row := astra.NewRow{
			"text": t.Key(0),
			"int":  0,
			"map":  map[int64]map[string]any{123: {"id": uuid}},
		}

		res, err := t.Table.InsertOne(t.Ctx, row)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// FindOne with primary key filter
		result := t.Table.FindOne(t.Ctx, filter.Eq("text", t.Key(0)))
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		err = result.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		// Verify the found document has all columns (even if null)
		// The table schema has many columns, so we check key fields
		testlib.FailIf(t, found.MustGet("text") != t.Key(0), "expected text=%s, got %v", t.Key(0), found.MustGet("text"))
		testlib.FailIf(t, found.MustGet("int") != int32(0), "expected int=0, got %v", found.MustGet("int"))

		// Verify map field
		mapField := found.MustGet("map").(map[int64]map[string]any)
		testlib.FailIf(t, len(mapField) != 1, "expected map with 1 entry, got %d", len(mapField))

		mapValue := mapField[123]
		testlib.FailIf(t, !mapValue["id"].(datatypes.UUID).Equals(uuid), "expected id=%s, got %v", uuid.String(), mapValue["id"])

		// Verify other fields are null or empty
		if setField, ok := found.Get("set"); ok && setField != nil {
			// Set should be empty
			setSlice := setField.([]any)
			testlib.FailIf(t, len(setSlice) != 0, "expected empty set, got %d items", len(setSlice))
		}

		if listField, ok := found.Get("list"); ok && listField != nil {
			// List should be empty
			listSlice := listField.([]any)
			testlib.FailIf(t, len(listSlice) != 0, "expected empty list, got %d items", len(listSlice))
		}

		// Verify inserted ID matches
		var insertedID map[string]any
		err = res.DecodeID(&insertedID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		testlib.FailIf(t, insertedID["text"] != t.Key(0), "expected insertedId text=%s, got %v", t.Key(0), insertedID["text"])
	})

	s.Run("should find one full row", func(t *harness.T) {
		now := time.Now().UTC().Truncate(time.Millisecond)
		decimal, _ := new(big.Float).SetString("12.34567890123456789012345678901234567890")

		row := astra.NewRow{
			"text":      t.Key(0),
			"int":       int32(0),
			"map":       map[int64]map[string]any{},
			"ascii":     "highway_star",
			"blob":      []byte("smoke_on_the_water"),
			"bigint":    int64(1231233),
			"date":      datatypes.DateOnlyFromTime(now),
			"decimal":   decimal,
			"double":    123.456,
			"duration":  datatypes.MustParseDuration("1y1mo1d1h1m1s1ms1us1ns"),
			"float":     float32(123.456),
			"inet":      net.ParseIP("::1"),
			"list":      []map[string]any{{"age": nil, "id": nil, "name": nil}, {"age": big.NewInt(3), "id": nil, "name": nil}},
			"set":       []datatypes.UUID{datatypes.NewUUID()},
			"smallint":  int16(123),
			"time":      datatypes.TimeOnlyFromTime(now),
			"timestamp": now,
			"tinyint":   int8(123),
			"uuid":      datatypes.NewUUIDv7(),
			"varint":    big.NewInt(0).SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}),
			"vector":    datatypes.NewVector([]float32{.123123, .123, .12321, .123123, .2132}),
			"boolean":   true,
			"udt": map[string]any{
				"name": "name",
				"id":   datatypes.NewUUIDv1(),
				"age":  big.NewInt(123),
			},
		}

		_, err := t.Table.InsertOne(t.Ctx, row)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// FindOne with primary key filter
		result := t.Table.FindOne(t.Ctx, filter.Eq("text", t.Key(0)))
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		err = result.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.NoDiff(t, row.ToMap(), found.ToMap())
	})

	s.Run("should return no documents error when not found", func(t *harness.T) {
		result := t.Table.FindOne(t.Ctx, filter.Eq("text", "nonexistent-key-12345"))
		testlib.FailIfErr(t, result.Err(), "FindOne should not error on no results")

		var found astra.Row
		err := result.Decode(&found)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got %v", err)
	})
}
