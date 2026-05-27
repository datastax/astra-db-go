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

import "testing"

func TestParseAstraTag(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    tagInfo
		wantErr bool
	}{
		{name: "empty", raw: "", want: tagInfo{}},
		{name: "skip", raw: "-", want: tagInfo{skip: true}},
		{name: "pk", raw: "pk", want: tagInfo{isPK: true}},
		{name: "pk with ordinal", raw: "pk,1", want: tagInfo{isPK: true, pkOrdinal: 1}},
		{name: "pk with ordinal and vector", raw: "pk,2,vector,dim=4", want: tagInfo{isPK: true, pkOrdinal: 2, isVector: true, dimension: 4}},
		{name: "pk with vectorize", raw: "pk,3,vectorize,provider=nvidia,model=NV-Embed-QA", want: tagInfo{isPK: true, pkOrdinal: 3, hasVectorize: true, provider: "nvidia", model: "NV-Embed-QA"}},
		{name: "ck asc", raw: "ck,1,asc", want: tagInfo{isCK: true, ckOrdinal: 1}},
		{name: "ck desc", raw: "ck,2,desc", want: tagInfo{isCK: true, ckOrdinal: 2, ckDescending: true}},
		{name: "type override", raw: "type=ascii", want: tagInfo{typeOverride: "ascii"}},
		{name: "dim only", raw: "dim=1536", want: tagInfo{dimension: 1536}},
		{name: "type=set", raw: "type=set", want: tagInfo{typeOverride: "set"}},
		{name: "jsonString", raw: "jsonString", want: tagInfo{isJSONString: true}},
		{name: "ck missing sort", raw: "ck,1", wantErr: true},
		{name: "ck bad ordinal", raw: "ck,abc,asc", wantErr: true},
		{name: "ck bad direction", raw: "ck,1,up", wantErr: true},
		{name: "bad dim", raw: "dim=abc", wantErr: true},
		{name: "unknown token", raw: "foobar", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAstraTag(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
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
