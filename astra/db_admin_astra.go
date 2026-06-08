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

package astra

import (
	"context"
	"net/http"

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/astra/results"
)

// AstraDatabaseAdmin provides admin operations for a specific Astra database
// via the DevOps API.
// Obtain an AstraDatabaseAdmin from [AstraAdmin.CreateDatabase] or [AstraAdmin.DatabaseAdmin].
//
// Example:
//
//	admin, err := client.Admin()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	dbAdmin, err := admin.CreateDatabase(ctx, astra.CreateDatabaseParams{
//	    Name:          "my-database",
//	    CloudProvider: "gcp",
//	    Region:        "us-east1",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
type AstraDatabaseAdmin struct {
	admin *AstraAdmin
	db    *Db
}

// Db returns the underlying Db handle.
func (a *AstraDatabaseAdmin) Db() *Db {
	return a.db
}

// ID returns the database ID.
func (a *AstraDatabaseAdmin) ID() string {
	return *a.db.id
}

// Info retrieves full database information from the DevOps API.
//
// Example:
//
//	info, err := dbAdmin.Info(ctx)
//	fmt.Println("Status:", info.Status)
func (a *AstraDatabaseAdmin) Info(ctx context.Context, opts ...options.DatabaseInfoOption) (*FullAstraDatabaseInfo, error) {
	return a.admin.DatabaseInfo(ctx, a.ID(), opts...)
}

// Drop terminates the database, permanently deleting all of its data.
//
// By default, this method blocks until the database is fully terminated.
// Use SetBlocking(false) to return immediately after the termination
// request is accepted.
//
// WARNING: This action cannot be undone.
//
// Example:
//
//	err := dbAdmin.Drop(ctx)
func (a *AstraDatabaseAdmin) Drop(ctx context.Context, opts ...options.DropDatabaseOption) error {
	return a.admin.DropDatabase(ctx, a.ID(), opts...)
}

// ListKeyspaces returns the keyspace names for this database, with the
// default keyspace first.
//
// Example:
//
//	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
func (a *AstraDatabaseAdmin) ListKeyspaces(ctx context.Context, opts ...options.ListKeyspacesOption) ([]string, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	db, err := a.admin.DatabaseInfo(ctx, a.ID(), options.DatabaseInfo().SetAPIOptions(merged.APIOptions))
	if err != nil {
		return nil, err
	}
	return db.Keyspaces, nil
}

// CreateKeyspace creates a new keyspace in this database.
//
// By default, this method blocks until the keyspace is visible. Use
// SetBlocking(false) to return immediately after the request is accepted.
//
// Example:
//
//	err := dbAdmin.CreateKeyspace(ctx, "my_keyspace")
func (a *AstraDatabaseAdmin) CreateKeyspace(ctx context.Context, keyspace string, opts ...options.CreateKeyspaceOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	if ptr.From(merged.UpdateDbKeyspace) {
		a.db.UseKeyspace(keyspace)
	}

	cmd := a.admin.createCommand(http.MethodPost, "/databases/"+a.ID()+"/keyspaces/"+keyspace, nil, nil, merged.APIOptions)
	_, err = cmd.Execute(ctx)
	if err != nil {
		return err
	}

	if !merged.GetBlocking() {
		return nil
	}

	awaitOpts := AwaitStatusOptions{
		PollInterval: merged.GetPollInterval(),
		Target:       DatabaseStatusActive,
		LegalStates:  []DatabaseStatus{DatabaseStatusMaintenance},
		APIOptions:   merged.APIOptions,
	}
	err = a.admin.awaitStatus(ctx, a.ID(), awaitOpts)
	return err
}

// DropKeyspace drops a keyspace from this database.
//
// Example:
//
//	err := dbAdmin.DropKeyspace(ctx, "my_keyspace")
func (a *AstraDatabaseAdmin) DropKeyspace(ctx context.Context, keyspace string, opts ...options.DropKeyspaceOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := a.admin.createCommand(http.MethodDelete, "/databases/"+a.ID()+"/keyspaces/"+keyspace, nil, nil, merged.APIOptions)
	_, err = cmd.Execute(ctx)
	if err != nil {
		return err
	}

	if !merged.GetBlocking() {
		return nil
	}

	awaitOpts := AwaitStatusOptions{
		PollInterval: merged.GetPollInterval(),
		Target:       DatabaseStatusActive,
		LegalStates:  []DatabaseStatus{DatabaseStatusMaintenance},
		APIOptions:   merged.APIOptions,
	}
	err = a.admin.awaitStatus(ctx, a.ID(), awaitOpts)
	return err
}

// FindEmbeddingProviders returns information about all available embedding providers
// and their supported models, authentication methods, and parameters.
//
// By default only SUPPORTED models are included. Use [options.FindEmbeddingProvidersOptions]
// to filter by a different lifecycle status, or pass [options.ModelLifecycleStatusAll] to
// include models of every status.
//
// Example:
//
//	result, err := dbAdmin.FindEmbeddingProviders(ctx)
//	if err != nil {
//	    return err
//	}
//	for name, provider := range result.EmbeddingProviders {
//	    fmt.Printf("Provider: %s (%s)\n", name, provider.DisplayName)
//	    for _, model := range provider.Models {
//	        fmt.Printf("  Model: %s\n", model.Name)
//	    }
//	}
//
// Note: Warnings are accessible via the WarningHandler option callback only.
func (a *AstraDatabaseAdmin) FindEmbeddingProviders(ctx context.Context, opts ...options.FindEmbeddingProvidersOption) (*results.FindEmbeddingProvidersResult, error) {
	return findEmbeddingProviders(a.db, ctx, opts...)
}
