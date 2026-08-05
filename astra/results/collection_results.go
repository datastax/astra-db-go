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

package results

import "github.com/datastax/astra-db-go/v2/astra/internal/constants"

// CollectionDescriptor represents the descriptor for a collection, including its name and definition.
type CollectionDescriptor struct {
	// Name of the collection.
	Name string `json:"name"`

	// Definition of the collection.
	Definition CollectionDefinition `json:"definition"`
}

// CollectionDefinition represents the definition of a collection.
type CollectionDefinition struct {
	// Settings for generating ids
	DefaultId *CollectionDefaultIdDefinition `json:"defaultId,omitempty"`

	// Vector specifications for the collection
	Vector *VectorDefinition `json:"vector,omitempty"`

	// Overrides for document indexing
	Indexing *IndexingDefinition `json:"indexing,omitempty"`

	// Lexical analysis definition for the collection
	Lexical *LexicalDefinition `json:"lexical,omitempty"`

	// Reranking definition for the collection
	Rerank *RerankDefinition `json:"rerank,omitempty"`
}

// CollectionIdType specifies the type of auto-generated document ID when no _id is
// provided in an inserted document. If not set, the default is a string UUID v4.
type CollectionIdType string

const (
	// CollectionIdTypeUUID uses a [UUID v4] as the default document ID.
	//
	// [UUID v4]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-4
	CollectionIdTypeUUID CollectionIdType = CollectionIdType(constants.CollectionIdTypeUUID)
	// CollectionIdTypeUUIDv6 uses a UUID v6 as the default document ID.
	// UUID v6 is field-compatible with UUID v1 and supports lexicographic sorting.
	//
	// [UUID v6]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-6
	CollectionIdTypeUUIDv6 CollectionIdType = CollectionIdType(constants.CollectionIdTypeUUIDv6)
	// CollectionIdTypeUUIDv7 uses a [UUID v7] as the default document ID.
	// UUID v7 is recommended for new systems as a replacement for UUID v1.
	//
	// [UUID v7]: https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-14.html#name-uuid-version-7
	CollectionIdTypeUUIDv7 CollectionIdType = CollectionIdType(constants.CollectionIdTypeUUIDv7)
	// CollectionIdTypeObjectId uses an ObjectID as the default document ID.
	CollectionIdTypeObjectId CollectionIdType = CollectionIdType(constants.CollectionIdTypeObjectId)
)

// CollectionDefaultIdDefinition represents the definition for a collection's default ID.
//
// If `type` is not specified, the default ID will be a string UUID.
type CollectionDefaultIdDefinition struct {
	// Type is the type of the default ID that the API should generate if no ID is provided in the inserted document.
	// Valid values: "uuid", "uuidv6", "uuidv7", "objectId".
	// If not specified, the default ID will be a string UUID.
	Type *CollectionIdType `json:"type,omitempty"`
}

// -- Placeholder structs for the types referenced above --
// TODO: flesh these out. I shallow-ported some C# code to Go to get things working but need
// to come back and finish these definition.

// VectorDefinition configures vector search for a collection.
type VectorDefinition struct {
	// Dimension specifies the dimension of vectors stored in this collection.
	// Required for vector-enabled collections.
	Dimension *int `json:"dimension,omitempty"`

	// Metric specifies the similarity metric used for vector search.
	// Valid values are "cosine", "euclidean", or "dot_product".
	// Default is "cosine".
	Metric *string `json:"metric,omitempty"`

	// Service configures automatic vector embedding generation (vectorize).
	Service *VectorServiceDefinition `json:"service,omitempty"`
}

// VectorServiceDefinition configures the embedding service for vectorize.
type VectorServiceDefinition struct {
	// Provider is the embedding provider name (e.g., "openai", "huggingface").
	Provider *string `json:"provider,omitempty"`

	// ModelName is the name of the embedding model to use.
	ModelName *string `json:"modelName,omitempty"`
}

// IndexingDefinition holds definition for collection indexing. Only one of `Allow` or `Deny` can be specified.
type IndexingDefinition struct {
	// Allow is a list of field paths to index, or ["*"] to index all fields.
	// Mutually exclusive with Deny.
	Allow []string `json:"allow,omitempty"`

	// Deny is a list of field paths to exclude from indexing, or ["*"] to disable indexing entirely.
	// Mutually exclusive with Allow.
	Deny []string `json:"deny,omitempty"`
}

type LexicalDefinition struct{}

type RerankDefinition struct{}
