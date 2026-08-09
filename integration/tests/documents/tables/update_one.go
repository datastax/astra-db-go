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
	"github.com/datastax/astra-db-go/v2/astra/update"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("update-one")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should error on empty $set/$unset", func(t *harness.T) {
		err1 := t.Table.UpdateOne(t.Ctx, filter.F{}, update.Table().Set("", nil))
		testlib.FailIf(t, err1 == nil, "expected error on empty $set")

		err2 := t.Table.UpdateOne(t.Ctx, filter.F{}, update.Table().Unset())
		testlib.FailIf(t, err2 == nil, "expected error on empty $unset")
	})

	s.Run("should error when trying to change pk", func(t *harness.T) {
		err1 := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Set("text", "new"))
		var dataAPIErr *results.DataAPIError
		testlib.FailIf(t, !errors.As(err1, &dataAPIErr), "expected DataAPIError when trying to change pk with $set")

		err2 := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Unset("text"))
		testlib.FailIf(t, !errors.As(err2, &dataAPIErr), "expected DataAPIError when trying to change pk with $unset")
	})

	s.Run("should error when exact pk not set", func(t *harness.T) {
		err := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0)}, update.Table().Set("tinyint", int8(3)))
		var dataAPIErr *results.DataAPIError
		testlib.FailIf(t, !errors.As(err, &dataAPIErr), "expected DataAPIError when pk is incomplete")
	})

	s.Run("should upsert w/ $set when no matching pk", func(t *harness.T) {
		now := time.Now().UTC().Truncate(time.Millisecond)
		decimal, _ := new(big.Float).SetString("12.34567890123456789012345678901234567890")

		expected := astra.NewRow{
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

		err := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().
			Set("ascii", expected.MustGet("ascii")).
			Set("blob", expected.MustGet("blob")).
			Set("bigint", expected.MustGet("bigint")).
			Set("date", expected.MustGet("date")).
			Set("decimal", expected.MustGet("decimal")).
			Set("double", expected.MustGet("double")).
			Set("duration", expected.MustGet("duration")).
			Set("float", expected.MustGet("float")).
			Set("inet", expected.MustGet("inet")).
			Set("list", expected.MustGet("list")).
			Set("set", expected.MustGet("set")).
			Set("map", expected.MustGet("map")).
			Set("smallint", expected.MustGet("smallint")).
			Set("time", expected.MustGet("time")).
			Set("timestamp", expected.MustGet("timestamp")).
			Set("tinyint", expected.MustGet("tinyint")).
			Set("uuid", expected.MustGet("uuid")).
			Set("varint", expected.MustGet("varint")).
			Set("vector", expected.MustGet("vector")).
			Set("boolean", expected.MustGet("boolean")).
			Set("udt", expected.MustGet("udt")))
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		// Verify the document was inserted
		result := t.Table.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		testlib.FailIfErr(t, result.Decode(&found), "Decode failed")

	})

	s.Run("should not upsert w/ $unset/null-$set when no matching pk", func(t *harness.T) {
		// Try $unset - should not create document
		err1 := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Unset("tinyint"))
		testlib.FailIfErr(t, err1, "UpdateOne with $unset failed: %v", err1)

		result1 := t.Table.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		var doc1 astra.Row
		err := result1.Decode(&doc1)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments after $unset on non-existent row")

		// Try $set with null - should not create document
		err2 := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Set("tinyint", nil))
		testlib.FailIfErr(t, err2, "UpdateOne with null $set failed: %v", err2)

		result2 := t.Table.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		var doc2 astra.Row
		err = result2.Decode(&doc2)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments after null $set on non-existent row")
	})

	s.Run("should error if $in is used", func(t *harness.T) {
		err := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": filter.F{"$in": []int{1, 2, 3}}}, update.Table().Set("tinyint", int8(3)))
		var dataAPIErr *results.DataAPIError
		testlib.FailIf(t, !errors.As(err, &dataAPIErr), "expected DataAPIError when using $in operator")
	})

	s.Run("should upsert w/ vectorize", func(t *harness.T) {
		// Vector length is 1024 for vectorize (from prelude.go)
		vector := make([]float32, 1024)
		for i := range vector {
			vector[i] = 0.1
		}

		err := t.Table_.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().
			Set("vector1", "hello world!!!").
			Set("vector2", datatypes.NewVector(vector)))
		testlib.FailIfErr(t, err, "UpdateOne with vectorize failed: %v", err)

		result := t.Table_.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		testlib.FailIfErr(t, result.Decode(&found), "Decode failed")

		// Verify vector1 was vectorized
		vector1Val := found.MustGet("vector1").(datatypes.Vector)
		testlib.FailIf(t, vector1Val.Dimension() != 1024, "vector1 should be vectorized")

		// Verify vector2 matches input
		vector2Val := found.MustGet("vector2").(datatypes.Vector)
		testlib.FailIf(t, vector2Val.Dimension() != 1024, "vector2 should be vectorized")

		// Verify text field
		testlib.FailIf(t, found.MustGet("text").(string) != t.Key(0), "text mismatch")
	})

	s.Run("should $set one", func(t *harness.T) {
		// Insert initial document
		_, err := t.Table.InsertOne(t.Ctx, astra.NewRow{"text": t.Key(0), "int": 0, "tinyint": int8(3)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// Update tinyint
		err = t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Set("tinyint", int8(4)))
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		// Verify update
		result := t.Table.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		testlib.FailIfErr(t, result.Decode(&found), "Decode failed")

		testlib.FailIf(t, found.MustGet("tinyint").(int8) != 4, "tinyint should be 4")
		testlib.FailIf(t, found.MustGet("text").(string) != t.Key(0), "text mismatch")
	})

	s.Run("should $set one after being upserted", func(t *harness.T) {
		// Upsert with initial value
		err := t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Set("tinyint", int8(3)))
		testlib.FailIfErr(t, err, "UpdateOne (upsert) failed: %v", err)

		// Update tinyint
		err = t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Set("tinyint", int8(4)))
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		// Verify update
		result := t.Table.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		testlib.FailIfErr(t, result.Decode(&found), "Decode failed")

		testlib.FailIf(t, found.MustGet("tinyint").(int8) != 4, "tinyint should be 4")
		testlib.FailIf(t, found.MustGet("text").(string) != t.Key(0), "text mismatch")
	})

	s.Run("should $unset one", func(t *harness.T) {
		// Insert initial document
		_, err := t.Table.InsertOne(t.Ctx, astra.NewRow{"text": t.Key(0), "int": 0, "tinyint": int8(3)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// Unset tinyint
		err = t.Table.UpdateOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0}, update.Table().Unset("tinyint"))
		testlib.FailIfErr(t, err, "UpdateOne failed: %v", err)

		// Verify unset
		result := t.Table.FindOne(t.Ctx, filter.F{"text": t.Key(0), "int": 0})
		testlib.FailIfErr(t, result.Err(), "FindOne failed: %v", result.Err())

		var found astra.Row
		testlib.FailIfErr(t, result.Decode(&found), "Decode failed")

		testlib.FailIf(t, found.MustGet("tinyint") != nil, "tinyint should be null")
		testlib.FailIf(t, found.MustGet("text").(string) != t.Key(0), "text mismatch")
	})
}
