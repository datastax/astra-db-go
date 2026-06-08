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

	"github.com/datastax/astra-db-go/astra/internal/command"
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

// newCmd creates a database-level command.
func (d *Db) newCmd(name string, payload any, opts ...options.APIOption) command.DataAPI {
	return command.NewDataAPICommand(d.endpoint, "", name, payload, serdes.TargetNone, options.Join(d.options, opts...))
}

// newAdminCmd creates a database-level admin command.
func (d *Db) newAdminCmd(name string, payload any, opts ...options.APIOption) command.DataAPI {
	return command.NewDataAPIAdminCommand(d.endpoint, "", name, payload, serdes.TargetNone, options.Join(d.options, opts...))
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

// UseKeyspace permanently switches the keyspace for this Db instance.
//
// All future operations on this database, and any collections/tables created
// from it, will use the new keyspace by default.
//
// Example:
//
//	db.UseKeyspace("new_keyspace")
func (d *Db) UseKeyspace(keyspace string) {
	d.options = options.Join(d.options, options.API().SetKeyspace(keyspace))
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
func (d *Db) Collection(name string, opts ...options.APIOption) *Collection { // TODO need to entirely rework options because trying to use GetCollectionOption which wraps APIOption causes major issues w/ the options hierarchy
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

// region Table/Collection/Type Creation

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

	cmd := d.newCmd("createCollection", map[string]any{
		"name":    name,
		"options": merged,
	}, merged.APIOptions)

	_, _, _, err = cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	return &Collection{
		db:      d,
		name:    name,
		options: options.Join(d.options, merged.APIOptions), // TODO this breaks things; will need to address as part of an options rework
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
	}, merged.APIOptions)

	_, _, _, err = cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	return &Table{
		db:      d,
		name:    name,
		options: options.Join(d.options, merged.APIOptions),
	}, nil
}

// CreateType creates a new user-defined type (UDT) in the database.
//
// Example usage:
//
//	definition := table.UDTDefinition{
//		Fields: table.Columns{
//			{Name: "street", Column: table.Text()},
//			{Name: "city", Column: table.Text()},
//			{Name: "zip_code", Column: table.Int()},
//		},
//	}
//	err := db.CreateType(ctx, "address", definition)
func (d *Db) CreateType(ctx context.Context, name string, definition table.UDTDefinition, opts ...options.CreateTypeOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := d.newCmd("createType", map[string]any{
		"name":       name,
		"definition": definition,
		"options":    merged,
	}, merged.APIOptions)

	_, _, _, err = cmd.Execute(ctx)
	return err
}

// alterTypePayload is the payload for the alterType command.
type alterTypePayload struct {
	Name      string                   `json:"name"`
	Operation table.AlterTypeOperation `json:"operation"`
}

// AlterType alters an existing user-defined type (UDT) in the database.
//
// Example usage:
//
//	err := db.AlterType(ctx, "address", table.AddTypeFields{
//		Fields: table.Columns{
//			{Name: "country", Column: table.Text()},
//		},
//	})
func (d *Db) AlterType(ctx context.Context, name string, op table.AlterTypeOperation, opts ...options.AlterTypeOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := d.newCmd("alterType", alterTypePayload{
		Name:      name,
		Operation: op,
	}, merged.APIOptions)

	_, _, _, err = cmd.Execute(ctx)
	return err
}

// endregion

// region Table/Collection/Index/Type Deletion

// DropCollection drops a collection from the database.
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropCollection(ctx context.Context, name string, opts ...options.DropCollectionOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}
	cmd := d.newCmd("deleteCollection", map[string]any{
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
	cmd := d.newCmd("dropTable", map[string]any{
		"name": name,
		"options": map[string]any{
			"ifExists": merged.IfExists,
		},
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
	cmd := d.newCmd("dropIndex", map[string]any{
		"name": name,
		"options": map[string]any{
			"ifExists": merged.IfExists,
		},
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// DropType drops (deletes) a user-defined type (UDT) from the database.
//
// Example usage:
//
//	err := db.DropType(ctx, "address")
func (d *Db) DropType(ctx context.Context, name string, opts ...options.DropTypeOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}
	cmd := d.newCmd("dropType", map[string]any{
		"name": name,
		"options": map[string]any{
			"ifExists": merged.IfExists,
		},
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// endregion

// region Table/Collection/Type Listing

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
	cmd := d.newCmd("findCollections", map[string]any{
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
	err = serdes.Deserialize(b, &resp, nil, serdes.TargetNone, opts.GetDesFlags())
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
	cmd := d.newCmd("listTables", map[string]any{
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
	err = serdes.Deserialize(b, &resp, nil, serdes.TargetNone, opts.GetDesFlags())
	return resp.Status.Tables, err
}

// ListTypes lists all user-defined types (UDTs) in the database with their full definitions.
//
// Example:
//
//	udts, err := db.ListTypes(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, u := range udts {
//	    fmt.Printf("UDT: %s (%d fields)\n", u.Name, len(u.Definition.Fields))
//	}
func (d *Db) ListTypes(ctx context.Context, opts ...options.ListTypesOption) ([]results.UDTDescriptor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listTypes[[]results.UDTDescriptor](d, ctx, true, merged.APIOptions)
}

// ListTypeNames lists the names of all user-defined types (UDTs) in the database.
//
// Example:
//
//	names, err := db.ListTypeNames(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, name := range names {
//	    fmt.Printf("UDT: %s\n", name)
//	}
func (d *Db) ListTypeNames(ctx context.Context, opts ...options.ListTypeNamesOption) ([]string, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listTypes[[]string](d, ctx, false, merged.APIOptions)
}

// listTypesResponse is the response from the listTypes command
type listTypesResponse[T any] struct {
	Status struct {
		Types T `json:"types"`
	} `json:"status"`
}

func listTypes[T any](d *Db, ctx context.Context, explain bool, opts *options.APIOptions) (T, error) {
	cmd := d.newCmd("listTypes", map[string]any{
		"options": map[string]any{
			"explain": explain,
		},
	}, opts)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	var resp listTypesResponse[T]
	err = serdes.Deserialize(b, &resp, nil, serdes.TargetNone, opts.GetDesFlags())
	return resp.Status.Types, err
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

// Info retrieves partial database metadata based on the database's endpoint.
// This operation requires a call to the DevOps API, which is only available on Astra databases.
func (d *Db) Info(ctx context.Context, opts ...options.DatabaseInfoOption) (*PartialAstraDatabaseInfo, error) {
	if !d.client.dataAPIBackend.IsAstra() {
		return nil, fmt.Errorf("info() is only available for Astra databases")
	}

	admin, err := d.DatabaseAdmin()
	if err != nil {
		return nil, err
	}

	astraAdmin, ok := admin.(*AstraDatabaseAdmin)
	if !ok {
		return nil, fmt.Errorf("expected AstraDatabaseAdmin, got %T", admin)
	}

	info, err := astraAdmin.Info(ctx, opts...)
	if err != nil {
		return nil, err
	}

	region, err := d.Region()
	if err != nil {
		return nil, err
	}

	return &PartialAstraDatabaseInfo{
		BaseAstraDatabaseInfo: info.BaseAstraDatabaseInfo,
		Region:                region,
		APIEndpoint:           d.Endpoint(),
	}, nil
}

// endregion
