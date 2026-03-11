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

package options

// CreateTableOptions represents options for creating a table
type CreateTableOptions struct {
	// IfNotExists if true, the command will silently succeed even if a table
	// with the given name already exists. This only checks table names, not schemas.
	IfNotExists *bool `json:"ifNotExists,omitempty"`

	// Keyspace specifies the keyspace in which to create the table.
	// If not provided, defaults to the working keyspace for the database.
	Keyspace *string `json:"-"`
}

// List implements Builder[CreateTableOptions].
func (o *CreateTableOptions) List() []func(*CreateTableOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for CreateTableOptions.
func (o CreateTableOptions) Validate() error { return nil }

// CreateTableOptionsBuilder is a builder for CreateTableOptions.
type CreateTableOptionsBuilder struct {
	Opts []func(*CreateTableOptions)
}

// CreateTable creates a new CreateTableOptionsBuilder.
func CreateTable() *CreateTableOptionsBuilder {
	return &CreateTableOptionsBuilder{}
}

// List implements Builder[CreateTableOptions].
func (b *CreateTableOptionsBuilder) List() []func(*CreateTableOptions) {
	return b.Opts
}

// SetIfNotExists sets whether the operation should silently succeed if the table already exists.
func (b *CreateTableOptionsBuilder) SetIfNotExists(v bool) *CreateTableOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateTableOptions) { o.IfNotExists = &v })
	return b
}

// SetKeyspace sets the keyspace for the table operation.
func (b *CreateTableOptionsBuilder) SetKeyspace(v string) *CreateTableOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateTableOptions) { o.Keyspace = &v })
	return b
}

// TableFindOptions represents options for finding rows in a table
type TableFindOptions struct {
	// Sort specifies how to sort the results. Can be used for:
	// - Ascending/descending sort on columns (e.g., {"rating": 1, "title": -1})
	// - Vector search with a vector (e.g., {"vector_column": [0.1, 0.2, 0.3]})
	// - Vector search with vectorize (e.g., {"vector_column": "search text"})
	Sort map[string]any `json:"sort,omitempty"`

	// Projection controls which columns are included or excluded in the returned rows
	// Use true to include a column, false to exclude it
	Projection map[string]bool `json:"projection,omitempty"`

	// Limit limits the total number of rows returned
	Limit *int `json:"limit,omitempty"`

	// Skip specifies the number of rows to bypass before returning rows.
	// Only valid with ascending/descending sort, not with vector search.
	Skip *int `json:"skip,omitempty"`

	// IncludeSimilarity if true, includes a $similarity property in the response
	// for vector searches. Only works with direct vector search, not vectorize.
	IncludeSimilarity *bool `json:"includeSimilarity,omitempty"`

	// InitialPageState is used for pagination to fetch the next page of results
	InitialPageState *string `json:"pageState,omitempty"`
}

// List implements Builder[TableFindOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[TableFindOptions].
func (o *TableFindOptions) List() []func(*TableFindOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for TableFindOptions.
func (o TableFindOptions) Validate() error { return nil }

// TableFindOptionsBuilder is a builder for TableFindOptions.
type TableFindOptionsBuilder struct {
	Opts []func(*TableFindOptions)
}

// TableFind creates a new TableFindOptionsBuilder.
func TableFind() *TableFindOptionsBuilder {
	return &TableFindOptionsBuilder{}
}

// List implements Builder[TableFindOptions].
func (b *TableFindOptionsBuilder) List() []func(*TableFindOptions) {
	return b.Opts
}

// SetSort sets the sort option for the find operation.
func (b *TableFindOptionsBuilder) SetSort(sort map[string]any) *TableFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TableFindOptions) { o.Sort = sort })
	return b
}

// SetProjection sets the projection option for the find operation.
func (b *TableFindOptionsBuilder) SetProjection(projection map[string]bool) *TableFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TableFindOptions) { o.Projection = projection })
	return b
}

// SetLimit sets the limit option for the find operation.
func (b *TableFindOptionsBuilder) SetLimit(limit int) *TableFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TableFindOptions) { o.Limit = &limit })
	return b
}

// SetSkip sets the skip option for the find operation.
func (b *TableFindOptionsBuilder) SetSkip(skip int) *TableFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TableFindOptions) { o.Skip = &skip })
	return b
}

// SetIncludeSimilarity sets the includeSimilarity option for vector search.
func (b *TableFindOptionsBuilder) SetIncludeSimilarity(include bool) *TableFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TableFindOptions) { o.IncludeSimilarity = &include })
	return b
}

// SetPageState sets the initial page state for pagination.
func (b *TableFindOptionsBuilder) SetPageState(pageState string) *TableFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TableFindOptions) { o.InitialPageState = &pageState })
	return b
}

// SortAscending is the sort order value for ascending (1)
const SortAscending = 1

// SortDescending is the sort order value for descending (-1)
const SortDescending = -1
