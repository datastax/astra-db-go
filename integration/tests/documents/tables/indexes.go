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

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/table"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("indexes")

	indexCreationTestIndex := 0

	testIndexCreation := func(
		testName string,
		testColumnType table.Column,
		indexType string,
		createIndex func(ctx context.Context, table *astra.Table, indexName string, ifNotExists bool) error,
	) {
		s.Run(testName, func(t *harness.T) {
			name := fmt.Sprintf("create_index_table_test_%d", indexCreationTestIndex)
			indexCreationTestIndex++

			def := table.Definition{
				Columns: table.Columns{
					{"pkey", table.Text()},
					{"col", testColumnType},
				},
				PrimaryKey: table.PrimaryKey{
					PartitionBy: []string{"pkey"},
				},
			}

			tbl, err := t.Db.CreateTable(t.Ctx, name, def)
			testlib.FailIfErr(t, err, "failed to create table")

			err = createIndex(t.Ctx, tbl, name+"_index", false)
			testlib.FailIfErr(t, err, "failed to create index")

			err = createIndex(t.Ctx, tbl, name+"_index", false)
			testlib.FailIf(t, err == nil, "expected error when creating existing index")

			err = createIndex(t.Ctx, tbl, name+"_index", true)
			testlib.FailIfErr(t, err, "failed to create existing index with ifNotExists")

			indexes, err := tbl.ListIndexes(t.Ctx)
			testlib.FailIfErr(t, err, "failed to list indexes")
			testlib.FailIf(t, len(indexes) != 1, "expected 1 index, got %d", len(indexes))

			indexNames, err := tbl.ListIndexNames(t.Ctx)
			testlib.FailIfErr(t, err, "failed to list index names")
			testlib.FailIf(t, len(indexNames) != 1, "expected 1 index name, got %d", len(indexNames))

			t.NoDiff(name+"_index", indexes[0].Name)
			t.NoDiff(indexType, indexes[0].IndexType)
			testlib.FailIf(t, indexes[0].Definition == nil, "expected index definition")
			t.NoDiff(name+"_index", indexNames[0])

			err = t.Db.DropTableIndex(t.Ctx, name+"_index")
			testlib.FailIfErr(t, err, "failed to drop index")

			err = t.Db.DropTableIndex(t.Ctx, name+"_index")
			testlib.FailIf(t, err == nil, "expected error when dropping non-existent index")

			err = t.Db.DropTableIndex(t.Ctx, name+"_index", options.DropTableIndex().SetIfExists(true))
			testlib.FailIfErr(t, err, "failed to drop non-existent index with ifExists")

			err = tbl.Drop(t.Ctx)
			testlib.FailIfErr(t, err, "failed to drop table")
		})
	}

	testIndexCreation(
		"should work when createIndex-ing a scalar",
		table.Text(),
		"regular",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateIndex().SetCaseSensitive(false).SetNormalize(false).SetAscii(true)
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateIndex(ctx, indexName, "col", opts)
		},
	)

	testIndexCreation(
		"should work when createIndex-ing map entries",
		table.Map("text", table.Decimal()),
		"regular",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateIndex(ctx, indexName, "col", opts)
		},
	)

	testIndexCreation(
		"should work when createIndex-ing map keys",
		table.Map("text", table.Blob()),
		"regular",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateIndex(ctx, indexName, map[string]string{"col": "$keys"}, opts)
		},
	)

	testIndexCreation(
		"should work when createIndex-ing map values",
		table.Map("text", table.BigInt()),
		"regular",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateIndex(ctx, indexName, map[string]string{"col": "$values"}, opts)
		},
	)

	testIndexCreation(
		"should work when createIndex-ing list values",
		table.List(table.Inet()),
		"regular",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateIndex(ctx, indexName, map[string]string{"col": "$values"}, opts)
		},
	)

	testIndexCreation(
		"should work when createIndex-ing set values",
		table.Set(table.Timestamp()),
		"regular",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateIndex(ctx, indexName, map[string]string{"col": "$values"}, opts)
		},
	)

	testIndexCreation(
		"should work when createVectorIndex-ing",
		table.Vector(3),
		"vector",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateVectorIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateVectorIndex(ctx, indexName, "col", opts)
		},
	)

	testIndexCreation(
		"should work when createTextIndex-ing",
		table.Text(),
		"text",
		func(ctx context.Context, tbl *astra.Table, indexName string, ifNotExists bool) error {
			opts := options.CreateTextIndex()
			if ifNotExists {
				opts = opts.SetIfNotExists(true)
			}
			return tbl.CreateTextIndex(ctx, indexName, "col", opts)
		},
	)
}
