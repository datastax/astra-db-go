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
	"testing"
)

func TestParseAstraTag(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		want     fieldInfo
		wantErr  string
		wantSkip bool
	}{
		{name: "empty", raw: "", want: fieldInfo{}},
		{name: "pk", raw: "pk", want: fieldInfo{modifier: pkFieldMod{}}},
		{name: "pk with ordinal", raw: "pk[0]", want: fieldInfo{modifier: pkFieldMod{}}},
		{name: "pk with ordinal 2", raw: "pk[2]", want: fieldInfo{modifier: pkFieldMod{ord: 2}}},
		{name: "pk with ordinal -1", raw: "pk[-1]", want: fieldInfo{modifier: pkFieldMod{ord: -1}}},
		{name: "ck asc", raw: "ck[1]", want: fieldInfo{modifier: ckFieldMod{ord: 1}}},
		{name: "ck asc explicit", raw: "ck[1,asc]", want: fieldInfo{modifier: ckFieldMod{ord: 1}}},
		{name: "ck desc", raw: "ck[2,desc]", want: fieldInfo{modifier: ckFieldMod{ord: 2, desc: true}}},
		{name: "dim only", raw: "dim=1536", want: fieldInfo{modifier: dimFieldMod{dim: 1536}}},
		{name: "type=ascii", raw: "type=ascii", want: fieldInfo{typeExpr: "ascii"}},
		{name: "type=set", raw: "type=set", want: fieldInfo{typeExpr: "set"}},
		{name: "type=random___#23m", raw: "type=random___#23m", want: fieldInfo{typeExpr: "random___#23m"}},
		{name: "type=ascii,pk", raw: "type=ascii,pk", want: fieldInfo{typeExpr: "ascii", modifier: pkFieldMod{}}},
		{name: "type=ascii,ck[1,desc]", raw: "type=ascii,ck[1,desc]", want: fieldInfo{typeExpr: "ascii", modifier: ckFieldMod{ord: 1, desc: true}}},
		{name: "type=ascii,dim=3", raw: "type=ascii,dim=3", want: fieldInfo{typeExpr: "ascii", modifier: dimFieldMod{dim: 3}}},
		
		{name: "pk missing bracket", raw: "pk[1", wantErr: `error parsing "pk[...]" modifier: expected brackets in after "pk" in "[1"`},
		{name: "pk bad ord", raw: "pk[abc]", wantErr: `error parsing "pk[...]" modifier: invalid ordinal "abc": strconv.Atoi: parsing "abc": invalid syntax`},
		{name: "ck bad dir", raw: "ck[1,up]", wantErr: `error parsing "ck[...]" modifier: invalid column order "up" (expected either 'asc' or 'desc')`},
		{name: "ck bad ord", raw: "ck[abc]", wantErr: `error parsing "ck[...]" modifier: invalid ordinal "abc": strconv.Atoi: parsing "abc": invalid syntax`},
		{name: "old pk syntax", raw: "pk,1", wantErr: `unknown modifier "1"`},
		{name: "old ck syntax", raw: "ck,1,asc", wantErr: `unknown modifier "1"`},
		{name: "ck desc no ordinal", raw: "ck[desc]", wantErr: `error parsing "ck[...]" modifier: invalid ordinal "desc": strconv.Atoi: parsing "desc": invalid syntax`},
		{name: "bad dim", raw: "dim=abc", wantErr: `error parsing "dim=N" modifier: invalid dim value: strconv.Atoi: parsing "abc": invalid syntax`},
		{name: "unknown token", raw: "foobar", wantErr: `unknown modifier "foobar"`},
		{name: "pk and ck", raw: "pk,ck", wantErr: `conflicting modifiers "pk[...]" and "ck[...]" present`},
		{name: "pk and ck with ord", raw: "pk[0],ck[0]", wantErr: `conflicting modifiers "pk[...]" and "ck[...]" present`},
		{name: "pk and dim", raw: "pk,dim=3", wantErr: `conflicting modifiers "pk[...]" and "dim=N" present`},
		{name: "skip and pk", raw: "-,pk", wantErr: `skip '-' cannot be combined with other modifiers`},

		{name: "skip", raw: "-", wantSkip: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err, skip := parseAstraTag(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.wantErr {
					t.Errorf("\n got err '%v'\nwant err '%v'", err, tt.wantErr)
				}
				return
			}
			if tt.wantSkip != skip {
				t.Errorf("expected skip=%t, got skip=%t", tt.wantSkip, skip)
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
