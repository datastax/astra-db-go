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
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("insert-many")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should throw a TableInsertManyError on failure", func(t *harness.T) {
		// Insert an empty row - this should fail because it's missing the primary key
		_, err := t.Table.InsertMany(t.Ctx, []astra.NewRow{{}})
		testlib.FailIf(t, err == nil, "expected InsertMany to fail with empty row")

		// Unwrap to InsertManyError
		var insertManyErr *results.InsertManyError
		testlib.FailIf(t, !errors.As(err, &insertManyErr), "expected error to be InsertManyError, got %T", err)

		// Check that no documents were inserted
		testlib.FailIf(t, insertManyErr.InsertedCount() != 0, "expected InsertedCount to be 0, got %d", insertManyErr.InsertedCount())

		// Check that there's at least one error
		testlib.FailIf(t, len(insertManyErr.Errors) == 0, "expected at least one error in InsertManyError.Errors")

		// Check that the first error is a DataAPIError (it's a value in the slice, so we check its type directly)
		testlib.FailIf(t, len(insertManyErr.Errors) == 0, "expected at least one error")
		// DataAPIErrors is []DataAPIError, so we can access the first error directly
		firstErr := insertManyErr.Errors[0]
		testlib.FailIf(t, firstErr.ErrorCode == "", "expected DataAPIError to have an ErrorCode")
	})
}
