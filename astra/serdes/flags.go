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

	// ExtendedErrorSnippet can be used to include more of the JSON snippet in error messages (default is 16 chars, ExtendedErrorSnippet allows up to 64 chars).
	//
	// Note that extending the error context has a higher chance of leaking sensitive data in error messages; be mindful of its usage.
	ExtendedErrorSnippet

	// UseJSONUnmarshal can be used to recognize the standard library's json.Unmarshaler interface.
	// Custom Astra unmarshalers still take precedence.
	UseJSONUnmarshal

	// CaseInsensitiveFieldMatching can be used to fallback to case-insensitive
	// struct field lookups if an exact match isn't found.
	CaseInsensitiveFieldMatching
)
