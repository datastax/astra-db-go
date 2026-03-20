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

package astradb

import (
	"context"
	"net/http"

	"github.com/datastax/astra-db-go/options"
)

// AstraDbAdmin provides admin operations for a specific Astra database
// via the DevOps API.
// Obtain an AstraDbAdmin from [AstraAdmin.CreateDatabase] or [AstraAdmin.DbAdmin].
//
// Example:
//
//	admin, err := client.Admin()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	dbAdmin, err := admin.CreateDatabase(ctx, astradb.CreateDatabaseParams{
//	    Name:          "my-database",
//	    CloudProvider: "gcp",
//	    Region:        "us-east1",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
type AstraDbAdmin struct {
	id    string
	admin *AstraAdmin
}

// ID returns the database ID.
func (d *AstraDbAdmin) ID() string {
	return d.id
}

// Info retrieves full database information from the DevOps API.
//
// Example:
//
//	info, err := dbAdmin.Info(ctx)
//	fmt.Println("Status:", info.Status)
func (d *AstraDbAdmin) Info(ctx context.Context) (*DatabaseInfo, error) {
	return d.admin.GetDatabase(ctx, d.id)
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
func (d *AstraDbAdmin) Drop(ctx context.Context, opts ...options.DropDatabaseOption) error {
	return d.admin.DropDatabase(ctx, d.id, opts...)
}

// ListKeyspaces returns the keyspace names for this database, with the
// default keyspace first.
//
// Example:
//
//	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
func (d *AstraDbAdmin) ListKeyspaces(ctx context.Context) ([]string, error) {
	db, err := d.admin.GetDatabase(ctx, d.id)
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
func (d *AstraDbAdmin) CreateKeyspace(ctx context.Context, keyspace string, opts ...options.CreateKeyspaceOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := d.admin.createCommand(http.MethodPost, "/databases/"+d.id+"/keyspaces/"+keyspace, nil)
	_, err = cmd.execute(ctx)
	if err != nil {
		return err
	}

	if !*merged.Blocking {
		return nil
	}

	awaitOpts := AwaitStatusOptions{
		PollInterval: *merged.PollInterval,
		Target:       DatabaseStatusActive,
		LegalStates:  []DatabaseStatus{DatabaseStatusMaintenance},
	}
	err = d.admin.awaitStatus(ctx, d.id, awaitOpts)
	return err
}

// DropKeyspace drops a keyspace from this database.
//
// Example:
//
//	err := dbAdmin.DropKeyspace(ctx, "my_keyspace")
func (d *AstraDbAdmin) DropKeyspace(ctx context.Context, keyspace string, opts ...options.DropKeyspaceOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := d.admin.createCommand(http.MethodDelete, "/databases/"+d.id+"/keyspaces/"+keyspace, nil)
	_, err = cmd.execute(ctx)
	if err != nil {
		return err
	}

	if !*merged.Blocking {
		return nil
	}

	awaitOpts := AwaitStatusOptions{
		PollInterval: *merged.PollInterval,
		Target:       DatabaseStatusActive,
		LegalStates:  []DatabaseStatus{DatabaseStatusMaintenance},
	}
	err = d.admin.awaitStatus(ctx, d.id, awaitOpts)
	return err
}
