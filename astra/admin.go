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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/internal/command"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
)

// DefaultAdminAPIVersion is the default version of the Astra DevOps API.
const DefaultAdminAPIVersion = "v2"

// AstraAdmin provides access to Astra DevOps API operations.
// Obtain an AstraAdmin instance from DataAPIClient.Admin().
// Only valid for Astra environments.
type AstraAdmin struct {
	client           *DataAPIClient
	options          *options.APIOptions // Cumulative options from Client -> Admin
	apiVersion       string
	astraEnvironment options.AstraEnvironment
}

func (a *AstraAdmin) createCommand(method string, path string, payload any, params url.Values, opts ...options.Builder[options.APIOptions]) *command.DevOpsAPI {
	return command.NewDevOpsAPICommand(a.astraEnvironment.DevOpsURL(), a.apiVersion, path, method, payload, params, options.Merge(append([]options.APIOption{a.options}, opts...)...))
}

// Region represents an available serverless region from the DevOps API.
type Region struct {
	// Classification indicates the region's classification level (e.g., "standard").
	Classification string `json:"classification"`
	// CloudProvider is the cloud provider (e.g., "aws", "gcp", "azure").
	CloudProvider string `json:"cloudProvider"`
	// DisplayName is the human-readable name of the region.
	DisplayName string `json:"displayName"`
	// Enabled indicates whether the region is currently available.
	Enabled bool `json:"enabled"`
	// Name is the region identifier used in API calls.
	Name string `json:"name"`
	// RegionType indicates the type of region (e.g., "serverless", "vector").
	RegionType string `json:"region_type"`
	// ReservedForQualifiedUsers indicates if region is restricted.
	ReservedForQualifiedUsers bool `json:"reservedForQualifiedUsers"`
	// Zone is the geographic zone (e.g., "na", "eu", "apac").
	Zone string `json:"zone"`
}

// DatabaseStatus is a type alias for options.DatabaseStatus, representing
// the status of an Astra database.
type DatabaseStatus = options.DatabaseStatus

// Re-export DatabaseStatus constants for convenience.
const (
	DatabaseStatusActive        = options.DatabaseStatusActive
	DatabaseStatusAssociating   = options.DatabaseStatusAssociating
	DatabaseStatusPending       = options.DatabaseStatusPending
	DatabaseStatusInitializing  = options.DatabaseStatusInitializing
	DatabaseStatusTerminating   = options.DatabaseStatusTerminating
	DatabaseStatusTerminated    = options.DatabaseStatusTerminated
	DatabaseStatusMaintenance   = options.DatabaseStatusMaintenance
	DatabaseStatusError         = options.DatabaseStatusError
	DatabaseStatusParking       = options.DatabaseStatusParking
	DatabaseStatusParked        = options.DatabaseStatusParked
	DatabaseStatusUnparking     = options.DatabaseStatusUnparking
	DatabaseStatusPreparing     = options.DatabaseStatusPreparing
	DatabaseStatusPrepared      = options.DatabaseStatusPrepared
	DatabaseStatusResizing      = options.DatabaseStatusResizing
	DatabaseStatusSuspended     = options.DatabaseStatusSuspended
	DatabaseStatusSuspending    = options.DatabaseStatusSuspending
	DatabaseStatusNonTerminated = options.DatabaseStatusNonTerminated
	DatabaseStatusAll           = options.DatabaseStatusAll
)

// BaseAstraDatabaseInfo contains the common properties shared by both PartialAstraDatabaseInfo and FullAstraDatabaseInfo.
type BaseAstraDatabaseInfo struct {
	// ID is the unique database identifier.
	ID string
	// Name is the database name.
	Name string
	// Status is the current database status.
	Status DatabaseStatus
	// Keyspaces is the merged list of all keyspaces (default + additional).
	Keyspaces []string
	// CloudProvider is the cloud provider (e.g., "aws", "gcp", "azure").
	CloudProvider string
	// Environment is the Astra environment.
	Environment options.AstraEnvironment
	// Raw is the raw DevOps API response, provided as an escape hatch
	// for fields not in the curated view.
	Raw *rawDatabaseResponse
}

// FullAstraDatabaseInfo is the complete metadata returned for an Astra database, flattening and simplifying the raw DevOps API response.
type FullAstraDatabaseInfo struct {
	BaseAstraDatabaseInfo
	// Regions contains information about the regions where the database is deployed.
	// It will have at least one value, and may have more for multi-region deployments.
	Regions []AstraDatabaseRegionInfo
	// CreatedAt is when the database was created.
	CreatedAt time.Time
	// LastUsed is when the database was last used (zero if unknown).
	LastUsed time.Time
	// OrgID is the organization identifier.
	OrgID string
	// OwnerID is the owner's identifier.
	OwnerID string
}

// PartialAstraDatabaseInfo is the partial metadata of a database, as returned from Db.Info, flattening and simplifying the raw DevOps API response.
type PartialAstraDatabaseInfo struct {
	BaseAstraDatabaseInfo
	// Region is the region being used by the [Db] instance.
	Region string
	// APIEndpoint is the API endpoint for the region.
	APIEndpoint string
}

// AstraDatabaseRegionInfo represents information about a region in which an Astra database is hosted.
//
// This includes the region name, the API endpoint to use when interacting with that region, and the created-at timestamp.
//
// Used within the regions field of FullAstraDatabaseInfo or similar types, which may include multiple region entries for multi-region databases.
type AstraDatabaseRegionInfo struct {
	// Name is the name of the region where the database is hosted, e.g. "us-east1".
	Name string `json:"name"`
	// APIEndpoint is the API endpoint for the region, e.g. "https://<db-id>-<region>.apps.astra.datastax.com".
	APIEndpoint string `json:"apiEndpoint"`
	// CreatedAt is the timestamp representing when this region was created.
	CreatedAt time.Time `json:"createdAt"`
}

// rawDatabaseResponse represents the full database response from the DevOps API.
// Used internally for JSON deserialization; the curated [FullAstraDatabaseInfo] is the public type.
type rawDatabaseResponse struct {
	AvailableActions []string `json:"availableActions"`
	Cost             struct {
		CostPerDayCents         int     `json:"costPerDayCents"`
		CostPerDayMRCents       int     `json:"costPerDayMRCents"`
		CostPerDayParkedCents   int     `json:"costPerDayParkedCents"`
		CostPerHourCents        int     `json:"costPerHourCents"`
		CostPerHourMRCents      int     `json:"costPerHourMRCents"`
		CostPerHourParkedCents  int     `json:"costPerHourParkedCents"`
		CostPerMinCents         int     `json:"costPerMinCents"`
		CostPerMinMRCents       int     `json:"costPerMinMRCents"`
		CostPerMinParkedCents   int     `json:"costPerMinParkedCents"`
		CostPerMonthCents       int     `json:"costPerMonthCents"`
		CostPerMonthMRCents     int     `json:"costPerMonthMRCents"`
		CostPerMonthParkedCents int     `json:"costPerMonthParkedCents"`
		CostPerNetworkGbCents   int     `json:"costPerNetworkGbCents"`
		CostPerReadGbCents      float64 `json:"costPerReadGbCents"`
		CostPerWrittenGbCents   float64 `json:"costPerWrittenGbCents"`
	} `json:"cost"`
	CqlshURL        string    `json:"cqlshUrl"`
	CreationTime    time.Time `json:"creationTime"`
	DataEndpointURL string    `json:"dataEndpointUrl"`
	GrafanaURL      string    `json:"grafanaUrl"`
	GraphqlURL      string    `json:"graphqlUrl"`
	ID              string    `json:"id"`
	Info            struct {
		AdditionalKeyspaces []string `json:"additionalKeyspaces"`
		CapacityUnits       int      `json:"capacityUnits"`
		CloudProvider       string   `json:"cloudProvider"`
		Datacenters         []struct {
			CapacityUnits                         int       `json:"capacityUnits"`
			CloudAccount                          string    `json:"cloudAccount"`
			CloudProvider                         string    `json:"cloudProvider"`
			DateCreated                           time.Time `json:"dateCreated"`
			ID                                    string    `json:"id"`
			IsPrimary                             bool      `json:"isPrimary"`
			Name                                  string    `json:"name"`
			Region                                string    `json:"region"`
			RegionClassification                  string    `json:"regionClassification"`
			RegionZone                            string    `json:"regionZone"`
			RequestedNodeCount                    int       `json:"requestedNodeCount"`
			SecureBundleInternalURL               string    `json:"secureBundleInternalUrl"`
			SecureBundleMigrationProxyInternalURL string    `json:"secureBundleMigrationProxyInternalUrl"`
			SecureBundleMigrationProxyURL         string    `json:"secureBundleMigrationProxyUrl"`
			SecureBundleURL                       string    `json:"secureBundleUrl"`
			Status                                string    `json:"status"`
			StreamingTenant                       struct {
				StreamingClusterName string `json:"streamingClusterName"`
				StreamingTenantName  string `json:"streamingTenantName"`
			} `json:"streamingTenant"`
			TargetAccount string `json:"targetAccount"`
			Tier          string `json:"tier"`
		} `json:"datacenters"`
		DbType    string   `json:"dbType"`
		Keyspace  string   `json:"keyspace"`
		Keyspaces []string `json:"keyspaces"`
		Name      string   `json:"name"`
		Region    string   `json:"region"`
		Tier      string   `json:"tier"`
	} `json:"info"`
	LastUsageTime time.Time `json:"lastUsageTime"`
	Metrics       struct {
		ErrorsTotalCount        int `json:"errorsTotalCount"`
		LiveDataSizeBytes       int `json:"liveDataSizeBytes"`
		ReadRequestsTotalCount  int `json:"readRequestsTotalCount"`
		WriteRequestsTotalCount int `json:"writeRequestsTotalCount"`
	} `json:"metrics"`
	ObservedStatus string                 `json:"observedStatus"`
	OrgID          string                 `json:"orgId"`
	OwnerID        string                 `json:"ownerId"`
	Status         options.DatabaseStatus `json:"status"`
	Storage        struct {
		DisplayStorage    int `json:"displayStorage"`
		NodeCount         int `json:"nodeCount"`
		ReplicationFactor int `json:"replicationFactor"`
		TotalStorage      int `json:"totalStorage"`
	} `json:"storage"`
	TerminationTime time.Time `json:"terminationTime"`
}

// toDatabaseInfo converts a raw DevOps API response to the curated FullAstraDatabaseInfo.
func (r *rawDatabaseResponse) toDatabaseInfo(env options.AstraEnvironment) *FullAstraDatabaseInfo {
	var keyspaces []string
	if r.Info.Keyspace != "" {
		keyspaces = append(keyspaces, r.Info.Keyspace)
	}
	keyspaces = append(keyspaces, r.Info.AdditionalKeyspaces...)

	regions := make([]AstraDatabaseRegionInfo, len(r.Info.Datacenters))
	for i, dc := range r.Info.Datacenters {
		regions[i] = AstraDatabaseRegionInfo{
			Name:        dc.Region,
			APIEndpoint: env.AstraDBEndpoint(r.ID, dc.Region),
			CreatedAt:   dc.DateCreated,
		}
	}

	return &FullAstraDatabaseInfo{
		BaseAstraDatabaseInfo: BaseAstraDatabaseInfo{
			ID:            r.ID,
			Name:          r.Info.Name,
			Status:        r.Status,
			Keyspaces:     keyspaces,
			CloudProvider: r.Info.CloudProvider,
			Environment:   env,
			Raw:           r,
		},
		Regions:   regions,
		CreatedAt: r.CreationTime,
		LastUsed:  r.LastUsageTime,
		OrgID:     r.OrgID,
		OwnerID:   r.OwnerID,
	}
}

// ClientOptions returns the admin's options as a resolved struct with defaults.
func (a *AstraAdmin) ClientOptions() *options.APIOptions {
	return a.options
}

// FindAvailableRegions retrieves available serverless regions from the DevOps API.
//
// Example - get all regions:
//
//	admin, err := client.Admin()
//	regions, err := admin.FindAvailableRegions(ctx)
//
// Example - filter by organization access:
//
//	regions, err := admin.FindAvailableRegions(ctx,
//	    options.FindAvailableRegions().SetFilterByOrg(true))
func (a *AstraAdmin) FindAvailableRegions(ctx context.Context, opts ...options.FindAvailableRegionsOption) ([]Region, error) {
	// Merge options
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	// Build command with query parameters.
	// Hard-coding to region-type=vector because classic isn't relevant to this client.
	params := url.Values{}
	params.Set("region-type", "vector")
	if ptr.From(merged.FilterByOrg) {
		params.Set("filter-by-org", "enabled")
	}
	cmd := a.createCommand(http.MethodGet, "/regions/serverless", nil, params, merged.APIOptions)

	// Execute request
	resp, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	// Parse response - the API returns a JSON array of regions
	var regions []Region
	if err := json.Unmarshal(resp.Body, &regions); err != nil {
		return nil, fmt.Errorf("failed to parse regions response: %w", err)
	}

	return regions, nil
}

// ListDatabases retrieves databases accessible to the caller.
//
// By default, only non-terminated databases are returned (up to 25).
// Use SetLimit (up to 100) and SetStartingAfter to control pagination.
//
// Example - list databases:
//
//	admin, err := client.Admin()
//	databases, err := admin.ListDatabases(ctx)
//
// Example - list only active GCP databases:
//
//	databases, err := admin.ListDatabases(ctx,
//	    options.ListDatabases().
//	        SetInclude(options.DatabaseStatusActive).
//	        SetProvider(options.CloudProviderGCP))
//
// Example - paginate through results and retrieve all databases:
//
//	func listAll(ctx context.Context, admin *astra.AstraAdmin) ([]astra.DatabaseInfo, error) {
//		var all []astra.DatabaseInfo
//		pageSize := 100
//		opts := options.ListDatabases().SetInclude(options.DatabaseStatusAll).SetLimit(pageSize)
//		for {
//			databases, err := admin.ListDatabases(ctx, opts)
//			if err != nil {
//				return nil, fmt.Errorf("admin.ListDatabases failed: %w", err)
//			}
//			all = append(all, databases...)
//			if len(databases) < pageSize {
//				break
//			}
//			// Set up cursor for next page
//			opts.SetStartingAfter(databases[len(databases)-1].ID)
//		}
//		return all, nil
//	}
func (a *AstraAdmin) ListDatabases(ctx context.Context, opts ...options.ListDatabasesOption) ([]FullAstraDatabaseInfo, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if merged.Include != nil {
		params.Set("include", string(*merged.Include))
	}
	if merged.Provider != nil {
		params.Set("provider", string(*merged.Provider))
	}
	if merged.Limit != nil {
		params.Set("limit", fmt.Sprintf("%d", *merged.Limit))
	}
	if merged.StartingAfter != nil {
		params.Set("starting_after", *merged.StartingAfter)
	}
	cmd := a.createCommand(http.MethodGet, "/databases", nil, params, merged.APIOptions)

	resp, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var raw []rawDatabaseResponse
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse databases response: %w", err)
	}

	databases := make([]FullAstraDatabaseInfo, len(raw))
	for i := range raw {
		databases[i] = *raw[i].toDatabaseInfo(a.astraEnvironment)
	}

	return databases, nil
}

// DatabaseInfo retrieves information about a specific database.
//
// Example:
//
//	admin, err := client.Admin()
//	db, err := admin.DatabaseInfo(ctx, "database-id")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Status:", db.Status)
func (a *AstraAdmin) DatabaseInfo(ctx context.Context, databaseID string, opts ...options.DatabaseInfoOption) (*FullAstraDatabaseInfo, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	cmd := a.createCommand(http.MethodGet, "/databases/"+databaseID, nil, nil, merged.APIOptions)
	resp, err := cmd.Execute(ctx)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Wrap error to provide more context and allow callers to check with errors.Is(err, ErrNotFound)
			return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
		}
		return nil, err
	}

	var raw rawDatabaseResponse
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse database response: %w", err)
	}

	return raw.toDatabaseInfo(a.astraEnvironment), nil
}

// CreateDatabaseParams contains the required parameters for creating a database.
type CreateDatabaseParams struct {
	// Name is the database name. Must start and end with a letter or number.
	// Can contain letters, numbers, and special characters: & + - _ ( ) < > . , @
	// Cannot exceed 50 characters.
	Name string
	// CloudProvider is the cloud provider (e.g., "aws", "gcp", "azure").
	CloudProvider string
	// Region is the cloud provider region for the database location.
	Region string
}

// createDatabaseRequest is the request payload for the create database API.
type createDatabaseRequest struct {
	Name          string `json:"name"`
	CloudProvider string `json:"cloudProvider"`
	Region        string `json:"region"`
	Keyspace      string `json:"keyspace,omitempty"`
	DbType        string `json:"dbType"`
	Tier          string `json:"tier"`
	CapacityUnits int    `json:"capacityUnits"`
}

// DatabaseAdmin returns an AstraDatabaseAdmin handle for the given database ID and region.
//
// No API calls are made; this simply creates a handle for performing
// admin operations on the specified database. Options provided override the
// defaults set on the DataAPIClient for the underlying Db instance.
//
// Example:
//
//	dbAdmin := admin.DatabaseAdmin("a6a1d8d6-...-377566f345bf", "us-east1")
//	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
func (a *AstraAdmin) DatabaseAdmin(id, region string, opts ...options.APIOption) *AstraDatabaseAdmin {
	return &AstraDatabaseAdmin{a, newDbFromID(id, region, a.astraEnvironment,
		a.client, options.Merge(append([]options.APIOption{a.options}, opts...)...))}
}

// DatabaseAdminFromEndpoint returns an AstraDatabaseAdmin handle for the given database endpoint.
//
// The endpoint should be in the form https://<db_id>-<region>.apps.astra.datastax.com.
// No API calls are made; this simply creates a handle for performing admin operations
// on the specified database. Options provided override the defaults set on the
// DataAPIClient for the underlying Db instance.
//
// Example:
//
//	dbAdmin := admin.DatabaseAdminFromEndpoint("https://<db_id>-<region>.apps.astra.datastax.com")
//	keyspaces, err := dbAdmin.ListKeyspaces(ctx)
func (a *AstraAdmin) DatabaseAdminFromEndpoint(endpoint string, opts ...options.APIOption) *AstraDatabaseAdmin {
	return &AstraDatabaseAdmin{a, newDbFromEndpoint(endpoint, a.client, options.Merge(append([]options.APIOption{a.options}, opts...)...))}
}

type AwaitStatusOptions struct {
	// Will default to sane value
	PollInterval time.Duration
	// The status we are waiting for
	Target DatabaseStatus
	// Legal statuses that DB can/will enter before entering target status.
	LegalStates []DatabaseStatus
	// APIOptions are the API options to use for status checks.
	APIOptions *options.APIOptions
}

// Case-insensitive compare out of an abundance of caution
func compareStatus(s DatabaseStatus, t DatabaseStatus) bool {
	return strings.EqualFold(string(s), string(t))
}

// Interval returns PollInterval if non-zero and falls back to default.
func (o *AwaitStatusOptions) Interval() time.Duration {
	if o.PollInterval <= 0 {
		return options.DefaultDatabasePollInterval
	}
	return o.PollInterval
}

// IsStatusLegal returns true if the given status is in the list of legal states.
func (o *AwaitStatusOptions) IsStatusLegal(s DatabaseStatus) bool {
	for _, legal := range o.LegalStates {
		if compareStatus(s, legal) {
			return true
		}
	}
	return false
}

// awaitStatus polls DatabaseInfo until the status matches a target or hits a failure state.
// See [AwaitStatusOptions] for configuration.
func (a *AstraAdmin) awaitStatus(ctx context.Context, databaseID string, opts AwaitStatusOptions) error {
	ticker := time.NewTicker(opts.Interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			db, err := a.DatabaseInfo(ctx, databaseID, options.DatabaseInfo().UpdateAPIOptions(opts.APIOptions))
			if err != nil {
				return err
			}
			if db.Status == opts.Target {
				// We hit our desired status
				return nil
			}
			// Otherwise, let's make sure this is an allowed status.
			if opts.IsStatusLegal(db.Status) {
				continue
			}
			return fmt.Errorf("database entered unexpected status: %s", db.Status)
		}
	}
}

// CreateDatabase creates a new serverless vector database and returns an
// [AstraDatabaseAdmin] for performing admin operations on it.
//
// The DevOps API endpoint is: POST https://api.astra.datastax.com/v2/databases
//
// By default, this method blocks until the database reaches ACTIVE status (typically
// about 2 minutes). Use SetBlocking(false) to return immediately after the creation
// request is accepted.
//
// Example - create a database (blocking by default):
//
//	admin, err := client.Admin()
//	dbAdmin, err := admin.CreateDatabase(ctx, astra.CreateDatabaseParams{
//	    Name:          "my-database",
//	    CloudProvider: "gcp",
//	    Region:        "us-east1",
//	})
//
// Example - create without waiting:
//
//	dbAdmin, err := admin.CreateDatabase(ctx, astra.CreateDatabaseParams{
//	    Name:          "my-database",
//	    CloudProvider: "gcp",
//	    Region:        "us-east1",
//	}, options.CreateDatabase().SetBlocking(false))
//
// Example - create with custom keyspace and poll interval:
//
//	dbAdmin, err := admin.CreateDatabase(ctx, astra.CreateDatabaseParams{
//	    Name:          "my-database",
//	    CloudProvider: "aws",
//	    Region:        "us-east-1",
//	}, options.CreateDatabase().
//	    SetKeyspace("my_keyspace").
//	    SetPollInterval(5 * time.Second))
func (a *AstraAdmin) CreateDatabase(ctx context.Context, params CreateDatabaseParams, opts ...options.CreateDatabaseOption) (*AstraDatabaseAdmin, error) {
	// Merge options
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	// Build request payload
	payload := createDatabaseRequest{
		Name:          params.Name,
		CloudProvider: params.CloudProvider,
		Region:        params.Region,
		DbType:        "vector",
		Tier:          "serverless",
		CapacityUnits: 1,
	}
	if merged.Keyspace != nil {
		payload.Keyspace = *merged.Keyspace
	}

	// Execute request
	cmd := a.createCommand(http.MethodPost, "/databases", payload, nil, merged.APIOptions)
	httpResp, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	// Database ID is in the location header.
	dbID := httpResp.Headers.Get("Location")
	if dbID == "" {
		return nil, fmt.Errorf("missing Location header in response")
	}

	region := params.Region

	dbAdmin := a.DatabaseAdmin(dbID, region)

	if !merged.GetBlocking() {
		return dbAdmin, nil
	}
	// Poll until database is ACTIVE
	awaitOpts := AwaitStatusOptions{
		PollInterval: merged.GetPollInterval(),
		Target:       DatabaseStatusActive,
		LegalStates:  []DatabaseStatus{DatabaseStatusInitializing, DatabaseStatusPending, DatabaseStatusAssociating},
		APIOptions:   merged.APIOptions,
	}
	err = a.awaitStatus(ctx, dbID, awaitOpts)
	return dbAdmin, err
}

// DropDatabase terminates a database, permanently deleting all of its data.
//
// The DevOps API endpoint is: POST https://api.astra.datastax.com/v2/databases/{id}/terminate
//
// By default, this method blocks until the database is fully terminated (typically
// about 6-7 minutes). Use SetBlocking(false) to return immediately after the termination
// request is accepted.
//
// WARNING: This action cannot be undone. All data, including automatic backups, will be
// permanently deleted.
//
// Example - drop database (blocking by default):
//
//	admin, err := client.Admin()
//	err = admin.DropDatabase(ctx, "database-id")
//
// Example - drop without waiting:
//
//	err := admin.DropDatabase(ctx, "database-id",
//	    options.DropDatabase().SetBlocking(false))
func (a *AstraAdmin) DropDatabase(ctx context.Context, databaseID string, opts ...options.DropDatabaseOption) error {
	// Merge options
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	cmd := a.createCommand(http.MethodPost, "/databases/"+databaseID+"/terminate", nil, nil, merged.APIOptions)
	_, err = cmd.Execute(ctx)
	if err != nil {
		return err
	}

	if !merged.GetBlocking() {
		return nil
	}
	// Poll until database is terminated
	awaitOpts := AwaitStatusOptions{
		PollInterval: merged.GetPollInterval(),
		Target:       DatabaseStatusTerminated,
		LegalStates:  []DatabaseStatus{DatabaseStatusTerminating},
		APIOptions:   merged.APIOptions,
	}

	return a.awaitStatus(ctx, databaseID, awaitOpts)
}
