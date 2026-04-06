// Copyright DataStax, Inc.
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

package tests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	astradb "github.com/datastax/astra-db-go"
	"github.com/datastax/astra-db-go/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/options"
)

func init() {
	// Register our tests
	t := []harness.IntegrationTest{
		{Name: "AdminFindAvailableRegionsNoFilter", Run: AdminFindAvailableRegionsNoFilter},
		{Name: "AdminFindAvailableRegionsFilterByOrg", Run: AdminFindAvailableRegionsFilterByOrg},
		{Name: "AdminListDatabases", Run: AdminListDatabases},
		{Name: "AdminListDatabasesPaginated", Run: AdminListDatabasesPaginated},
		{Name: "AdminCreateDropDatabase", Run: AdminCreateDropDatabase},
		{Name: "AdminKeyspaceCreateListDrop", Run: AdminKeyspaceCreateListDrop},
		{Name: "AdminGetDatabaseNotFound", Run: AdminGetDatabaseNotFound},
	}
	harness.Register(t...)
}

func AdminFindAvailableRegionsNoFilter(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	_, err = admin.FindAvailableRegions(ctx)
	return err
}

func AdminFindAvailableRegionsFilterByOrg(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	_, err = admin.FindAvailableRegions(ctx,
		options.FindAvailableRegions().SetFilterByOrg(true))
	return err
}

func AdminListDatabases(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	databases, err := admin.ListDatabases(ctx, options.ListDatabases().SetProvider(options.CloudProviderAzure))
	if err != nil {
		return fmt.Errorf("ListDatabases failed: %w", err)
	}
	slog.Info("Listed databases", "count", len(databases))
	return nil
}

func AdminListDatabasesPaginated(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	// Using low page-size to try to  ensure pagination is exercised in tests.
	pageSize := 5
	var all []astradb.DatabaseInfo
	opts := options.ListDatabases().SetInclude(options.DatabaseStatusAll).SetLimit(pageSize)

	for page := 1; ; page++ {
		databases, err := admin.ListDatabases(ctx, opts)
		if err != nil {
			return fmt.Errorf("ListDatabases page %d failed: %w", page, err)
		}
		slog.Info("Listed databases page", "page", page, "count", len(databases))
		all = append(all, databases...)

		if len(databases) < pageSize {
			break
		}
		// Set up cursor for next page
		opts.SetStartingAfter(databases[len(databases)-1].ID)
	}

	slog.Info("Total databases found via pagination", "count", len(all))
	for _, db := range all {
		slog.Info("Database", "id", db.ID, "name", db.Name, "status", db.Status)
	}

	return nil
}

func AdminCreateDropDatabase(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	// Create a database (non-blocking to keep test fast)
	params := astradb.CreateDatabaseParams{
		Name:          "go-sdk-integration-test",
		CloudProvider: "gcp",
		Region:        "us-east1",
	}

	slog.Info("Creating database", "name", params.Name, "provider", params.CloudProvider, "region", params.Region)
	dbAdmin, err := admin.CreateDatabase(ctx, params,
		options.CreateDatabase())
	if err != nil {
		return fmt.Errorf("CreateDatabase failed: %w", err)
	}
	slog.Info("Database creation initiated", "id", dbAdmin.ID())

	// Verify we can get the database info via AstraDbAdmin.
	// Note: this is just a pass-through for AstraAdmin.GetDatabase,
	// so we don't need to also test that function directly.
	db, err := dbAdmin.Info(ctx)
	if err != nil {
		return fmt.Errorf("AstraDbAdmin.Info failed: %w", err)
	}
	slog.Info("Database info retrieved", "id", dbAdmin.ID(), "status", db.Status, "name", db.Name)

	// Drop the database via AstraDbAdmin (non-blocking)
	slog.Info("Dropping database (non-blocking)", "id", dbAdmin.ID())
	err = dbAdmin.Drop(ctx,
		options.DropDatabase().SetBlocking(false))
	if err != nil {
		return fmt.Errorf("AstraDbAdmin.Drop failed: %w", err)
	}
	slog.Info("Database drop initiated", "id", dbAdmin.ID())

	return nil
}

func AdminKeyspaceCreateListDrop(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	// Find an active database to test against
	databases, err := admin.ListDatabases(ctx,
		options.ListDatabases().SetInclude(options.DatabaseStatusActive).SetLimit(1))
	if err != nil {
		return fmt.Errorf("ListDatabases failed: %w", err)
	}
	if len(databases) == 0 {
		return fmt.Errorf("no active databases found to test keyspace operations")
	}

	db := databases[0]

	// Get an AstraDbAdmin for this database
	dbAdmin := admin.DbAdmin(db.ID)

	// Create a test keyspace with a unique name
	ksName := fmt.Sprintf("go_sdk_test_ks_%d", time.Now().UnixMilli())
	slog.Info("Creating keyspace", "name", ksName)
	err = dbAdmin.CreateKeyspace(ctx, ksName, options.CreateKeyspace().SetBlocking(true))
	if err != nil {
		return fmt.Errorf("CreateKeyspace failed: %w", err)
	}
	slog.Info("Keyspace created", "name", ksName)

	// List keyspaces and verify ours exists
	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
	if err != nil {
		return fmt.Errorf("ListKeyspaces failed: %w", err)
	}
	slog.Info("Listed keyspaces", "count", len(keyspaces), "keyspaces", keyspaces)

	found := slices.Contains(keyspaces, ksName)
	if !found {
		return fmt.Errorf("keyspace %q not found in ListKeyspaces result", ksName)
	}

	// Drop the keyspace
	slog.Info("Dropping keyspace", "name", ksName)
	err = dbAdmin.DropKeyspace(ctx, ksName)
	if err != nil {
		return fmt.Errorf("DropKeyspace failed: %w", err)
	}
	slog.Info("Keyspace dropped", "name", ksName)

	return nil
}

func AdminGetDatabaseNotFound(e *harness.TestEnv) error {
	ctx := context.Background()
	client := e.DefaultClient()
	admin, err := client.Admin()
	if err != nil {
		return fmt.Errorf("Admin() failed: %w", err)
	}

	// Attempt to get a database with an ID that doesn't exist
	_, err = admin.GetDatabase(ctx, "nonexistent-id")
	if err == nil {
		return fmt.Errorf("expected error when getting nonexistent database, got nil")
	}
	if !errors.Is(err, astradb.ErrNotFound) {
		return fmt.Errorf("expected ErrNotFound when getting nonexistent database, got: %w", err)
	}
	slog.Info("GetDatabase with nonexistent ID correctly returned error", "error", err)

	return nil
}
