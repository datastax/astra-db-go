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

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/internal/testlib"
	"github.com/fatih/color"
)

var (
	createTCWG  sync.WaitGroup
	createUDTWG sync.WaitGroup
	listTCWG    sync.WaitGroup
	deleteTCWG  sync.WaitGroup

	collectionsToDelete sync.Map // map[string][]string
	tablesToDelete      sync.Map // map[string][]string
)

func prelude() {
	db := GlobalFixtures.Db
	dbAdmin := GlobalFixtures.DbAdmin

	PrintlnBold("Running prelude...")

	awaitKeyspacesSetup(dbAdmin) // creates necessary keyspaces; deletes the 'slania' keyspace for keyspace lifecycle tests

	startCreateUDTs(db) // sets up UDTS/tables/collections in parallel
	startCreateCollections(db)
	startCreateTables(db)
	startListingCollections(db)
	startListingTables(db)
	startDeletingCollections(db)
	startDeletingTables(db)

	awaitUDTSetup()
	awaitCollectionTableSetup()

	PrintlnBold(color.GreenString("\n✓ Prelude finished."))
}

// obscenity warning: the rest of the code under this comment is not very pleasant to the eyes
// if you're just reading to get a gist of the file, you can just trust the imperative steps of prelude()

func awaitKeyspacesSetup(dbAdmin astra.DatabaseAdmin) {
	PrintlnChecklist("Setting up keyspaces")

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

func startCreateUDTs(db *astra.Db) {
	PrintlnChecklist("Started creating UDTs")

	for _, keyspace := range TestKeyspaces {
		createUDTWG.Add(1)
		go func(ks string) {
			defer createUDTWG.Done()
			err := db.CreateType(Ctx, "example_udt", ExampleUDTSchema, options.CreateType().SetIfNotExists(true).SetKeyspace(ks))
			testlib.PanicIfErr(err, "failed to create UDT in keyspace %s during prelude", ks)
		}(keyspace)
	}
}

func startCreateCollections(db *astra.Db) {
	for _, keyspace := range TestKeyspaces {
		createTCWG.Add(1)
		go func(ks string) {
			defer createTCWG.Done()

			builder := options.CreateCollection().SetKeyspace(ks)
			if ks == TestKeyspaces[0] {
				builder.SetVector(&options.VectorOptions{
					Dimension: ptr.To(5),
					Metric:    ptr.To("cosine"),
				})
			} else {
				builder.SetVector(&options.VectorOptions{
					Dimension: ptr.To(1024),
					Service: &options.VectorServiceOptions{
						Provider:  ptr.To("openai"),
						ModelName: ptr.To("text-embedding-3-small"),
					},
				})
			}

			coll, err := db.CreateCollection(Ctx, DefaultCollectionName, builder)
			testlib.PanicIfErr(err, "failed to create collection %s in keyspace %s", DefaultCollectionName, ks)

			_, err = coll.DeleteMany(Ctx, filter.F{}, options.CollectionDeleteMany().SetAPIOptions(options.API().SetKeyspace(ks)))
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

			err = tbl.DeleteMany(Ctx, filter.F{}, options.TableDeleteMany().SetAPIOptions(options.API().SetKeyspace(ks)))
			testlib.PanicIfErr(err, "failed to clear table %s in keyspace %s", DefaultTableName, ks)

			if ks == TestKeyspaces[0] {
				err = tbl.CreateVectorIndex(Ctx, fmt.Sprintf("vector_idx_%s", ks), "vector", options.CreateVectorIndex().SetMetric(options.MetricDotProduct).SetIfNotExists(true).SetAPIOptions(options.API().SetKeyspace(ks)))
				testlib.PanicIfErr(err, "failed to create vector index in keyspace %s", ks)
			}

			err = tbl.CreateIndex(Ctx, fmt.Sprintf("bigint_idx_%s", ks), "bigint", options.CreateIndex().SetIfNotExists(true).SetAPIOptions(options.API().SetKeyspace(ks)))
			testlib.PanicIfErr(err, "failed to create bigint index in keyspace %s", ks)
		}(keyspace)
	}
}

func startListingCollections(db *astra.Db) {
	PrintlnChecklist("Started listing collections")

	for _, keyspace := range TestKeyspaces {
		listTCWG.Add(1)
		go func(ks string) {
			defer listTCWG.Done()
			names, err := db.ListCollectionNames(Ctx, options.ListCollectionNames().SetKeyspace(ks))
			testlib.PanicIfErr(err, "failed to list collections in keyspace %s", ks)
			collectionsToDelete.Store(ks, names)
		}(keyspace)
	}
}

func startListingTables(db *astra.Db) {
	PrintlnChecklist("Started listing tables")

	for _, keyspace := range TestKeyspaces {
		listTCWG.Add(1)
		go func(ks string) {
			defer listTCWG.Done()
			names, err := db.ListTableNames(Ctx, options.ListTableNames().SetKeyspace(ks))
			testlib.PanicIfErr(err, "failed to list tables in keyspace %s", ks)
			tablesToDelete.Store(ks, names)
		}(keyspace)
	}
}

func startDeletingCollections(db *astra.Db) {
	PrintlnChecklist("Started deleting collections")

	deleteTCWG.Add(1)
	go func() {
		defer deleteTCWG.Done()
		listTCWG.Wait()

		collectionsToDelete.Range(func(key, value any) bool {
			ks := key.(string)
			names := value.([]string)

			for _, name := range names {
				if slices.Contains(TestKeyspaces, ks) && name == DefaultCollectionName {
					continue
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting collection '%s.%s'", ks, name))

				deleteTCWG.Add(1)
				go func(ks, name string) {
					defer deleteTCWG.Done()
					err := db.DropCollection(Ctx, name, options.DropCollection().SetKeyspace(ks))
					testlib.PanicIfErr(err, "failed to drop collection '%s.%s' during prelude cleanup", ks, name)
				}(ks, name)
			}
			return true
		})
	}()
}

func startDeletingTables(db *astra.Db) {
	PrintlnChecklist("Started deleting tables")

	deleteTCWG.Add(1)
	go func() {
		defer deleteTCWG.Done()
		listTCWG.Wait()

		tablesToDelete.Range(func(key, value any) bool {
			ks := key.(string)
			names := value.([]string)

			for _, name := range names {
				if slices.Contains(TestKeyspaces, ks) && name == DefaultTableName {
					continue
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting table '%s.%s'", ks, name))

				deleteTCWG.Add(1)
				go func(ks, name string) {
					defer deleteTCWG.Done()
					err := db.DropTable(Ctx, name, options.DropTable().SetIfExists(true).SetKeyspace(ks))
					testlib.PanicIfErr(err, "failed to drop table '%s.%s' during prelude cleanup", ks, name)
				}(ks, name)
			}
			return true
		})
	}()
}

func awaitUDTSetup() {
	PrintlnChecklist("Waiting for UDT creation to complete")

	createUDTWG.Wait()
	PrintlnNestedChecklist("Finished creation")
	PrintlnNestedChecklist("Done!")
}

func awaitCollectionTableSetup() {
	PrintlnChecklist("Waiting for collection/table setup to complete")

	createTCWG.Wait()
	PrintlnNestedChecklist("Finished creation")

	listTCWG.Wait()
	PrintlnNestedChecklist("Finished listing")

	deleteTCWG.Wait()
	PrintlnNestedChecklist("Finished deletion")
	PrintlnNestedChecklist("Done!")
}
