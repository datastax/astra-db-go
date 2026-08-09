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
	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/cursors"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("delete-one")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should deleteOne document", func(t *harness.T) {
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedID any
		err = res.DecodeID(&insertedID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		deleteRes, _ := t.Collection.DeleteOne(t.Ctx, filter.Eq("_id", insertedID))
		testlib.FailIf(t, deleteRes.DeletedCount != 1, "expected DeletedCount to be 1, got %d", deleteRes.DeletedCount)
	})

	s.Run("should not delete any when no match in deleteOne", func(t *harness.T) {
		_, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		deleteRes, _ := t.Collection.DeleteOne(t.Ctx, filter.Eq("name", "band-maid"))
		testlib.FailIf(t, deleteRes.DeletedCount != 0, "expected DeletedCount to be 0, got %d", deleteRes.DeletedCount)
	})

	s.Run("should deleteOne with sort", func(t *harness.T) {
		_, err := t.Collection.InsertMany(t.Ctx, []astra.NewDocument{
			{"name": "a", "key": t.Key(0)},
			{"name": "c", "key": t.Key(0)},
			{"name": "b", "key": t.Key(0)},
		})
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		deleteRes, _ := t.Collection.DeleteOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionDeleteOne().SetSort(sort.Asc("name")),
		)
		testlib.FailIf(t, deleteRes.DeletedCount != 1, "expected DeletedCount to be 1, got %d", deleteRes.DeletedCount)

		all, err := cursors.DecodeAll[map[string]any](t.Ctx, t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Asc("name")).SetProjection(map[string]any{"_id": 0, "name": 1}),
		))

		t.NoDiff([]map[string]any{
			{"name": "b"},
			{"name": "c"},
		}, all)
	})
}
