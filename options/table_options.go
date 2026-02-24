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

// TODO: move towards new naming scheme as well as using builder pattern.

// CreateTableOptions represents options for creating a table
type CreateTableOptions struct {
	// IfNotExists if true, the command will silently succeed even if a table
	// with the given name already exists. This only checks table names, not schemas.
	IfNotExists bool `json:"ifNotExists,omitempty"`

	// Keyspace specifies the keyspace in which to create the table.
	// If not provided, defaults to the working keyspace for the database.
	Keyspace string `json:"-"`
}

// TableOption is a functional option for configuring CreateTableOptions
type TableOption func(*CreateTableOptions)

// WithIfNotExists sets the ifNotExists option
func WithIfNotExists(ifNotExists bool) TableOption {
	return func(opts *CreateTableOptions) {
		opts.IfNotExists = ifNotExists
	}
}

// WithTableKeyspace sets the keyspace option for the table operation
func WithTableKeyspace(keyspace string) TableOption {
	return func(opts *CreateTableOptions) {
		opts.Keyspace = keyspace
	}
}

// NewCreateTableOptions creates a CreateTableOptions with the provided options applied
func NewCreateTableOptions(opts ...TableOption) *CreateTableOptions {
	options := &CreateTableOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	return options
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

// TableFindOption is a functional option for configuring TableFindOptions
type TableFindOption func(*TableFindOptions)

// WithSort sets the sort option for the find operation
func WithSort(sort map[string]any) TableFindOption {
	return func(opts *TableFindOptions) {
		opts.Sort = sort
	}
}

// WithProjection sets the projection option for the find operation
func WithProjection(projection map[string]bool) TableFindOption {
	return func(opts *TableFindOptions) {
		opts.Projection = projection
	}
}

// WithLimit sets the limit option for the find operation
func WithLimit(limit int) TableFindOption {
	return func(opts *TableFindOptions) {
		opts.Limit = &limit
	}
}

// WithSkip sets the skip option for the find operation
func WithSkip(skip int) TableFindOption {
	return func(opts *TableFindOptions) {
		opts.Skip = &skip
	}
}

// WithIncludeSimilarity sets the includeSimilarity option for vector search
func WithIncludeSimilarity(include bool) TableFindOption {
	return func(opts *TableFindOptions) {
		opts.IncludeSimilarity = &include
	}
}

// WithInitialPageState sets the initial page state for pagination
func WithInitialPageState(pageState string) TableFindOption {
	return func(opts *TableFindOptions) {
		opts.InitialPageState = &pageState
	}
}

// NewTableFindOptions creates a TableFindOptions with the provided options applied
func NewTableFindOptions(opts ...TableFindOption) *TableFindOptions {
	options := &TableFindOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	return options
}

// SortAscending is the sort order value for ascending (1)
const SortAscending = 1

// SortDescending is the sort order value for descending (-1)
const SortDescending = -1

// CollectionFindOptions represents options for finding documents in a collection
type CollectionFindOptions struct {
	// Sort specifies how to sort the results. Can be used for:
	// - Ascending/descending sort on fields (e.g., {"rating": 1, "title": -1})
	// - Vector search with a vector (e.g., {"$vector": [0.1, 0.2, 0.3]})
	// - Vector search with vectorize (e.g., {"$vectorize": "search text"})
	Sort map[string]any `json:"sort,omitempty"`

	// Projection controls which fields are included or excluded in the returned documents
	// Use true to include a field, false to exclude it
	Projection map[string]any `json:"projection,omitempty"`

	// Limit limits the total number of documents returned
	Limit *int `json:"limit,omitempty"`

	// Skip specifies the number of documents to bypass before returning results.
	// Only valid with ascending/descending sort, not with vector search.
	Skip *int `json:"skip,omitempty"`

	// IncludeSimilarity if true, includes a $similarity property in the response
	// for vector searches.
	IncludeSimilarity *bool `json:"includeSimilarity,omitempty"`

	// IncludeSortVector if true, includes the sort vector in the response.
	// Useful for vector searches using $vectorize.
	IncludeSortVector *bool `json:"includeSortVector,omitempty"`

	// InitialPageState is used for pagination to fetch the next page of results
	InitialPageState *string `json:"pageState,omitempty"`
}

// CollectionFindOption is a functional option for configuring CollectionFindOptions
type CollectionFindOption func(*CollectionFindOptions)

// WithCollectionSort sets the sort option for the find operation
func WithCollectionSort(sort map[string]any) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.Sort = sort
	}
}

// WithCollectionProjection sets the projection option for the find operation
func WithCollectionProjection(projection map[string]any) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.Projection = projection
	}
}

// WithCollectionLimit sets the limit option for the find operation
func WithCollectionLimit(limit int) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.Limit = &limit
	}
}

// WithCollectionSkip sets the skip option for the find operation
func WithCollectionSkip(skip int) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.Skip = &skip
	}
}

// WithCollectionIncludeSimilarity sets the includeSimilarity option for vector search
func WithCollectionIncludeSimilarity(include bool) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.IncludeSimilarity = &include
	}
}

// WithCollectionIncludeSortVector sets the includeSortVector option for vectorize searches
func WithCollectionIncludeSortVector(include bool) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.IncludeSortVector = &include
	}
}

// WithCollectionPageState sets the initial page state for pagination
func WithCollectionPageState(pageState string) CollectionFindOption {
	return func(opts *CollectionFindOptions) {
		opts.InitialPageState = &pageState
	}
}

// NewCollectionFindOptions creates a CollectionFindOptions with the provided options applied
func NewCollectionFindOptions(opts ...CollectionFindOption) *CollectionFindOptions {
	options := &CollectionFindOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}
