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

	"time"

	"github.com/datastax/astra-db-go/v2/astra/internal/command"
	"github.com/datastax/astra-db-go/v2/astra/internal/timeout"
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
	// PCUTypes lists the PCU types available in this region.
	// May be nil if the region has no PCU type information.
	PCUTypes []PCUGroupType `json:"pcu_types,omitempty"`
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

// PCUGroupTypeDetails contains detailed hardware specifications for a PCU group type.
type PCUGroupTypeDetails struct {
	// VCPU is the number of virtual CPUs.
	VCPU *int `json:"vCPU,omitempty"`
	// Memory is the memory specification (e.g., "32GiB").
	Memory *string `json:"memory,omitempty"`
	// DiskCache is the disk cache specification.
	DiskCache *string `json:"disk_cache,omitempty"`
}

// PCUGroupType describes the type/tier of a PCU group.
type PCUGroupType struct {
	// Type is the PCU type identifier.
	Type string `json:"type"`
	// Region is the region this type applies to.
	Region *string `json:"region,omitempty"`
	// CloudProvider is the cloud provider for this PCU type.
	CloudProvider *options.CloudProvider `json:"provider,omitempty"`
	// Details contains the hardware specifications for this PCU type.
	Details *PCUGroupTypeDetails `json:"details,omitempty"`
}

// PCUGroup represents a Provisioned Capacity Unit (PCU) group in an Astra organization.
//
// PCU groups are used to attach databases to provisioned capacity during creation.
// See: https://docs.datastax.com/en/astra-db-serverless/administration/provisioned-capacity-units.html
type PCUGroup struct {
	// ID is the unique identifier of the PCU group.
	ID string `json:"id"`
	// OrgID is the organization identifier.
	OrgID *string `json:"orgId,omitempty"`
	// Title is the human-readable name of the PCU group.
	Title *string `json:"title,omitempty"`
	// CloudProvider is the cloud provider for this PCU group (e.g., "aws", "gcp", "azure").
	CloudProvider options.CloudProvider `json:"cloudProvider"`
	// Region is the region where this PCU group is deployed.
	Region string `json:"region"`
	// InstanceType is the instance type of the PCU group.
	InstanceType *string `json:"instanceType,omitempty"`
	// PCUType describes the type/tier of this PCU group.
	PCUType *PCUGroupType `json:"pcuType,omitempty"`
	// ProvisionType indicates how the PCU is provisioned.
	ProvisionType *string `json:"provisionType,omitempty"`
	// Min is the minimum number of PCUs.
	Min *int `json:"min,omitempty"`
	// Max is the maximum number of PCUs.
	Max *int `json:"max,omitempty"`
	// Description is a human-readable description.
	Description *string `json:"description,omitempty"`
	// CreatedAt is the creation timestamp.
	CreatedAt *string `json:"createdAt,omitempty"`
	// UpdatedAt is the last updated timestamp.
	UpdatedAt *string `json:"updatedAt,omitempty"`
	// CreatedBy is the identifier of who created this group.
	CreatedBy *string `json:"createdBy,omitempty"`
	// UpdatedBy is the identifier of who last updated this group.
	UpdatedBy *string `json:"updatedBy,omitempty"`
	// Status is the current status of the PCU group.
	Status *string `json:"status,omitempty"`
	// Reserved is the number of reserved PCUs.
	Reserved *int `json:"reserved,omitempty"`
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
	resp, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
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
//	        SetProvider(options.CloudProviderFilterGCP))
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

	resp, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
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
	resp, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
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
	// CloudProvider is the cloud provider (e.g., "aws", "gcp", "azure").
	CloudProvider options.CloudProvider
	// Region is the cloud provider region for the database location.
	Region string
	// PCUGroupUUID is the optional UUID of a PCU group to attach the database to upon creation.
	// If set, CreateDatabase will validate that a PCU group with this ID exists and matches
	// the specified CloudProvider and Region before submitting the creation request.
	PCUGroupUUID string
	// Tier is the database tier (e.g., "serverless"). Defaults to "serverless".
	Tier string
	// CapacityUnits is the number of capacity units. Defaults to 1.
	CapacityUnits int
	// DbType is the type of database. Defaults to "vector".
	// If set to "nonvector", the field is omitted from the request.
	DbType string
}

// createDatabaseRequest is the request payload for the create database API.
type createDatabaseRequest struct {
	Name          string                `json:"name"`
	CloudProvider options.CloudProvider `json:"cloudProvider"`
	Region        string                `json:"region"`
	Keyspace      string                `json:"keyspace,omitempty"`
	DbType        string                `json:"dbType,omitempty"`
	Tier          string                `json:"tier"`
	CapacityUnits int                   `json:"capacityUnits"`
	PCUGroupUUID  string                `json:"pcuGroupUUID,omitempty"`
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
	return &AstraDatabaseAdmin{a, newDbFromID(id, region, a.astraEnvironment, a.client, options.Merge(append([]options.APIOption{a.options}, opts...)...))}
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

type awaitStatusOptions struct {
	// Will default to sane value
	PollInterval time.Duration
	// The status we are waiting for
	Target DatabaseStatus
	// Legal statuses that DB can/will enter before entering target status.
	LegalStates []DatabaseStatus
	// APIOptions are the API options to use for status checks.
	APIOptions *options.APIOptions
}

// Interval returns PollInterval if non-zero and falls back to default.
func (o *awaitStatusOptions) Interval() time.Duration {
	if o.PollInterval <= 0 {
		return options.DefaultDatabasePollInterval
	}
	return o.PollInterval
}

// IsStatusLegal returns true if the given status is in the list of legal states.
func (o *awaitStatusOptions) IsStatusLegal(s DatabaseStatus) bool {
	for _, legal := range o.LegalStates {
		if s == legal {
			return true
		}
	}
	return false
}

// awaitStatus polls DatabaseInfo until the status matches a target or hits a failure state.
// See [awaitStatusOptions] for configuration.
func (a *AstraAdmin) awaitStatus(ctx context.Context, databaseID string, opts awaitStatusOptions) error {
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

// ListPCUGroups retrieves PCU groups for the current organization from the DevOps API.
//
// Query the DevOps API to get a listing of the PCU groups for subsequent use in
// database creation. The return value can be filtered to a specific cloud provider
// and region, or include every PCU group in the org.
//
// If Region is set in options, CloudProvider must also be set.
//
// Example - list all PCU groups:
//
//	admin, err := client.Admin()
//	groups, err := admin.ListPCUGroups(ctx)
//
// Example - filter by provider and region:
//
//	groups, err := admin.ListPCUGroups(ctx,
//	    options.ListPCUGroups().SetCloudProvider("gcp").SetRegion("us-east1"))
func (a *AstraAdmin) ListPCUGroups(ctx context.Context, opts ...options.ListPCUGroupsOption) ([]PCUGroup, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	cmd := a.createCommand(http.MethodPost, "/pcus/actions/get", struct{}{}, nil, merged.APIOptions)

	resp, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
	if err != nil {
		return nil, err
	}

	var groups []PCUGroup
	if err := json.Unmarshal(resp.Body, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse PCU groups response: %w", err)
	}

	if merged.CloudProvider != nil {
		filtered := groups[:0]
		for _, g := range groups {
			if g.CloudProvider == *merged.CloudProvider && (merged.Region == nil || g.Region == *merged.Region) {
				filtered = append(filtered, g)
			}
		}
		return filtered, nil
	}

	return groups, nil
}

// validatePCUGroupExists checks that the given PCU group UUID exists and matches
// the cloud provider and region specified in params. Called by CreateDatabase
// when params.PCUGroupUUID is set.
func (a *AstraAdmin) validatePCUGroupExists(ctx context.Context, params CreateDatabaseParams, apiOpts *options.APIOptions) error {
	var opts []options.ListPCUGroupsOption
	if apiOpts != nil {
		opts = append(opts, options.ListPCUGroups().UpdateAPIOptions(apiOpts))
	}
	groups, err := a.ListPCUGroups(ctx, opts...)
	if err != nil {
		return err
	}

	var found *PCUGroup
	for i := range groups {
		if groups[i].ID == params.PCUGroupUUID {
			found = &groups[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("requested PCU group ID %q not found for cloud provider/region (%q / %q): aborting database creation",
			params.PCUGroupUUID, params.CloudProvider, params.Region)
	}

	if found.CloudProvider != params.CloudProvider || found.Region != params.Region {
		return fmt.Errorf("requested PCU group ID %q is in another cloud provider and region (%q / %q): aborting database creation",
			params.PCUGroupUUID, found.CloudProvider, found.Region)
	}

	return nil
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
//	dbAdmin, err := admin.CreateDatabase(ctx, "my-database", astra.CreateDatabaseParams{
//	    CloudProvider: "gcp",
//	    Region:        "us-east1",
//	})
//
// Example - create without waiting:
//
//	dbAdmin, err := admin.CreateDatabase(ctx, "my-database", astra.CreateDatabaseParams{
//	    CloudProvider: "gcp",
//	    Region:        "us-east1",
//	}, options.CreateDatabase().SetBlocking(false))
//
// Example - create with custom keyspace and poll interval:
//
//	dbAdmin, err := admin.CreateDatabase(ctx, "my-database", astra.CreateDatabaseParams{
//	    CloudProvider: "aws",
//	    Region:        "us-east-1",
//	}, options.CreateDatabase().
//	    SetKeyspace("my_keyspace").
//	    SetPollInterval(5 * time.Second))
func (a *AstraAdmin) CreateDatabase(ctx context.Context, name string, params CreateDatabaseParams, opts ...options.CreateDatabaseOption) (*AstraDatabaseAdmin, error) {
	// Merge options
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	if params.PCUGroupUUID != "" {
		if err := a.validatePCUGroupExists(ctx, params, merged.APIOptions); err != nil {
			return nil, err
		}
	}

	tier := params.Tier
	if tier == "" {
		tier = "serverless"
	}
	capacityUnits := params.CapacityUnits
	if capacityUnits == 0 {
		capacityUnits = 1
	}
	dbType := params.DbType
	if dbType == "" {
		dbType = "vector"
	}
	if dbType == "nonvector" {
		dbType = ""
	}

	// Build request payload
	payload := createDatabaseRequest{
		Name:          name,
		CloudProvider: params.CloudProvider,
		Region:        params.Region,
		DbType:        dbType,
		Tier:          tier,
		CapacityUnits: capacityUnits,
		PCUGroupUUID:  params.PCUGroupUUID,
	}
	if merged.Keyspace != nil {
		payload.Keyspace = *merged.Keyspace
	}

	// Execute request
	cmd := a.createCommand(http.MethodPost, "/databases", payload, nil, merged.APIOptions)
	httpResp, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
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
	awaitOpts := awaitStatusOptions{
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
	_, err = cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
	if err != nil {
		return err
	}

	if !merged.GetBlocking() {
		return nil
	}
	// Poll until database is terminated
	awaitOpts := awaitStatusOptions{
		PollInterval: merged.GetPollInterval(),
		Target:       DatabaseStatusTerminated,
		LegalStates:  []DatabaseStatus{DatabaseStatusTerminating},
		APIOptions:   merged.APIOptions,
	}

	return a.awaitStatus(ctx, databaseID, awaitOpts)
}
