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
	"context"
	"fmt"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/table"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("alter")

	tableAlterTestIndex := 0

	testTableAlter := func(
		testName string,
		tableDefinition table.Definition,
		performAlter func(ctx context.Context, tbl *astra.Table) error,
		undoAlter func(ctx context.Context, tbl *astra.Table) error,
		testBeforeAlter func(ctx context.Context, tbl *astra.Table, t *harness.T),
		testAfterAlter func(ctx context.Context, tbl *astra.Table, t *harness.T),
	) {
		s.Run(testName, func(t *harness.T) {
			name := fmt.Sprintf("alter_table_test_%d", tableAlterTestIndex)
			tableAlterTestIndex++

			tbl, err := t.Db.CreateTable(t.Ctx, name, tableDefinition)
			testlib.FailIfErr(t, err, "failed to create table")

			testBeforeAlter(t.Ctx, tbl, t)

			err = performAlter(t.Ctx, tbl)
			testlib.FailIfErr(t, err, "failed to perform alter")

			time.Sleep(250 * time.Millisecond) // helps stop tests from occasionally failing

			testAfterAlter(t.Ctx, tbl, t)

			err = undoAlter(t.Ctx, tbl)
			testlib.FailIfErr(t, err, "failed to undo alter")

			time.Sleep(250 * time.Millisecond)

			testBeforeAlter(t.Ctx, tbl, t)

			err = tbl.Drop(t.Ctx)
			testlib.FailIfErr(t, err, "failed to drop table")
		})
	}

	testTableAlter(
		"should add & drop columns",
		table.Definition{
			Columns: table.Columns{
				{"pkey", table.Text()},
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"pkey"},
			},
		},
		func(ctx context.Context, tbl *astra.Table) error {
			return tbl.Alter(ctx, table.AddColumns{
				Columns: table.Columns{
					{"cars", table.List(table.Text())},
				},
			})
		},
		func(ctx context.Context, tbl *astra.Table) error {
			return tbl.Alter(ctx, table.DropColumns{
				Columns: []string{"cars"},
			})
		},
		func(ctx context.Context, tbl *astra.Table, t *harness.T) {
			def, err := tbl.Definition(ctx)
			testlib.FailIfErr(t, err, "failed to get definition")
			_, ok := def.Definition.Columns.Get("cars")
			testlib.FailIf(t, ok, "cars column should not exist")
		},
		func(ctx context.Context, tbl *astra.Table, t *harness.T) {
			def, err := tbl.Definition(ctx)
			testlib.FailIfErr(t, err, "failed to get definition")
			col, ok := def.Definition.Columns.Get("cars")
			testlib.FailIf(t, !ok, "cars column should exist")
			testlib.FailIf(t, col.Type != table.TypeList, "expected type list")
			testlib.FailIf(t, col.ValueType == nil || col.ValueType.Type != table.TypeText, "expected valueType text")
		},
	)

	testTableAlter(
		"should add & drop vectorize",
		table.Definition{
			Columns: table.Columns{
				{"pkey", table.Text()},
				{"vector", table.Vector(1024)},
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"pkey"},
			},
		},
		func(ctx context.Context, tbl *astra.Table) error {
			return tbl.Alter(ctx, table.AddVectorize{
				Columns: map[string]table.VectorService{
					"vector": {Provider: "openai", ModelName: "text-embedding-3-small"},
				},
			})
		},
		func(ctx context.Context, tbl *astra.Table) error {
			return tbl.Alter(ctx, table.DropVectorize{
				Columns: []string{"vector"},
			})
		},
		func(ctx context.Context, tbl *astra.Table, t *harness.T) {
			def, err := tbl.Definition(ctx)
			testlib.FailIfErr(t, err, "failed to get definition")
			col, ok := def.Definition.Columns.Get("vector")
			testlib.FailIf(t, !ok, "vector column should exist")
			testlib.FailIf(t, col.Type != table.TypeVector, "expected type vector")
			testlib.FailIf(t, col.Service != nil, "vector service should be nil")
		},
		func(ctx context.Context, tbl *astra.Table, t *harness.T) {
			def, err := tbl.Definition(ctx)
			testlib.FailIfErr(t, err, "failed to get definition")
			col, ok := def.Definition.Columns.Get("vector")
			testlib.FailIf(t, !ok, "vector column should exist")
			testlib.FailIf(t, col.Type != table.TypeVector, "expected type vector")
			testlib.FailIf(t, col.Service == nil, "vector service should not be nil")
		},
	)
}
