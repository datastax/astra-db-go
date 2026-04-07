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

import "testing"

func TestParseAstraEnvironmentFromEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     AstraEnvironment
	}{
		{
			name:     "prod endpoint",
			endpoint: "https://abc-def.apps.astra.datastax.com",
			want:     AstraEnvironmentProd,
		},
		{
			name:     "dev endpoint",
			endpoint: "https://abc-def.apps.astra-dev.datastax.com",
			want:     AstraEnvironmentDev,
		},
		{
			name:     "test endpoint",
			endpoint: "https://abc-def.apps.astra-test.datastax.com",
			want:     AstraEnvironmentTest,
		},
		{
			name:     "unknown endpoint defaults to prod",
			endpoint: "https://my-database.example.com",
			want:     AstraEnvironmentProd,
		},
		{
			name:     "empty string defaults to prod",
			endpoint: "",
			want:     AstraEnvironmentProd,
		},
		{
			name:     "invalid URL defaults to prod",
			endpoint: "://bad",
			want:     AstraEnvironmentProd,
		},
		{
			name:     "prod endpoint with path",
			endpoint: "https://abc-def.apps.astra.datastax.com/api/json/v1",
			want:     AstraEnvironmentProd,
		},
		{
			name:     "dev endpoint with port",
			endpoint: "https://abc-def.apps.astra-dev.datastax.com:443",
			want:     AstraEnvironmentDev,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAstraEnvironmentFromEndpoint(tt.endpoint)
			if got != tt.want {
				t.Errorf("ParseAstraEnvironmentFromEndpoint(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}
