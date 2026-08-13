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

package administration

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.BackgroundSuite()

	s.Run("(ADMIN) (LONG) (NOT-DEV) (ASTRA) admin-lifecycle", func(t *harness.T) {
		if t.Admin == nil {
			return
		}

		admin := t.Admin
		dbName := fmt.Sprintf("test_db_go_%d", time.Now().UnixNano())
		syncDbName := fmt.Sprintf("test_db_sync_go_%d", time.Now().UnixNano())

		// cleanup function
		cleanup := func() {
			dbs, _ := admin.ListDatabases(context.Background())
			for _, db := range dbs {
				if (strings.HasPrefix(db.Name, "test_db_go_") || strings.HasPrefix(db.Name, "test_db_sync_go_")) && db.Status != astra.DatabaseStatusTerminating {
					_ = admin.DropDatabase(context.Background(), db.ID, options.DropDatabase().SetBlocking(false))
				}
			}
		}
		cleanup()       // clean before
		defer cleanup() // clean after

		// Create database async
		asyncDbAdmin, err := admin.CreateDatabase(t.Ctx, dbName, astra.CreateDatabaseParams{
			CloudProvider: options.CloudProviderGCP,
			Region:        "us-east1",
		}, options.CreateDatabase().
			SetKeyspace("my_keyspace").
			SetBlocking(false).
			UpdateAPIOptions(options.API().SetDatabaseAdminTimeout(12*time.Minute)))

		testlib.FailIfErr(t, err, "failed to create async db: %v", err)

		asyncDb := asyncDbAdmin.Db()

		id, err := asyncDb.ID()
		testlib.FailIfErr(t, err, "failed to get id")
		testlib.FailIf(t, id == "", "db id missing")
		testlib.FailIf(t, asyncDbAdmin.ID() == "", "db admin id missing")

		// Check db info
		dbInfo1, err := asyncDbAdmin.Info(t.Ctx)
		testlib.FailIfErr(t, err, "failed to get db info: %v", err)
		testlib.FailIf(t, dbInfo1.Status != astra.DatabaseStatusPending && dbInfo1.Status != astra.DatabaseStatusInitializing, "wrong status: %s", dbInfo1.Status)
		testlib.FailIf(t, dbInfo1.Name != dbName, "wrong name")
		testlib.FailIf(t, dbInfo1.CloudProvider != string(options.CloudProviderGCP), "wrong cloud provider")
		testlib.FailIf(t, len(dbInfo1.Regions) != 1, "wrong regions length")
		testlib.FailIf(t, dbInfo1.Regions[0].Name != "us-east1", "wrong region name")
		testlib.FailIf(t, len(dbInfo1.Keyspaces) != 1 || dbInfo1.Keyspaces[0] != "my_keyspace", "wrong keyspaces")

		dbInfo2, err := admin.DatabaseInfo(t.Ctx, asyncDbAdmin.ID())
		testlib.FailIfErr(t, err, "failed to get db info: %v", err)
		t.NoDiff(dbInfo1.Name, dbInfo2.Name)
		t.NoDiff(dbInfo1.Keyspaces, dbInfo2.Keyspaces)
		testlib.FailIf(t, dbInfo2.Status != astra.DatabaseStatusPending && dbInfo2.Status != astra.DatabaseStatusInitializing, "wrong status: %s", dbInfo2.Status)

		// Create database sync
		syncDbAdmin, err := admin.CreateDatabase(t.Ctx, syncDbName, astra.CreateDatabaseParams{
			CloudProvider: options.CloudProviderGCP,
			Region:        "us-east1",
		}, options.CreateDatabase().
			SetBlocking(true).
			SetPollInterval(10*time.Second).
			UpdateAPIOptions(options.API().SetDatabaseAdminTimeout(12*time.Minute)))

		testlib.FailIfErr(t, err, "failed to create sync db: %v", err)
		syncDb := syncDbAdmin.Db()

		syncId, err := syncDb.ID()
		testlib.FailIfErr(t, err, "failed to get sync id")
		testlib.FailIf(t, syncId == "", "db id missing")
		testlib.FailIf(t, syncDbAdmin.ID() == "", "db admin id missing")

		// syncDbInfo
		dbInfo3, err := syncDbAdmin.Info(t.Ctx)
		testlib.FailIfErr(t, err, "failed to get db info: %v", err)
		testlib.FailIf(t, dbInfo3.Name != syncDbName, "wrong name")
		testlib.FailIf(t, dbInfo3.CloudProvider != string(options.CloudProviderGCP), "wrong provider")
		testlib.FailIf(t, dbInfo3.Regions[0].Name != "us-east1", "wrong region")
		testlib.FailIf(t, len(dbInfo3.Keyspaces) != 1 || dbInfo3.Keyspaces[0] != "default_keyspace", "wrong keyspace")

		// Wait for async db
		awaitStatus := func(admin *astra.AstraDatabaseAdmin, target astra.DatabaseStatus) {
			for i := 0; i < 180; i++ {
				info, err := admin.Info(t.Ctx)
				testlib.FailIfErr(t, err, "info failed")
				if info.Status == target {
					return
				}
				time.Sleep(10 * time.Second)
			}
			t.Fatalf("database did not reach status %s", target)
		}
		awaitStatus(asyncDbAdmin, astra.DatabaseStatusActive)

		dbs := []struct {
			admin *astra.AstraDatabaseAdmin
			db    *astra.Db
			typ   string
			ks    string
		}{
			{syncDbAdmin, syncDb, "sync", "default_keyspace"},
			{asyncDbAdmin, asyncDb, "async", "my_keyspace"},
		}

		for _, dbData := range dbs {
			dbAdmin := dbData.admin
			db := dbData.db
			expectedKs := dbData.ks

			info, err := dbAdmin.Info(t.Ctx)
			testlib.FailIfErr(t, err, "info failed")
			testlib.FailIf(t, info.Status != astra.DatabaseStatusActive, "status not active")
			testlib.FailIf(t, info.CloudProvider != string(options.CloudProviderGCP), "wrong provider")
			testlib.FailIf(t, len(info.Regions) != 1, "wrong regions")
			testlib.FailIf(t, info.Regions[0].Name != "us-east1", "wrong region name")
			testlib.FailIf(t, len(info.Keyspaces) != 1 || info.Keyspaces[0] != expectedKs, "wrong keyspaces")

			collections1, err := db.ListCollectionNames(t.Ctx)
			testlib.FailIfErr(t, err, "list collections failed")
			testlib.FailIf(t, len(collections1) != 0, "expected 0 collections")

			coll, err := db.CreateCollection(t.Ctx, "test_collection", nil)
			testlib.FailIfErr(t, err, "create collection failed")
			testlib.FailIf(t, coll == nil || coll.Name() != "test_collection", "wrong collection name")

			collections2, err := db.ListCollectionNames(t.Ctx)
			testlib.FailIfErr(t, err, "list collections failed")
			testlib.FailIf(t, len(collections2) != 1 || collections2[0] != "test_collection", "expected 1 collection")

			dbsList, err := admin.ListDatabases(t.Ctx)
			testlib.FailIfErr(t, err, "list dbs failed")
			idx := slices.IndexFunc(dbsList, func(d astra.FullAstraDatabaseInfo) bool { return d.ID == dbAdmin.ID() })
			testlib.FailIf(t, idx == -1, "db not found in list")

			keyspaces, err := dbAdmin.ListKeyspaces(t.Ctx)
			testlib.FailIfErr(t, err, "list keyspaces failed")
			testlib.FailIf(t, len(keyspaces) != 1, "expected 1 keyspace")
		}

		// Keyspace tests
		err = asyncDbAdmin.CreateKeyspace(t.Ctx, "other_keyspace", options.CreateKeyspace().SetBlocking(false))
		testlib.FailIfErr(t, err, "create keyspace failed")

		info4, err := asyncDbAdmin.Info(t.Ctx)
		testlib.FailIfErr(t, err, "info failed")
		testlib.FailIf(t, info4.Status != astra.DatabaseStatusMaintenance, "status not maintenance")
		t.NoDiff([]string{"my_keyspace"}, info4.Keyspaces)

		err = syncDbAdmin.CreateKeyspace(t.Ctx, "other_keyspace", options.CreateKeyspace().SetBlocking(true))
		testlib.FailIfErr(t, err, "create keyspace sync failed")

		awaitStatus(asyncDbAdmin, astra.DatabaseStatusActive)

		for _, dbData := range dbs {
			keyspaces, err := dbData.admin.ListKeyspaces(t.Ctx)
			testlib.FailIfErr(t, err, "list keyspaces failed")
			slices.Sort(keyspaces)
			expected := []string{dbData.ks, "other_keyspace"}
			slices.Sort(expected)
			t.NoDiff(expected, keyspaces)
		}

		err = asyncDbAdmin.DropKeyspace(t.Ctx, "other_keyspace", options.DropKeyspace().SetBlocking(false))
		testlib.FailIfErr(t, err, "drop keyspace failed")

		info5, err := asyncDbAdmin.Info(t.Ctx)
		testlib.FailIfErr(t, err, "info failed")
		testlib.FailIf(t, info5.Status != astra.DatabaseStatusMaintenance, "status not maintenance")

		err = syncDbAdmin.DropKeyspace(t.Ctx, "other_keyspace", options.DropKeyspace().SetBlocking(true))
		testlib.FailIfErr(t, err, "drop keyspace sync failed")

		awaitStatus(asyncDbAdmin, astra.DatabaseStatusActive)

		for _, dbData := range dbs {
			keyspaces, err := dbData.admin.ListKeyspaces(t.Ctx)
			testlib.FailIfErr(t, err, "list keyspaces failed")
			t.NoDiff([]string{dbData.ks}, keyspaces)
		}

		// Drop dbs
		err = asyncDbAdmin.Drop(t.Ctx, options.DropDatabase().SetBlocking(false))
		testlib.FailIfErr(t, err, "drop async db failed")

		info6, err := asyncDbAdmin.Info(t.Ctx)
		testlib.FailIfErr(t, err, "info failed")
		testlib.FailIf(t, info6.Status != astra.DatabaseStatusTerminating, "status not terminating")

		err = syncDbAdmin.Drop(t.Ctx, options.DropDatabase().SetBlocking(true).UpdateAPIOptions(options.API().SetDatabaseAdminTimeout(12*time.Minute)))
		testlib.FailIfErr(t, err, "drop sync db failed")

		awaitStatus(asyncDbAdmin, astra.DatabaseStatusTerminated)

		dbsList3, err := admin.ListDatabases(t.Ctx)
		testlib.FailIfErr(t, err, "list dbs failed")
		for _, dbData := range dbs {
			idx := slices.IndexFunc(dbsList3, func(d astra.FullAstraDatabaseInfo) bool { return d.ID == dbData.admin.ID() })
			testlib.FailIf(t, idx != -1, "db found in list after termination")
		}

		// Errors
		err = admin.DropDatabase(t.Ctx, syncDbAdmin.ID(), options.DropDatabase().SetBlocking(true))
		testlib.FailIf(t, err == nil, "expected dev ops api error")
	})
}
