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
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.SequentialSuite("options")

	s.Run("lists its own options", func(t *harness.T) {
		// Get the default collection
		collection := t.Db.Collection(harness.DefaultCollectionName)

		// Call Options() to get the collection descriptor
		descriptor, err := collection.Options(t.Ctx)
		testlib.FailIfErr(t, err, "Options() failed: %v", err)

		// Verify we got a valid descriptor
		testlib.FailIf(t, descriptor == nil, "expected non-nil descriptor")
		testlib.FailIf(t, descriptor.Name != harness.DefaultCollectionName, "expected collection name %s, got %s", harness.DefaultCollectionName, descriptor.Name)
	})

	s.Run("error is thrown when doing .options() on non-existent collections", func(t *harness.T) {
		// Get a reference to a non-existent collection
		collection := t.Db.Collection("non_existent_collection")

		// Try to get options - should fail with ErrNotFound
		_, err := collection.Options(t.Ctx)
		testlib.FailIf(t, err == nil, "expected error when calling Options() on non-existent collection")

		// Verify it's ErrNotFound
		testlib.FailIf(t, !errors.Is(err, astra.ErrNotFound), "expected ErrNotFound, got: %v", err)
	})
}
