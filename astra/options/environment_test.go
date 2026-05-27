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

func TestParseAstraEndpoint(t *testing.T) {
	const (
		uuid1   = "6b7e7c43-7a5d-4c7e-a9b4-8f8a1b2c3d4e"
		uuid2   = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
		uuid3   = "ffffffff-0000-1111-2222-333333333333"
		region1 = "us-east1"
		region2 = "eu-west1"
		region3 = "ap-southeast1"
	)

	tests := []struct {
		name       string
		endpoint   string
		wantID     string
		wantRegion string
		wantEnv    AstraEnvironment
	}{
		{
			name:       "prod endpoint",
			endpoint:   "https://" + uuid1 + "-" + region1 + ".apps.astra.datastax.com",
			wantID:     uuid1,
			wantRegion: region1,
			wantEnv:    AstraEnvironmentProd,
		},
		{
			name:       "dev endpoint",
			endpoint:   "https://" + uuid2 + "-" + region2 + ".apps.astra-dev.datastax.com",
			wantID:     uuid2,
			wantRegion: region2,
			wantEnv:    AstraEnvironmentDev,
		},
		{
			name:       "test endpoint",
			endpoint:   "https://" + uuid3 + "-" + region3 + ".apps.astra-test.datastax.com",
			wantID:     uuid3,
			wantRegion: region3,
			wantEnv:    AstraEnvironmentTest,
		},
		{
			name:       "unknown endpoint defaults to prod",
			endpoint:   "https://my-database.example.com",
			wantID:     "",
			wantRegion: "",
			wantEnv:    AstraEnvironmentProd,
		},
		{
			name:       "empty string defaults to prod",
			endpoint:   "",
			wantID:     "",
			wantRegion: "",
			wantEnv:    AstraEnvironmentProd,
		},
		{
			name:       "invalid URL defaults to prod",
			endpoint:   "://bad",
			wantID:     "",
			wantRegion: "",
			wantEnv:    AstraEnvironmentProd,
		},
		{
			name:       "prod endpoint with path",
			endpoint:   "https://" + uuid2 + "-" + region1 + ".apps.astra.datastax.com/api/json/v1",
			wantID:     uuid2,
			wantRegion: region1,
			wantEnv:    AstraEnvironmentProd,
		},
		{
			name:       "dev endpoint with port",
			endpoint:   "https://" + uuid3 + "-" + region2 + ".apps.astra-dev.datastax.com:443",
			wantID:     uuid3,
			wantRegion: region2,
			wantEnv:    AstraEnvironmentDev,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotRegion, gotEnv := ParseAstraEndpoint(tt.endpoint)
			if gotID != tt.wantID {
				t.Errorf("ParseAstraEndpoint(%q) id = %q, want %q", tt.endpoint, gotID, tt.wantID)
			}
			if gotRegion != tt.wantRegion {
				t.Errorf("ParseAstraEndpoint(%q) region = %q, want %q", tt.endpoint, gotRegion, tt.wantRegion)
			}
			if gotEnv != tt.wantEnv {
				t.Errorf("ParseAstraEndpoint(%q) env = %q, want %q", tt.endpoint, gotEnv, tt.wantEnv)
			}
		})
	}
}
