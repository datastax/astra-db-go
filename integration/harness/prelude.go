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
	createWG sync.WaitGroup
	listWG   sync.WaitGroup
	deleteWG sync.WaitGroup

	collectionsToDelete sync.Map // map[string][]string
	tablesToDelete      sync.Map // map[string][]string
)

func prelude() {
	db := GlobalFixtures.Db
	dbAdmin := GlobalFixtures.DbAdmin

	PrintlnBold("Running prelude...")

	awaitKeyspacesSetup(dbAdmin) // creates necessary keyspaces; deletes the 'slania' keyspace for keyspace lifecycle tests

	startCreateCollections(db) // sets up tables/collections in parallel
	startCreateTables(db)
	startListingCollections(db)
	startListingTables(db)
	startDeletingCollections(db)
	startDeletingTables(db)

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

func startCreateCollections(db *astra.Db) {
	PrintlnChecklist("Creating collections")

	for _, keyspace := range TestKeyspaces {
		createWG.Add(1)
		go func(ks string) {
			defer createWG.Done()

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

	PrintlnNestedChecklist("Moved to background...")
}

func startCreateTables(db *astra.Db) {
	PrintlnChecklist("Creating tables")

	for _, keyspace := range TestKeyspaces {
		createWG.Add(1)
		go func(ks string) {
			defer createWG.Done()

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

	PrintlnNestedChecklist("Moved to background...")
}

func startListingCollections(db *astra.Db) {
	PrintlnChecklist("Listing collections")

	for _, keyspace := range TestKeyspaces {
		listWG.Add(1)
		go func(ks string) {
			defer listWG.Done()
			names, err := db.ListCollectionNames(Ctx, options.ListCollectionNames().SetKeyspace(ks))
			testlib.PanicIfErr(err, "failed to list collections in keyspace %s", ks)
			collectionsToDelete.Store(ks, names)
		}(keyspace)
	}

	PrintlnNestedChecklist("Moved to background...")
}

func startListingTables(db *astra.Db) {
	PrintlnChecklist("Listing tables")

	for _, keyspace := range TestKeyspaces {
		listWG.Add(1)
		go func(ks string) {
			defer listWG.Done()
			names, err := db.ListTableNames(Ctx, options.ListTableNames().SetKeyspace(ks))
			testlib.PanicIfErr(err, "failed to list tables in keyspace %s", ks)
			tablesToDelete.Store(ks, names)
		}(keyspace)
	}

	PrintlnNestedChecklist("Moved to background...")
}

func startDeletingCollections(db *astra.Db) {
	PrintlnChecklist("Deleting collections")

	deleteWG.Add(1)
	go func() {
		defer deleteWG.Done()
		listWG.Wait()

		collectionsToDelete.Range(func(key, value any) bool {
			ks := key.(string)
			names := value.([]string)

			for _, name := range names {
				if slices.Contains(TestKeyspaces, ks) && name == DefaultCollectionName {
					continue
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting collection '%s.%s'", ks, name))

				deleteWG.Add(1)
				go func(ks, name string) {
					defer deleteWG.Done()
					err := db.DropCollection(Ctx, name, options.DropCollection().SetKeyspace(ks))
					testlib.PanicIfErr(err, "failed to drop collection '%s.%s' during prelude cleanup", ks, name)
				}(ks, name)
			}
			return true
		})
	}()

	PrintlnNestedChecklist("Moved to background...")
}

func startDeletingTables(db *astra.Db) {
	PrintlnChecklist("Deleting tables")

	deleteWG.Add(1)
	go func() {
		defer deleteWG.Done()
		listWG.Wait()

		tablesToDelete.Range(func(key, value any) bool {
			ks := key.(string)
			names := value.([]string)

			for _, name := range names {
				if slices.Contains(TestKeyspaces, ks) && name == DefaultTableName {
					continue
				}

				PrintlnNestedChecklist(fmt.Sprintf("Deleting table '%s.%s'", ks, name))

				deleteWG.Add(1)
				go func(ks, name string) {
					defer deleteWG.Done()
					err := db.DropTable(Ctx, name, options.DropTable().SetIfExists(true).SetKeyspace(ks))
					testlib.PanicIfErr(err, "failed to drop table '%s.%s' during prelude cleanup", ks, name)
				}(ks, name)
			}
			return true
		})
	}()

	PrintlnNestedChecklist("Moved to background...")
}

func awaitCollectionTableSetup() {
	PrintlnChecklist("Waiting for collection/table setup to complete")

	createWG.Wait()
	PrintlnNestedChecklist("Finished creation")

	listWG.Wait()
	PrintlnNestedChecklist("Finished listing")

	deleteWG.Wait()
	PrintlnNestedChecklist("Finished deletion")
	PrintlnNestedChecklist("Done!")
}
