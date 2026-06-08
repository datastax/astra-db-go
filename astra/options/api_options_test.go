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

package options_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/options"
)

func TestDefaultAPIOptions(t *testing.T) {
	opts := options.Merge[options.APIOptions]()

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
	if opts.TokenProvider != nil {
		t.Error("expected default token provider to be nil")
	}
}

func TestNewAPIOptions(t *testing.T) {
	token := "test-token"
	keyspace := "my_keyspace"

	// Merge is for resolution.
	opts := options.Merge(
		options.API().SetToken(token),
		options.API().SetKeyspace(keyspace),
	)

	if gotToken, _ := opts.TokenProvider.Token(context.Background()); gotToken != token {
		t.Errorf("expected token %q, got %q", token, gotToken)
	}
	if *opts.Keyspace != keyspace {
		t.Errorf("expected keyspace %q, got %q", keyspace, *opts.Keyspace)
	}
}

func TestMerge_SingleLayer(t *testing.T) {
	token := "layer-token"
	// Use builder for layer
	layer := options.API().SetToken(token)

	result := options.Merge(layer)

	if gotToken, _ := result.TokenProvider.Token(context.Background()); gotToken != token {
		t.Errorf("expected token %q, got %q", token, gotToken)
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

	// Use builders for layers to avoid clobbering with intermediate defaults
	clientOpts := options.API().SetToken(clientToken)
	dbOpts := options.API().SetKeyspace(dbKeyspace)
	collOpts := options.API().SetRequestTimeout(collectionTimeout)

	result := options.Merge(clientOpts, dbOpts, collOpts)

	// Token from client layer
	if gotToken, _ := result.TokenProvider.Token(context.Background()); gotToken != clientToken {
		t.Errorf("expected token %q, got %q", clientToken, gotToken)
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

	clientOpts := options.API().SetKeyspace(clientKeyspace)
	dbOpts := options.API().SetKeyspace(dbKeyspace)

	result := options.Merge(clientOpts, dbOpts)

	// DB keyspace should override client keyspace
	if result.GetKeyspace() != dbKeyspace {
		t.Errorf("expected keyspace %q from db layer to override, got %q", dbKeyspace, result.GetKeyspace())
	}
}

func TestMerge_NilLayers(t *testing.T) {
	token := "my-token"
	opts := options.API().SetToken(token)

	// Should handle nil layers gracefully
	result := options.Merge(nil, opts, nil)

	if gotToken, _ := result.TokenProvider.Token(context.Background()); gotToken != token {
		t.Errorf("expected token %q, got %q", token, gotToken)
	}
}

func TestMerge_Headers(t *testing.T) {
	clientOpts := options.API().
		SetHeader("X-Client-Header", "client-value").
		SetHeader("X-Shared-Header", "client-shared")
	dbOpts := options.API().
		SetHeader("X-DB-Header", "db-value").
		SetHeader("X-Shared-Header", "db-shared") // Override

	result := options.Merge(clientOpts, dbOpts)

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

	opts := options.Merge(options.API().SetHTTPClient(customClient))

	if opts.HTTPClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 45 * time.Second

	opts := options.Merge(options.API().SetRequestTimeout(timeout))

	if opts.Timeout == nil || opts.Timeout.Request == nil {
		t.Fatal("expected timeout to be set")
	}
	if *opts.Timeout.Request != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, *opts.Timeout.Request)
	}
}

func TestWithAPIVersion(t *testing.T) {
	version := "vdoesntexist"

	opts := options.Merge(options.API().SetAPIVersion(version))

	if opts.APIVersion == nil || *opts.APIVersion != version {
		t.Errorf("expected API version %q, got %v", version, opts.APIVersion)
	}
}

func TestGetters_NilSafety(t *testing.T) {
	var nilOpts *options.APIOptions

	// All getters should be safe to call on nil and return default-values
	if nilOpts.GetTokenProvider() != nil {
		t.Error("expected nil token provider for nil options")
	}
	if nilOpts.GetKeyspace() != "default_keyspace" {
		t.Errorf("expected 'default_keyspace' for nil options, got %v", nilOpts.GetKeyspace())
	}
	if nilOpts.GetAPIVersion() != "v1" {
		t.Errorf("expected 'v1' API version for nil options, got %v", nilOpts.GetAPIVersion())
	}
	if nilOpts.GetHTTPClient() == nil {
		t.Error("expected non-nil HTTP client for nil options")
	}
	if nilOpts.GetRequestTimeout() != 30*time.Second {
		t.Errorf("expected 30s timeout for nil options, got %v", nilOpts.GetRequestTimeout())
	}
}

func TestMerge_FullHierarchy(t *testing.T) {
	// Simulate full hierarchy: Client -> Database -> Collection -> Command
	clientOpts := options.API().
		SetToken("client-token").
		SetKeyspace("client_keyspace").
		SetHeader("X-Client", "true")

	dbOpts := options.API().
		SetKeyspace("db_keyspace") // Override

	collOpts := options.API().
		SetRequestTimeout(60*time.Second).
		SetHeader("X-Collection", "true")

	cmdOpts := options.API().
		SetRequestTimeout(5 * time.Second) // Override for specific command

	result := options.Merge(clientOpts, dbOpts, collOpts, cmdOpts)

	// Token from client (unchanged)
	if gotToken, _ := result.GetTokenProvider().Token(context.Background()); gotToken != "client-token" {
		t.Errorf("expected client token, got %q", gotToken)
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

	opts := options.Merge(
		options.API().SetConnectionTimeout(connTimeout),
		options.API().SetRequestTimeout(reqTimeout),
		options.API().SetBulkOperationTimeout(bulkTimeout),
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

func TestGeneralMethodTimeout(t *testing.T) {
	generalTimeout := 5 * time.Minute

	opts := options.Merge(
		options.API().SetGeneralMethodTimeout(generalTimeout),
	)

	if opts.Timeout == nil {
		t.Fatal("expected timeout options to be set")
	}
	if opts.Timeout.GeneralMethod == nil {
		t.Fatal("expected GeneralMethod timeout to be set")
	}
	if *opts.Timeout.GeneralMethod != generalTimeout {
		t.Errorf("expected GeneralMethod timeout %v, got %v", generalTimeout, *opts.Timeout.GeneralMethod)
	}
}

func TestGeneralMethodTimeoutMerge(t *testing.T) {
	clientTimeout := 5 * time.Minute
	collTimeout := 2 * time.Minute

	clientOpts := options.API().SetGeneralMethodTimeout(clientTimeout)
	collOpts := options.API().SetGeneralMethodTimeout(collTimeout)

	result := options.Merge(clientOpts, collOpts)

	if result.Timeout == nil || result.Timeout.GeneralMethod == nil {
		t.Fatal("expected GeneralMethod timeout after merge")
	}
	if *result.Timeout.GeneralMethod != collTimeout {
		t.Errorf("expected collection layer to override: want %v, got %v", collTimeout, *result.Timeout.GeneralMethod)
	}
}

func TestGeneralMethodTimeoutMergePreservesNil(t *testing.T) {
	// When no layer sets GeneralMethod, it should remain nil
	clientOpts := options.API().SetToken("t")
	result := options.Merge(clientOpts)

	if result.GetGeneralMethodTimeout() != nil {
		t.Errorf("expected nil GeneralMethod timeout, got %v", result.GetGeneralMethodTimeout())
	}
}

func TestGetGeneralMethodTimeoutNilSafety(t *testing.T) {
	var nilOpts *options.APIOptions
	if nilOpts.GetGeneralMethodTimeout() != nil {
		t.Error("expected nil for nil options")
	}
}

func TestWithHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Header-1": "value1",
		"X-Header-2": "value2",
	}

	opts := options.Merge(options.API().SetHeaders(headers))

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
