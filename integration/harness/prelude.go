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

package harness

import (
	"fmt"
	"slices"
	"sync"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
	"github.com/fatih/color"
)

var (
	createTCWG sync.WaitGroup
	deleteTCWG sync.WaitGroup
)

func prelude() {
	db := GlobalFixtures.Db
	dbAdmin := GlobalFixtures.DbAdmin

	PrintlnBold("Running prelude...")

	awaitKeyspacesSetup(dbAdmin) // creates necessary keyspaces; deletes the 'slania' keyspace for keyspace lifecycle tests

	awaitUDTCreation(db) // sets up UDTS (must be done before creating tables which use them)

	startCreateCollections(db) // creates up tables/collections in parallel
	startCreateTables(db)

	startPruningCollections(db) // deletes leftover tables/collections/udts in parallel
	startPruningTables(db)
	startPruningUDTs(db)

	awaitCollectionTableUDTSetup()

	PrintlnBold(color.GreenString("\n✓ Prelude finished."))
}

// obscenity warning: the rest of the code under this comment is not very pleasant to the eyes
// if you're just reading to get a gist of the file, you can just trust the imperative steps of prelude()

func awaitKeyspacesSetup(dbAdmin astra.DatabaseAdmin) {
	PrintlnChecklist("Creating keyspaces")

	allKeyspaces, err := dbAdmin.ListKeyspaces(Ctx)
	testlib.PanicIfErr(err, "failed to list keyspaces during prelude")

	if slices.Contains(allKeyspaces, "slania") {
		PrintlnNestedChecklist("Deleting keyspace 'slania'")
		testlib.PanicIfErr(dbAdmin.DropKeyspace(Ctx, "slania"), "failed to drop keyspace 'slania' during prelude")
	}

	for _, keyspace := range TestKeyspaces {
		if !slices.Contains(allKeyspaces, keyspace) {
			PrintlnNestedChecklist(fmt.Sprintf("Creating keyspace '%s'", keyspace))
			testlib.PanicIfErr(dbAdmin.CreateKeyspace(Ctx, keyspace), "failed to create keyspace '%s' during prelude", keyspace)
		}
	}

	PrintlnNestedChecklist("Done!")
}

func awaitUDTCreation(db *astra.Db) {
	PrintlnChecklist("Creating UDTs")

	testlib.AwaitAll(nil, TestKeyspaces, func(ks string) (any, error) {
		return nil, db.CreateType(Ctx, DefaultUDTName, ExampleUDTSchema, options.CreateType().SetIfNotExists(true).SetKeyspace(ks))
	})

	PrintlnNestedChecklist("Done!")
}

func startCreateCollections(db *astra.Db) {
	for _, keyspace := range TestKeyspaces {
		createTCWG.Add(1)
		go func(ks string) {
			defer createTCWG.Done()

			builder := options.CreateCollection().SetKeyspace(ks)
			if ks == TestKeyspaces[0] {
				builder.UpdateVector(&options.VectorOptions{
					Dimension: ptr.To(5),
					Metric:    ptr.To("cosine"),
				})
			} else {
				builder.UpdateVector(&options.VectorOptions{
					Dimension: ptr.To(1024),
					Service: &options.VectorServiceOptions{
						Provider:  ptr.To("openai"),
						ModelName: ptr.To("text-embedding-3-small"),
					},
				})
			}

			coll, err := db.CreateCollection(Ctx, DefaultCollectionName, builder)
			testlib.PanicIfErr(err, "failed to create collection %s in keyspace %s", DefaultCollectionName, ks)

			_, err = coll.DeleteMany(Ctx, filter.F{}, options.CollectionDeleteMany().UpdateAPIOptions(options.API().SetKeyspace(ks)))
			testlib.PanicIfErr(err, "failed to clear collection %s in keyspace %s", DefaultCollectionName, ks)
		}(keyspace)
	}
}

func startCreateTables(db *astra.Db) {
	PrintlnChecklist("Started creating tables")

	for _, keyspace := range TestKeyspaces {
		createTCWG.Add(1)
		go func(ks string) {
			defer createTCWG.Done()

			schema := EverythingTableSchema
			if ks != TestKeyspaces[0] {
				schema = EverythingTableSchemaWithVectorize
			}

			tbl, err := db.CreateTable(Ctx, DefaultTableName, schema, options.CreateTable().SetIfNotExists(true).SetKeyspace(ks))
			testlib.PanicIfErr(err, "failed to create table %s in keyspace %s", DefaultTableName, ks)

			err = tbl.DeleteMany(Ctx, filter.F{}, options.TableDeleteMany().UpdateAPIOptions(options.API().SetKeyspace(ks)))
			testlib.PanicIfErr(err, "failed to clear table %s in keyspace %s", DefaultTableName, ks)

			if ks == TestKeyspaces[0] {
				err = tbl.CreateVectorIndex(Ctx, fmt.Sprintf("vector_idx_%s", ks), "vector", options.CreateVectorIndex().SetMetric(options.MetricDotProduct).SetIfNotExists(true).UpdateAPIOptions(options.API().SetKeyspace(ks)))
				testlib.PanicIfErr(err, "failed to create vector index in keyspace %s", ks)
			}

			err = tbl.CreateIndex(Ctx, fmt.Sprintf("bigint_idx_%s", ks), "bigint", options.CreateIndex().SetIfNotExists(true).UpdateAPIOptions(options.API().SetKeyspace(ks)))
			testlib.PanicIfErr(err, "failed to create bigint index in keyspace %s", ks)
		}(keyspace)
	}
}

func startPruningCollections(db *astra.Db) {
	PrintlnChecklist("Started pruning collections")

	deleteTCWG.Add(1)
	go func() {
		defer deleteTCWG.Done()

		testlib.AwaitAll(nil, TestKeyspaces, func(ks string) (any, error) {
			names, err := db.ListCollectionNames(Ctx, options.ListCollections().SetKeyspace(ks))
			if err != nil {
				return nil, err
			}

			return testlib.AwaitAll(nil, names, func(name string) (any, error) {
				if name == DefaultCollectionName {
					return nil, nil
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting collection '%s.%s'", ks, name))
				return nil, db.DropCollection(Ctx, name, options.DropCollection().SetKeyspace(ks))
			}), nil
		})
	}()
}

func startPruningTables(db *astra.Db) {
	PrintlnChecklist("Started pruning tables")

	deleteTCWG.Add(1)
	go func() {
		defer deleteTCWG.Done()

		testlib.AwaitAll(nil, TestKeyspaces, func(ks string) (any, error) {
			names, err := db.ListTableNames(Ctx, options.ListTables().SetKeyspace(ks))
			if err != nil {
				return nil, err
			}

			return testlib.AwaitAll(nil, names, func(name string) (any, error) {
				if name == DefaultTableName {
					return nil, nil
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting table '%s.%s'", ks, name))
				return nil, db.DropTable(Ctx, name, options.DropTable().SetKeyspace(ks).SetIfExists(true))
			}), nil
		})
	}()
}

func startPruningUDTs(db *astra.Db) {
	PrintlnChecklist("Started pruning UDTs")

	deleteTCWG.Add(1)
	go func() {
		defer deleteTCWG.Done()

		testlib.AwaitAll(nil, TestKeyspaces, func(ks string) (any, error) {
			names, err := db.ListTypeNames(Ctx, options.ListTypes().SetKeyspace(ks))
			if err != nil {
				return nil, err
			}

			return testlib.AwaitAll(nil, names, func(name string) (any, error) {
				if name == DefaultUDTName {
					return nil, nil
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting UDT '%s.%s'", ks, name))
				return nil, db.DropType(Ctx, name, options.DropType().SetKeyspace(ks).SetIfExists(true))
			}), nil
		})
	}()
}

func awaitCollectionTableUDTSetup() {
	PrintlnChecklist("Waiting for collection/table setup to complete")

	deleteTCWG.Wait()
	PrintlnNestedChecklist("Finished pruning")

	createTCWG.Wait()
	PrintlnNestedChecklist("Finished creation")
	PrintlnNestedChecklist("Done!")
}
