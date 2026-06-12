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

package astra

import (
	"fmt"

	"github.com/datastax/astra-db-go/v2/astra/options"
)

// DataAPIClient is a client for interacting with an Astra DB database.
// Construct a new client using [NewClient].
//
// Options set on the client are inherited by all databases, collections,
// tables, and commands created from it, unless overridden at a lower level.
type DataAPIClient struct {
	options        *options.APIOptions
	dataAPIBackend options.DataAPIBackend
}

// NewClient returns a new DataAPIClient with the given options.
//
// Example:
//
//	client := astra.NewClient(
//	    options.API().SetToken("AstraCS:..."),
//	)
//
// Example with non-Astra backend:
//
//	client := astra.NewClient(
//	    options.API().SetToken("AstraCS:..."),
//	    options.API().SetDataAPIBackend(options.DataAPIBackendHCD),
//	)
func NewClient(opts ...options.APIOption) *DataAPIClient {
	resolved := options.Merge(opts...)
	return &DataAPIClient{
		options:        resolved,
		dataAPIBackend: resolved.GetDataAPIBackend(),
	}
}

// ClientOptions returns the client's options as a resolved struct with defaults.
func (c *DataAPIClient) ClientOptions() *options.APIOptions {
	return c.options
}

// Database returns a handle for the given database endpoint.
//
// Options set here override those set on the client.
//
// Example:
//
//	db := client.Database("https://...",
//	    options.API().SetKeyspace("my_keyspace"),
//	)
func (c *DataAPIClient) Database(endpoint string, opts ...options.APIOption) *Db {
	return newDbFromEndpoint(endpoint, c, options.Merge(append([]options.APIOption{c.options}, opts...)...))
}

// Admin returns an AstraAdmin handle for DevOps API operations.
// Returns an error if the client's environment is not an Astra environment.
//
// Options set on the client are inherited by the AstraAdmin.
//
// Example:
//
//	admin, err := client.Admin()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	regions, err := admin.FindAvailableRegions(ctx)
func (c *DataAPIClient) Admin(opts ...options.APIOption) (*AstraAdmin, error) {
	if !c.dataAPIBackend.IsAstra() {
		return nil, fmt.Errorf("Admin is only available with the Astra backend (current: %s)", c.dataAPIBackend)
	}
	combined := options.Merge(append([]options.APIOption{c.options}, opts...)...)
	env := combined.GetAstraEnvironment()
	return &AstraAdmin{
		client:           c,
		options:          combined,
		apiVersion:       DefaultAdminAPIVersion,
		astraEnvironment: env,
	}, nil
}
