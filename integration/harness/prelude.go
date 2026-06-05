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

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/internal/testlib"
)

func prelude() {
	db := GlobalFixtures.Db
	dbAdmin := GlobalFixtures.DbAdmin

	PrintlnBold("Running prelude...")

	awaitKeyspacesSetup(dbAdmin)

	//// 3. Setup default collection
	//fmt.Printf("Ensuring default collection '%s' exists...\n", DefaultCollectionName)
	//_, _ = db.CreateCollection(Ctx, DefaultCollectionName, options.CreateCollection().
	//	SetVector(&options.VectorOptions{
	//		Dimension: ptr.To(5),
	//		Metric:    ptr.To("cosine"),
	//	}))
	//
	//coll := db.Collection(DefaultCollectionName)
	//_, _ = coll.DeleteMany(Ctx, filter.F{})
	//
	//// 4. Setup default table
	//fmt.Printf("Ensuring default table '%s' exists...\n", DefaultTableName)
	//_, _ = db.CreateTable(Ctx, DefaultTableName, EverythingTableSchema, options.CreateTable().SetIfNotExists(true))
	//
	//tbl := db.Table(DefaultTableName)
	//_ = tbl.DeleteMany(Ctx, filter.F{})
	//
	//// 5. Cleanup other resources (Crash-only recovery)
	//fmt.Println("Cleaning up other collections...")
	//colls, err := db.ListCollectionNames(Ctx)
	//if err == nil {
	//	for _, name := range colls {
	//		if name != DefaultCollectionName {
	//			_ = db.DropCollection(Ctx, name)
	//		}
	//	}
	//}
	//
	//fmt.Println("Cleaning up other tables...")
	//tables, err := db.ListTableNames(Ctx)
	//if err == nil {
	//	for _, name := range tables {
	//		if name != DefaultTableName {
	//			_ = db.DropTable(Ctx, name)
	//		}
	//	}
	//}

	PrintlnBold("\n✓ Prelude finished.\n")
}

//	for (const keyspace of TEST_KEYSPACES) {
//	    if (!allKeyspaces.includes(keyspace)) {
//	      readline.clearLine(process.stdout, 0);
//	      console.warn(`\rcreating keyspace '${keyspace}'`);
//	      await dbAdmin.createKeyspace(keyspace);
//	    }
//	  }
func awaitKeyspacesSetup(dbAdmin astra.DatabaseAdmin) {
	PrintlnChecklist("Setting up keyspaces")

	allKeyspaces, err := dbAdmin.ListKeyspaces(Ctx)
	testlib.PanicIfErr(err, "failed to list keyspaces during prelude")

	if slices.Contains(allKeyspaces, "slania") {
		PrintlnNestedChecklist("Deleting keyspace 'slania'...")
		testlib.PanicIfErr(dbAdmin.DropKeyspace(Ctx, "slania"), "failed to drop keyspace 'slania' during prelude")
	}

	for _, keyspace := range TestKeyspaces {
		if !slices.Contains(allKeyspaces, keyspace) {
			PrintlnNestedChecklist(fmt.Sprintf("Creating keyspace '%s'...", keyspace))
			testlib.PanicIfErr(dbAdmin.CreateKeyspace(Ctx, keyspace), fmt.Sprintf("failed to create keyspace '%s' during prelude", keyspace))
		}
	}

	PrintlnNestedChecklist("Done!")
}

func startCreateCollections(db astra.Db) {
	PrintlnChecklist("Creating collections")

	PrintlnNestedChecklist("Moved to background...")
}

func startCreateTables(db astra.Db) {
	PrintlnChecklist("Creating tables")

	PrintlnNestedChecklist("Moved to background...")
}

func startListingCollections(db astra.Db) {
	PrintlnChecklist("Listing collections")

	PrintlnNestedChecklist("Moved to background...")
}

func startListingTables(db astra.Db) {
	PrintlnChecklist("Listing tables")

	PrintlnNestedChecklist("Moved to background...")
}

func startDeletingCollections(db astra.Db) {
	PrintlnChecklist("Deleting collections")

	PrintlnNestedChecklist("Moved to background...")
}

func startDeletingTables(db astra.Db) {
	PrintlnChecklist("Deleting tables")

	PrintlnNestedChecklist("Moved to background...")
}

func awaitCollectionTableSetup() {
	PrintlnChecklist("Waiting for collection/table setup to complete")

	PrintlnNestedChecklist("Finished creation")
	PrintlnNestedChecklist("Finished listing")
	PrintlnNestedChecklist("Finished deletion")
	PrintlnNestedChecklist("Done!")
}
