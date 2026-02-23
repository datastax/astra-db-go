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

// Validate implements Validator for CreateCollectionOptions.
func (o CreateCollectionOptions) Validate() error {
	return nil
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
func (b *CreateCollectionOptionsBuilder) SetDefaultId(v *CollectionDefaultIdOptions) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		o.DefaultId = v
	})
	return b
}

// SetVector sets the vector options for the collection.
func (b *CreateCollectionOptionsBuilder) SetVector(v *VectorOptions) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		o.Vector = v
	})
	return b
}

// SetIndexing sets the indexing options for the collection.
func (b *CreateCollectionOptionsBuilder) SetIndexing(v *IndexingOptions) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		o.Indexing = v
	})
	return b
}

// SetLexical sets the lexical analysis options for the collection.
func (b *CreateCollectionOptionsBuilder) SetLexical(v *LexicalOptions) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		o.Lexical = v
	})
	return b
}

// SetRerank sets the reranking options for the collection.
func (b *CreateCollectionOptionsBuilder) SetRerank(v *RerankOptions) *CreateCollectionOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateCollectionOptions) {
		o.Rerank = v
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
	Type DefaultIdType `json:"type,omitempty"`
}

// -- Placeholder structs for the types referenced above --
// TODO: flesh these out. I shallow-ported some C# code to Go to get things working but need
// to come back and finish these options.

// VectorOptions configures vector search for a collection.
type VectorOptions struct {
	// Dimension specifies the dimension of vectors stored in this collection.
	// Required for vector-enabled collections.
	Dimension int `json:"dimension,omitempty"`

	// Metric specifies the similarity metric used for vector search.
	// Valid values are "cosine", "euclidean", or "dot_product".
	// Default is "cosine".
	Metric string `json:"metric,omitempty"`

	// Service configures automatic vector embedding generation (vectorize).
	Service *VectorServiceOptions `json:"service,omitempty"`
}

// VectorServiceOptions configures the embedding service for vectorize.
type VectorServiceOptions struct {
	// Provider is the embedding provider name (e.g., "openai", "huggingface").
	Provider string `json:"provider,omitempty"`

	// ModelName is the name of the embedding model to use.
	ModelName string `json:"modelName,omitempty"`
}

type IndexingOptions struct {
}

type LexicalOptions struct {
}

type RerankOptions struct {
}
