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

	"github.com/datastax/astra-db-go/sort"
)

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
}

// SetIndexingAllow sets the list of field paths to index. Use "*" to index all fields.
// Mutually exclusive with SetIndexingDeny.
func (b *createCollectionOptionsBuilder) SetIndexingAllow(v ...string) *createCollectionOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateCollectionOptions) {
		if o.Indexing == nil {
			o.Indexing = &IndexingOptions{}
		}
		o.Indexing.Allow = v
	})
	return b
}

// SetIndexingDeny sets the list of field paths to exclude from indexing. Use "*" to
// disable indexing entirely. Mutually exclusive with SetIndexingAllow.
func (b *createCollectionOptionsBuilder) SetIndexingDeny(v ...string) *createCollectionOptionsBuilder {
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
	DefaultIdTypeUUID DefaultIdType = "uuid"
	// DefaultIdTypeUUIDv6 uses a UUID v6 as the default document ID.
	// UUID v6 is field-compatible with UUID v1 and supports lexicographic sorting.
	//
	// [UUID v6]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-6
	DefaultIdTypeUUIDv6 DefaultIdType = "uuidv6"
	// DefaultIdTypeUUIDv7 uses a [UUID v7] as the default document ID.
	// UUID v7 is recommended for new systems as a replacement for UUID v1.
	//
	// [UUID v7]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-7
	DefaultIdTypeUUIDv7 DefaultIdType = "uuidv7"
	// DefaultIdTypeObjectId uses an ObjectID as the default document ID.
	DefaultIdTypeObjectId DefaultIdType = "objectId"
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

// VectorServiceOptions configures the embedding service for vectorize.
type VectorServiceOptions struct {
	// Provider is the embedding provider name (e.g., "openai", "huggingface").
	Provider *string `json:"provider,omitempty"`

	// ModelName is the name of the embedding model to use.
	ModelName *string `json:"modelName,omitempty"`
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

type LexicalOptions struct{}

type RerankOptions struct{}

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

// SetPageState sets the initial page state for pagination.
func (b *collectionFindOptionsBuilder) SetPageState(pageState string) *collectionFindOptionsBuilder {
	b.setters = append(b.setters, func(o *CollectionFindOptions) { o.InitialPageState = &pageState })
	return b
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
