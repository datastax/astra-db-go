// Copyright DataStax, Inc.
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

package options_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/options"
)

func TestDefaultAPIOptions(t *testing.T) {
	opts := options.DefaultAPIOptions()

	if opts.GetAPIVersion() != "v1" {
		t.Errorf("expected default API version 'v1', got %q", opts.GetAPIVersion())
	}
	if opts.GetKeyspace() != "default_keyspace" {
		t.Errorf("expected default keyspace 'default_keyspace', got %q", opts.GetKeyspace())
	}
	if opts.GetHTTPClient() == nil {
		t.Error("expected default HTTP client to be non-nil")
	}
	if opts.GetRequestTimeout() != 30*time.Second {
		t.Errorf("expected default request timeout 30s, got %v", opts.GetRequestTimeout())
	}
}

func TestNewAPIOptions(t *testing.T) {
	token := "test-token"
	keyspace := "my_keyspace"

	opts := options.NewAPIOptions(
		options.WithToken(token),
		options.WithKeyspace(keyspace),
	)

	if opts.GetToken() != token {
		t.Errorf("expected token %q, got %q", token, opts.GetToken())
	}
	if *opts.Keyspace != keyspace {
		t.Errorf("expected keyspace %q, got %q", keyspace, *opts.Keyspace)
	}
}

func TestMerge_SingleLayer(t *testing.T) {
	token := "layer-token"
	layer := options.NewAPIOptions(options.WithToken(token))

	result := options.MergeAPILayers(layer)

	if result.GetToken() != token {
		t.Errorf("expected token %q, got %q", token, result.GetToken())
	}
	// Should have defaults for unset values
	if result.GetKeyspace() != "default_keyspace" {
		t.Errorf("expected default keyspace, got %q", result.GetKeyspace())
	}
}

func TestMerge_MultipleLayers(t *testing.T) {
	clientToken := "client-token"
	dbKeyspace := "db_keyspace"
	collectionTimeout := 60 * time.Second

	clientOpts := options.NewAPIOptions(options.WithToken(clientToken))
	dbOpts := options.NewAPIOptions(options.WithKeyspace(dbKeyspace))
	collOpts := options.NewAPIOptions(options.WithRequestTimeout(collectionTimeout))

	result := options.MergeAPILayers(clientOpts, dbOpts, collOpts)

	// Token from client layer
	if result.GetToken() != clientToken {
		t.Errorf("expected token %q, got %q", clientToken, result.GetToken())
	}
	// Keyspace from db layer
	if result.GetKeyspace() != dbKeyspace {
		t.Errorf("expected keyspace %q, got %q", dbKeyspace, result.GetKeyspace())
	}
	// Timeout from collection layer
	if result.GetRequestTimeout() != collectionTimeout {
		t.Errorf("expected timeout %v, got %v", collectionTimeout, result.GetRequestTimeout())
	}
}

func TestMerge_LaterLayerOverrides(t *testing.T) {
	clientKeyspace := "client_ks"
	dbKeyspace := "db_ks"

	clientOpts := options.NewAPIOptions(options.WithKeyspace(clientKeyspace))
	dbOpts := options.NewAPIOptions(options.WithKeyspace(dbKeyspace))

	result := options.MergeAPILayers(clientOpts, dbOpts)

	// DB keyspace should override client keyspace
	if result.GetKeyspace() != dbKeyspace {
		t.Errorf("expected keyspace %q from db layer to override, got %q", dbKeyspace, result.GetKeyspace())
	}
}

func TestMerge_NilLayers(t *testing.T) {
	token := "my-token"
	opts := options.NewAPIOptions(options.WithToken(token))

	// Should handle nil layers gracefully
	result := options.MergeAPILayers(nil, opts, nil)

	if result.GetToken() != token {
		t.Errorf("expected token %q, got %q", token, result.GetToken())
	}
}

func TestMerge_Headers(t *testing.T) {
	clientOpts := options.NewAPIOptions(
		options.WithHeader("X-Client-Header", "client-value"),
		options.WithHeader("X-Shared-Header", "client-shared"),
	)
	dbOpts := options.NewAPIOptions(
		options.WithHeader("X-DB-Header", "db-value"),
		options.WithHeader("X-Shared-Header", "db-shared"), // Override
	)

	result := options.MergeAPILayers(clientOpts, dbOpts)

	// Client header preserved
	if result.Headers["X-Client-Header"] != "client-value" {
		t.Errorf("expected client header to be preserved")
	}
	// DB header added
	if result.Headers["X-DB-Header"] != "db-value" {
		t.Errorf("expected db header to be added")
	}
	// Shared header overridden by db layer
	if result.Headers["X-Shared-Header"] != "db-shared" {
		t.Errorf("expected shared header to be overridden by db layer, got %q", result.Headers["X-Shared-Header"])
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 120 * time.Second}

	opts := options.NewAPIOptions(options.WithHTTPClient(customClient))

	if opts.HTTPClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 45 * time.Second

	opts := options.NewAPIOptions(options.WithTimeout(timeout))

	if opts.Timeout == nil || opts.Timeout.Request == nil {
		t.Fatal("expected timeout to be set")
	}
	if *opts.Timeout.Request != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, *opts.Timeout.Request)
	}
}

func TestWithAPIVersion(t *testing.T) {
	version := "vdoesntexist"

	opts := options.NewAPIOptions(options.WithAPIVersion(version))

	if opts.APIVersion == nil || *opts.APIVersion != version {
		t.Errorf("expected API version %q, got %v", version, opts.APIVersion)
	}
}

func TestGetters_NilSafety(t *testing.T) {
	var nilOpts *options.APIOptions

	// All getters should be safe to call on nil
	if nilOpts.GetToken() != "" {
		t.Error("expected empty token for nil options")
	}
	if nilOpts.GetKeyspace() != "default_keyspace" {
		t.Error("expected default keyspace for nil options")
	}
	if nilOpts.GetAPIVersion() != "v1" {
		t.Error("expected default API version for nil options")
	}
	if nilOpts.GetHTTPClient() == nil {
		t.Error("expected default HTTP client for nil options")
	}
	if nilOpts.GetRequestTimeout() != 30*time.Second {
		t.Error("expected default timeout for nil options")
	}
}

func TestMerge_FullHierarchy(t *testing.T) {
	// Simulate full hierarchy: Client -> Database -> Collection -> Command
	clientOpts := options.NewAPIOptions(
		options.WithToken("client-token"),
		options.WithKeyspace("client_keyspace"),
		options.WithHeader("X-Client", "true"),
	)

	dbOpts := options.NewAPIOptions(
		options.WithKeyspace("db_keyspace"), // Override
	)

	collOpts := options.NewAPIOptions(
		options.WithTimeout(60*time.Second),
		options.WithHeader("X-Collection", "true"),
	)

	cmdOpts := options.NewAPIOptions(
		options.WithTimeout(5 * time.Second), // Override for specific command
	)

	result := options.MergeAPILayers(clientOpts, dbOpts, collOpts, cmdOpts)

	// Token from client (unchanged)
	if result.GetToken() != "client-token" {
		t.Errorf("expected client token, got %q", result.GetToken())
	}

	// Keyspace from db (overridden)
	if result.GetKeyspace() != "db_keyspace" {
		t.Errorf("expected db keyspace, got %q", result.GetKeyspace())
	}

	// Timeout from command (overridden)
	if result.GetRequestTimeout() != 5*time.Second {
		t.Errorf("expected command timeout 5s, got %v", result.GetRequestTimeout())
	}

	// Both headers preserved
	if result.Headers["X-Client"] != "true" {
		t.Error("expected client header to be preserved")
	}
	if result.Headers["X-Collection"] != "true" {
		t.Error("expected collection header to be preserved")
	}
}

func TestTimeoutOptions(t *testing.T) {
	connTimeout := 10 * time.Second
	reqTimeout := 30 * time.Second
	bulkTimeout := 120 * time.Second

	opts := options.NewAPIOptions(
		options.WithConnectionTimeout(connTimeout),
		options.WithRequestTimeout(reqTimeout),
		options.WithBulkOperationTimeout(bulkTimeout),
	)

	if opts.Timeout == nil {
		t.Fatal("expected timeout options to be set")
	}
	if *opts.Timeout.Connection != connTimeout {
		t.Errorf("expected connection timeout %v, got %v", connTimeout, *opts.Timeout.Connection)
	}
	if *opts.Timeout.Request != reqTimeout {
		t.Errorf("expected request timeout %v, got %v", reqTimeout, *opts.Timeout.Request)
	}
	if *opts.Timeout.BulkOperation != bulkTimeout {
		t.Errorf("expected bulk operation timeout %v, got %v", bulkTimeout, *opts.Timeout.BulkOperation)
	}
}

func TestWithHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Header-1": "value1",
		"X-Header-2": "value2",
	}

	opts := options.NewAPIOptions(options.WithHeaders(headers))

	if len(opts.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(opts.Headers))
	}
	if opts.Headers["X-Header-1"] != "value1" {
		t.Error("expected header 1 to be set")
	}
	if opts.Headers["X-Header-2"] != "value2" {
		t.Error("expected header 2 to be set")
	}
}
