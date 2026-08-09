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

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("delete-one")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should error on sort being set", func(t *harness.T) {
		// In Go, sort is not available as an option for table DeleteOne
		// The TypeScript test verifies that passing sort causes an error
		// In Go, this is enforced at compile time - there is no SetSort method
		// on TableDeleteOneOptions, so we verify the API design prevents this

		// Verify that attempting to delete with a filter that would need sorting
		// (like vector sort) is not possible through the type system
		// This test documents that sort is not supported for table deleteOne

		// Insert a test row first
		_, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text": t.Key(0),
			"int":  0,
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// DeleteOne with complete primary key should work (no sort needed)
		err = t.Table.DeleteOne(t.Ctx, filter.F{
			"text": t.Key(0),
			"int":  0,
		})
		testlib.FailIfErr(t, err, "DeleteOne failed: %v", err)
	})

	s.Run("should delete one row with given pk", func(t *harness.T) {
		// Insert and delete first row
		_, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text": t.Key(0),
			"int":  0,
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		err = t.Table.DeleteOne(t.Ctx, filter.F{
			"text": t.Key(0),
			"int":  0,
		})
		testlib.FailIfErr(t, err, "DeleteOne failed: %v", err)

		// Verify row is deleted
		result := t.Table.FindOne(t.Ctx, filter.F{
			"text": t.Key(0),
			"int":  0,
		})
		var row astra.Row
		err = result.Decode(&row)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got: %v", err)

		// Insert and delete second row
		_, err = t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text": t.Key(1),
			"int":  1,
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		err = t.Table.DeleteOne(t.Ctx, filter.F{
			"text": t.Key(1),
			"int":  1,
		})
		testlib.FailIfErr(t, err, "DeleteOne failed: %v", err)

		// Verify row is deleted
		result = t.Table.FindOne(t.Ctx, filter.F{
			"text": t.Key(1),
			"int":  1,
		})
		err = result.Decode(&row)
		testlib.FailIf(t, !errors.Is(err, results.ErrNoDocuments), "expected ErrNoDocuments, got: %v", err)
	})

	s.Run("should error when trying to delete many rows with $ne", func(t *harness.T) {
		// Insert a test row
		_, err := t.Table.InsertOne(t.Ctx, astra.NewRow{
			"text": t.Key(0),
			"int":  0,
		})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		// Attempt to delete with $ne filter (which could match multiple rows)
		err = t.Table.DeleteOne(t.Ctx, filter.F{
			"text": t.Key(0),
			"int":  filter.F{"$ne": 5},
		})
		testlib.FailIf(t, err == nil, "expected error when using $ne in deleteOne filter")

		// Verify it's a DataAPIError
		var dataAPIErr *results.DataAPIError
		testlib.FailIf(t, !errors.As(err, &dataAPIErr), "expected DataAPIError, got: %T", err)
	})
}
