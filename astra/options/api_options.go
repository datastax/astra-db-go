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
	"maps"
	"net/http"
	"time"

	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/astra/results"
)

// APIOptions contains all configurable options that can be set at any level
// in the client hierarchy (Client -> Database -> Collection/Table -> Command).
// Options set at a lower level override those set at a higher level.
type APIOptions struct {
	// Token is the authentication token for Astra DB
	Token *string

	// Keyspace is the keyspace to use for operations
	Keyspace *string

	// APIVersion is the Data API version (e.g., "v1")
	APIVersion *string

	// HTTPClient is the HTTP client to use for requests
	HTTPClient *http.Client

	// Headers contains custom headers to include in requests
	// (e.g., for embedding API keys like "x-embedding-api-key")
	Headers map[string]string

	// Timeout contains timeout configuration
	Timeout *TimeoutOptions `optlift:"Request:RequestTimeout,Connection:ConnectionTimeout,BulkOperation:BulkOperationTimeout,GeneralMethod:GeneralMethodTimeout"`

	// Serdes contains serialization/deserialization options
	Serdes *SerdesOptions

	// WarningHandler is called for each warning received from the API.
	// Set this at any level (Client, Database, Collection/Table, or Command).
	WarningHandler WarningHandler

	// AstraEnvironment is the Astra environment (prod, dev, test).
	// Controls the DevOps API URL. Defaults to prod.
	AstraEnvironment *AstraEnvironment

	// DataAPIBackend is the database backend (astra, hcd, dse, cassandra, other).
	// Controls the Data API path. Defaults to astra.
	DataAPIBackend *DataAPIBackend
}

func (o *APIOptions) SetDefaults() {
	o.APIVersion = ptr.To("v1")
	o.Keyspace = ptr.To("default_keyspace")
	o.HTTPClient = &http.Client{}
	o.Headers = make(map[string]string)
	o.Timeout = &TimeoutOptions{}
	o.Timeout.SetDefaults()
	o.AstraEnvironment = ptr.To(AstraEnvironmentProd)
	o.DataAPIBackend = ptr.To(DataAPIBackendAstra)
}

// TimeoutOptions contains timeout configuration for API operations.
type TimeoutOptions struct {
	// Request is the timeout for individual HTTP requests
	Request *time.Duration
	// Connection is the timeout for establishing connections
	Connection *time.Duration
	// BulkOperation is the timeout for bulk operations like insertMany
	BulkOperation *time.Duration
	// GeneralMethod is the overall timeout for paginated operations like deleteMany and updateMany.
	// When set, the entire multi-page operation must complete within this duration.
	GeneralMethod *time.Duration
}

func (o *TimeoutOptions) SetDefaults() {
	o.Request = ptr.To(30 * time.Second)
}

// SerdesOptions contains options for serialization and deserialization behavior.
// This is a placeholder for future extensibility.
type SerdesOptions struct {
	// Future options:
	// - Custom date/time handling
	// - Map encoding modes
	// - Custom type converters
}

// WarningHandler is a callback function invoked for each warning in API responses.
// warnings indicate non-fatal conditions such as missing indexes or deprecated features.
type WarningHandler func(w results.Warning)

// APIOption is a Builder that modifies APIOptions.
type APIOption = Builder[APIOptions]

func (b *apiOptionsBuilder) SetHeader(key, value string) *apiOptionsBuilder {
	b.setters = append(b.setters, func(o *APIOptions) {
		newHeaders := make(map[string]string, len(o.Headers)+1)
		maps.Copy(newHeaders, o.Headers)
		newHeaders[key] = value
		o.Headers = newHeaders
	})
	return b
}

func (b *apiOptionsBuilder) SetWarningHandler(handler WarningHandler) *apiOptionsBuilder {
	b.setters = append(b.setters, func(o *APIOptions) {
		o.WarningHandler = handler
	})
	return b
}

// Helper functions for getting values safely.

// GetToken returns the token or empty string if not set.
func (o *APIOptions) GetToken() string {
	if o == nil || o.Token == nil {
		return ""
	}
	return *o.Token
}

// GetKeyspace returns the keyspace or empty string if not set.
func (o *APIOptions) GetKeyspace() string {
	if o == nil || o.Keyspace == nil {
		return ""
	}
	return *o.Keyspace
}

// GetAPIVersion returns the API version or empty string if not set.
func (o *APIOptions) GetAPIVersion() string {
	if o == nil || o.APIVersion == nil {
		return ""
	}
	return *o.APIVersion
}

// GetHTTPClient returns the HTTP client or nil if not set.
func (o *APIOptions) GetHTTPClient() *http.Client {
	if o == nil {
		return nil
	}
	return o.HTTPClient
}

// GetAstraEnvironment returns the Astra environment or zero-value if not set.
func (o *APIOptions) GetAstraEnvironment() AstraEnvironment {
	if o == nil || o.AstraEnvironment == nil {
		return ""
	}
	return *o.AstraEnvironment
}

// GetDataAPIBackend returns the database backend or zero-value if not set.
func (o *APIOptions) GetDataAPIBackend() DataAPIBackend {
	if o == nil || o.DataAPIBackend == nil {
		return ""
	}
	return *o.DataAPIBackend
}

// GetRequestTimeout returns the request timeout or 0 if not set.
func (o *APIOptions) GetRequestTimeout() time.Duration {
	if o == nil || o.Timeout == nil || o.Timeout.Request == nil {
		return 0
	}
	return *o.Timeout.Request
}

// GetGeneralMethodTimeout returns the general method timeout or nil if not set.
func (o *APIOptions) GetGeneralMethodTimeout() *time.Duration {
	if o == nil || o.Timeout == nil {
		return nil
	}
	return o.Timeout.GeneralMethod
}
