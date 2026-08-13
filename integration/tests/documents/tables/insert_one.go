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
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("insert-one")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should fail to insert a document into a table", func(t *harness.T) {
		_, err := t.Table.InsertOne(t.Ctx, astra.NewDocument{})
		testlib.FailIf(t, err == nil, "expected InsertOne to fail when inserting a document into a table")
	})

	s.Run("should fail to insert collection-only datatypes into a table", func(t *harness.T) {
		types := []any{
			datatypes.NewObjectId(),
		}

		testlib.AwaitAll(t, types, func(ty any) (any, error) {
			_, err := t.Table_.InsertOne(t.Ctx, astra.NewRow{"val": ty})
			testlib.ErrMustBe[*serdes.EncodeError](t, err, "expected %T insertion to fail", ty)
			return nil, nil
		})
	})

	s.Run("should insert one partial row", func(t *harness.T) {
		res, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text": t.Key(0),
			"int":  0,
			"map":  map[int64]map[string]any{4: {}},
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var id astra.Row
		testlib.FailIfErr(t, res.DecodeID(&id), "DecodeID failed")
		t.NoDiff(map[string]any{"text": t.Key(0), "int": int32(0)}, id.ToMap())
	})

	s.Run("should insert one full row", func(t *harness.T) {
		res, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text":      t.Key(0),
			"int":       0,
			"map":       map[int64]map[string]any{},
			"ascii":     "highway_star",
			"blob":      []byte("smoke_on_the_water"),
			"bigint":    int64(1231233),
			"date":      datatypes.DateOnlyNow(),
			"decimal":   12.3456789012345678901234567890,
			"double":    123.456,
			"duration":  datatypes.MustParseDuration("P1Y1M1DT1H1M1.001001001S"),
			"float":     float32(123.456),
			"inet":      "::1",
			"list":      []map[string]any{{}, {}},
			"set":       datatypes.NewSet(datatypes.NewUUIDv4(), datatypes.NewUUIDv7(), datatypes.NewUUIDv7()),
			"smallint":  int16(123),
			"time":      datatypes.TimeOnlyNow(),
			"timestamp": time.Now(),
			"tinyint":   int8(123),
			"uuid":      datatypes.NewUUIDv4(),
			"varint":    123123123123123,
			"vector":    datatypes.NewVector([]float32{.123123, .123, .12321, .123123, .2132}),
			"boolean":   true,
			"udt": map[string]any{
				"name": "ac",
				"age":  123,
			},
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var id astra.Row
		testlib.FailIfErr(t, res.DecodeID(&id), "DecodeID failed")
		t.NoDiff(map[string]any{"text": t.Key(0), "int": int32(0)}, id.ToMap())
	})

	s.Run("should insert one full row using raw representations", func(t *harness.T) {
		res, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text":      t.Key(0),
			"int":       0,
			"map":       [][]any{{int64(4), map[string]any{}}},
			"ascii":     "highway_star",
			"blob":      map[string]any{"$binary": "c21va2Vfb25fdGhlX3dhdGVy"},
			"bigint":    1231233,
			"date":      "2021-01-01",
			"double":    123.456,
			"duration":  "1y1mo1d1h1m1s1ms1us1ns",
			"float":     123.456,
			"inet":      "::1",
			"list":      []any{},
			"set":       []string{datatypes.NewUUIDv4().String(), datatypes.NewUUIDv7().String(), datatypes.NewUUIDv7().String()},
			"smallint":  123,
			"time":      "12:34:56",
			"timestamp": "2021-01-01T12:34:56.789Z",
			"tinyint":   123,
			"uuid":      datatypes.NewUUIDv4(),
			"varint":    1231231231231231,
			"vector":    []float32{.123123, .123, .12321, .123123, .2132},
			"boolean":   true,
			"udt": map[string]any{
				"name": "ac",
				"age":  123,
				"id":   datatypes.NewUUIDv6().String(),
			},
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var id astra.Row
		testlib.FailIfErr(t, res.DecodeID(&id), "DecodeID failed")
		t.NoDiff(map[string]any{"text": t.Key(0), "int": int32(0)}, id.ToMap())
	})

	s.Run("should upsert rows", func(t *harness.T) {
		res1, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text": t.Key(0),
			"int":  0,
			"map":  map[int64]map[string]any{123: {"age": 5}},
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var id1 astra.Row
		testlib.FailIfErr(t, res1.DecodeID(&id1), "DecodeID failed")
		t.NoDiff(map[string]any{"text": t.Key(0), "int": int32(0)}, id1.ToMap())

		res2, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text":  t.Key(0),
			"int":   0,
			"ascii": "highway_star",
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var id2 astra.Row
		testlib.FailIfErr(t, res2.DecodeID(&id2), "DecodeID failed")
		t.NoDiff(map[string]any{"text": t.Key(0), "int": int32(0)}, id2.ToMap())
	})

	s.Run("(VECTORIZE) should insert w/ vectorize", func(t *harness.T) {
		res, err := t.Table_.InsertOne(t.Ctx, astra.NewRow{
			"text":    t.Key(0),
			"int":     0,
			"vector1": "hardest button",
			"vector2": "to button",
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var id astra.Row
		testlib.FailIfErr(t, res.DecodeID(&id), "DecodeID failed")
		t.NoDiff(map[string]any{"text": t.Key(0), "int": int32(0)}, id.ToMap())
	})
}
