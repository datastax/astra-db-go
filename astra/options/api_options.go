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

	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

// APIOptions contains all configurable options that can be set at any level
// in the client hierarchy (Client -> Database -> Collection/Table -> Command).
// Options set at a lower level override those set at a higher level.
type APIOptions struct {
	// TokenProvider is the authentication token provider for Astra DB
	TokenProvider TokenProvider

	// Keyspace is the keyspace to use for operations
	Keyspace *string

	// APIVersion is the Data API version (e.g., "v1")
	APIVersion *string

	// HTTPClient is the HTTP client to use for requests
	HTTPClient *http.Client

	// Headers contains custom headers to include in requests
	// (e.g., for embedding API keys like "x-embedding-api-key")
	Headers Headers

	// Timeout contains timeout configuration
	Timeout *TimeoutOptions `optlift:"Request:RequestTimeout,GeneralMethod:GeneralMethodTimeout,CollectionAdmin:CollectionAdminTimeout,TableAdmin:TableAdminTimeout,DatabaseAdmin:DatabaseAdminTimeout,KeyspaceAdmin:KeyspaceAdminTimeout"`

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

	// Callers contains information about the application making the request
	Callers Callers

	// EmbeddingHeaderProvider provides headers for embedding services (e.g. $vectorize).
	EmbeddingHeadersProvider EmbeddingHeadersProvider

	// RerankingHeaderProvider provides headers for reranking services (e.g. $lexical).
	RerankingHeadersProvider RerankingHeadersProvider
}

// Caller represents information about the application making the request.
type Caller struct {
	Name    string
	Version string
}

// Headers is a custom map type that implements shouldMerge to accumulate headers additively.
type Headers map[string]string

// Merge implements shouldMerge for Headers.
func (h Headers) merge(other shouldMerge) shouldMerge {
	otherHeaders := other.(Headers)
	res := make(Headers, len(h)+len(otherHeaders))
	maps.Copy(res, h)
	maps.Copy(res, otherHeaders)
	return res
}

// Callers is a custom slice type that implements shouldMerge to accumulate callers additively.
type Callers []Caller

// Merge implements shouldMerge for Callers.
func (c Callers) merge(other shouldMerge) shouldMerge {
	otherCallers := other.(Callers)
	res := make(Callers, 0, len(c)+len(otherCallers))
	res = append(res, c...)
	res = append(res, otherCallers...)
	return res
}

// SerdesOptions contains options for serialization and deserialization behavior.
type SerdesOptions struct {
	// Serialization flags
	TrustRawMessage *bool
	SortMapKeys     *bool
	SerNoCache      *bool
	UseJSONMarshal  *bool

	// Deserialization flags
	SparseRows                   *bool
	UseNumber                    *bool
	DesNoCache                   *bool
	ExtendedErrorSnippet         *bool
	UseJSONUnmarshal             *bool
	CaseInsensitiveFieldMatching *bool
}

// Merge implements shouldMerge for SerdesOptions.
func (o *SerdesOptions) merge(other shouldMerge) shouldMerge {
	casted, _ := other.(*SerdesOptions)
	return Merge[SerdesOptions](o, casted)
}

// GetSerFlags returns the aggregated serialization flags.
func (o *SerdesOptions) GetSerFlags() serdes.SerFlags {
	var f serdes.SerFlags
	o.walkSerFlags(func(flag serdes.SerFlags, field **bool) {
		if ptr.From(*field) {
			f |= flag
		}
	})
	return f
}

// GetDesFlags returns the aggregated deserialization flags.
func (o *SerdesOptions) GetDesFlags() serdes.DesFlags {
	var f serdes.DesFlags
	o.walkDesFlags(func(flag serdes.DesFlags, field **bool) {
		if ptr.From(*field) {
			f |= flag
		}
	})
	return f
}

func (o *SerdesOptions) walkSerFlags(fn func(serdes.SerFlags, **bool)) {
	if o == nil {
		return
	}
	fn(serdes.TrustRawMessage, &o.TrustRawMessage)
	fn(serdes.SortMapKeys, &o.SortMapKeys)
	fn(serdes.SerNoCache, &o.SerNoCache)
	fn(serdes.UseJSONMarshal, &o.UseJSONMarshal)
}

func (o *SerdesOptions) walkDesFlags(fn func(serdes.DesFlags, **bool)) {
	if o == nil {
		return
	}
	fn(serdes.SparseRows, &o.SparseRows)
	fn(serdes.UseNumber, &o.UseNumber)
	fn(serdes.DesNoCache, &o.DesNoCache)
	fn(serdes.ExtendedErrorSnippet, &o.ExtendedErrorSnippet)
	fn(serdes.UseJSONUnmarshal, &o.UseJSONUnmarshal)
	fn(serdes.CaseInsensitiveFieldMatching, &o.CaseInsensitiveFieldMatching)
}

// TimeoutOptions contains timeout configuration for API operations.
type TimeoutOptions struct {
	// Request is the timeout for individual HTTP requests
	Request *time.Duration
	// GeneralMethod is the overall timeout for general method operations (DML operations).
	// When set, the entire operation must complete within this duration.
	GeneralMethod *time.Duration
	// CollectionAdmin is the timeout for collection admin operations
	CollectionAdmin *time.Duration
	// TableAdmin is the timeout for table admin operations
	TableAdmin *time.Duration
	// DatabaseAdmin is the timeout for database admin operations
	DatabaseAdmin *time.Duration
	// KeyspaceAdmin is the timeout for keyspace admin operations
	KeyspaceAdmin *time.Duration
}

// Merge implements shouldMerge for TimeoutOptions.
func (o *TimeoutOptions) merge(other shouldMerge) shouldMerge {
	casted, _ := other.(*TimeoutOptions)
	return Merge[TimeoutOptions](o, casted)
}

// GetRequest returns the request timeout or 15 seconds if not set.
func (o *TimeoutOptions) GetRequest() time.Duration {
	if o == nil || o.Request == nil {
		return 15 * time.Second
	}
	return *o.Request
}

// GetGeneralMethod returns the general method timeout or 30 seconds if not set.
func (o *TimeoutOptions) GetGeneralMethod() time.Duration {
	if o == nil || o.GeneralMethod == nil {
		return 30 * time.Second
	}
	return *o.GeneralMethod
}

// GetCollectionAdmin returns the collection admin timeout or 60 seconds if not set.
func (o *TimeoutOptions) GetCollectionAdmin() time.Duration {
	if o == nil || o.CollectionAdmin == nil {
		return 60 * time.Second
	}
	return *o.CollectionAdmin
}

// GetTableAdmin returns the table admin timeout or 30 seconds if not set.
func (o *TimeoutOptions) GetTableAdmin() time.Duration {
	if o == nil || o.TableAdmin == nil {
		return 30 * time.Second
	}
	return *o.TableAdmin
}

// GetDatabaseAdmin returns the database admin timeout or 10 minutes if not set.
func (o *TimeoutOptions) GetDatabaseAdmin() time.Duration {
	if o == nil || o.DatabaseAdmin == nil {
		return 10 * time.Minute
	}
	return *o.DatabaseAdmin
}

// GetKeyspaceAdmin returns the keyspace admin timeout or 30 seconds if not set.
func (o *TimeoutOptions) GetKeyspaceAdmin() time.Duration {
	if o == nil || o.KeyspaceAdmin == nil {
		return 30 * time.Second
	}
	return *o.KeyspaceAdmin
}

// EnableSerFlags sets the provided serialization flags to true.
func (b *serdesOptionsBuilder) EnableSerFlags(flags serdes.SerFlags) *serdesOptionsBuilder {
	b.setters = append(b.setters, func(o *SerdesOptions) {
		o.walkSerFlags(func(flag serdes.SerFlags, field **bool) {
			if flags&flag != 0 {
				*field = ptr.To(true)
			}
		})
	})
	return b
}

// DisableSerFlags sets the provided serialization flags to false.
func (b *serdesOptionsBuilder) DisableSerFlags(flags serdes.SerFlags) *serdesOptionsBuilder {
	b.setters = append(b.setters, func(o *SerdesOptions) {
		o.walkSerFlags(func(flag serdes.SerFlags, field **bool) {
			if flags&flag != 0 {
				*field = ptr.To(false)
			}
		})
	})
	return b
}

// EnableDesFlags sets the provided deserialization flags to true.
func (b *serdesOptionsBuilder) EnableDesFlags(flags serdes.DesFlags) *serdesOptionsBuilder {
	b.setters = append(b.setters, func(o *SerdesOptions) {
		o.walkDesFlags(func(flag serdes.DesFlags, field **bool) {
			if flags&flag != 0 {
				*field = ptr.To(true)
			}
		})
	})
	return b
}

// DisableDesFlags sets the provided deserialization flags to false.
func (b *serdesOptionsBuilder) DisableDesFlags(flags serdes.DesFlags) *serdesOptionsBuilder {
	b.setters = append(b.setters, func(o *SerdesOptions) {
		o.walkDesFlags(func(flag serdes.DesFlags, field **bool) {
			if flags&flag != 0 {
				*field = ptr.To(false)
			}
		})
	})
	return b
}

// WarningHandler is a callback function invoked for each warning in API responses.
// warnings indicate non-fatal conditions such as missing indexes or deprecated features.
type WarningHandler func(w results.Warning)

// APIOption is a Builder that modifies APIOptions.
type APIOption = Builder[APIOptions]

func (b *apiOptionsBuilder) AddHeader(key, value string) *apiOptionsBuilder {
	b.setters = append(b.setters, func(o *APIOptions) {
		newHeaders := make(Headers, len(o.Headers)+1)
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

// SetToken sets the authentication token for Astra DB.
func (b *apiOptionsBuilder) SetToken(token string) *apiOptionsBuilder {
	b.setters = append(b.setters, func(o *APIOptions) {
		o.TokenProvider = NewStaticTokenProvider(token)
	})
	return b
}

// SetTokenProvider sets the authentication token provider for Astra DB.
func (b *apiOptionsBuilder) SetTokenProvider(provider TokenProvider) *apiOptionsBuilder {
	b.setters = append(b.setters, func(o *APIOptions) {
		o.TokenProvider = provider
	})
	return b
}

// AddCaller adds caller information to the existing list.
func (b *apiOptionsBuilder) AddCaller(name, version string) *apiOptionsBuilder {
	b.setters = append(b.setters, func(o *APIOptions) {
		o.Callers = append(append([]Caller(nil), o.Callers...), Caller{Name: name, Version: version})
	})
	return b
}

// Helper functions for getting values safely.

// GetTokenProvider returns the token provider or nil if not set.
func (o *APIOptions) GetTokenProvider() TokenProvider {
	if o == nil {
		return nil
	}
	return o.TokenProvider
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

// GetHTTPClient returns the HTTP client or a default one if not set.
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

// GetRequestTimeout returns the request timeout or 30 seconds if not set.
func (o *APIOptions) GetRequestTimeout() time.Duration {
	return o.GetTimeout().GetRequest()
}

// GetGeneralMethodTimeout returns the general method timeout or 30 seconds if not set.
func (o *APIOptions) GetGeneralMethodTimeout() time.Duration {
	return o.GetTimeout().GetGeneralMethod()
}

// GetTimeout returns the timeout configuration.
func (o *APIOptions) GetTimeout() *TimeoutOptions {
	if o == nil {
		return nil
	}
	return o.Timeout
}

// GetSerFlags returns the serialization flags or 0 if not set.
func (o *APIOptions) GetSerFlags() serdes.SerFlags {
	if o == nil || o.Serdes == nil {
		return 0
	}
	return o.Serdes.GetSerFlags()
}

// GetDesFlags returns the deserialization flags or 0 if not set.
func (o *APIOptions) GetDesFlags() serdes.DesFlags {
	if o == nil || o.Serdes == nil {
		return 0
	}
	return o.Serdes.GetDesFlags()
}

// Merge implements shouldMerge for APIOptions.
func (o *APIOptions) merge(other shouldMerge) shouldMerge {
	casted, _ := other.(*APIOptions)
	return Merge[APIOptions](o, casted)
}
