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
	"net/http"
	"time"

	"github.com/datastax/astra-db-go/results"
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
	Timeout *TimeoutOptions

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

// APIOption is a function that modifies APIOptions.
// Use the With* functions to create APIOption values.
type APIOption func(*APIOptions)

// DefaultAPIOptions returns the default options used as the base for merging.
func DefaultAPIOptions() *APIOptions {
	apiVersion := "v1"
	keyspace := "default_keyspace"
	httpClient := &http.Client{}
	requestTimeout := 30 * time.Second

	return &APIOptions{
		APIVersion: &apiVersion,
		Keyspace:   &keyspace,
		HTTPClient: httpClient,
		Headers:    make(map[string]string),
		Timeout: &TimeoutOptions{
			Request: &requestTimeout,
		},
	}
}

// NewAPIOptions creates an APIOptions with the given options applied.
func NewAPIOptions(opts ...APIOption) *APIOptions {
	o := &APIOptions{
		Headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// MergeAPILayers combines multiple APIOptions layers, with later options overriding earlier ones.
// The merge order should be: Defaults -> Client -> Database -> Collection/Table -> Command
// Returns a new APIOptions with all non-nil values from the layers applied.
func MergeAPILayers(layers ...*APIOptions) *APIOptions {
	result := DefaultAPIOptions()

	for _, layer := range layers {
		if layer == nil {
			continue
		}

		if layer.Token != nil {
			result.Token = layer.Token
		}
		if layer.Keyspace != nil {
			result.Keyspace = layer.Keyspace
		}
		if layer.APIVersion != nil {
			result.APIVersion = layer.APIVersion
		}
		if layer.HTTPClient != nil {
			result.HTTPClient = layer.HTTPClient
		}

		// Merge headers (layer headers override/add to existing)
		if layer.Headers != nil {
			if result.Headers == nil {
				result.Headers = make(map[string]string)
			}
			for k, v := range layer.Headers {
				result.Headers[k] = v
			}
		}

		// Merge timeout options
		if layer.Timeout != nil {
			if result.Timeout == nil {
				result.Timeout = &TimeoutOptions{}
			}
			if layer.Timeout.Request != nil {
				result.Timeout.Request = layer.Timeout.Request
			}
			if layer.Timeout.Connection != nil {
				result.Timeout.Connection = layer.Timeout.Connection
			}
			if layer.Timeout.BulkOperation != nil {
				result.Timeout.BulkOperation = layer.Timeout.BulkOperation
			}
			if layer.Timeout.GeneralMethod != nil {
				result.Timeout.GeneralMethod = layer.Timeout.GeneralMethod
			}
		}

		// Merge serdes options
		if layer.Serdes != nil {
			result.Serdes = layer.Serdes
		}

		// Merge warning handler (later layers override)
		if layer.WarningHandler != nil {
			result.WarningHandler = layer.WarningHandler
		}

		// Merge environment
		if layer.AstraEnvironment != nil {
			result.AstraEnvironment = layer.AstraEnvironment
		}
		if layer.DataAPIBackend != nil {
			result.DataAPIBackend = layer.DataAPIBackend
		}
	}

	return result
}

// WithToken sets the authentication token.
func WithToken(token string) APIOption {
	return func(o *APIOptions) {
		o.Token = &token
	}
}

// WithKeyspace sets the keyspace.
func WithKeyspace(keyspace string) APIOption {
	return func(o *APIOptions) {
		o.Keyspace = &keyspace
	}
}

// WithAPIVersion sets the API version.
func WithAPIVersion(version string) APIOption {
	return func(o *APIOptions) {
		o.APIVersion = &version
	}
}

// WithHTTPClient sets the HTTP client.
func WithHTTPClient(client *http.Client) APIOption {
	return func(o *APIOptions) {
		o.HTTPClient = client
	}
}

// WithHeader adds a single header.
func WithHeader(key, value string) APIOption {
	return func(o *APIOptions) {
		if o.Headers == nil {
			o.Headers = make(map[string]string)
		}
		o.Headers[key] = value
	}
}

// WithHeaders sets multiple headers at once.
func WithHeaders(headers map[string]string) APIOption {
	return func(o *APIOptions) {
		if o.Headers == nil {
			o.Headers = make(map[string]string)
		}
		for k, v := range headers {
			o.Headers[k] = v
		}
	}
}

// WithRequestTimeout sets the per-request timeout.
func WithRequestTimeout(d time.Duration) APIOption {
	return func(o *APIOptions) {
		if o.Timeout == nil {
			o.Timeout = &TimeoutOptions{}
		}
		o.Timeout.Request = &d
	}
}

// WithConnectionTimeout sets the connection timeout.
func WithConnectionTimeout(d time.Duration) APIOption {
	return func(o *APIOptions) {
		if o.Timeout == nil {
			o.Timeout = &TimeoutOptions{}
		}
		o.Timeout.Connection = &d
	}
}

// WithBulkOperationTimeout sets the bulk operation timeout (for insertMany, etc.).
func WithBulkOperationTimeout(d time.Duration) APIOption {
	return func(o *APIOptions) {
		if o.Timeout == nil {
			o.Timeout = &TimeoutOptions{}
		}
		o.Timeout.BulkOperation = &d
	}
}

// WithTimeout is a convenience function that sets the request timeout.
// This is the most commonly used timeout setting.
func WithTimeout(d time.Duration) APIOption {
	return WithRequestTimeout(d)
}

// WithGeneralMethodTimeout sets the overall timeout for paginated operations
// like deleteMany and updateMany. When set, the entire multi-page operation
// must complete within this duration.
func WithGeneralMethodTimeout(d time.Duration) APIOption {
	return func(o *APIOptions) {
		if o.Timeout == nil {
			o.Timeout = &TimeoutOptions{}
		}
		o.Timeout.GeneralMethod = &d
	}
}

// WithAstraEnvironment sets the Astra environment (prod, dev, test).
func WithAstraEnvironment(env AstraEnvironment) APIOption {
	return func(o *APIOptions) {
		o.AstraEnvironment = &env
	}
}

// WithDataAPIBackend sets the database backend (astra, hcd, dse, cassandra, other).
func WithDataAPIBackend(backend DataAPIBackend) APIOption {
	return func(o *APIOptions) {
		o.DataAPIBackend = &backend
	}
}

// WithWarningHandler sets a callback to be invoked for each API warning.
// The handler is called synchronously before the method returns.
//
// Example usage:
//
//	client := astradb.NewClient(
//		options.WithToken("..."),
//		options.WithWarningHandler(func(w results.Warning) {
//			slog.Warn("API warning", "code", w.ErrorCode, "message", w.Message)
//		}),
//	)
//
// warnings can indicate missing indexes, deprecated features, or other
// non-fatal conditions that don't prevent the operation from completing.
func WithWarningHandler(handler WarningHandler) APIOption {
	return func(o *APIOptions) {
		o.WarningHandler = handler
	}
}

// Helper functions for getting values with defaults

// GetToken returns the token or empty string if not set.
func (o *APIOptions) GetToken() string {
	if o == nil || o.Token == nil {
		return ""
	}
	return *o.Token
}

// GetKeyspace returns the keyspace or "default_keyspace" if not set.
func (o *APIOptions) GetKeyspace() string {
	if o == nil || o.Keyspace == nil {
		return "default_keyspace"
	}
	return *o.Keyspace
}

// GetAPIVersion returns the API version or "v1" if not set.
func (o *APIOptions) GetAPIVersion() string {
	if o == nil || o.APIVersion == nil {
		return "v1"
	}
	return *o.APIVersion
}

// GetHTTPClient returns the HTTP client or a default client if not set.
func (o *APIOptions) GetHTTPClient() *http.Client {
	if o == nil || o.HTTPClient == nil {
		return &http.Client{}
	}
	return o.HTTPClient
}

// GetAstraEnvironment returns the Astra environment or AstraEnvironmentProd if not set.
func (o *APIOptions) GetAstraEnvironment() AstraEnvironment {
	if o == nil || o.AstraEnvironment == nil {
		return AstraEnvironmentProd
	}
	return *o.AstraEnvironment
}

// GetDataAPIBackend returns the database backend or DataAPIBackendAstra if not set.
func (o *APIOptions) GetDataAPIBackend() DataAPIBackend {
	if o == nil || o.DataAPIBackend == nil {
		return DataAPIBackendAstra
	}
	return *o.DataAPIBackend
}

// GetRequestTimeout returns the request timeout or 30s if not set.
func (o *APIOptions) GetRequestTimeout() time.Duration {
	if o == nil || o.Timeout == nil || o.Timeout.Request == nil {
		return 30 * time.Second
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
