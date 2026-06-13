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

import (
	"fmt"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/internal/constants"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/sort"
)

// GetCollectionOptions represents options for getting a collection handle.
type GetCollectionOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-" optlift:"Keyspace,EmbeddingHeadersProvider,RerankingHeadersProvider"`
}

// SetEmbeddingAPIKey sets the API key to use for embedding generation for this collection.
func (b *getCollectionOptionsBuilder) SetEmbeddingAPIKey(apiKey string) *getCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *GetCollectionOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.EmbeddingHeadersProvider = NewEmbeddingAPIKeyHeadersProvider(apiKey)
	})
	return b
}

// UpdateRerankingAPIKey sets the API key to use for reranking generation for this collection.
func (b *getCollectionOptionsBuilder) UpdateRerankingAPIKey(apiKey string) *getCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *GetCollectionOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.RerankingHeadersProvider = NewRerankingAPIKeyHeadersProvider(apiKey)
	})
	return b
}

// CreateCollectionOptions represents options for a collection's behavior.
type CreateCollectionOptions struct {
	// Settings for generating ids
	DefaultId *CollectionDefaultIdOptions `json:"defaultId,omitempty"`

	// Vector specifications for the collection
	Vector *VectorOptions `json:"vector,omitempty"`

	// Overrides for document indexing
	Indexing *IndexingOptions `json:"indexing,omitempty"`

	// Lexical analysis options for the collection
	Lexical *LexicalOptions `json:"lexical,omitempty"`

	// Reranking options for the collection
	Rerank *RerankOptions `json:"rerank,omitempty"`

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-" optlift:"Keyspace,EmbeddingHeadersProvider,RerankingHeadersProvider"`
}

// SetEmbeddingAPIKey sets the API key to use for embedding generation for this collection.
func (b *createCollectionOptionsBuilder) SetEmbeddingAPIKey(apiKey string) *createCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateCollectionOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.EmbeddingHeadersProvider = NewEmbeddingAPIKeyHeadersProvider(apiKey)
	})
	return b
}

// UpdateRerankingAPIKey sets the API key to use for reranking generation for this collection.
func (b *createCollectionOptionsBuilder) UpdateRerankingAPIKey(apiKey string) *createCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateCollectionOptions) {
		if o.APIOptions == nil {
			o.APIOptions = &APIOptions{}
		}
		o.APIOptions.RerankingHeadersProvider = NewRerankingAPIKeyHeadersProvider(apiKey)
	})
	return b
}

// UpdateIndexingAllow sets the list of field paths to index. Use "*" to index all fields.
// Mutually exclusive with UpdateIndexingDeny.
func (b *createCollectionOptionsBuilder) UpdateIndexingAllow(v ...string) *createCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateCollectionOptions) {
		if o.Indexing == nil {
			o.Indexing = &IndexingOptions{}
		}
		o.Indexing.Allow = v
	})
	return b
}

// UpdateIndexingDeny sets the list of field paths to exclude from indexing. Use "*" to
// disable indexing entirely. Mutually exclusive with UpdateIndexingAllow.
func (b *createCollectionOptionsBuilder) UpdateIndexingDeny(v ...string) *createCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateCollectionOptions) {
		if o.Indexing == nil {
			o.Indexing = &IndexingOptions{}
		}
		o.Indexing.Deny = v
	})
	return b
}

// DefaultIdType specifies the type of auto-generated document ID when no _id is
// provided in an inserted document. If not set, the default is a string UUID v4.
type DefaultIdType string

const (
	// DefaultIdTypeUUID uses a [UUID v4] as the default document ID.
	//
	// [UUID v4]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-4
	DefaultIdTypeUUID DefaultIdType = DefaultIdType(constants.DefaultIdTypeUUID)
	// DefaultIdTypeUUIDv6 uses a UUID v6 as the default document ID.
	// UUID v6 is field-compatible with UUID v1 and supports lexicographic sorting.
	//
	// [UUID v6]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-6
	DefaultIdTypeUUIDv6 DefaultIdType = DefaultIdType(constants.DefaultIdTypeUUIDv6)
	// DefaultIdTypeUUIDv7 uses a [UUID v7] as the default document ID.
	// UUID v7 is recommended for new systems as a replacement for UUID v1.
	//
	// [UUID v7]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-7
	DefaultIdTypeUUIDv7 DefaultIdType = DefaultIdType(constants.DefaultIdTypeUUIDv7)
	// DefaultIdTypeObjectId uses an ObjectID as the default document ID.
	DefaultIdTypeObjectId DefaultIdType = DefaultIdType(constants.DefaultIdTypeObjectId)
)

// CollectionDefaultIdOptions represents the options for a collection's default ID.
//
// If `type` is not specified, the default ID will be a string UUID.
type CollectionDefaultIdOptions struct {
	// Type is the type of the default ID that the API should generate if no ID is provided in the inserted document.
	// Valid values: "uuid", "uuidv6", "uuidv7", "objectId".
	// If not specified, the default ID will be a string UUID.
	Type *DefaultIdType `json:"type,omitempty"`
}

// -- Placeholder structs for the types referenced above --
// TODO: flesh these out. I shallow-ported some C# code to Go to get things working but need
// to come back and finish these options.

// VectorOptions configures vector search for a collection.
type VectorOptions struct {
	// Dimension specifies the dimension of vectors stored in this collection.
	// Required for vector-enabled collections.
	Dimension *int `json:"dimension,omitempty"`

	// Metric specifies the similarity metric used for vector search.
	// Valid values are "cosine", "euclidean", or "dot_product".
	// Default is "cosine".
	Metric *string `json:"metric,omitempty"`

	// Service configures automatic vector embedding generation (vectorize).
	Service *VectorServiceOptions `json:"service,omitempty"`
}

// VectorServiceOptions configures the embedding service for vectorize, to automatically
// transform text into a vector ready for semantic vector searching.
//
// You can find out more information about each provider/model in the DataStax docs, or
// through [DatabaseAdmin.FindEmbeddingProviders].
type VectorServiceOptions struct {
	// Provider is the name of the embedding provider which provides the model to use
	// (e.g., "openai", "nvidia").
	Provider *string `json:"provider,omitempty"`

	// ModelName is the name of the embedding model to use.
	// Use "endpoint-defined-model" for providers like huggingfaceDedicated where the
	// model is defined by the endpoint rather than selected by name.
	ModelName *string `json:"modelName,omitempty"`

	// Authentication holds any necessary collection-bound authentication credentials.
	//
	// Most commonly, set providerKey to the name of a key stored in the Astra KMS
	// (Astra portal integration) to use for SHARED_SECRET authentication:
	//
	//	Authentication: map[string]any{"providerKey": "*KEY_NAME*"}
	Authentication map[string]any `json:"authentication,omitempty"`

	// Parameters holds arbitrary parameters that may be required on a per-model or
	// per-provider basis. Not all providers require parameters.
	//
	// Example (openai, optional projectId):
	//
	//	Parameters: map[string]any{"projectId": "my-project"}
	Parameters map[string]any `json:"parameters,omitempty"`
}

// Validate implements Validator for VectorServiceOptions.
// Provider and ModelName must both be set or both be unset.
func (o *VectorServiceOptions) Validate() error {
	if (o.Provider != nil) != (o.ModelName != nil) {
		return fmt.Errorf("vectorize service: provider and modelName must both be set or both be unset")
	}
	return nil
}

// IndexingOptions holds options for collection indexing. Only one of `Allow` or `Deny` can be specified.
type IndexingOptions struct {
	// Allow is a list of field paths to index, or ["*"] to index all fields.
	// Mutually exclusive with Deny.
	Allow []string `json:"allow,omitempty"`

	// Deny is a list of field paths to exclude from indexing, or ["*"] to disable indexing entirely.
	// Mutually exclusive with Allow.
	Deny []string `json:"deny,omitempty"`
}

// Validate implements Validator for IndexingOptions.
func (o *IndexingOptions) Validate() error {
	// Allow/Deny must be mutually exclusive. If both are set, return an error.
	if o.Allow != nil && o.Deny != nil {
		return fmt.Errorf("allow and deny cannot both be set")
	}
	return nil
}

// LexicalOptions configures lexical search (BM25) for a collection.
type LexicalOptions struct {
	// Enabled specifies whether lexical search is enabled for the collection.
	Enabled *bool `json:"enabled,omitempty"`

	// Analyzer specifies the analyzer to use for lexical search.
	Analyzer any `json:"analyzer,omitempty"`
}

// RerankOptions configures reranking for a collection.
type RerankOptions struct {
	// Enabled specifies whether reranking is enabled for the collection.
	Enabled *bool `json:"enabled,omitempty"`

	// Service configures the reranking service.
	Service *RerankServiceOptions `json:"service,omitempty"`
}

// RerankServiceOptions configures the reranking service.
type RerankServiceOptions struct {
	// Provider is the name of the reranking provider (e.g., "nvidia").
	Provider *string `json:"provider,omitempty"`

	// ModelName is the name of the reranking model to use.
	ModelName *string `json:"modelName,omitempty"`

	// Authentication holds any necessary collection-bound authentication credentials.
	Authentication map[string]any `json:"authentication,omitempty"`

	// Parameters holds arbitrary parameters that may be required on a per-model or
	// per-provider basis.
	Parameters map[string]any `json:"parameters,omitempty"`
}

// UpdateAnalyzer sets the analyzer name for lexical search (e.g., "standard").
func (b *lexicalOptionsBuilder) UpdateAnalyzer(v string) *lexicalOptionsBuilder {
	b.setters = append(b.setters, func(o *LexicalOptions) {
		o.Analyzer = v
	})
	return b
}

// SetCustomAnalyzer sets a complex analyzer configuration for lexical search.
func (b *lexicalOptionsBuilder) SetCustomAnalyzer(v map[string]any) *lexicalOptionsBuilder {
	b.setters = append(b.setters, func(o *LexicalOptions) {
		o.Analyzer = v
	})
	return b
}

// CollectionInsertOneOptions represents options for inserting a single document.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type CollectionInsertOneOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

type CollectionInsertManyOptions struct {
	Ordered     *bool       `json:"ordered,omitempty"`
	ChunkSize   *int        `json:"-"`
	Concurrency *int        `json:"-"`
	APIOptions  *APIOptions `json:"-"`
}

func (o *CollectionInsertManyOptions) Validate() error {
	if ptr.FromWithDefault(o.Ordered, false) && ptr.FromWithDefault(o.Concurrency, 1) != 1 {
		return fmt.Errorf("concurrency must be unset or 1 when ordered is true")
	}
	if o.Concurrency != nil && *o.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}
	return nil
}

// CollectionFindOptions represents options for finding documents in a collection
type CollectionFindOptions struct {
	// Sort specifies how to sort the results. Can be used for:
	//  - Ascending/descending sort on fields: sort.Asc("rating").Desc("title")
	//  - Vector search with a vector: sort.Vector([]float32{0.1, 0.2, 0.3})
	//  - Vector search with vectorize: sort.Vectorize("search text")
	Sort sort.Sortable `json:"sort,omitempty"`

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

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionFindAndRerankOptions represents options for finding and reranking documents in a collection.
type CollectionFindAndRerankOptions struct {
	// Sort specifies how to sort the results. For findAndRerank, this is typically a hybrid sort.
	Sort sort.Sortable `json:"sort,omitempty"`

	// Projection controls which fields are included or excluded in the returned documents.
	Projection map[string]any `json:"projection,omitempty"`

	// Limit limits the total number of documents returned.
	Limit *int `json:"limit,omitempty"`

	// HybridLimits provides additional limits for hybrid search.
	// This can be a single number (for both vector and lexical) or a map[string]int.
	HybridLimits any `json:"hybridLimits,omitempty"`

	// IncludeScores if true, includes a $scores property in the response.
	IncludeScores *bool `json:"includeScores,omitempty"`

	// IncludeSortVector if true, includes the sort vector in the response.
	IncludeSortVector *bool `json:"includeSortVector,omitempty"`

	// RerankOn specifies the field to rerank on.
	RerankOn *string `json:"rerankOn,omitempty"`

	// RerankQuery provides the query to use for reranking.
	RerankQuery *string `json:"rerankQuery,omitempty"`

	// InitialPageState is used for pagination (if supported by the API in the future).
	InitialPageState *string `json:"pageState,omitempty"`

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command.
	APIOptions *APIOptions `json:"-"`
}

// SetHybridLimits sets the HybridLimits option.
// This can be a single number (for both vector and lexical) or a map[string]int.
func (b *collectionFindAndRerankOptionsBuilder) SetHybridLimits(v any) *collectionFindAndRerankOptionsBuilder {
	b.setters = append(b.setters, func(o *CollectionFindAndRerankOptions) {
		o.HybridLimits = v
	})
	return b
}

// Validate implements Validator for CollectionFindAndRerankOptions.
// If HybridLimits is set, it must be either an int or a map[string]int.
func (o *CollectionFindAndRerankOptions) Validate() error {
	if o.HybridLimits != nil {
		switch o.HybridLimits.(type) {
		case int, map[string]int:
		default:
			return fmt.Errorf("HybridLimits must be either an int or a map[string]int")
		}
	}
	return nil
}

// CollectionUpdateOneOptions represents options for an updateOne operation.
type CollectionUpdateOneOptions struct {
	// Sort specifies the sort order to apply before selecting the document to update.
	// This determines which document is updated when the filter matches multiple documents.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Upsert if true, inserts a new document if no document matches the filter.
	Upsert *bool `json:"upsert,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionUpdateManyOptions represents options for an updateMany operation.
type CollectionUpdateManyOptions struct {
	// Upsert if true, inserts a new document if no document matches the filter.
	Upsert *bool `json:"upsert,omitempty"`
	// Timeout is the overall timeout for the entire paginated operation.
	// Overrides the GeneralMethod timeout from the hierarchy. Client-side only.
	Timeout *time.Duration `json:"-"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionDeleteOneOptions represents options for a deleteOne operation.
type CollectionDeleteOneOptions struct {
	// Sort specifies the sort order to apply before selecting the document to delete.
	// This determines which document is deleted when the filter matches multiple documents.
	Sort sort.Sortable `json:"sort,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionDeleteManyOptions represents options for a deleteMany operation.
type CollectionDeleteManyOptions struct {
	// Timeout is the overall timeout for the entire paginated operation.
	// Overrides the GeneralMethod timeout from the hierarchy. Client-side only.
	Timeout *time.Duration `json:"-"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionFindOneOptions represents options for a findOne operation.
type CollectionFindOneOptions struct {
	// Sort specifies the sort order to apply before selecting the document to update.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Projection controls which fields are included or excluded in the returned document.
	Projection map[string]any `json:"projection,omitempty"`
	// IncludeSimilarity if true, include the similarity score in the result via the
	// $similarity field.
	IncludeSimilarity *bool `json:"includeSimilarity,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionReplaceOneOptions represents options for a replaceOne operation.
type CollectionReplaceOneOptions struct {
	// Sort specifies the sort order to apply before selecting the document to replace.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Upsert if true, inserts a new document if no document matches the filter.
	Upsert *bool `json:"upsert,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// ReturnDocument specifies whether to return the document before or after the update.
type ReturnDocument string

const (
	// ReturnDocumentBefore returns the document as it was before the update.
	ReturnDocumentBefore ReturnDocument = "before"
	// ReturnDocumentAfter returns the document as it is after the update.
	ReturnDocumentAfter ReturnDocument = "after"
)

// CollectionFindOneAndUpdateOptions represents options for a findOneAndUpdate operation.
type CollectionFindOneAndUpdateOptions struct {
	// Sort specifies the sort order to apply before selecting the document to update.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Projection controls which fields are included or excluded in the returned document.
	Projection map[string]any `json:"projection,omitempty"`
	// Upsert if true, inserts a new document if no document matches the filter.
	Upsert *bool `json:"upsert,omitempty"`
	// ReturnDocument specifies whether to return the document before or after the update.
	ReturnDocument *ReturnDocument `json:"returnDocument,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionFindOneAndReplaceOptions represents options for a findOneAndReplace operation.
type CollectionFindOneAndReplaceOptions struct {
	// Sort specifies the sort order to apply before selecting the document to replace.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Projection controls which fields are included or excluded in the returned document.
	Projection map[string]any `json:"projection,omitempty"`
	// Upsert if true, inserts a new document if no document matches the filter.
	Upsert *bool `json:"upsert,omitempty"`
	// ReturnDocument specifies whether to return the document before or after the replacement.
	ReturnDocument *ReturnDocument `json:"returnDocument,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionFindOneAndDeleteOptions represents options for a findOneAndDelete operation.
type CollectionFindOneAndDeleteOptions struct {
	// Sort specifies the sort order to apply before selecting the document to delete.
	Sort sort.Sortable `json:"sort,omitempty"`
	// Projection controls which fields are included or excluded in the returned document.
	Projection map[string]any `json:"projection,omitempty"`
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// Side note: REALLY don't like the following name. But not sure what can be done about it.

// CollectionOptionsOptions represents options for fetching a collection's descriptor.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type CollectionOptionsOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionCountDocumentsOptions represents options for the countDocuments command.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type CollectionCountDocumentsOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CollectionEstimatedDocumentCountOptions represents options for the estimatedDocumentCount command.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type CollectionEstimatedDocumentCountOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Collection→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}
