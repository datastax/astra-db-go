package results

import "github.com/datastax/astra-db-go/astra/serdes"

// IndexDescriptor describes an index on a table.
// When listing indexes with explain=true, all fields are populated.
// When explain=false, only Name is populated.
type IndexDescriptor struct {
	// Name is the index identifier.
	Name string `json:"name"`
	// Definition contains the column and options for the index.
	// Only populated when explain=true.
	Definition *IndexDefinition `json:"definition,omitempty"`
	// IndexType is either "regular" or "vector".
	// Only populated when explain=true.
	IndexType string `json:"indexType,omitempty"`
}

// UnmarshalAstraRaw implements custom unmarshaling for IndexDescriptor.
// The API returns either a string (name only) or an object (full metadata)
// depending on the explain option.
func (d *IndexDescriptor) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	// Try to unmarshal as a string first (names only response)
	var name string
	if err := serdes.Deserialize(value, &name, nil, ctx.Target); err == nil {
		d.Name = name
		return nil
	}

	// Otherwise unmarshal as an object (explain=true response)
	type indexDescriptorAlias IndexDescriptor
	var alias indexDescriptorAlias
	if err := serdes.Deserialize(value, &alias, nil, ctx.Target); err != nil {
		return err
	}
	*d = IndexDescriptor(alias)
	return nil
}

// IndexDefinition describes which column is indexed and its options.
type IndexDefinition struct {
	// Column is the name of the indexed column.
	Column string `json:"column"`
	// Options contains index-specific configuration.
	Options *IndexDefinitionOptions `json:"options,omitempty"`
}

// IndexDefinitionOptions contains configuration for an index.
type IndexDefinitionOptions struct {
	// Metric is the similarity metric for vector indexes (cosine, dot_product, euclidean).
	Metric string `json:"metric,omitempty"`
	// SourceModel is the embedding model identifier for vector indexes.
	SourceModel string `json:"sourceModel,omitempty"`
	// Ascii if true, converts non-ASCII characters to US-ASCII before indexing.
	Ascii *bool `json:"ascii,omitempty"`
	// Normalize if true, applies Unicode character normalization before indexing.
	Normalize *bool `json:"normalize,omitempty"`
	// CaseSensitive if true, enforces case-sensitive matching.
	CaseSensitive *bool `json:"caseSensitive,omitempty"`
}
