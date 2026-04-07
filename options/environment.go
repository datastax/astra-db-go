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

import (
	"fmt"
	"net/url"
	"strings"
)

// AstraEnvironment represents which Astra environment to target (controls DevOps API URL).
type AstraEnvironment string

const (
	// AstraEnvironmentProd is the Astra production environment.
	AstraEnvironmentProd AstraEnvironment = "prod"
	// AstraEnvironmentDev is the Astra development environment.
	AstraEnvironmentDev AstraEnvironment = "dev"
	// AstraEnvironmentTest is the Astra test environment.
	AstraEnvironmentTest AstraEnvironment = "test"
)

// DevOpsURL returns the Astra DevOps API base URL for this environment.
func (e AstraEnvironment) DevOpsURL() string {
	switch e {
	case AstraEnvironmentDev:
		return "https://api.dev.cloud.datastax.com"
	case AstraEnvironmentTest:
		return "https://api.test.cloud.datastax.com"
	default:
		return "https://api.astra.datastax.com"
	}
}

// AstraDBEndpoint returns the Data API endpoint for an Astra database.
func (e AstraEnvironment) AstraDBEndpoint(id, region string) string {
	switch e {
	case AstraEnvironmentDev:
		return fmt.Sprintf("https://%s-%s.apps.astra-dev.datastax.com", id, region)
	case AstraEnvironmentTest:
		return fmt.Sprintf("https://%s-%s.apps.astra-test.datastax.com", id, region)
	default:
		return fmt.Sprintf("https://%s-%s.apps.astra.datastax.com", id, region)
	}
}

// ParseAstraEnvironmentFromEndpoint detects the AstraEnvironment from an endpoint URL's hostname.
// Returns AstraEnvironmentDev for *.apps.astra-dev.datastax.com,
// AstraEnvironmentTest for *.apps.astra-test.datastax.com,
// and AstraEnvironmentProd for everything else.
func ParseAstraEnvironmentFromEndpoint(endpoint string) AstraEnvironment {
	u, err := url.Parse(endpoint)
	if err != nil {
		return AstraEnvironmentProd
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case strings.HasSuffix(host, ".apps.astra-dev.datastax.com"):
		return AstraEnvironmentDev
	case strings.HasSuffix(host, ".apps.astra-test.datastax.com"):
		return AstraEnvironmentTest
	default:
		return AstraEnvironmentProd
	}
}

// DataAPIBackend represents the database backend (controls Data API path).
type DataAPIBackend string

const (
	// DataAPIBackendAstra is the Astra backend.
	DataAPIBackendAstra DataAPIBackend = "astra"
	// DataAPIBackendHCD is the Hyper-Converged Database backend.
	DataAPIBackendHCD DataAPIBackend = "hcd"
	// DataAPIBackendDSE is the DataStax Enterprise backend.
	DataAPIBackendDSE DataAPIBackend = "dse"
	// DataAPIBackendCassandra is the open-source Cassandra backend.
	DataAPIBackendCassandra DataAPIBackend = "cassandra"
	// DataAPIBackendOther is any other backend.
	DataAPIBackendOther DataAPIBackend = "other"
)

// DataAPIPath returns the Data API URL path prefix for this backend.
// Astra uses "api/json", while other backends have no prefix (the version
// is appended separately by the command layer).
//
// So, for astra, the URL will be something like:
// https://{database-id}-{region}.apps.astra.datastax.com/api/json/{version}/
//
// Non-astra DBs don't have /api/json. See also:
// https://github.com/datastax/astra-db-ts/blob/45f1c7fd9d46d82eee802947a285437c70d419bf/src/lib/api/constants.ts#L62
func (b DataAPIBackend) DataAPIPath() string {
	if b == DataAPIBackendAstra {
		return "api/json"
	}
	return ""
}

// IsAstra returns true if the backend is Astra.
func (b DataAPIBackend) IsAstra() bool {
	return b == DataAPIBackendAstra
}
