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
	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("delete-many")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should deleteMany on a single column", func(t *harness.T) {
		// Insert 5 rows with same text key but different int values
		rows := make([]astra.NewRow, 5)
		for i := 0; i < 5; i++ {
			rows[i] = astra.NewRow{
				"text": t.Key(0),
				"int":  i,
			}
		}
		_, err := t.Table.InsertMany(t.Ctx, rows)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Delete the row where int=3
		err = t.Table.DeleteMany(t.Ctx, filter.F{
			"text": t.Key(0),
			"int":  3,
		})
		testlib.FailIfErr(t, err, "DeleteMany failed: %v", err)

		// Verify 4 rows remain
		cursor := t.Table.Find(filter.F{"text": t.Key(0)})
		var found []astra.Row
		err = cursor.DecodeAll(t.Ctx, &found)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(found) != 4, "expected 4 rows, got %d", len(found))

		// Verify the remaining int values are [0, 1, 2, 4]
		intValues := make([]int32, len(found))
		for i, doc := range found {
			intValues[i] = doc.MustGet("int").(int32)
		}
		t.NoDiff([]int32{0, 1, 2, 4}, intValues)
	})

	s.Run("should deleteMany with a range", func(t *harness.T) {
		// Insert 50 rows
		rows := make([]astra.NewRow, 50)
		for i := 0; i < 50; i++ {
			rows[i] = astra.NewRow{
				"text": t.Key(0),
				"int":  i,
			}
		}
		_, err := t.Table.InsertMany(t.Ctx, rows)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Delete rows where int < 25
		err = t.Table.DeleteMany(t.Ctx, filter.F{
			"text": t.Key(0),
			"int":  filter.F{"$lt": 25},
		})
		testlib.FailIfErr(t, err, "DeleteMany failed: %v", err)

		// Verify 25 rows remain
		cursor := t.Table.Find(filter.F{"text": t.Key(0)})
		var found []astra.Row // TODO better error if using Document instead of Row
		err = cursor.DecodeAll(t.Ctx, &found)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(found) != 25, "expected 25 rows, got %d", len(found))

		// Verify the remaining int values are [25, 26, ..., 49]
		intValues := make([]int32, len(found))
		for i, doc := range found {
			intValues[i] = doc.MustGet("int").(int32)
		}
		expectedValues := make([]int32, 25)
		for i := 0; i < 25; i++ {
			expectedValues[i] = int32(i + 25)
		}
		t.NoDiff(expectedValues, intValues)
	})

	s.Run("should delete all documents given an empty filter", func(t *harness.T) {
		// Insert 100 rows
		rows := make([]astra.NewRow, 100)
		for i := 0; i < 100; i++ {
			rows[i] = astra.NewRow{
				"text": t.Key(0),
				"int":  i,
			}
		}
		_, err := t.Table_.InsertMany(t.Ctx, rows)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Delete all rows with empty filter
		err = t.Table_.DeleteMany(t.Ctx, filter.F{})
		testlib.FailIfErr(t, err, "DeleteMany failed: %v", err)

		// Verify no rows remain
		cursor := t.Table_.Find(filter.F{})
		var found []astra.Row
		err = cursor.DecodeAll(t.Ctx, &found)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(found) != 0, "expected 0 rows, got %d", len(found))
	})
}
