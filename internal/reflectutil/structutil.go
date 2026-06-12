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

package reflectutil

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// FieldMeta contains the fully parsed metadata for a single mapped field.
type FieldMeta struct {
	Name      string
	OmitEmpty bool

	Field reflect.StructField
	Index []int // The sequence of struct field indices to reach this field

	tagged bool
}

// GetFields flattens a struct type, resolving embedded fields and name collisions.
// It follows encoding/json shadowing and ambiguity rules.
func GetFields(t reflect.Type) ([]FieldMeta, error) {
	seen := make(map[reflect.Type]bool)
	fields, err := getFieldsRecursive(t, seen, nil)
	if err != nil {
		return nil, err
	}

	// Sort fields lexicographically by their Index to preserve struct definition order
	sort.Slice(fields, func(i, j int) bool {
		a, b := fields[i].Index, fields[j].Index
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		for k := 0; k < minLen; k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})

	return fields, nil
}

func getFieldsRecursive(t reflect.Type, seen map[reflect.Type]bool, basePath []int) ([]FieldMeta, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	if seen[t] {
		return nil, nil // embedded cycle detected
	}
	seen[t] = true
	defer delete(seen, t)

	type embeddedField struct {
		subfield *FieldMeta
	}

	topLevelNames := make(map[string]struct{})
	ambiguousNames := make(map[string]int)
	ambiguousTags := make(map[string]int)

	fields := make([]FieldMeta, 0, t.NumField())
	var embeddedFields []embeddedField

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		embedded := f.Anonymous
		unexported := len(f.PkgPath) != 0

		name, ignored, omitempty, tagged := parseJSONTag(f)

		if ignored {
			continue
		}

		if unexported && !embedded {
			continue
		}

		if embedded && !tagged {
			typ := f.Type
			ptr := f.Type.Kind() == reflect.Ptr

			if ptr {
				typ = typ.Elem()
			}

			if typ.Kind() == reflect.Struct {
				idx := make([]int, len(basePath), len(basePath)+1)
				copy(idx, basePath)
				idx = append(idx, i)

				subtypeFields, err := getFieldsRecursive(typ, seen, idx)
				if err != nil {
					return nil, err
				}

				for j := range subtypeFields {
					embeddedFields = append(embeddedFields, embeddedField{
						subfield: &subtypeFields[j],
					})
				}
				continue
			}

			if unexported {
				continue
			}
		}

		idx := make([]int, len(basePath), len(basePath)+1)
		copy(idx, basePath)
		idx = append(idx, i)

		fields = append(fields, FieldMeta{
			Name:      name,
			OmitEmpty: omitempty,
			Field:     f,
			Index:     idx,
			tagged:    tagged,
		})

		topLevelNames[name] = struct{}{}
		ambiguousNames[name]++
		ambiguousTags[name]++
	}

	for _, embfield := range embeddedFields {
		ambiguousNames[embfield.subfield.Name]++
		if embfield.subfield.tagged {
			ambiguousTags[embfield.subfield.Name]++
		}
	}

	for _, embfield := range embeddedFields {
		subfield := *embfield.subfield

		switch resolveEmbeddedAmbiguity(subfield.Name, subfield.tagged, topLevelNames, ambiguousNames, ambiguousTags) {
		case shadowed:
			continue
		case ambiguous:
			return nil, fmt.Errorf("unresolvable ambiguity for field %q in struct %s", subfield.Name, t.String())
		case unambiguous:
			// all good
		}

		// prevents dominant flags more than one level below the embedded one
		subfield.tagged = false

		fields = append(fields, subfield)
	}

	return fields, nil
}

// parseJSONTag parses standard encoding/json style struct tags.
func parseJSONTag(f reflect.StructField) (name string, ignored, omitempty, tagged bool) {
	name = f.Name

	if parts := strings.Split(f.Tag.Get("json"), ","); len(parts) != 0 {
		if len(parts[0]) > 0 {
			name = parts[0]
			tagged = true
		}

		if name == "-" && len(parts) == 1 {
			ignored = true
			return
		}

		for _, opt := range parts[1:] {
			switch opt {
			case "omitempty":
				omitempty = true
			}
		}
	}

	return
}

type embeddedAmbiguity int

const (
	unambiguous embeddedAmbiguity = iota
	shadowed
	ambiguous
)

func resolveEmbeddedAmbiguity(name string, tagged bool, topLevelNames map[string]struct{}, nameCounts, tagCounts map[string]int) embeddedAmbiguity {
	if _, exists := topLevelNames[name]; exists {
		return shadowed // top level field with the same name exists so ignore this embedded field
	}

	if nameCounts[name] == 1 {
		return unambiguous // no collisions so all good to go
	}

	if tagCounts[name] == 1 && tagged {
		return unambiguous // multiple fields with the same name, so the field with the tag wins
	}

	if tagCounts[name] != 1 {
		return ambiguous // zero or multiple tags w/ the same name so we can't resolve anything
	}

	return shadowed // field collided and lost to a tagged field.
}
