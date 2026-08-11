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
	"slices"

	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.SequentialSuite("drop")

	s.Run("(LONG) should drop a collection using the collection method", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "purple_gassy_balloon",
			options.CreateCollection().
				SetKeyspace(harness.TestKeyspaces[1]).
				SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "CreateCollection failed: %v", err)

		err = coll.Drop(t.Ctx)
		testlib.FailIfErr(t, err, "Drop failed: %v", err)

		collections, err := t.Db.ListCollectionNames(t.Ctx, options.ListCollections().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "ListCollectionNames failed: %v", err)

		found := slices.Contains(collections, "purple_gassy_balloon")
		testlib.FailIf(t, !found, "expected collection to be dropped, but it still exists")
	})
}
