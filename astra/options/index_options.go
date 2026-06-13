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

// VectorMetric represents the similarity measurement for vector search.
type VectorMetric string

// Metric constants for vector index similarity measurement
const (
	MetricCosine     VectorMetric = "cosine"
	MetricDotProduct VectorMetric = "dot_product"
	MetricEuclidean  VectorMetric = "euclidean"
)

// CreateIndexOptions represents options for creating a regular index.
type CreateIndexOptions struct {
	// IfNotExists if true, the command will silently succeed even if an index
	// with the given name already exists. This only checks index names, not definitions.
	IfNotExists *bool

	// Ascii if true, converts non-ASCII characters to US-ASCII before indexing.
	// Only applicable to text columns.
	Ascii *bool

	// Normalize if true, applies Unicode character normalization before indexing.
	// Only applicable to text columns.
	Normalize *bool

	// CaseSensitive if true (default), enforces case-sensitive matching.
	// Only applicable to text columns.
	CaseSensitive *bool

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions
}

// CreateVectorIndexOptions represents options for creating a vector index.
type CreateVectorIndexOptions struct {
	// IfNotExists if true, the command will silently succeed even if an index
	// with the given name already exists. This only checks index names, not definitions.
	IfNotExists *bool

	// Metric is the similarity measurement for vector search.
	// Valid values: "cosine" (default), "dot_product", "euclidean"
	Metric *VectorMetric

	// SourceModel is the embedding generation model, enabling optimizations.
	// Valid values: "ada002", "bert", "cohere-v3", "gecko", "nv-qa-4",
	// "openai-v3-large", "openai-v3-small", "other" (default)
	//
	// NOTE: following the other libraries' patterns, we are using a enum-like option for Metric, but
	// this is a string. For reference:
	// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#parameters
	SourceModel *string

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions
}

// CreateTextIndexOptions represents options for creating a text index.
type CreateTextIndexOptions struct {
	// IfNotExists if true, the command will silently succeed even if an index
	// with the given name already exists. This only checks index names, not definitions.
	IfNotExists *bool

	// Analyzer is the name of the analyzer to use for the index, or a configuration map.
	Analyzer any

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions
}

// UpdateAnalyzer sets the built-in analyzer to use for the text index (e.g. "standard", "simple", "whitespace", etc.)
func (b *createTextIndexOptionsBuilder) UpdateAnalyzer(v string) *createTextIndexOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateTextIndexOptions) {
		o.Analyzer = v
	})
	return b
}

// SetCustomAnalyzer sets a custom analyzer configuration for the text index. The map should follow the structure defined by the API, e.g.:
//
//	{
//	  "tokenizer": {...},
//	  "filters": [...],
//	  "charFilters": [...],
//	}
func (b *createTextIndexOptionsBuilder) SetCustomAnalyzer(v map[string]any) *createTextIndexOptionsBuilder {
	b.setters = append(b.setters, func(o *CreateTextIndexOptions) {
		o.Analyzer = v
	})
	return b
}

// ListIndexesOptions represents options for listing indexes.
type ListIndexesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions
}
