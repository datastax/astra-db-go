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

package astradb

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/options"
)

const sampleRegionsResponse = `[
	{
		"classification": "general-purpose",
		"cloudProvider": "aws",
		"displayName": "US East (N. Virginia)",
		"enabled": true,
		"name": "us-east-1",
		"region_type": "vector",
		"reservedForQualifiedUsers": false,
		"zone": "na"
	},
	{
		"classification": "general-purpose",
		"cloudProvider": "gcp",
		"displayName": "US Central (Iowa)",
		"enabled": true,
		"name": "us-central1",
		"region_type": "serverless",
		"reservedForQualifiedUsers": false,
		"zone": "na"
	}
]`

func TestRegionUnmarshal(t *testing.T) {
	var regions []Region
	if err := json.Unmarshal([]byte(sampleRegionsResponse), &regions); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(regions))
	}

	// Verify first region
	region := regions[0]
	if region.Name != "us-east-1" {
		t.Errorf("expected name 'us-east-1', got %s", region.Name)
	}
	if region.CloudProvider != "aws" {
		t.Errorf("expected cloudProvider 'aws', got %s", region.CloudProvider)
	}
	if region.DisplayName != "US East (N. Virginia)" {
		t.Errorf("expected displayName 'US East (N. Virginia)', got %s", region.DisplayName)
	}
	if !region.Enabled {
		t.Error("expected enabled to be true")
	}
	if region.RegionType != "vector" {
		t.Errorf("expected region_type 'vector', got %s", region.RegionType)
	}
	if region.Classification != "general-purpose" {
		t.Errorf("expected classification 'general-purpose', got %s", region.Classification)
	}
	if region.Zone != "na" {
		t.Errorf("expected zone 'na', got %s", region.Zone)
	}
	if region.ReservedForQualifiedUsers {
		t.Error("expected reservedForQualifiedUsers to be false")
	}

	// Verify second region
	region2 := regions[1]
	if region2.Name != "us-central1" {
		t.Errorf("expected name 'us-central1', got %s", region2.Name)
	}
	if region2.CloudProvider != "gcp" {
		t.Errorf("expected cloudProvider 'gcp', got %s", region2.CloudProvider)
	}
}

func TestFindAvailableRegionsOptionsBuilder(t *testing.T) {
	t.Run("filter by org", func(t *testing.T) {
		opts := options.FindAvailableRegions().SetFilterByOrg(true)
		merged, err := options.MergeAndValidate(opts)
		if err != nil {
			t.Fatalf("MergeAndValidate: %v", err)
		}
		if merged.FilterByOrg == nil || *merged.FilterByOrg != true {
			t.Error("expected FilterByOrg to be true")
		}
	})

	t.Run("combined options", func(t *testing.T) {
		opts := options.FindAvailableRegions().
			SetFilterByOrg(true)
		merged, err := options.MergeAndValidate(opts)
		if err != nil {
			t.Fatalf("MergeAndValidate: %v", err)
		}
		if merged.FilterByOrg == nil || *merged.FilterByOrg != true {
			t.Error("expected FilterByOrg to be true")
		}
	})
}

func TestAdminResolveOptions(t *testing.T) {
	// Verify that AstraAdmin inherits options from client
	client := NewClient(options.WithToken("client-token"))
	admin, err := client.Admin()
	if err != nil {
		t.Fatalf("Admin() returned unexpected error: %v", err)
	}

	opts := admin.resolveOptions()
	if opts.GetToken() != "client-token" {
		t.Errorf("expected token 'client-token', got %s", opts.GetToken())
	}
}

func TestAdminOptionOverride(t *testing.T) {
	// Verify that AstraAdmin-level options override client options
	client := NewClient(options.WithToken("client-token"))
	admin, err := client.Admin(options.WithToken("admin-token"))
	if err != nil {
		t.Fatalf("Admin() returned unexpected error: %v", err)
	}

	opts := admin.resolveOptions()
	if opts.GetToken() != "admin-token" {
		t.Errorf("expected token 'admin-token', got %s", opts.GetToken())
	}
}

func TestFindAvailableRegionsOptionsStruct(t *testing.T) {
	// Test that the raw struct can be used directly (implements Builder)
	opts := &options.FindAvailableRegionsOptions{
		FilterByOrg: boolPtr(true),
	}

	merged, err := options.MergeAndValidate(opts)
	if err != nil {
		t.Fatalf("MergeAndValidate: %v", err)
	}
	if merged.FilterByOrg == nil || *merged.FilterByOrg != true {
		t.Error("expected FilterByOrg to be true")
	}
}

/*

curl -sS -L -X GET "https://api.astra.datastax.com/v2/regions/serverless?region-type=REGION_TYPE&filter-by-org=FILTER_BY_ORG" \
--header "Authorization: Bearer APPLICATION_TOKEN" \
--header "Content-Type: application/json"
*/

func TestSTuff(t *testing.T) {
	// Verify that AstraAdmin inherits options from client
	client := NewClient(options.WithToken("client-token"))
	admin, err := client.Admin()
	if err != nil {
		t.Fatalf("Admin() returned unexpected error: %v", err)
	}
	cmd := admin.createCommand("GET", "/regions/serverless", nil)
	url, err := cmd.url()
	if err != nil {
		t.Fatalf("cmd.url() producted unexpected error: %v", err)
	}
	expectedURL := "https://api.astra.datastax.com/v2/regions/serverless"
	if url != expectedURL {
		t.Errorf("expected: %s\ngot: %s", expectedURL, url)
	}
}

func TestAdminEnvironmentDefaultsProd(t *testing.T) {
	client := NewClient(options.WithToken("token"))
	admin, err := client.Admin()
	if err != nil {
		t.Fatalf("Admin() returned unexpected error: %v", err)
	}
	if admin.astraEnvironment != options.AstraEnvironmentProd {
		t.Errorf("expected prod environment, got %s", admin.astraEnvironment)
	}
}

func TestAdminEnvironmentFromClientOptions(t *testing.T) {
	client := NewClient(
		options.WithToken("token"),
		options.WithAstraEnvironment(options.AstraEnvironmentDev),
	)
	admin, err := client.Admin()
	if err != nil {
		t.Fatalf("Admin() returned unexpected error: %v", err)
	}
	if admin.astraEnvironment != options.AstraEnvironmentDev {
		t.Errorf("expected dev environment, got %s", admin.astraEnvironment)
	}
}

func TestAdminEnvironmentOverriddenAtAdminLevel(t *testing.T) {
	client := NewClient(
		options.WithToken("token"),
		options.WithAstraEnvironment(options.AstraEnvironmentDev),
	)
	admin, err := client.Admin(options.WithAstraEnvironment(options.AstraEnvironmentTest))
	if err != nil {
		t.Fatalf("Admin() returned unexpected error: %v", err)
	}
	if admin.astraEnvironment != options.AstraEnvironmentTest {
		t.Errorf("expected test environment, got %s", admin.astraEnvironment)
	}
}

func TestAdminNotAvailableForNonAstra(t *testing.T) {
	client := NewClient(
		options.WithToken("token"),
		options.WithDataAPIBackend(options.DataAPIBackendHCD),
	)
	_, err := client.Admin()
	if err == nil {
		t.Error("expected error for non-Astra backend, got nil")
	}
}

func TestExtractDevopsError(t *testing.T) {
	// Test that a DevOps error response is properly extracted and parsed
	errorBody := "{\"errors\":[{\"ID\":340002,\"message\":\"no bearer token in request\"}]}"
	err := extractDevOpsError(401, []byte(errorBody))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// TODO: switch to errors.AsType when it becomes more widely available:
	// https://go.dev/doc/go1.26#errorspkgerrors
	var errs DataAPIErrors
	if !errors.As(err, &errs) {
		t.Fatalf("expecting error of type DataAPIErrors. Got %s", err)
	}
	if len(errs) != 1 {
		t.Fatalf("expecting len(errs) to = 1. Got %d", len(errs))
	}
	expected := "no bearer token in request"
	if errs[0].Message != expected {
		t.Fatalf("expecting Message %q. got %q", expected, errs[0].Message)
	}
}

func TestAwaitStatusOptions(t *testing.T) {
	opts := &AwaitStatusOptions{
		Target:      "ACTIVE",
		LegalStates: []DatabaseStatus{"MAINTENANCE"},
	}

	if !opts.IsStatusLegal(DatabaseStatusMaintenance) {
		t.Error("expected IsStatusLegal to return true for MAINTENANCE")
	}

	if opts.IsStatusLegal(DatabaseStatusTerminated) {
		t.Error("expected IsStatusLegal to return false for TERMINATED")
	}
}

// Actual response from ListDatabases.
const exampleRespListDatabases = `[{"availableActions":["getCreds","addDatacenters","cloneDatabaseFromSnapshot","terminateDatacenter","setEngineType","terminate","addKeyspace","removeKeyspace","suspend","updateVectorAgent","getVectorAgentConfig","hibernate","migrate"],"cost":{"costPerDayCents":0,"costPerDayMRCents":0,"costPerDayParkedCents":0,"costPerHourCents":0,"costPerHourMRCents":0,"costPerHourParkedCents":0,"costPerMinCents":0,"costPerMinMRCents":0,"costPerMinParkedCents":0,"costPerMonthCents":0,"costPerMonthMRCents":0,"costPerMonthParkedCents":0,"costPerNetworkGbCents":0,"costPerReadGbCents":0.1,"costPerWrittenGbCents":0.1},"cqlshUrl":"https://f4c9684a-75f9-4674-8d10-664f36af90f9-us-east-2.apps.astra.datastax.com/cqlsh","creationTime":"2025-11-03T22:47:51Z","dataEndpointUrl":"https://f4c9684a-75f9-4674-8d10-664f36af90f9-us-east-2.apps.astra.datastax.com/api/rest","grafanaUrl":"https://f4c9684a-75f9-4674-8d10-664f36af90f9-us-east-2.dashboard.astra.datastax.com/d/cloud/dse-cluster-condensed?refresh=30s\u0026orgId=1\u0026kiosk=tv","graphqlUrl":"https://f4c9684a-75f9-4674-8d10-664f36af90f9-us-east-2.apps.astra.datastax.com/api/graphql","id":"f4c9684a-75f9-4674-8d10-664f36af90f9","info":{"additionalKeyspaces":["go_sdk_test_ks_1770849991507","go_sdk_test_ks_1771364762314","go_sdk_test_ks_1771365114440"],"capacityUnits":1,"cloudProvider":"AWS","datacenters":[{"capacityUnits":1,"cloudAccount":"551367681185","cloudProvider":"AWS","dateCreated":"2025-11-03T22:47:51Z","id":"f4c9684a-75f9-4674-8d10-664f36af90f9-1","isPrimary":true,"name":"dc-1","region":"us-east-2","regionClassification":"standard","regionZone":"na","requestedNodeCount":3,"secureBundleInternalUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/f4c9684a-75f9-4674-8d10-664f36af90f9-1/secure-connect-internal-deantest.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=171e5fb9c23f29881fbd0d77341c47b944af462143db0757d0b71e144b0f5b5d","secureBundleMigrationProxyInternalUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/f4c9684a-75f9-4674-8d10-664f36af90f9-1/secure-connect-proxy-internal-deantest.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=ab8d0b5bd6ad7f7717bedf7a42e2972b278dbef4c4097aa07e4206567b8c69b8","secureBundleMigrationProxyUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/f4c9684a-75f9-4674-8d10-664f36af90f9-1/secure-connect-proxy-deantest.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=c5b835f989e1f80817417d2a86903cbe97791b0ccc8f36bd21e276579777d01e","secureBundleUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/f4c9684a-75f9-4674-8d10-664f36af90f9-1/secure-connect-deantest.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=1d96e357950121182a7947320a470d0d421344a387451830233aff468683e3cd","status":"","streamingTenant":{"streamingClusterName":"","streamingTenantName":""},"targetAccount":"551367681185","tier":"serverless"}],"dbType":"vector","keyspace":"default_keyspace","keyspaces":["default_keyspace","go_sdk_test_ks_1770849991507","go_sdk_test_ks_1771364762314","go_sdk_test_ks_1771365114440"],"name":"deantest","region":"us-east-2","tier":"serverless"},"lastUsageTime":"2026-02-17T22:15:27Z","metrics":{"errorsTotalCount":0,"liveDataSizeBytes":0,"readRequestsTotalCount":0,"writeRequestsTotalCount":0},"observedStatus":"ACTIVE","orgId":"36b475ee-0fee-4253-ab76-65c2fa06cbe8","ownerId":"431ae7d6-05e3-4b0a-af9c-540d8ab2372d","status":"ACTIVE","storage":{"displayStorage":10,"nodeCount":3,"replicationFactor":1,"totalStorage":5},"terminationTime":"0001-01-01T00:00:00Z"},{"cost":{"costPerDayCents":0,"costPerDayMRCents":0,"costPerDayParkedCents":0,"costPerHourCents":0,"costPerHourMRCents":0,"costPerHourParkedCents":0,"costPerMinCents":0,"costPerMinMRCents":0,"costPerMinParkedCents":0,"costPerMonthCents":0,"costPerMonthMRCents":0,"costPerMonthParkedCents":0,"costPerNetworkGbCents":0,"costPerReadGbCents":0.1,"costPerWrittenGbCents":0.1},"cqlshUrl":"https://5dab41cb-9e58-45dd-9049-797128af992a-us-east1.apps.astra.datastax.com/cqlsh","creationTime":"2026-02-06T22:20:58Z","dataEndpointUrl":"https://5dab41cb-9e58-45dd-9049-797128af992a-us-east1.apps.astra.datastax.com/api/rest","grafanaUrl":"https://5dab41cb-9e58-45dd-9049-797128af992a-us-east1.dashboard.astra.datastax.com/d/cloud/dse-cluster-condensed?refresh=30s\u0026orgId=1\u0026kiosk=tv","graphqlUrl":"https://5dab41cb-9e58-45dd-9049-797128af992a-us-east1.apps.astra.datastax.com/api/graphql","id":"5dab41cb-9e58-45dd-9049-797128af992a","info":{"capacityUnits":1,"cloudProvider":"GCP","datacenters":[{"capacityUnits":1,"cloudAccount":"astra-serverless-prod-63","cloudProvider":"GCP","dateCreated":"2026-02-06T22:20:58Z","id":"5dab41cb-9e58-45dd-9049-797128af992a-1","isPrimary":true,"name":"dc-1","region":"us-east1","regionClassification":"standard","regionZone":"na","requestedNodeCount":3,"secureBundleInternalUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/5dab41cb-9e58-45dd-9049-797128af992a-1/secure-connect-internal-go-sdk-integration-test.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=e4a5d6dc1d425dd37b234fe04bec2ceb11af3fa4eb26b8c183c481fe5761869b","secureBundleMigrationProxyInternalUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/5dab41cb-9e58-45dd-9049-797128af992a-1/secure-connect-proxy-internal-go-sdk-integration-test.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=81492b480f57032c6ea4450539fa1f8e5366f583b858f6cf3c2628a05751ac3e","secureBundleMigrationProxyUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/5dab41cb-9e58-45dd-9049-797128af992a-1/secure-connect-proxy-go-sdk-integration-test.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=1766f2b98f5800d06eefeca6ecce3b8612dfd18d1e30c6bac7401623d6e74b9f","secureBundleUrl":"https://datastax-cluster-config-prod.s3.us-east-2.amazonaws.com/5dab41cb-9e58-45dd-9049-797128af992a-1/secure-connect-go-sdk-integration-test.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=AKIA2AIQRQ76WSWBKT4M%2F20260218%2Fus-east-2%2Fs3%2Faws4_request\u0026X-Amz-Date=20260218T214251Z\u0026X-Amz-Expires=300\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=2362afbee5387d6f8f623e3d730dddf39b9ae9d22cd908ac35b51136438059e5","status":"","streamingTenant":{"streamingClusterName":"","streamingTenantName":""},"targetAccount":"astra-serverless-prod-63","tier":"serverless"}],"dbType":"vector","keyspace":"default_keyspace","keyspaces":["default_keyspace"],"name":"go-sdk-integration-test","region":"us-east1","tier":"serverless"},"lastUsageTime":"2026-02-06T22:20:58Z","metrics":{"errorsTotalCount":0,"liveDataSizeBytes":0,"readRequestsTotalCount":0,"writeRequestsTotalCount":0},"observedStatus":"TERMINATED","orgId":"36b475ee-0fee-4253-ab76-65c2fa06cbe8","ownerId":"rFKTZxWCZbnyojzvJatPQjUk","status":"TERMINATED","storage":{"displayStorage":10,"nodeCount":3,"replicationFactor":1,"totalStorage":5},"terminationTime":"2026-02-07T00:28:20Z"}]`

func TestListDatbasesJSON(t *testing.T) {
	var raw []rawDatabaseResponse
	if err := json.Unmarshal([]byte(exampleRespListDatabases), &raw); err != nil {
		t.Errorf("failed to parse databases response: %v", err)
	}
	databases := make([]DatabaseInfo, len(raw))
	for i := range raw {
		databases[i] = *raw[i].toDatabaseInfo(options.AstraEnvironmentDev)
	}
	// Make sure we got 2 DBs.
	if len(databases) != 2 {
		t.Errorf("expected 2 databases, got %d", len(databases))
	}
	// Here's raw JSON: "creationTime":"2025-11-03T22:47:51Z".
	// Make sure it unmarshals to the expected time.Time value.
	expected := time.Date(2025, 11, 3, 22, 47, 51, 0, time.UTC)
	if !databases[0].CreatedAt.Equal(expected) {
		t.Errorf("expected creation time %s, got %s", expected.Format(time.RFC3339), databases[0].CreatedAt.Format(time.RFC3339))
	}
	// Examine regions
	if len(databases[0].Regions) != 1 {
		t.Errorf("expected 1 region, got %d", len(databases[0].Regions))
		t.FailNow()
	}
	if databases[0].Regions[0].APIEndpoint != "https://f4c9684a-75f9-4674-8d10-664f36af90f9-us-east-2.apps.astra-dev.datastax.com" {
		t.Errorf("unexpected API endpoint: %s", databases[0].Regions[0].APIEndpoint)
	}
}
