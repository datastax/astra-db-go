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
	"fmt"

	"github.com/datastax/astra-db-go/cursors"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/serdes"
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

// region Meta

// Name returns the table name.
func (t *Table) Name() string {
	return t.name
}

// ClientOptions returns the table's options (or empty options if nil).
func (t *Table) ClientOptions() *options.APIOptions {
	if t.options == nil {
		return &options.APIOptions{}
	}
	return t.options
}

// Database returns the parent database.
func (t *Table) Database() *Db {
	return t.db
}

// newCmd creates a command for this table. Will merge opts (if any) and apply them
// as command-level options.
func (t *Table) newCmd(name string, payload any, cmdOpts ...options.APIOption) command {
	return newCmdWithOptions(t.db, t.name, name, payload, t.options, serdes.TargetTable, cmdOpts...)
}

// newCmdWithMergedOptions creates a command with a pre-built *APIOptions override,
// used by builder-pattern methods where API options flow through the struct.
func (t *Table) newCmdWithMergedOptions(name string, payload any, cmdOpts *options.APIOptions) command {
	return newCmdWithMergedOptions(t.db, t.name, name, payload, t.options, serdes.TargetTable, cmdOpts)
}

// endregion

// region Definition

// Definition retrieves the table's descriptor including its definition.
// This method calls the database's ListTables and returns the descriptor
// for this specific table.
//
// Options passed here override those set on the table.
func (t *Table) Definition(ctx context.Context, opts ...options.TableDefinitionOption) (*results.TableDescriptor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	tables, err := t.db.ListTables(ctx, &options.ListTablesOptions{APIOptions: merged.APIOptions})
	if err != nil {
		return nil, err
	}

	for _, tbl := range tables {
		if tbl.Name == t.name {
			return &tbl, nil
		}
	}

	return nil, ErrNotFound
}

// endregion

// region Insertions

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
func (t *Table) InsertOne(ctx context.Context, row any, opts ...options.TableInsertOneOption) (*results.InsertOneResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return insertOne(ctx, row, t.newCmdWithMergedOptions, (insertOneOptions)(*merged), serdes.TargetTable)
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
func (t *Table) InsertMany(ctx context.Context, rows any, opts ...options.TableInsertManyOption) (*results.InsertManyResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return insertMany(ctx, rows, t.newCmdWithMergedOptions, (insertManyOptions)(*merged), serdes.TargetCollection)
}

// endregion

// region Finds

// FindOne finds a single row in a table matching the filter criteria.
//
// Example usage:
//
//	result := table.FindOne(ctx, filter.Eq("id", "some-uuid"))
//	var row MyRow
//	err := result.Decode(&row)
func (t *Table) FindOne(ctx context.Context, f TableFilter, opts ...options.TableFindOneOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, nil, serdes.TargetTable, err)
	}
	return findOne(ctx, f, t.newCmdWithMergedOptions, (findOneOptions)(*merged), serdes.TargetTable)
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
//	if err := cursor.DecodeAll(ctx, &rows); err != nil {
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
//
// In the unlikely case of an option validation error while creating the cursor,
// the cursor will be returned in an unclearable errored state.
func (t *Table) Find(f TableFilter, opts ...options.TableFindOption) *cursors.TableFindCursor {
	merged, err := options.MergeAndValidate(opts...)

	fetcher := func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, *table.LazySchema, error) {
		cmd := t.newCmdWithMergedOptions("find", payload, merged.APIOptions)
		return cmd.Execute(ctx)
	}

	return cursors.NewTableFindCursor(f, merged, fetcher, err)
}

// endregion

// region Updates

// UpdateOne updates a single row matching the filter.
//
// The filter must describe the complete primary key using equality on
// primary-key columns. The update parameter should be an [update.U]
// expression built via update.Table(), e.g.
// update.Table().Set("rating", 4.5).Unset("borrower").
//
// If no row matches and the update sets at least one non-null value, a new
// row is created (implicit upsert). You cannot update primary key values.
//
// Options passed here override those set on the table.
//
// Example:
//
//	err := tbl.UpdateOne(ctx,
//	    filter.F{"title": "Hidden Shadows of the Past", "author": "John Anthony"},
//	    update.Table().Set("rating", 4.5).Unset("borrower"),
//	)
func (t *Table) UpdateOne(ctx context.Context, f TableFilter, u TableUpdate, opts ...options.TableUpdateOneOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}
	_, err = updateOne(ctx, f, u, t.newCmdWithMergedOptions, updateOneOptions{nil, nil, merged.APIOptions}, serdes.TargetTable)
	return err
}

// endregion

// region Deletions

// DeleteOne deletes a single row matching the filter.
//
// The filter must describe the complete primary key using equality on
// primary-key columns. If no row matches, DeleteOne is a no-op and returns
// nil.
//
// Options passed here override those set on the table.
//
// Example:
//
//	err := tbl.DeleteOne(ctx,
//	    filter.F{"title": "Hidden Shadows of the Past", "author": "John Anthony"},
//	)
func (t *Table) DeleteOne(ctx context.Context, f TableFilter, opts ...options.TableDeleteOneOption) error {
	deleteOpts, err := options.MergeAndValidate(opts...)
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}
	// Note: warnings are accessible via the WarningHandler option callback only.
	_, err = deleteOne(ctx, f, t.newCmdWithMergedOptions, deleteOneOptions{Sort: nil, APIOptions: deleteOpts.APIOptions}, serdes.TargetTable)
	return err
}

// DeleteMany deletes all rows in the table matching the filter.
//
// The filter must reference only primary-key columns per the Data API rules
// for table deleteMany. An empty filter (filter.F{}) deletes every row in the
// table; a nil filter is rejected to avoid accidental total deletes.
//
// The Data API always returns deletedCount = -1 for this command, so no count
// is surfaced to the caller; the method returns only an error.
//
// Options passed here override those set on the table.
//
// Example:
//
//	err := tbl.DeleteMany(ctx,
//		filter.F{"title": "Hidden Shadows of the Past", "author": "John Anthony"},
//	)
func (t *Table) DeleteMany(ctx context.Context, f TableFilter, opts ...options.TableDeleteManyOption) error {
	deleteOpts, err := options.MergeAndValidate(opts...)
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}
	if f == nil {
		// Force the user to pass empty filter to avoid accidental delete all.
		return ErrNilFilter
	}

	cmd := t.newCmdWithMergedOptions("deleteMany", map[string]any{
		"filter": f,
	}, deleteOpts.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// endregion

// region Indexe Creation

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
	// Note: warnings are accessible via the WarningHandler option callback only.
	_, _, _, err = cmd.Execute(ctx)
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
	// Note: warnings are accessible via the WarningHandler option callback only.
	_, _, _, err = cmd.Execute(ctx)
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

// UnmarshalAstraRaw implements custom unmarshaling for IndexDescriptor.
// The API returns either a string (name only) or an object (full metadata)
// depending on the explain option.
func (d *IndexDescriptor) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	// Try to unmarshal as a string first (names only response)
	var name string
	if err := serdes.Deserialize(value, &name, nil, ctx.Target); err == nil {
		d.Name = name
		return nil
	}

	// Otherwise unmarshal as an object (explain=true response)
	type indexDescriptorAlias IndexDescriptor
	var alias indexDescriptorAlias
	if err := serdes.Deserialize(value, &alias, nil, ctx.Target); err != nil {
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

// endregion

// region Index Listing

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
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp listIndexesResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetTable); err != nil {
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

// endregion

// region Index Deletion

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
// Note: warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropTableIndex(ctx context.Context, name string) error {
	cmd := dropTableIndexCommand(d, name)
	_, _, _, err := cmd.Execute(ctx)
	return err
}

// dropTableIndexCommand builds the dropIndex command for the database
func dropTableIndexCommand(d *Db, name string) command {
	return d.newCmd("dropIndex", dropIndexPayload{Name: name})
}

// endregion

// region Altering

// alterTablePayload is the payload for the alterTable command.
type alterTablePayload struct {
	Operation table.AlterOperation `json:"operation"`
}

// Alter modifies the table's schema. Exactly one operation must be set on
// op — Add, Drop, AddVectorize, or DropVectorize. The four operations are
// mutually exclusive per call.
//
// Note that the Data API does not allow column type changes (drop and re-add
// instead) and does not support renaming a table. Dropping a vectorize
// integration preserves any embeddings already stored in the column; only
// the auto-embedding integration is removed.
//
// After adding columns, index any new columns you intend to filter or sort on.
//
// Example — add columns:
//
//	err := tbl.Alter(ctx, table.AlterOperation{
//	    Add: &table.AddColumns{
//	        Columns: table.Columns{
//	            "is_summer_reading": table.Boolean(),
//	            "library_branch":    table.Text(),
//	        },
//	    },
//	})
//
// Example — drop columns:
//
//	err := tbl.Alter(ctx, table.AlterOperation{
//	    Drop: &table.DropColumns{Columns: []string{"borrower"}},
//	})
//
// Example — add vectorize on a vector column:
//
//	err := tbl.Alter(ctx, table.AlterOperation{
//	    AddVectorize: &table.AddVectorize{
//	        Columns: map[string]table.VectorService{
//	            "summary_vec": {
//	                Provider:  "openai",
//	                ModelName: "text-embedding-3-small",
//	                Authentication: map[string]string{
//	                    "providerKey": "OPENAI_API_KEY",
//	                },
//	            },
//	        },
//	    },
//	})
//
// Example — drop vectorize:
//
//	err := tbl.Alter(ctx, table.AlterOperation{
//	    DropVectorize: &table.DropVectorize{Columns: []string{"summary_vec"}},
//	})
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (t *Table) Alter(ctx context.Context, op table.AlterOperation, opts ...options.AlterTableOption) error {
	if err := validateAlterOperation(op); err != nil {
		return err
	}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	cmd := t.newCmdWithMergedOptions("alterTable", alterTablePayload{
		Operation: op,
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// validateAlterOperation enforces that exactly one operation field is set on
// the alterTable operation. The Data API rejects payloads with zero or more
// than one operation; we surface that locally to give callers a clear error.
func validateAlterOperation(op table.AlterOperation) error {
	count := 0
	if op.Add != nil {
		count++
	}
	if op.Drop != nil {
		count++
	}
	if op.AddVectorize != nil {
		count++
	}
	if op.DropVectorize != nil {
		count++
	}
	if count == 0 {
		return fmt.Errorf("alterTable: operation must set one of Add, Drop, AddVectorize, DropVectorize")
	}
	if count > 1 {
		return fmt.Errorf("alterTable: only one operation may be set per call (Add, Drop, AddVectorize, DropVectorize are mutually exclusive)")
	}
	return nil
}

// endregion

// region Misc

// Drop deletes the table and all its rows. Use with caution.
func (t *Table) Drop(ctx context.Context) error {
	return t.db.DropTable(ctx, t.name)
}

// endregion
