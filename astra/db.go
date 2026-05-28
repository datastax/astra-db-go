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

// Package astra implements the astra database client.
package astra

import (
	"context"
	"fmt"

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/table"
)

// Db represents a connection to a specific Astra DB database.
//
// Options set on the database are inherited by all collections, tables,
// and commands created from it, unless overridden at a lower level.
type Db struct {
	endpoint string
	id       *string
	region   *string
	env      options.AstraEnvironment
	client   *DataAPIClient
	options  options.Joined[options.APIOptions]
}

// Constructors

func newDbFromID(id, region string, env options.AstraEnvironment, client *DataAPIClient, opts options.Joined[options.APIOptions]) *Db {
	return &Db{env.AstraDBEndpoint(id, region), ptr.To(id), ptr.To(region), env, client, opts}
}

func newDbFromEndpoint(endpoint string, client *DataAPIClient, opts options.Joined[options.APIOptions]) *Db {
	id, region, env := options.ParseAstraEndpoint(endpoint)
	if id != "" {
		return &Db{endpoint, ptr.To(id), ptr.To(region), env, client, opts}
	}
	return &Db{endpoint, nil, nil, env, client, opts}
}

// region Meta

// newCmdWithMergedOptions creates a database-level command with a pre-merged
// *APIOptions for the command-level overrides.
func (d *Db) newCmdWithMergedOptions(name string, payload any, cmdAPIOpt *options.APIOptions) command {
	return newCmdWithMergedOptions(d, "", name, payload, d.options, serdes.TargetNone, cmdAPIOpt)
}

// Endpoint returns the database API endpoint.
func (d *Db) Endpoint() string {
	return d.endpoint
}

// ID returns the database UUID.
//
// Only available for Astra databases connected via a standard endpoint (not a private endpoint).
//
// Example:
//
//	db := client.Database("https://<db_id>-<region>.apps.astra.datastax.com")
//	id, err := db.ID() // "<db_id>"
//
// Returns an error if the database is not an Astra database, or if the ID cannot be parsed
// from the endpoint URL.
func (d *Db) ID() (string, error) {
	if !d.client.dataAPIBackend.IsAstra() {
		return "", fmt.Errorf("db.ID() is only available for Astra databases (current backend: %s)", d.client.dataAPIBackend)
	}
	if d.id == nil {
		return "", fmt.Errorf("unexpected Astra endpoint URL %q: database ID could not be parsed", d.endpoint)
	}
	return *d.id, nil
}

// Region returns the database region (e.g. "us-east-1").
//
// Only available for Astra databases connected via a standard endpoint (not a private endpoint).
//
// Example:
//
//	db := client.Database("https://<db_id>-<region>.apps.astra.datastax.com")
//	region, err := db.Region() // "<region>"
//
// Returns an error if the database is not an Astra database, or if the region cannot be parsed
// from the endpoint URL.
func (d *Db) Region() (string, error) {
	if !d.client.dataAPIBackend.IsAstra() {
		return "", fmt.Errorf("db.Region() is only available for Astra databases (current backend: %s)", d.client.dataAPIBackend)
	}
	if d.region == nil {
		return "", fmt.Errorf("unexpected Astra endpoint URL %q: database region could not be parsed", d.endpoint)
	}
	return *d.region, nil
}

// ClientOptions returns the database's options as a resolved struct with defaults.
func (d *Db) ClientOptions() *options.APIOptions {
	return options.Merge(d.options...)
}

// Client returns the parent DataAPIClient.
func (d *Db) Client() *DataAPIClient {
	return d.client
}

// endregion

// region Table/Collection Getters

// Collection returns a handle for the named collection.
//
// Options set here override those set on the database.
//
// Example:
//
//	coll := db.Collection("my_collection",
//	    options.API().SetRequestTimeout(60 * time.Second),
//	)
func (d *Db) Collection(name string, opts ...options.APIOption) *Collection {
	return &Collection{d, name, options.Join(d.options, opts...)}
}

// Table returns a Table object for the specified table name.
// This does not create the table or verify its existence.
//
// Options set here override those set on the database.
//
// Example:
//
//	tbl := db.Table("my_table",
//	    options.API().SetRequestTimeout(60 * time.Second),
//	)
func (d *Db) Table(name string, opts ...options.APIOption) *Table {
	return &Table{d, name, options.Join(d.options, opts...)}
}

// endregion

// region Table/Collection Creation

// CreateCollection creates a collection in the database.
//
// Options can be passed using the builder pattern or as raw structs:
//
//	// No options (simple collection)
//	coll, err := db.CreateCollection(ctx, "my_collection")
//
//	// With vector options
//	coll, err := db.CreateCollection(ctx, "my_collection",
//	    options.CreateCollection().SetVector(&options.VectorOptions{
//	        Dimension: 1024,
//	        Metric:    "cosine",
//	    })
//	)
//
//	// Passing in options as raw struct
//	opts := &options.CreateCollectionOptions{
//		DefaultId: &options.CollectionDefaultIdOptions{
//			Type: options.DefaultIdTypeUUIDv7,
//		},
//	}
//	coll, err := db.CreateCollection(ctx, "my_collection", opts)
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) CreateCollection(ctx context.Context, name string, opts ...options.CreateCollectionOption) (*Collection, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	cmd := d.newCmdWithMergedOptions("createCollection", map[string]any{
		"name":    name,
		"options": merged,
	}, merged.APIOptions)

	_, _, _, err = cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	return &Collection{
		db:   d,
		name: name,
	}, nil
}

// CreateTable creates a new table in the database with the specified definition.
//
// The definition includes column names, data types, and the primary key configuration.
// After creating a table, you should index columns that you want to sort or filter
// to optimize queries.
//
// Example usage:
//
//	definition := table.Definition{
//		Columns: table.Columns{
//			{Name: "title", Column: table.Text()},
//			{Name: "number_of_pages", Column: table.Int()},
//			{Name: "rating", Column: table.Float()},
//			{Name: "is_checked_out", Column: table.Boolean()},
//		},
//		PrimaryKey: table.PrimaryKey{
//			PartitionBy: []string{"title"},
//		},
//	}
//	tbl, err := db.CreateTable(ctx, "my_table", definition)
func (d *Db) CreateTable(ctx context.Context, name string, definition table.Definition, opts ...options.CreateTableOption) (*Table, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	cmd := d.newCmdWithMergedOptions("createTable", map[string]any{
		"name":       name,
		"definition": definition,
		"options":    merged,
	}, merged.APIOptions)

	_, _, _, err = cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	return &Table{
		db:   d,
		name: name,
	}, nil
}

// endregion

// region Table/Collection/Index Deletion

// DropCollection drops a collection from the database.
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropCollection(ctx context.Context, name string, opts ...options.DropCollectionOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}
	cmd := d.newCmdWithMergedOptions("deleteCollection", map[string]any{
		"name": name,
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// DropTable drops (deletes) a table from the database.
//
// Example usage:
//
//	err := db.DropTable(ctx, "my_table")
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropTable(ctx context.Context, name string, opts ...options.DropTableOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}
	cmd := d.newCmdWithMergedOptions("dropTable", map[string]any{
		"name": name,
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// DropTableIndex drops (deletes) an index from the database.
//
// Example usage:
//
//	err := db.DropTableIndex(ctx, "rating_idx")
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropTableIndex(ctx context.Context, name string, opts ...options.DropTableIndexOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}
	cmd := d.newCmdWithMergedOptions("dropIndex", map[string]any{
		"name": name,
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// endregion

// region Table/Collection Listing

// ListCollections lists all collections in the database with their full definitions.
//
// You can specify a keyspace in the options parameter, which will override the
// working keyspace for this Db instance.
//
// Example:
//
//	collections, err := db.ListCollections(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, coll := range collections {
//	    fmt.Printf("Collection: %s\n", coll.Name)
//	    if coll.Definition.Vector != nil {
//	        fmt.Printf("  Vector dimension: %d\n", *coll.Definition.Vector.Dimension)
//	    }
//	}
//
// Options passed here override those set on the database.
func (d *Db) ListCollections(ctx context.Context, opts ...options.ListCollectionsOption) ([]results.CollectionDescriptor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listCollections[[]results.CollectionDescriptor](d, ctx, true, merged.APIOptions)
}

// ListCollectionNames lists the names of all collections in the database.
//
// You can specify a keyspace in the options parameter, which will override the
// working keyspace for this Db instance.
//
// Example:
//
//	names, err := db.ListCollectionNames(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, name := range names {
//	    fmt.Printf("Collection: %s\n", name)
//	}
//
// Options passed here override those set on the database.
func (d *Db) ListCollectionNames(ctx context.Context, opts ...options.ListCollectionNamesOption) ([]string, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listCollections[[]string](d, ctx, false, merged.APIOptions)
}

// listTablesResponse is the response from the listTables command
type listCollectionsResponse[T any] struct {
	Status struct {
		Collections T `json:"collections"`
	} `json:"status"`
}

func listCollections[T any](d *Db, ctx context.Context, explain bool, opts *options.APIOptions) (T, error) {
	cmd := d.newCmdWithMergedOptions("findCollections", map[string]any{
		"options": map[string]any{
			"explain": explain,
		},
	}, opts)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	var resp listCollectionsResponse[T]
	err = serdes.Deserialize(b, &resp, nil, serdes.TargetNone)
	return resp.Status.Collections, err
}

// ListTables lists all tables in the database with their full definitions.
//
// You can specify API options via the options parameter to override settings
// for this command.
//
// Example:
//
//	tables, err := db.ListTables(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, t := range tables {
//	    fmt.Printf("Table: %s (%d columns)\n", t.Name, len(t.Definition.Columns))
//	}
//
// Options passed here override those set on the database.
func (d *Db) ListTables(ctx context.Context, opts ...options.ListTablesOption) ([]results.TableDescriptor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listTables[[]results.TableDescriptor](d, ctx, true, merged.APIOptions)
}

// ListTableNames lists the names of all tables in the database.
//
// Example:
//
//	names, err := db.ListTableNames(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, name := range names {
//	    fmt.Printf("Table: %s\n", name)
//	}
//
// Options passed here override those set on the database.
func (d *Db) ListTableNames(ctx context.Context, opts ...options.ListTableNamesOption) ([]string, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listTables[[]string](d, ctx, false, merged.APIOptions)
}

// listTablesResponse is the response from the listTables command
type listTablesResponse[T any] struct {
	Status struct {
		Tables T `json:"tables"`
	} `json:"status"`
}

func listTables[T any](d *Db, ctx context.Context, explain bool, opts *options.APIOptions) (T, error) {
	cmd := d.newCmdWithMergedOptions("listTables", map[string]any{
		"options": map[string]any{
			"explain": explain,
		},
	}, opts)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	var resp listTablesResponse[T]
	err = serdes.Deserialize(b, &resp, nil, serdes.TargetNone)
	return resp.Status.Tables, err
}

// endregion

// region Admin

// DatabaseAdmin returns a DatabaseAdmin for managing keyspaces on this database.
// The concrete implementation depends on the environment:
//   - Astra environments return an [AstraDatabaseAdmin] (DevOps API)
//   - Non-Astra environments return a [DataAPIDatabaseAdmin] (Data API)
func (d *Db) DatabaseAdmin() (DatabaseAdmin, error) {
	// Astra backends use the DevOps API.
	if d.client.dataAPIBackend.IsAstra() {
		if _, err := d.ID(); err != nil {
			return nil, err
		}
		admin, err := d.client.Admin(options.API().SetAstraEnvironment(d.env))
		if err != nil {
			return nil, err
		}
		return &AstraDatabaseAdmin{admin: admin, db: d}, nil
	}
	// Non-Astra backends use the Data API.
	return &DataAPIDatabaseAdmin{db: d}, nil
}

// endregion
