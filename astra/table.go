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
	"fmt"

	"github.com/datastax/astra-db-go/v2/astra/cursors"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/internal/command"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/table"
	"github.com/datastax/astra-db-go/v2/astra/update"
)

// TableFilter is implemented by [filter.F] and [filter.Filter].
// See the [filter package] for more details.
//
//	f := filter.F{"num_pages": filter.F{"$gt": 300}}
//
// [filter package]: https://pkg.go.dev/github.com/datastax/astra-db-go/v2/astra/filter
type TableFilter = filter.Filterable

// TableUpdate is implemented by [update.TableUpdateBuilder] and [update.U].
// See the [update package] for more details.
//
// [update package]: https://pkg.go.dev/github.com/datastax/astra-db-go/v2/astra/update
type TableUpdate = update.TableUpdate

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

// ClientOptions returns the table's options as a resolved struct with defaults.
func (t *Table) ClientOptions() *options.APIOptions {
	return t.options
}

// Database returns the parent database.
func (t *Table) Database() *Db {
	return t.db
}

// newCmd creates a command for this table. Will merge opts (if any) and apply them
// as command-level options.
func (t *Table) newCmd(name string, payload any, opts ...options.APIOption) command.DataAPI {
	return command.NewDataAPICommand(t.db.endpoint, t.name, name, payload, serdes.TargetTable, options.Merge(append([]options.APIOption{t.options}, opts...)...))
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
		return nil, err
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
	return insertOne(ctx, row, t.newCmd, (insertOneOptions)(*merged), serdes.TargetTable)
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
	return insertMany(ctx, rows, t.newCmd, (insertManyOptions)(*merged), serdes.TargetTable)
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
		return results.NewSingleResult(nil, nil, nil, serdes.TargetTable, err, 0)
	}
	return findOne(ctx, f, t.newCmd, (findOneOptions)(*merged), serdes.TargetTable)
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

	fetcher := func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		cmd := t.newCmd("find", payload, merged.APIOptions)
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
	_, err = updateOne(ctx, f, u, t.newCmd, updateOneOptions{nil, nil, merged.APIOptions})
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
		return err
	}
	// Note: warnings are accessible via the WarningHandler option callback only.
	_, err = deleteOne(ctx, f, t.newCmd, deleteOneOptions{Sort: nil, APIOptions: deleteOpts.APIOptions})
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
		return err
	}
	if f == nil {
		// Force the user to pass empty filter to avoid accidental delete all.
		return ErrNilFilter
	}

	cmd := t.newCmd("deleteMany", map[string]any{
		"filter": f,
	}, deleteOpts.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// endregion

// region Index Creation

// CreateIndex creates an index on a column in the table.
//
// The column parameter can be:
//   - A string for regular column indexes: "column_name"
//   - A map for indexing map column keys or values: map[string]string{"map_col": "$keys"}
//
// For text columns, you can configure index behavior using SetAscii, SetNormalize,
// and SetCaseSensitive on the option builder.
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
		return fmt.Errorf("invalid index column type: %t", column)
	}
	// All good.
	return nil
}

// createIndexCommand builds the createIndex command for the table
func createIndexCommand(t *Table, name string, column any, opts ...options.CreateIndexOption) (command.DataAPI, error) {
	if err := validateIndexName(name); err != nil {
		return command.DataAPI{}, err
	}
	if err := validateIndexColumn(column); err != nil {
		return command.DataAPI{}, err
	}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return command.DataAPI{}, err
	}

	return t.newCmd("createIndex", map[string]any{
		"name": name,
		"definition": map[string]any{
			"column": column,
			"options": map[string]any{
				"caseSensitive": merged.CaseSensitive,
				"normalize":     merged.Normalize,
				"ascii":         merged.Ascii,
			},
		},
		"options": map[string]any{
			"ifNotExists": merged.IfNotExists,
		},
	}, merged.APIOptions), nil
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
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// createVectorIndexCommand builds the createVectorIndex command for the table
func createVectorIndexCommand(t *Table, name string, column string, opts ...options.CreateVectorIndexOption) (command.DataAPI, error) {
	if err := validateIndexName(name); err != nil {
		return command.DataAPI{}, err
	}
	if err := validateIndexColumn(column); err != nil {
		return command.DataAPI{}, err
	}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return command.DataAPI{}, err
	}

	return t.newCmd("createVectorIndex", map[string]any{
		"name": name,
		"definition": map[string]any{
			"column": column,
			"options": map[string]any{
				"metric":      merged.Metric,
				"sourceModel": merged.SourceModel,
			},
		},
		"options": map[string]any{
			"ifNotExists": merged.IfNotExists,
		},
	}, merged.APIOptions), nil
}

// CreateTextIndex creates a text index on a text column in the table.
//
// Example - basic text index:
//
//	err := tbl.CreateTextIndex(ctx, "content_idx", "content")
//
// Example - with analyzer:
//
//	err := tbl.CreateTextIndex(ctx, "content_idx", "content",
//	    options.CreateTextIndex().UpdateAnalyzer("standard"))
//
// Example - with ifNotExists:
//
//	err := tbl.CreateTextIndex(ctx, "content_idx", "content",
//	    options.CreateTextIndex().SetIfNotExists(true))
func (t *Table) CreateTextIndex(ctx context.Context, name string, column string, opts ...options.CreateTextIndexOption) error {
	cmd, err := createTextIndexCommand(t, name, column, opts...)
	if err != nil {
		return err
	}
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// createTextIndexCommand builds the createTextIndex command for the table
func createTextIndexCommand(t *Table, name string, column string, opts ...options.CreateTextIndexOption) (command.DataAPI, error) {
	if err := validateIndexName(name); err != nil {
		return command.DataAPI{}, err
	}
	if err := validateIndexColumn(column); err != nil {
		return command.DataAPI{}, err
	}

	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return command.DataAPI{}, err
	}

	return t.newCmd("createTextIndex", map[string]any{
		"name": name,
		"definition": map[string]any{
			"column": column,
			"options": map[string]any{
				"analyzer": merged.Analyzer,
			},
		},
		"options": map[string]any{
			"ifNotExists": merged.IfNotExists,
		},
	}, merged.APIOptions), nil
}

// endregion

// region Index Listing

// ListIndexes lists all indexes on the table with their full definitions.
//
// Example:
//
//	indexes, err := tbl.ListIndexes(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, idx := range indexes {
//	    fmt.Printf("Index: %s on column %s\n", idx.Name, idx.Definition.Column)
//	}
//
// Options passed here override those set on the database.
func (t *Table) ListIndexes(ctx context.Context, opts ...options.ListIndexesOption) ([]results.IndexDescriptor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listIndexes[[]results.IndexDescriptor](t, ctx, true, merged.APIOptions)
}

// ListIndexNames lists the names of all indexes on the table.
//
// Example:
//
//	names, err := tbl.ListIndexNames(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, name := range names {
//	    fmt.Printf("Index: %s\n", name)
//	}
//
// Options passed here override those set on the database.
func (t *Table) ListIndexNames(ctx context.Context, opts ...options.ListIndexesOption) ([]string, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return listIndexes[[]string](t, ctx, false, merged.APIOptions)
}

// listIndexesResponse is the response from the listIndexes command
type listIndexesResponse[T any] struct {
	Status struct {
		Indexes T `json:"indexes"`
	} `json:"status"`
}

func listIndexes[T any](t *Table, ctx context.Context, explain bool, opts *options.APIOptions) (T, error) {
	cmd := t.newCmd("listIndexes", map[string]any{
		"options": map[string]any{
			"explain": explain,
		},
	}, opts)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	var resp listIndexesResponse[T]
	err = serdes.Deserialize(b, &resp, nil, serdes.TargetTable, opts.GetDesFlags())
	return resp.Status.Indexes, err
}

// endregion

// region Altering

// alterTablePayload is the payload for the alterTable command.
type alterTablePayload struct {
	Operation table.AlterOperation `json:"operation"`
}

// Alter modifies the table's schema. The operation provided (AddColumns,
// DropColumns, AddVectorize, or DropVectorize) determines the action taken.
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
//	err := tbl.Alter(ctx, table.AddColumns{
//	    Columns: table.Columns{
//	        "is_summer_reading": table.Boolean(),
//	        "library_branch":    table.Text(),
//	    },
//	})
//
// Example — drop columns:
//
//	err := tbl.Alter(ctx, table.DropColumns{
//	    Columns: []string{"borrower"},
//	})
//
// Example — add vectorize on a vector column:
//
//	err := tbl.Alter(ctx, table.AddVectorize{
//	    Columns: map[string]table.VectorService{
//	        "summary_vec": {
//	            Provider:  "openai",
//	            ModelName: "text-embedding-3-small",
//	            Authentication: map[string]string{
//	                "providerKey": "OPENAI_API_KEY",
//	            },
//	        },
//	    },
//	})
//
// Example — drop vectorize:
//
//	err := tbl.Alter(ctx, table.DropVectorize{
//	    Columns: []string{"summary_vec"},
//	})
//
// Note: warnings are accessible via the WarningHandler option callback only.
func (t *Table) Alter(ctx context.Context, op table.AlterOperation, opts ...options.AlterTableOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := t.newCmd("alterTable", alterTablePayload{
		Operation: op,
	}, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// endregion

// region Misc

// Drop deletes the table and all its rows. Use with caution.
func (t *Table) Drop(ctx context.Context, opts ...options.DropTableOption) error {
	return t.db.DropTable(ctx, t.name, opts...)
}

// endregion
