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

import "fmt"

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

// List implements Builder[CreateCollectionOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[CreateCollectionOptions].
func (o *CreateCollectionOptions) List() []func(*CreateCollectionOptions) {
	return NoopBuilder(o)
}

// validate calls Validate on v if it is non-nil.
// TODO: probably will end up moving this. It's undecided what approach we
// want to take for child struct validation still so leaving it here until that
// conversation is resolved.
func validate[T Validator](v *T) error {
	if v == nil {
		return nil
	}
	return (*v).Validate()
}

// Just a helper to return first error in variadic input to prevent a bunch of if err != nil.
func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (o CreateCollectionOptions) Validate() error {
	err := firstErr(
		validate(o.DefaultId),
		validate(o.Vector),
		validate(o.Indexing),
		validate(o.Lexical),
		validate(o.Rerank),
	)
	return err
}

// CreateCollectionOptionsBuilder is a builder for CreateCollectionOptions that implements
// Builder[CreateCollectionOptions] following the MongoDB Go driver pattern.
type CreateCollectionOptionsBuilder struct {
	Opts []func(*CreateCollectionOptions)
}

// CreateCollection creates a new CreateCollectionOptionsBuilder.
func CreateCollection() *CreateCollectionOptionsBuilder {
	return &CreateCollectionOptionsBuilder{}
}

// List implements Builder[CreateCollectionOptions].
func (b *CreateCollectionOptionsBuilder) List() []func(*CreateCollectionOptions) {
	return b.Opts
}

// SetDefaultId sets the default ID options for the collection.
func (b *CreateCollectionOptionsBuilder) SetDefaultId(v ...Builder[CollectionDefaultIdOptions]) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		merged, _ := MergeOptions(v...)
		o.DefaultId = merged
	})
	return b
}

// SetVector sets the vector options for the collection.
func (b *CreateCollectionOptionsBuilder) SetVector(v ...Builder[VectorOptions]) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		merged, _ := MergeOptions(v...)
		o.Vector = merged
	})
	return b
}

// SetIndexingAllow sets the list of field paths to index. Use "*" to index all fields.
// Mutually exclusive with SetIndexingDeny.
func (b *CreateCollectionOptionsBuilder) SetIndexingAllow(v ...string) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		if o.Indexing == nil {
			o.Indexing = &IndexingOptions{}
		}
		o.Indexing.Allow = v
	})
	return b
}

// SetIndexingDeny sets the list of field paths to exclude from indexing. Use "*" to
// disable indexing entirely. Mutually exclusive with SetIndexingAllow.
func (b *CreateCollectionOptionsBuilder) SetIndexingDeny(v ...string) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		if o.Indexing == nil {
			o.Indexing = &IndexingOptions{}
		}
		o.Indexing.Deny = v
	})
	return b
}

// SetIndexing sets the indexing options for the collection. Example:
//
//	opts := options.CreateCollection().SetIndexing(&options.IndexingOptions{
//		Allow: []string{"field1", "field2"},
//	})
func (b *CreateCollectionOptionsBuilder) SetIndexing(v ...Builder[IndexingOptions]) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		merged, _ := MergeOptions(v...)
		o.Indexing = merged
	})
	return b
}

// SetLexical sets the lexical analysis options for the collection.
func (b *CreateCollectionOptionsBuilder) SetLexical(v ...Builder[LexicalOptions]) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		merged, _ := MergeOptions(v...)
		o.Lexical = merged
	})
	return b
}

// SetRerank sets the reranking options for the collection.
func (b *CreateCollectionOptionsBuilder) SetRerank(v ...Builder[RerankOptions]) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		merged, _ := MergeOptions(v...)
		o.Rerank = merged
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

// List implements Builder[CollectionDefaultIdOptions].
func (o *CollectionDefaultIdOptions) List() []func(*CollectionDefaultIdOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for CollectionDefaultIdOptions.
func (o CollectionDefaultIdOptions) Validate() error { return nil }

// CollectionDefaultIdOptionsBuilder is a builder for CollectionDefaultIdOptions.
type CollectionDefaultIdOptionsBuilder struct {
	Opts []func(*CollectionDefaultIdOptions)
}

// CollectionDefaultId creates a new CollectionDefaultIdOptionsBuilder.
func CollectionDefaultId() *CollectionDefaultIdOptionsBuilder {
	return &CollectionDefaultIdOptionsBuilder{}
}

// List implements Builder[CollectionDefaultIdOptions].
func (b *CollectionDefaultIdOptionsBuilder) List() []func(*CollectionDefaultIdOptions) {
	return b.Opts
}

// SetType sets the default ID type.
func (b *CollectionDefaultIdOptionsBuilder) SetType(v DefaultIdType) *CollectionDefaultIdOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionDefaultIdOptions) { o.Type = &v })
	return b
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

// List implements Builder[VectorOptions].
func (o *VectorOptions) List() []func(*VectorOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for VectorOptions.
func (o VectorOptions) Validate() error { return nil }

// VectorOptionsBuilder is a builder for VectorOptions.
type VectorOptionsBuilder struct {
	Opts []func(*VectorOptions)
}

// Vector creates a new VectorOptionsBuilder.
func Vector() *VectorOptionsBuilder {
	return &VectorOptionsBuilder{}
}

// List implements Builder[VectorOptions].
func (b *VectorOptionsBuilder) List() []func(*VectorOptions) {
	return b.Opts
}

// SetDimension sets the vector dimension.
func (b *VectorOptionsBuilder) SetDimension(v int) *VectorOptionsBuilder {
	b.Opts = append(b.Opts, func(o *VectorOptions) { o.Dimension = &v })
	return b
}

// SetMetric sets the similarity metric.
func (b *VectorOptionsBuilder) SetMetric(v string) *VectorOptionsBuilder {
	b.Opts = append(b.Opts, func(o *VectorOptions) { o.Metric = &v })
	return b
}

// SetService sets the vectorize service options.
func (b *VectorOptionsBuilder) SetService(v ...Builder[VectorServiceOptions]) *VectorOptionsBuilder {
	b.Opts = append(b.Opts, func(o *VectorOptions) {
		merged, _ := MergeOptions(v...)
		o.Service = merged
	})
	return b
}

// VectorServiceOptions configures the embedding service for vectorize.
type VectorServiceOptions struct {
	// Provider is the embedding provider name (e.g., "openai", "huggingface").
	Provider *string `json:"provider,omitempty"`

	// ModelName is the name of the embedding model to use.
	ModelName *string `json:"modelName,omitempty"`
}

// List implements Builder[VectorServiceOptions].
func (o *VectorServiceOptions) List() []func(*VectorServiceOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for VectorServiceOptions.
func (o VectorServiceOptions) Validate() error { return nil }

// VectorServiceOptionsBuilder is a builder for VectorServiceOptions.
type VectorServiceOptionsBuilder struct {
	Opts []func(*VectorServiceOptions)
}

// VectorService creates a new VectorServiceOptionsBuilder.
func VectorService() *VectorServiceOptionsBuilder {
	return &VectorServiceOptionsBuilder{}
}

// List implements Builder[VectorServiceOptions].
func (b *VectorServiceOptionsBuilder) List() []func(*VectorServiceOptions) {
	return b.Opts
}

// SetProvider sets the embedding provider name.
func (b *VectorServiceOptionsBuilder) SetProvider(v string) *VectorServiceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *VectorServiceOptions) { o.Provider = &v })
	return b
}

// SetModelName sets the embedding model name.
func (b *VectorServiceOptionsBuilder) SetModelName(v string) *VectorServiceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *VectorServiceOptions) { o.ModelName = &v })
	return b
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

// List implements Builder[IndexingOptions].
func (o *IndexingOptions) List() []func(*IndexingOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for IndexingOptions.
func (o IndexingOptions) Validate() error {
	// Allow/Deny must be mutually exclusive. If both are set, return an error.
	if o.Allow != nil && o.Deny != nil {
		return fmt.Errorf("allow and deny cannot both be set")
	}
	return nil
}

// IndexingOptionsBuilder is a builder for IndexingOptions.
type IndexingOptionsBuilder struct {
	Opts []func(*IndexingOptions)
}

// Indexing creates a new IndexingOptionsBuilder.
func Indexing() *IndexingOptionsBuilder {
	return &IndexingOptionsBuilder{}
}

// List implements Builder[IndexingOptions].
func (b *IndexingOptionsBuilder) List() []func(*IndexingOptions) {
	return b.Opts
}

// SetAllow sets the list of field paths to index.
// Use "*" to index all fields. Mutually exclusive with SetDeny.
func (b *IndexingOptionsBuilder) SetAllow(fields ...string) *IndexingOptionsBuilder {
	b.Opts = append(b.Opts, func(o *IndexingOptions) { o.Allow = fields })
	return b
}

// SetDeny sets the list of field paths to exclude from indexing.
// Use "*" to disable indexing entirely. Mutually exclusive with SetAllow.
func (b *IndexingOptionsBuilder) SetDeny(fields ...string) *IndexingOptionsBuilder {
	b.Opts = append(b.Opts, func(o *IndexingOptions) { o.Deny = fields })
	return b
}

type LexicalOptions struct{}

// List implements Builder[LexicalOptions].
func (o *LexicalOptions) List() []func(*LexicalOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for LexicalOptions.
func (o LexicalOptions) Validate() error { return nil }

type RerankOptions struct{}

// List implements Builder[RerankOptions].
func (o *RerankOptions) List() []func(*RerankOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for RerankOptions.
func (o RerankOptions) Validate() error { return nil }

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

// List implements Builder[CollectionFindOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[CollectionFindOptions].
func (o *CollectionFindOptions) List() []func(*CollectionFindOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for CollectionFindOptions.
func (o CollectionFindOptions) Validate() error { return nil }

// CollectionFindOptionsBuilder is a builder for CollectionFindOptions.
type CollectionFindOptionsBuilder struct {
	Opts []func(*CollectionFindOptions)
}

// CollectionFind creates a new CollectionFindOptionsBuilder.
func CollectionFind() *CollectionFindOptionsBuilder {
	return &CollectionFindOptionsBuilder{}
}

// List implements Builder[CollectionFindOptions].
func (b *CollectionFindOptionsBuilder) List() []func(*CollectionFindOptions) {
	return b.Opts
}

// SetSort sets the sort option for the find operation.
func (b *CollectionFindOptionsBuilder) SetSort(sort map[string]any) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.Sort = sort })
	return b
}

// SetProjection sets the projection option for the find operation.
func (b *CollectionFindOptionsBuilder) SetProjection(projection map[string]any) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.Projection = projection })
	return b
}

// SetLimit sets the limit option for the find operation.
func (b *CollectionFindOptionsBuilder) SetLimit(limit int) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.Limit = &limit })
	return b
}

// SetSkip sets the skip option for the find operation.
func (b *CollectionFindOptionsBuilder) SetSkip(skip int) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.Skip = &skip })
	return b
}

// SetIncludeSimilarity sets the includeSimilarity option for vector search.
func (b *CollectionFindOptionsBuilder) SetIncludeSimilarity(include bool) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.IncludeSimilarity = &include })
	return b
}

// SetIncludeSortVector sets the includeSortVector option for vectorize searches.
func (b *CollectionFindOptionsBuilder) SetIncludeSortVector(include bool) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.IncludeSortVector = &include })
	return b
}

// SetPageState sets the initial page state for pagination.
func (b *CollectionFindOptionsBuilder) SetPageState(pageState string) *CollectionFindOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionFindOptions) { o.InitialPageState = &pageState })
	return b
}

// CollectionUpdateOneOptions represents options for an updateOne operation.
type CollectionUpdateOneOptions struct {
	Sort   map[string]any
	Upsert *bool
}

// List implements Builder[CollectionUpdateOneOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[CollectionUpdateOneOptions].
func (o *CollectionUpdateOneOptions) List() []func(*CollectionUpdateOneOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for CollectionUpdateOneOptions.
func (o CollectionUpdateOneOptions) Validate() error {
	return nil
}

// CollectionUpdateOneOptionsBuilder is a builder for CollectionUpdateOneOptions.
type CollectionUpdateOneOptionsBuilder struct {
	Opts []func(*CollectionUpdateOneOptions)
}

// CollectionUpdateOne creates a new CollectionUpdateOneOptionsBuilder.
func CollectionUpdateOne() *CollectionUpdateOneOptionsBuilder {
	return &CollectionUpdateOneOptionsBuilder{}
}

// List implements Builder[CollectionUpdateOneOptions].
func (b *CollectionUpdateOneOptionsBuilder) List() []func(*CollectionUpdateOneOptions) {
	return b.Opts
}

// SetSort sets the sort option for the updateOne operation.
func (b *CollectionUpdateOneOptionsBuilder) SetSort(sort map[string]any) *CollectionUpdateOneOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionUpdateOneOptions) { o.Sort = sort })
	return b
}

// SetUpsert sets the upsert option for the updateOne operation.
func (b *CollectionUpdateOneOptionsBuilder) SetUpsert(upsert bool) *CollectionUpdateOneOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CollectionUpdateOneOptions) { o.Upsert = &upsert })
	return b
}
