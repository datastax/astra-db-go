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
	"strings"
	"testing"
)

func TestParseTypeExpr(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want typeExpr
	}{
		{"scalar text", "text", typeExpr{name: "text"}},
		{"scalar ascii", "ascii", typeExpr{name: "ascii"}},
		{"scalar duration", "duration", typeExpr{name: "duration"}},
		{"scalar vector", "vector", typeExpr{name: "vector"}},

		{"bare set", "set", typeExpr{name: "set", elem: &typeExpr{name: "infer"}}},
		{"bare list", "list", typeExpr{name: "list", elem: &typeExpr{name: "infer"}}},
		{"bare map", "map", typeExpr{name: "map", key: &typeExpr{name: "infer"}, elem: &typeExpr{name: "infer"}}},

		{"set[ascii]", "set[ascii]", typeExpr{name: "set", elem: &typeExpr{name: "ascii"}}},
		{"list[blob]", "list[blob]", typeExpr{name: "list", elem: &typeExpr{name: "blob"}}},
		{"map[uuid]blob", "map[uuid]blob", typeExpr{name: "map", key: &typeExpr{name: "uuid"}, elem: &typeExpr{name: "blob"}}},

		{
			"map[uuid]set[text]",
			"map[uuid]set[text]",
			typeExpr{name: "map", key: &typeExpr{name: "uuid"}, elem: &typeExpr{name: "set", elem: &typeExpr{name: "text"}}},
		},
		{
			"set[udt[person]]",
			"set[udt[person]]",
			typeExpr{name: "set", elem: &typeExpr{name: "udt", udtName: "person"}},
		},
		{
			"map[text]udt[person]",
			"map[text]udt[person]",
			typeExpr{name: "map", key: &typeExpr{name: "text"}, elem: &typeExpr{name: "udt", udtName: "person"}},
		},

		{"map[infer]blob", "map[infer]blob", typeExpr{name: "map", key: &typeExpr{name: "infer"}, elem: &typeExpr{name: "blob"}}},
		{"map[text]infer", "map[text]infer", typeExpr{name: "map", key: &typeExpr{name: "text"}, elem: &typeExpr{name: "infer"}}},
		{"map[infer]infer", "map[infer]infer", typeExpr{name: "map", key: &typeExpr{name: "infer"}, elem: &typeExpr{name: "infer"}}},

		{"udt[person]", "udt[person]", typeExpr{name: "udt", udtName: "person"}},
		{"udt underscore digits", "udt[my_type_1]", typeExpr{name: "udt", udtName: "my_type_1"}},

		{"infer leaf", "infer", typeExpr{name: "infer"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTypeExpr(tt.raw)
			if err != nil {
				t.Fatalf("parseTypeExpr(%q) errored: %v", tt.raw, err)
			}
			if !typeExprEqual(got, tt.want) {
				t.Errorf("parseTypeExpr(%q):\n  got  = %s\n  want = %s", tt.raw, exprFormat(got), exprFormat(tt.want))
			}
		})
	}
}

func TestParseTypeExpr_Errors(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantSub string // substring expected in error message
	}{
		{"empty", "", "empty"},
		{"unterminated set", "set[", "expected identifier"},
		{"unterminated set after ident", "set[ascii", "expected"},
		{"unknown inner", "set[foo]", "unknown type"},
		{"map missing value", "map[text]", "map[K]V requires both"},
		{"map empty brackets", "map[]", "expected identifier"},
		{"udt without brackets", "udt", "udt requires a name"},
		{"udt empty brackets", "udt[]", "udt"},
		{"infer with brackets", "infer[text]", "infer is a leaf"},
		{"scalar with brackets", "text[ascii]", "cannot take bracket"},
		{"trailing garbage", "set[text]extra", "trailing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTypeExpr(tt.raw)
			if err == nil {
				t.Fatalf("parseTypeExpr(%q) = no error, want error containing %q", tt.raw, tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("parseTypeExpr(%q) error = %q, want containing %q", tt.raw, err.Error(), tt.wantSub)
			}
		})
	}
}

func typeExprEqual(a, b typeExpr) bool {
	if a.name != b.name || a.udtName != b.udtName {
		return false
	}
	if (a.elem == nil) != (b.elem == nil) || (a.key == nil) != (b.key == nil) {
		return false
	}
	if a.elem != nil && !typeExprEqual(*a.elem, *b.elem) {
		return false
	}
	if a.key != nil && !typeExprEqual(*a.key, *b.key) {
		return false
	}
	return true
}

// exprFormat renders a typeExpr in bracket notation. Used only by the test
// helpers here — production error messages quote the raw user input instead.
func exprFormat(e typeExpr) string {
	switch e.name {
	case "set", "list":
		if e.elem == nil {
			return e.name
		}
		return e.name + "[" + exprFormat(*e.elem) + "]"
	case "map":
		if e.key == nil || e.elem == nil {
			return "map"
		}
		return "map[" + exprFormat(*e.key) + "]" + exprFormat(*e.elem)
	case "udt":
		return "udt[" + e.udtName + "]"
	default:
		return e.name
	}
}
