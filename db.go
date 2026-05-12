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

// Package astradb implements the astra database client.
package astradb

import (
	"context"
	"fmt"
	"net/url"

	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/serdes"
	"github.com/datastax/astra-db-go/table"
)

// Db represents a connection to a specific Astra DB database.
//
// Options set on the database are inherited by all collections, tables,
// and commands created from it, unless overridden at a lower level.
type Db struct {
	endpoint string
	client   *DataAPIClient
	options  *options.APIOptions
}

// region Misc

func (d *Db) newCmd(name string, payload any, opts ...options.APIOption) command {
	return newCmdWithOptions(d, "", name, payload, d.options, serdes.TargetNone, opts...)
}

// newCmdWithMergedOptions creates a database-level command with a pre-merged
// *APIOptions for the command-level overrides.
func (d *Db) newCmdWithMergedOptions(name string, payload any, cmdOpts *options.APIOptions) command {
	return newCmdWithMergedOptions(d, "", name, payload, d.options, serdes.TargetNone, cmdOpts)
}

// Endpoint returns the database API endpoint.
func (d *Db) Endpoint() string {
	return d.endpoint
}

// Options returns the database's options (or empty options if nil).
func (d *Db) Options() *options.APIOptions {
	if d.options == nil {
		return &options.APIOptions{}
	}
	return d.options
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
//	    options.WithTimeout(60 * time.Second),
//	)
func (d *Db) Collection(name string, opts ...options.APIOption) *Collection {
	return &Collection{d, name, options.NewAPIOptions(opts...)}
}

// Table returns a Table object for the specified table name.
// This does not create the table or verify its existence.
//
// Options set here override those set on the database.
//
// Example:
//
//	tbl := db.Table("my_table",
//	    options.WithTimeout(60 * time.Second),
//	)
func (d *Db) Table(name string, opts ...options.APIOption) *Table {
	return &Table{d, name, options.NewAPIOptions(opts...)}
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
//	coll, err := db.CreateCollection(ctx, "my_collection", options.WithCreateCollectionOptions(opts))
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) CreateCollection(ctx context.Context, name string, opts ...options.CreateCollectionOption) (*Collection, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	cmd := d.newCmd("createCollection", map[string]any{
		"name":    name,
		"options": merged,
	})

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

	cmd := d.newCmd("createTable", map[string]any{
		"name":       name,
		"definition": definition,
		"options":    merged,
	})

	if merged.Keyspace != nil {
		cmd.keyspace = *merged.Keyspace
	}

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

// region Table/Collection Deletion

// DropCollection drops a collection from the database.
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropCollection(ctx context.Context, name string) error {
	cmd := d.newCmd("deleteCollection", map[string]any{
		"name": name,
	})
	_, _, _, err := cmd.Execute(ctx)
	return err
}

// DropTable drops (deletes) a table from the database.
//
// Example usage:
//
//	err := db.DropTable(ctx, "my_table")
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropTable(ctx context.Context, name string) error {
	cmd := d.newCmd("dropTable", map[string]any{
		"name": name,
	})
	_, _, _, err := cmd.Execute(ctx)
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
//   - Astra environments return an [AstraDbAdmin] (DevOps API)
//   - Non-Astra environments return a [DataAPIDatabaseAdmin] (Data API)
func (d *Db) DatabaseAdmin() (DatabaseAdmin, error) {
	// Astra backends use the DevOps API.
	if d.client.dataAPIBackend.IsAstra() {
		env := options.ParseAstraEnvironmentFromEndpoint(d.endpoint)
		admin, err := d.client.Admin(options.WithAstraEnvironment(env))
		if err != nil {
			return nil, err
		}
		id, err := parseAstraEndpoint(d.endpoint)
		if err != nil {
			return nil, fmt.Errorf("cannot parse database ID from endpoint %q: %w", d.endpoint, err)
		}
		return admin.DbAdmin(id), nil
	}
	// Non-Astra backends use the Data API.
	return &DataAPIDatabaseAdmin{db: d}, nil
}

// parseAstraEndpoint extracts the database UUID from an Astra Data API endpoint.
// The expected format is: https://{uuid}-{region}.apps.astra.datastax.com
// The UUID is always 36 characters (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func parseAstraEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint URL: %w", err)
	}
	host := u.Hostname()
	if len(host) < 36 {
		return "", fmt.Errorf("hostname too short to contain a UUID: %s", host)
	}
	return host[:36], nil
}

// endregion
