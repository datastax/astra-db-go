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

package astradb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/datastax/astra-db-go/cursors"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/sort"
	"github.com/datastax/astra-db-go/table"
)

// TableFilter is implemented by [filter.F] and [filter.Filter].
// See the [filter package] for more details.
//
// Example composing Filters:
//
//	f := filter.Gt("num_pages", 300)
//
// Example using filter.F:
//
//	f := filter.F{"num_pages": filter.F{"$gt": 300}}
//
// [filter package]: https://pkg.go.dev/github.com/datastax/astra-db-go/filter
type TableFilter = filter.Filterable

// Table represents a table in the Astra DB.
//
// Options set on the table are inherited by all commands
// executed on it, unless overridden at the command level.
type Table struct {
	db      *Db
	name    string
	options *options.APIOptions
}

// Name returns the table name.
func (t *Table) Name() string {
	return t.name
}

// Options returns the table's options (or empty options if nil).
func (t *Table) Options() *options.APIOptions {
	if t.options == nil {
		return &options.APIOptions{}
	}
	return t.options
}

// Database returns the parent database.
func (t *Table) Database() *Db {
	return t.db
}

// newCmd creates a command for this table
func (t *Table) newCmd(name string, payload any, opts ...options.APIOption) command {
	return newCmdWithOptions(t.db, t.name, name, payload, t.options, opts...)
}

// newCmdOverride creates a command with a pre-built *APIOptions override,
// used by builder-pattern methods where API options flow through the struct.
func (t *Table) newCmdOverride(name string, payload any, override *options.APIOptions) command {
	return command{
		db:              t.db,
		name:            name,
		resourceName:    t.name,
		payload:         payload,
		resourceOptions: t.options,
		commandOptions:  override,
	}
}

// createTablePayload is the payload for the createTable command
type createTablePayload struct {
	Name       string           `json:"name"`
	Definition table.Definition `json:"definition"`
	Options    *createTableOpts `json:"options,omitempty"`
}

// createTableOpts represents the options sub-object in createTable payload
type createTableOpts struct {
	IfNotExists bool `json:"ifNotExists,omitempty"`
}

// createTableResponse represents the response from createTable
type createTableResponse struct {
	Status struct {
		OK int `json:"ok"`
	} `json:"status"`
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
	return &Table{
		db:      d,
		name:    name,
		options: options.NewAPIOptions(opts...),
	}
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
//		Columns: map[string]table.Column{
//			"title":           table.Text(),
//			"number_of_pages": table.Int(),
//			"rating":          table.Float(),
//			"is_checked_out":  table.Boolean(),
//		},
//		PrimaryKey: table.PrimaryKey{
//			PartitionBy: []string{"title"},
//		},
//	}
//	tbl, err := db.CreateTable(ctx, "my_table", definition)
func (d *Db) CreateTable(ctx context.Context, name string, definition table.Definition, opts ...options.CreateTableOption) (*Table, error) {
	// Apply options
	tableOpts, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	payload := createTablePayload{
		Name:       name,
		Definition: definition,
	}

	// Add options if ifNotExists is set
	if ptr.From(tableOpts.IfNotExists) {
		payload.Options = &createTableOpts{
			IfNotExists: true,
		}
	}

	cmd := d.newCmd("createTable", payload)

	// Override keyspace if specified in options
	if tableOpts.Keyspace != nil {
		cmd.keyspace = *tableOpts.Keyspace
	}

	// Execute the command
	// Response is in format: {"status":{"ok":1}}
	// Note: Warnings are accessible via the WarningHandler option callback only.
	_, _, err = cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	return &Table{
		db:   d,
		name: name,
	}, nil
}

// dropTablePayload is the payload for the dropTable command
type dropTablePayload struct {
	Name string `json:"name"`
}

// DropTable drops (deletes) a table from the database.
//
// Example usage:
//
//	err := db.DropTable(ctx, "my_table")
//
// Note: Warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropTable(ctx context.Context, name string) error {
	cmd := d.newCmd("dropTable", dropTablePayload{Name: name})
	_, _, err := cmd.Execute(ctx)
	return err
}

// dropIndexPayload is the payload for the dropIndex command
type dropIndexPayload struct {
	Name string `json:"name"`
}

// DropTableIndex drops (deletes) an index from the database.
//
// Example usage:
//
//	err := db.DropTableIndex(ctx, "rating_idx")
//
// Note: Warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropTableIndex(ctx context.Context, name string) error {
	cmd := dropTableIndexCommand(d, name)
	_, _, err := cmd.Execute(ctx)
	return err
}

// dropTableIndexCommand builds the dropIndex command for the database
func dropTableIndexCommand(d *Db, name string) command {
	return d.newCmd("dropIndex", dropIndexPayload{Name: name})
}

// tableFindPayload is the payload for the find command on tables
type tableFindPayload struct {
	Filter     any             `json:"filter,omitempty"`
	Sort       sort.Sortable   `json:"sort,omitempty"`
	Projection map[string]bool `json:"projection,omitempty"`
	Options    *tableFindOpts  `json:"options,omitempty"`
}

// tableFindOpts represents the options sub-object in find payload
type tableFindOpts struct {
	Limit             *int    `json:"limit,omitempty"`
	Skip              *int    `json:"skip,omitempty"`
	IncludeSimilarity *bool   `json:"includeSimilarity,omitempty"`
	PageState         *string `json:"pageState,omitempty"`
}

// tableFindResponse is the response from the find command on tables
type tableFindResponse struct {
	Data struct {
		Documents     []json.RawMessage `json:"documents"`
		NextPageState *string           `json:"nextPageState"`
	} `json:"data"`
}

// Find returns a cursor for iterating over rows matching the filter criteria.
//
// The cursor automatically handles pagination, fetching new pages as needed.
//
// The filter parameter defines criteria for selecting rows. Pass an empty filter.F{}
// or nil to find all rows (not recommended for large tables).
//
// Use options to specify sorting, projection, limits, and other behaviors.
//
// Example using Next/Decode pattern:
//
//	cursor := tbl.Find(filter.Eq("is_checked_out", false))
//	defer cursor.Close()
//
//	for cursor.Next(ctx) {
//	    var row MyRow
//	    if err := cursor.Decode(&row); err != nil {
//	        return err
//	    }
//	    // Process row
//	}
//	if err := cursor.Err(); err != nil {
//	    return err
//	}
//
// Example getting all results at once:
//
//	cursor := tbl.Find(filter.F{})
//	var rows []MyRow
//	if err := cursor.All(ctx, &rows); err != nil {
//	    return err
//	}
//
// Example with vector search:
//
//	cursor := tbl.Find(filter.F{},
//	    options.TableFind().
//	        SetSort(sort.Vector([]float32{0.1, 0.2, 0.3})).
//	        SetIncludeSimilarity(true),
//	)
func (t *Table) Find(f TableFilter, opts ...options.TableFindOption) (*cursors.TableFindCursor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	fetcher := func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, error) {
		cmd := t.newCmdOverride("find", payload, merged.APIOptions)
		return cmd.Execute(ctx)
	}
	return cursors.NewTableFindCursor(f, merged, fetcher), nil
}

// FindOne finds a single row in a table matching the filter criteria.
//
// Example usage:
//
//	result := table.FindOne(ctx, filter.Eq("id", "some-uuid"))
//	var row MyRow
//	err := result.Decode(&row)
func (t *Table) FindOne(ctx context.Context, f any, opts ...options.TableFindOption) *results.SingleResult {
	// Validate filter type
	switch f.(type) {
	case filter.F, filter.Filter, map[string]any, nil:
		// Allowed filter types
	default:
		return results.NewSingleResult(nil, nil, fmt.Errorf("invalid filter type: %Raw", f))
	}

	// Build the find options
	findOpts, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, fmt.Errorf("invalid options: %w", err))
	}

	// Build the payload
	payload := tableFindPayload{
		Filter:     f,
		Sort:       findOpts.Sort,
		Projection: findOpts.Projection,
	}

	// Add options if any are set (limit is not applicable for findOne)
	if findOpts.IncludeSimilarity != nil {
		payload.Options = &tableFindOpts{
			IncludeSimilarity: findOpts.IncludeSimilarity,
		}
	}

	cmd := t.newCmdOverride("findOne", payload, findOpts.APIOptions)
	b, warnings, err := cmd.Execute(ctx)
	return results.NewSingleResult(b, warnings, err)
}

// tableInsertOnePayload is the payload for insertOne on tables
type tableInsertOnePayload struct {
	Document any `json:"document"`
}

// tableInsertManyPayload is the payload for insertMany on tables
type tableInsertManyPayload struct {
	Documents any `json:"documents"`
}

// TableInsertResponse represents the response from insert operations on tables.
// The InsertedIds contains the primary key values of inserted rows.
type TableInsertResponse struct {
	Status struct {
		// InsertedIds contains the primary key values of inserted rows.
		// For single-column primary keys, this will be an array of scalar values.
		// For composite/compound primary keys, this will be an array of objects
		// with the primary key column names as keys.
		InsertedIds []any `json:"insertedIds"`
		// PrimaryKeySchema describes the structure of the primary key.
		// Contains information about partition keys and clustering keys.
		PrimaryKeySchema *PrimaryKeySchema `json:"primaryKeySchema,omitempty"`
	} `json:"status"`
}

// PrimaryKeySchema describes the primary key structure returned in insert responses.
// It maps column names to their type information.
type PrimaryKeySchema map[string]ColumnTypeInfo

// ColumnTypeInfo describes the type of a column in the primary key schema
type ColumnTypeInfo struct {
	Type string `json:"type"`
}

// InsertOne inserts a single row into the table.
//
// The row parameter should be a struct or map representing the row data.
// The primary key columns must be included in the row data.
//
// Returns the inserted primary key value(s) in the response.
//
// Example usage:
//
//	type Book struct {
//		Title         string  `json:"title"`
//		Author        string  `json:"author"`
//		NumberOfPages int     `json:"number_of_pages"`
//		Rating        float32 `json:"rating"`
//	}
//
//	book := Book{
//		Title:         "The Great Gatsby",
//		Author:        "F. Scott Fitzgerald",
//		NumberOfPages: 180,
//		Rating:        4.5,
//	}
//	resp, err := table.InsertOne(ctx, book)
func (t *Table) InsertOne(ctx context.Context, row any, opts ...options.APIOption) (TableInsertResponse, error) {
	var resp TableInsertResponse
	cmd := t.newCmd("insertOne", tableInsertOnePayload{
		Document: row,
	}, opts...)
	// Note: Warnings are accessible via the WarningHandler option callback only.
	b, _, err := cmd.Execute(ctx)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(b, &resp)
	return resp, err
}

// InsertMany inserts multiple rows into the table.
//
// The rows parameter must be a non-empty slice of structs or maps representing the row data.
// The primary key columns must be included in each row.
//
// Returns the inserted primary key values in the response.
//
// Example usage:
//
//	books := []Book{
//		{Title: "Book 1", Author: "Author 1", NumberOfPages: 100, Rating: 4.0},
//		{Title: "Book 2", Author: "Author 2", NumberOfPages: 200, Rating: 4.5},
//	}
//	resp, err := table.InsertMany(ctx, books)
func (t *Table) InsertMany(ctx context.Context, rows any, opts ...options.APIOption) (TableInsertResponse, error) {
	var resp TableInsertResponse

	// Ensure we have a slice with rows
	err := ensureNonEmptySlice(rows)
	if err != nil {
		return resp, fmt.Errorf("rows: %w", err)
	}

	cmd := t.newCmd("insertMany", tableInsertManyPayload{
		Documents: rows,
	}, opts...)
	// Note: Warnings are accessible via the WarningHandler option callback only.
	b, _, err := cmd.Execute(ctx)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(b, &resp)
	return resp, err
}

// createIndexPayload is the payload for the createIndex command
type createIndexPayload struct {
	Name       string                `json:"name"`
	Definition createIndexDefinition `json:"definition"`
	Options    *createIndexOpts      `json:"options,omitempty"`
}

// createIndexDefinition defines which column to index and any index options
type createIndexDefinition struct {
	Column  any           `json:"column"` // string or map[string]string for $keys/$values
	Options *indexDefOpts `json:"options,omitempty"`
}

// indexDefOpts contains options for text index behavior
type indexDefOpts struct {
	Ascii         *bool `json:"ascii,omitempty"`
	Normalize     *bool `json:"normalize,omitempty"`
	CaseSensitive *bool `json:"caseSensitive,omitempty"`
}

// createIndexOpts contains command-level options for index creation
type createIndexOpts struct {
	IfNotExists bool `json:"ifNotExists,omitempty"`
}

// createVectorIndexPayload is the payload for the createVectorIndex command
type createVectorIndexPayload struct {
	Name       string                      `json:"name"`
	Definition createVectorIndexDefinition `json:"definition"`
	Options    *createIndexOpts            `json:"options,omitempty"`
}

// createVectorIndexDefinition defines which column to index and vector options
type createVectorIndexDefinition struct {
	Column  string              `json:"column"`
	Options *vectorIndexDefOpts `json:"options,omitempty"`
}

// vectorIndexDefOpts contains options for vector index behavior
type vectorIndexDefOpts struct {
	Metric      string `json:"metric,omitempty"`
	SourceModel string `json:"sourceModel,omitempty"`
}

// CreateIndex creates an index on a column in the table.
//
// The column parameter can be:
//   - A string for regular column indexes: "column_name"
//   - A map for indexing map column keys or values: map[string]string{"map_col": "$keys"}
//
// For text columns, you can configure index behavior using SetAscii, SetNormalize,
// and SetCaseSensitive on the options builder.
//
// Example - basic column index:
//
//	err := tbl.CreateIndex(ctx, "rating_idx", "rating")
//
// Example - text column with case-insensitive matching:
//
//	err := tbl.CreateIndex(ctx, "title_idx", "title",
//	    options.CreateIndex().SetCaseSensitive(false))
//
// Example - map column keys index:
//
//	err := tbl.CreateIndex(ctx, "tags_idx", map[string]string{"tags": "$keys"})
//
// Example - with ifNotExists:
//
//	err := tbl.CreateIndex(ctx, "rating_idx", "rating",
//	    options.CreateIndex().SetIfNotExists(true))
//
// Example - combining multiple option sources:
//
//	err := tbl.CreateIndex(ctx, "title_idx", "title",
//	    options.CreateIndex().SetAscii(true),
//	    options.CreateIndex().SetIfNotExists(true))
func (t *Table) CreateIndex(ctx context.Context, name string, column any, opts ...options.CreateIndexOption) error {
	cmd, err := createIndexCommand(t, name, column, opts...)
	if err != nil {
		return err
	}
	// Note: Warnings are accessible via the WarningHandler option callback only.
	_, _, err = cmd.Execute(ctx)
	return err
}

// Validate index name.
func validateIndexName(idxName string) error {
	// Right now we are only checking for empty names. Rationale:
	// https://github.com/datastax/astra-db-go/pull/7#discussion_r2743808855
	if idxName == "" {
		return fmt.Errorf("index name cannot be empty")
	}
	// All good.
	return nil
}

// Column can be a string or a map for $keys/$values. Examples:
//
//   - "column_name"
//   - map[string]string{"example_map_column": "$keys"}
func validateIndexColumn(column any) error {
	// Validate type
	switch column := column.(type) {
	case string:
		// OK. But for string, make sure it's not empty
		if column == "" {
			return fmt.Errorf("index column name cannot be empty")
		}
	case map[string]string:
		// OK. But make sure not empty map.
		if len(column) == 0 {
			return fmt.Errorf("index column map cannot be empty")
		}
	default:
		return fmt.Errorf("invalid index column type: %Raw", column)
	}
	// All good.
	return nil
}

// createIndexCommand builds the createIndex command for the table
func createIndexCommand(t *Table, name string, column any, opts ...options.CreateIndexOption) (command, error) {
	if err := validateIndexName(name); err != nil {
		return command{}, err
	}
	if err := validateIndexColumn(column); err != nil {
		return command{}, err
	}
	payload := createIndexPayload{
		Name: name,
		Definition: createIndexDefinition{
			Column: column,
		},
	}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return command{}, err
	}

	// Add definition options if any text index options are set
	if merged.Ascii != nil || merged.Normalize != nil || merged.CaseSensitive != nil {
		payload.Definition.Options = &indexDefOpts{
			Ascii:         merged.Ascii,
			Normalize:     merged.Normalize,
			CaseSensitive: merged.CaseSensitive,
		}
	}

	// Add command options if ifNotExists is set
	if ptr.From(merged.IfNotExists) {
		payload.Options = &createIndexOpts{
			IfNotExists: true,
		}
	}

	return t.newCmd("createIndex", payload), nil
}

// CreateVectorIndex creates a vector index on a vector column in the table.
//
// Vector indexes enable efficient similarity search on vector columns.
// You can configure the similarity metric and source model for optimization.
//
// Example - basic vector index:
//
//	err := tbl.CreateVectorIndex(ctx, "embedding_idx", "embedding")
//
// Example - with metric and source model:
//
//	err := tbl.CreateVectorIndex(ctx, "embedding_idx", "embedding",
//	    options.CreateVectorIndex().SetMetric(options.MetricDotProduct).SetSourceModel("ada002"))
//
// Example - with ifNotExists:
//
//	err := tbl.CreateVectorIndex(ctx, "embedding_idx", "embedding",
//	    options.CreateVectorIndex().SetIfNotExists(true))
func (t *Table) CreateVectorIndex(ctx context.Context, name string, column string, opts ...options.CreateVectorIndexOption) error {
	cmd, err := createVectorIndexCommand(t, name, column, opts...)
	if err != nil {
		return err
	}
	// Note: Warnings are accessible via the WarningHandler option callback only.
	_, _, err = cmd.Execute(ctx)
	return err
}

// createVectorIndexCommand builds the createVectorIndex command for the table
func createVectorIndexCommand(t *Table, name string, column string, opts ...options.CreateVectorIndexOption) (command, error) {
	if err := validateIndexName(name); err != nil {
		return command{}, err
	}
	if err := validateIndexColumn(column); err != nil {
		return command{}, err
	}
	payload := createVectorIndexPayload{
		Name: name,
		Definition: createVectorIndexDefinition{
			Column: column,
		},
	}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return command{}, err
	}

	// Add definition options if metric or sourceModel are set
	if merged.Metric != nil || merged.SourceModel != nil {
		defOpts := &vectorIndexDefOpts{}
		if merged.Metric != nil {
			defOpts.Metric = string(*merged.Metric)
		}
		if merged.SourceModel != nil {
			defOpts.SourceModel = *merged.SourceModel
		}
		payload.Definition.Options = defOpts
	}

	// Add command options if ifNotExists is set
	if ptr.From(merged.IfNotExists) {
		payload.Options = &createIndexOpts{
			IfNotExists: true,
		}
	}

	return t.newCmd("createVectorIndex", payload), nil
}

// IndexDescriptor describes an index on a table.
// When listing indexes with explain=true, all fields are populated.
// When explain=false, only Name is populated.
type IndexDescriptor struct {
	// Name is the index identifier.
	Name string `json:"name"`
	// Definition contains the column and options for the index.
	// Only populated when explain=true.
	Definition *IndexDefinition `json:"definition,omitempty"`
	// IndexType is either "regular" or "vector".
	// Only populated when explain=true.
	IndexType string `json:"indexType,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling for IndexDescriptor.
// The API returns either a string (name only) or an object (full metadata)
// depending on the explain option.
func (d *IndexDescriptor) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first (names only response)
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		d.Name = name
		return nil
	}

	// Otherwise unmarshal as an object (explain=true response)
	type indexDescriptorAlias IndexDescriptor
	var alias indexDescriptorAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*d = IndexDescriptor(alias)
	return nil
}

// IndexDefinition describes which column is indexed and its options.
type IndexDefinition struct {
	// Column is the name of the indexed column.
	Column string `json:"column"`
	// Options contains index-specific configuration.
	Options *IndexDefinitionOptions `json:"options,omitempty"`
}

// IndexDefinitionOptions contains configuration for an index.
type IndexDefinitionOptions struct {
	// Metric is the similarity metric for vector indexes (cosine, dot_product, euclidean).
	Metric string `json:"metric,omitempty"`
	// SourceModel is the embedding model identifier for vector indexes.
	SourceModel string `json:"sourceModel,omitempty"`
	// Ascii if true, converts non-ASCII characters to US-ASCII before indexing.
	Ascii *bool `json:"ascii,omitempty"`
	// Normalize if true, applies Unicode character normalization before indexing.
	Normalize *bool `json:"normalize,omitempty"`
	// CaseSensitive if true, enforces case-sensitive matching.
	CaseSensitive *bool `json:"caseSensitive,omitempty"`
}

// listIndexesPayload is the payload for the listIndexes command
type listIndexesPayload struct {
	Options *listIndexesOpts `json:"options,omitempty"`
}

// listIndexesOpts contains options for the listIndexes command
type listIndexesOpts struct {
	Explain bool `json:"explain,omitempty"`
}

// listIndexesResponse is the response from the listIndexes command
type listIndexesResponse struct {
	Status struct {
		Indexes []IndexDescriptor `json:"indexes"`
	} `json:"status"`
}

// ListIndexes lists indexes on the table.
//
// By default, only index names are returned. Use SetExplain(true) to get
// full index metadata including column definitions and options.
//
// Example - list index names only:
//
//	indexes, err := tbl.ListIndexes(ctx)
//	for _, idx := range indexes {
//	    fmt.Println(idx.Name)
//	}
//
// Example - list with full metadata:
//
//	indexes, err := tbl.ListIndexes(ctx, options.ListIndexes().SetExplain(true))
//	for _, idx := range indexes {
//	    fmt.Printf("Index %s on column %s (type: %s)\n",
//	        idx.Name, idx.Definition.Column, idx.IndexType)
//	}
func (t *Table) ListIndexes(ctx context.Context, opts ...options.ListIndexesOption) ([]IndexDescriptor, error) {
	cmd, err := listIndexesCommand(t, opts...)
	if err != nil {
		return nil, err
	}
	b, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp listIndexesResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}

	return resp.Status.Indexes, nil
}

// listIndexesCommand builds the listIndexes command for the table
func listIndexesCommand(t *Table, opts ...options.ListIndexesOption) (command, error) {
	payload := listIndexesPayload{}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return command{}, err
	}

	// Add options if explain is set
	if ptr.From(merged.Explain) {
		payload.Options = &listIndexesOpts{
			Explain: true,
		}
	}

	return t.newCmd("listIndexes", payload), nil
}
