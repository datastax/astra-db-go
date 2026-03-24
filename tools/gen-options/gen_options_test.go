package main

import (
	"testing"
)

func TestUnexportedName(t *testing.T) {
	// Created some tests juuuust in case we have some non-ASCII builder names
	// in the future. It is extremely unlikely, but, these are valid go identifiers.
	tests := []struct {
		name string
		want string
	}{
		{"CreateIndexOptionsBuilder", "createIndexOptionsBuilder"},
		{"ΔeltaBuilder", "δeltaBuilder"},
		{"运行Builder", "运行Builder"},
		{"πBuilder", "πBuilder"}, // pi builder!
		{"URL", "url"},
		{"APIOptionsBuilder", "apiOptionsBuilder"},
		{"AπBuilder", "aπBuilder"}, // a pi builder!
		{"", ""},
		{"A", "a"},
		{"builder", "builder"},           // already unexported, should stay the same
		{"StatusIMUsed", "statusIMUsed"}, // Real-world example from stdlib.
		{"HTTP2Config", "http2Config"},   // Real-world example from stdlib.
		{"ǱBuilder", "ǳBuilder"},         // Titlecase digraph example. Just to be thorough.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unexportedName(tt.name); got != tt.want {
				t.Errorf("unexportedName(%q): GOT %q. WANT %q.", tt.name, got, tt.want)
			}
		})
	}
}

// This is the "golden" output for simple bool field.
const simpleBoolFieldExample = `// CollectionUpdateOneOption configures a CollectionUpdateOne operation.
// You can use the fluent-style builder or a pointer to [CollectionUpdateOneOptions] interchangeably.
// 
// Example using the fluent builder ([CollectionUpdateOne]):
//
//	// No need to use pointer for builder; the builder handles that for you.
//	opts := options.CollectionUpdateOne().SetUpsert(false)
//
// Example using a pointer to [CollectionUpdateOneOptions] without the fluent builder:
//
//	opts := &options.CollectionUpdateOneOptions{Upsert: ptr.To(false)}
type CollectionUpdateOneOption = Builder[CollectionUpdateOneOptions]`

const methodOnlyExample = `// CreateCollectionOption configures a CreateCollection operation.
// You can use the fluent-style builder or a pointer to [CreateCollectionOptions] interchangeably.
// 
// Example using the fluent builder ([CreateCollection]):
//
//	opts := options.CreateCollection().SetDefaultId(...)
//
// Example using a pointer to [CreateCollectionOptions] without the fluent builder:
//
//	opts := &options.CreateCollectionOptions{...}
type CreateCollectionOption = Builder[CreateCollectionOptions]`

func TestAliasExampleString(t *testing.T) {
	tests := []struct {
		name string
		ex   aliasDef
		want string
	}{
		{
			name: "simple bool field",
			ex: aliasDef{
				Alias:          "CollectionUpdateOneOption",
				Constructor:    "CollectionUpdateOne",
				OptsType:       "CollectionUpdateOneOptions",
				HasSimpleField: true,
				Method:         "SetUpsert",
				Field:          "Upsert",
				BuilderArg:     "false",
				StructVal:      "ptr.To(false)",
			},
			want: simpleBoolFieldExample,
		},
		{
			name: "fallback with method only",
			ex: aliasDef{
				Alias:       "CreateCollectionOption",
				Constructor: "CreateCollection",
				OptsType:    "CreateCollectionOptions",
				Method:      "SetDefaultId",
			},
			want: methodOnlyExample,
		},
		{
			name: "zero value - no setters",
			ex: aliasDef{
				Alias:       "EmptyOption",
				Constructor: "Empty",
				OptsType:    "EmptyOptions",
			},
			want: "// EmptyOption configures a Empty operation.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ex.String()
			if got != tt.want {
				t.Errorf("GOT:\n%s\nWANT:\n%s", got, tt.want)
			}
		})
	}
}

func TestPickAliasExample(t *testing.T) {
	tests := []struct {
		name    string
		setters []setterDef
		want    aliasDef
	}{
		{
			name: "picks first simple field",
			setters: []setterDef{
				{Method: "SetVector", Field: "Vector", IsVariadicBuilder: true, InnerType: "VectorOptions"},
				{Method: "SetBlocking", Field: "Blocking", ParamType: "bool"},
				{Method: "SetLimit", Field: "Limit", ParamType: "int"},
			},
			want: aliasDef{
				HasSimpleField: true,
				Method:         "SetBlocking",
				Field:          "Blocking",
				BuilderArg:     "false",
				StructVal:      "ptr.To(false)",
			},
		},
		{
			name: "fallback to first setter when no simple field",
			setters: []setterDef{
				{Method: "SetVector", Field: "Vector", IsVariadicBuilder: true, InnerType: "VectorOptions"},
			},
			want: aliasDef{Method: "SetVector"},
		},
		{
			name:    "empty setters",
			setters: nil,
			want:    aliasDef{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickAliasExample(tt.setters)
			if got != tt.want {
				t.Errorf("GOT: %+v. WANT: %+v", got, tt.want)
			}
		})
	}
}
