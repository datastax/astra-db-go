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

package options

import "github.com/datastax/astra-db-go/v2/astra/sort"

// GetTableOptions represents options for getting a table handle.
type GetTableOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-" optlift:"Keyspace,EmbeddingHeadersProvider,RerankingHeadersProvider"`
}

// SetEmbeddingAPIKey sets the API key to use for embedding generation for this table.
func (b *getTableOptionsBuilder) SetEmbeddingAPIKey(apiKey string) *getTableOptionsBuilder {
	return b.SetEmbeddingHeadersProvider(NewEmbeddingAPIKeyHeaderProvider(apiKey))
}

// SetRerankingAPIKey sets the API key to use for reranking generation for this table.
func (b *getTableOptionsBuilder) SetRerankingAPIKey(apiKey string) *getTableOptionsBuilder {
	return b.SetRerankingHeadersProvider(NewRerankingAPIKeyHeaderProvider(apiKey))
}

// SetEmbeddingHeadersProvider sets the headers provider to use for embedding generation for this table.
func (b *getTableOptionsBuilder) SetEmbeddingHeadersProvider(provider EmbeddingHeadersProvider) *getTableOptionsBuilder {
	b.setters = append(b.setters, func(o *GetTableOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.EmbeddingHeadersProvider = provider
	})
	return b
}

// SetRerankingHeadersProvider sets the headers provider to use for reranking generation for this table.
func (b *getTableOptionsBuilder) SetRerankingHeadersProvider(provider RerankingHeadersProvider) *getTableOptionsBuilder {
	b.setters = append(b.setters, func(o *GetTableOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.RerankingHeadersProvider = provider
	})
	return b
}

// CreateTableOptions represents options for creating a table
type CreateTableOptions struct {
	// IfNotExists if true, the command will silently succeed even if a table
	// with the given name already exists. This only checks table names, not schemas.
	IfNotExists *bool `json:"ifNotExists,omitempty"`

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-" optlift:"Keyspace,EmbeddingHeadersProvider,RerankingHeadersProvider"`
}

// SetEmbeddingAPIKey sets the API key to use for embedding generation for this table.
func (b *createTableOptionsBuilder) SetEmbeddingAPIKey(apiKey string) *createTableOptionsBuilder {
	return b.SetEmbeddingHeadersProvider(NewEmbeddingAPIKeyHeaderProvider(apiKey))
}

// SetRerankingAPIKey sets the API key to use for reranking generation for this table.
func (b *createTableOptionsBuilder) SetRerankingAPIKey(apiKey string) *createTableOptionsBuilder {
	return b.SetRerankingHeadersProvider(NewRerankingAPIKeyHeaderProvider(apiKey))
}

// SetEmbeddingHeadersProvider sets the headers provider to use for embedding generation for this table.
func (b *createTableOptionsBuilder) SetEmbeddingHeadersProvider(provider EmbeddingHeadersProvider) *createTableOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateTableOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.EmbeddingHeadersProvider = provider
	})
	return b
}

// SetRerankingHeadersProvider sets the headers provider to use for reranking generation for this table.
func (b *createTableOptionsBuilder) SetRerankingHeadersProvider(provider RerankingHeadersProvider) *createTableOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateTableOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.RerankingHeadersProvider = provider
	})
	return b
}

// TableFindOneOptions represents options for a findOne operation.
type TableFindOneOptions struct {
	// Sort specifies the sort order to apply before selecting the document to update.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Projection controls which fields are included or excluded in the returned document.
	Projection map[string]any `json:"projection,omitempty"`
	// IncludeSimilarity if true, include the similarity score in the result via the
	// $similarity field.
	IncludeSimilarity *bool `json:"includeSimilarity,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// TableFindOptions represents options for finding rows in a table
type TableFindOptions struct {
	// Sort specifies how to sort the results. Can be used for:
	//  - Ascending/descending sort on columns: sort.Asc("rating").Desc("title")
	//  - Vector search with a vector: sort.Vector([]float32{0.1, 0.2, 0.3})
	//  - Vector search with vectorize: sort.Vectorize("search text")
	Sort sort.Sortable `json:"sort,omitempty"`

	// Projection controls which columns are included or excluded in the returned rows
	// Use true to include a column, false to exclude it
	Projection map[string]any `json:"projection,omitempty"`

	// Limit limits the total number of rows returned
	Limit *int `json:"limit,omitempty"`

	// Skip specifies the number of rows to bypass before returning rows.
	// Only valid with ascending/descending sort, not with vector search.
	Skip *int `json:"skip,omitempty"`

	// IncludeSimilarity if true, includes a $similarity property in the response
	// for vector searches. Only works with direct vector search, not vectorize.
	IncludeSimilarity *bool `json:"includeSimilarity,omitempty"`

	// IncludeSortVector if true, includes the sort vector in the response.
	// Useful for vector searches using $vectorize.
	IncludeSortVector *bool `json:"includeSortVector,omitempty"`

	// InitialPageState is used for pagination to fetch the next page of results
	InitialPageState *string `json:"pageState,omitempty"`

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// SetPageState sets the initial page state for pagination.
func (b *tableFindOptionsBuilder) SetPageState(pageState string) *tableFindOptionsBuilder {
	b.setters = append(b.setters, func(o *TableFindOptions) { o.InitialPageState = &pageState })
	return b
}

// TableInsertOneOptions represents options for inserting a single row in a table.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type TableInsertOneOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// TableInsertManyOptions represents options for inserting multiple rows in a table.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type TableInsertManyOptions struct {
	Ordered     *bool `json:"ordered,omitempty"`
	ChunkSize   *int  `json:"-"`
	Concurrency *int  `json:"-"`

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// TableUpdateOneOptions represents options for updating a single row in a table.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type TableUpdateOneOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// TableDeleteOneOptions represents options for deleting a single row in a table.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type TableDeleteOneOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// TableDeleteManyOptions represents options for deleting multiple rows in a table.
// Right now this is empty except for APIOptions; table deleteMany has no
// pagination currently, etc.
type TableDeleteManyOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// TableDefinitionOptions represents options for fetching a table's descriptor.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type TableDefinitionOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// AlterTableOptions represents options for altering a table.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type AlterTableOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}
