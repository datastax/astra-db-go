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

package collection_test

import (
	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/integration/harness"
	"github.com/datastax/astra-db-go/internal/testlib"
)

func init() {
	s := harness.ParallelSuite()
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should fail to insert a document into a table", func(t *harness.T) {
		_, err := t.Table.InsertOne(t.Ctx, astra.NewDocument{})
		testlib.FailIf(t, err == nil, "expected InsertOne to fail when inserting a document into a table")
	})
}

func insertOneDecodeAny(res *results.InsertOneResult) any {
	var id any
	if err := res.DecodeID(&id); err != nil {
		panic("failed to decode inserted ID: " + err.Error())
	}
	return id
}
