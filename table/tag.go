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

package table

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// tagInfo holds parsed astra struct tag data.
type tagInfo struct {
	isPK         bool
	pkOrdinal    int // 1-based; 0 = unset
	isCK         bool
	ckOrdinal    int // 1-based
	ckDescending bool
	typeOverride string
	dimension    int
	isVector     bool
	hasVectorize bool
	provider     string
	model        string
	isJSONString bool
	skip         bool
}

// parseAstraTag parses the value of an "astra" struct tag.
//
// Grammar: <role>[,<ordinal>][,<modifier>...]
//
// Roles: pk, ck, -
// Modifiers: type=<T>, dim=<N>, vector, vectorize, provider=<P>, model=<M>, jsonString
func parseAstraTag(raw string) (tagInfo, error) {
	var info tagInfo
	if raw == "" {
		return info, nil
	}

	tokens := strings.Split(raw, ",")
	i := 0

	switch tokens[0] {
	case "-":
		return tagInfo{skip: true}, nil

	case "pk":
		info.isPK = true
		i = 1
		// Optional ordinal (next numeric token)
		if i < len(tokens) {
			if n, err := strconv.Atoi(tokens[i]); err == nil {
				info.pkOrdinal = n
				i++
			}
		}

	case "ck":
		info.isCK = true
		i = 1
		// Required: ordinal
		if i >= len(tokens) {
			return info, fmt.Errorf("ck requires ordinal and sort direction (e.g. ck,1,asc)")
		}
		n, err := strconv.Atoi(tokens[i])
		if err != nil {
			return info, fmt.Errorf("ck ordinal must be a number, got %q", tokens[i])
		}
		info.ckOrdinal = n
		i++
		// Required: sort direction
		if i >= len(tokens) {
			return info, fmt.Errorf("ck requires sort direction (asc or desc)")
		}
		switch tokens[i] {
		case "asc":
			info.ckDescending = false
		case "desc":
			info.ckDescending = true
		default:
			return info, fmt.Errorf("ck sort direction must be asc or desc, got %q", tokens[i])
		}
		i++

	default:
		// No role prefix — all tokens are modifiers
		i = 0
	}

	// Parse remaining tokens as modifiers
	for ; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case strings.HasPrefix(tok, "type="):
			info.typeOverride = tok[len("type="):]
		case strings.HasPrefix(tok, "dim="):
			d, err := strconv.Atoi(tok[len("dim="):])
			if err != nil {
				return info, fmt.Errorf("dim= value must be a number, got %q", tok[len("dim="):])
			}
			info.dimension = d
		case tok == "vector":
			info.isVector = true
		case tok == "vectorize":
			info.hasVectorize = true
		case strings.HasPrefix(tok, "provider="):
			info.provider = tok[len("provider="):]
		case strings.HasPrefix(tok, "model="):
			info.model = tok[len("model="):]
		case tok == "jsonString":
			info.isJSONString = true
		default:
			return info, fmt.Errorf("unknown astra tag token %q", tok)
		}
	}

	return info, nil
}

// columnName extracts the column name from a struct field's json tag,
// falling back to the field name if no json tag is present.
func columnName(f reflect.StructField) (name string, include bool) {
	jt := f.Tag.Get("json")
	if jt == "" {
		return f.Name, true
	}
	name, _, _ = strings.Cut(jt, ",")
	if name == "-" {
		return "", false
	}
	if name == "" {
		return f.Name, true
	}
	return name, true
}
