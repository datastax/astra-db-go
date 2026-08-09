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

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.SequentialSuite("misc")

	s.Run("DataAPIError is thrown when doing data api operation on non-existent collections", func(t *harness.T) {
		// Get a reference to a non-existent collection
		collection := t.Db.Collection("non_existent_collection")

		// Try to insert a document - should fail with DataAPIError
		_, err := collection.InsertOne(t.Ctx, astra.NewDocument{"username": "test"})
		testlib.FailIf(t, err == nil, "expected error when inserting into non-existent collection")

		// Verify it's a DataAPIError
		var dataAPIErr *results.DataAPIError
		testlib.FailIf(t, !errors.As(err, &dataAPIErr), "expected DataAPIError, got: %T", err)
	})
}
