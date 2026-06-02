package serdes

type SerFlags int
type DesFlags int

const (
	// TrustRawMessage can be used as a slight optimization for when internal code has something it knows for sure is trusted.
	TrustRawMessage SerFlags = 1 << iota

	// SortMapKeys can be used to force the keys of a map to be sorted using datatypes.ComparatorFor when serializing
	SortMapKeys

	// SerNoCache can be used to disable the serdes cache for serialization.
	SerNoCache

	// UseJSONMarshal can be used to recognize the standard library's json.Marshaler interface.
	// Custom Astra marshalers still take precedence.
	UseJSONMarshal
)

const (
	// SparseRows can be used to disable populating missing columns in untyped rows.
	SparseRows DesFlags = 1 << iota

	// UseNumber can be used for untyped documents/rows in case they're expecting large numbers often.
	UseNumber

	// DesNoCache can be used to disable the serdes cache for deserialization.
	DesNoCache

	// ExtendedErrorContext can be used to include more of the JSON snippet in error messages (default is 16 chars, ExtendedErrorContext allows up to 64 chars).
	//
	// Note that extending the error context has a higher chance of leaking sensitive data in error messages; be mindful of its usage.
	ExtendedErrorContext

	// UseJSONUnmarshal can be used to recognize the standard library's json.Unmarshaler interface.
	// Custom Astra unmarshalers still take precedence.
	UseJSONUnmarshal
)
