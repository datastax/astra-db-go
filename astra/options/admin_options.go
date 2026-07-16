// Copyright IBM Corp.

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
	"time"
)

// DefaultDatabasePollInterval is the default interval for polling database status.
const DefaultDatabasePollInterval = 10 * time.Second

// DefaultKeyspacePollInterval is the default interval for polling keyspace operations.
const DefaultKeyspacePollInterval = 1 * time.Second

// FindAvailableRegionsOptions represents options for the FindAvailableRegions operation.
type FindAvailableRegionsOptions struct {
	// FilterByOrg filters by organization access. Whether to only return regions that
	// can be used by the caller's organization.
	FilterByOrg *bool

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→Admin hierarchy.
	APIOptions *APIOptions
}

// ListPCUGroupsOptions represents options for the ListPCUGroups operation.
type ListPCUGroupsOptions struct {
	// CloudProvider filters the returned PCU groups by cloud provider.
	CloudProvider *CloudProvider

	// Region filters the returned PCU groups by region.
	// If provided, CloudProvider must also be provided.
	Region *string

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→Admin hierarchy.
	APIOptions *APIOptions
}

// Validate ensures that Region is not set without CloudProvider.
func (o *ListPCUGroupsOptions) Validate() error {
	if o.Region != nil && o.CloudProvider == nil {
		return fmt.Errorf("ListPCUGroups: if Region is provided, CloudProvider must also be provided")
	}
	return nil
}

// DatabaseStatus represents the status of an Astra database.
// Also used as a filter value for ListDatabases (e.g., DatabaseStatusAll, DatabaseStatusNonTerminated).
type DatabaseStatus string

const (
	// DatabaseStatusActive indicates the database is ready for use.
	DatabaseStatusActive DatabaseStatus = "ACTIVE"
	// DatabaseStatusAssociating indicates the database is being associated to a [PCU] group.
	//
	// [PCU]: https://docs.datastax.com/en/astra-db-serverless/administration/provisioned-capacity-units.html
	DatabaseStatusAssociating DatabaseStatus = "ASSOCIATING"
	// DatabaseStatusPending indicates the database creation is pending.
	DatabaseStatusPending DatabaseStatus = "PENDING"
	// DatabaseStatusInitializing indicates the database is being initialized.
	DatabaseStatusInitializing DatabaseStatus = "INITIALIZING"
	// DatabaseStatusTerminating indicates the database is being terminated.
	DatabaseStatusTerminating DatabaseStatus = "TERMINATING"
	// DatabaseStatusTerminated indicates the database has been terminated.
	DatabaseStatusTerminated DatabaseStatus = "TERMINATED"
	// DatabaseStatusMaintenance indicates the database is under maintenance.
	DatabaseStatusMaintenance DatabaseStatus = "MAINTENANCE"
	// DatabaseStatusError indicates the database is in an error state.
	DatabaseStatusError DatabaseStatus = "ERROR"
	// DatabaseStatusParking indicates the database is being parked (hibernated).
	DatabaseStatusParking DatabaseStatus = "PARKING"
	// DatabaseStatusParked indicates the database is parked (hibernated).
	DatabaseStatusParked DatabaseStatus = "PARKED"
	// DatabaseStatusUnparking indicates the database is being unparked.
	DatabaseStatusUnparking DatabaseStatus = "UNPARKING"
	// DatabaseStatusPreparing indicates the database is preparing.
	DatabaseStatusPreparing DatabaseStatus = "PREPARING"
	// DatabaseStatusPrepared indicates the database is prepared.
	DatabaseStatusPrepared DatabaseStatus = "PREPARED"
	// DatabaseStatusResizing indicates the database is being resized.
	DatabaseStatusResizing DatabaseStatus = "RESIZING"
	// DatabaseStatusSuspended indicates the database is suspended.
	DatabaseStatusSuspended DatabaseStatus = "SUSPENDED"
	// DatabaseStatusSuspending indicates the database is being suspended.
	DatabaseStatusSuspending DatabaseStatus = "SUSPENDING"

	// DatabaseStatusNonTerminated is a special filter value for ListDatabases
	// that returns all databases that are not terminated (default).
	DatabaseStatusNonTerminated DatabaseStatus = "NONTERMINATED"
	// DatabaseStatusAll is a special filter value for ListDatabases
	// that returns all databases regardless of status.
	DatabaseStatusAll DatabaseStatus = "ALL"
)

// CloudProviderFilter controls which databases are returned by ListDatabases based on cloud provider.
type CloudProviderFilter string

const (
	// CloudProviderFilterAll returns databases from all cloud providers (default).
	CloudProviderFilterAll CloudProviderFilter = "ALL"
	// CloudProviderFilterGCP returns only GCP databases.
	CloudProviderFilterGCP CloudProviderFilter = CloudProviderFilter(CloudProviderGCP)
	// CloudProviderFilterAWS returns only AWS databases.
	CloudProviderFilterAWS CloudProviderFilter = CloudProviderFilter(CloudProviderAWS)
	// CloudProviderFilterAzure returns only Azure databases.
	CloudProviderFilterAzure CloudProviderFilter = CloudProviderFilter(CloudProviderAzure)
)

// CloudProvider represents a cloud provider hosting an Astra database.
type CloudProvider string

const (
	// CloudProviderGCP represents Google Cloud Platform.
	CloudProviderGCP CloudProvider = "GCP"
	// CloudProviderAWS represents Amazon Web Services.
	CloudProviderAWS CloudProvider = "AWS"
	// CloudProviderAzure represents Microsoft Azure.
	CloudProviderAzure CloudProvider = "AZURE"
)

// ListDatabasesOptions represents options for the ListDatabases operation.
type ListDatabasesOptions struct {
	// Include filters databases by status. Defaults to [DatabaseStatusNonTerminated].
	Include *DatabaseStatus
	// Provider filters databases by cloud provider. Defaults to [CloudProviderFilterAll].
	Provider *CloudProviderFilter
	// Limit is the maximum number of databases to return (1-100). Defaults to 25.
	Limit *int
	// StartingAfter is a database ID to use with pagination. Pass the DB ID of the
	// last item on the previous page to get the next page.
	StartingAfter *string

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→Admin hierarchy.
	APIOptions *APIOptions
}

// CreateDatabaseOptions represents options for the CreateDatabase operation.
type CreateDatabaseOptions struct {
	// Keyspace is the initial keyspace name. Defaults to "default_keyspace" if not specified.
	Keyspace *string
	// Blocking controls whether to wait for the database to become ACTIVE.
	// Defaults to true.
	Blocking *bool
	// PollInterval is how often to check the database status when blocking.
	// Defaults to DefaultDatabasePollInterval (10 seconds).
	PollInterval *time.Duration

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→Admin hierarchy.
	APIOptions *APIOptions
}

// GetBlocking returns the Blocking option or true if not set.
func (o *CreateDatabaseOptions) GetBlocking() bool {
	if o == nil || o.Blocking == nil {
		return true
	}
	return *o.Blocking
}

// GetPollInterval returns the PollInterval option or DefaultDatabasePollInterval if not set.
func (o *CreateDatabaseOptions) GetPollInterval() time.Duration {
	if o == nil || o.PollInterval == nil {
		return DefaultDatabasePollInterval
	}
	return *o.PollInterval
}

// DropDatabaseOptions represents options for the DropDatabase operation.
type DropDatabaseOptions struct {
	// Blocking controls whether to wait for the database to be fully terminated.
	// Defaults to true.
	Blocking *bool
	// PollInterval is how often to check the database status when blocking.
	// Defaults to DefaultDatabasePollInterval (10 seconds).
	PollInterval *time.Duration

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→Admin hierarchy.
	APIOptions *APIOptions
}

// GetBlocking returns the Blocking option or true if not set.
func (o *DropDatabaseOptions) GetBlocking() bool {
	if o == nil || o.Blocking == nil {
		return true
	}
	return *o.Blocking
}

// GetPollInterval returns the PollInterval option or DefaultDatabasePollInterval if not set.
func (o *DropDatabaseOptions) GetPollInterval() time.Duration {
	if o == nil || o.PollInterval == nil {
		return DefaultDatabasePollInterval
	}
	return *o.PollInterval
}

// ListKeyspacesOptions represents options for listing the keyspaces in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListKeyspacesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// CreateKeyspaceOptions represents options for the CreateKeyspace operation.
type CreateKeyspaceOptions struct {
	// Blocking controls whether to wait for the keyspace to become visible.
	// Defaults to true. Only used by the Astra (DevOps API) path.
	Blocking *bool
	// PollInterval is how often to check whether the keyspace exists when blocking.
	// Defaults to DefaultKeyspacePollInterval (1 second). Only used by the Astra (DevOps API) path.
	PollInterval *time.Duration
	// ReplicationFactor sets the replication factor for the keyspace.
	// Only used by the Data API path (non-Astra environments).
	ReplicationFactor *int
	// UpdateDbKeyspace controls whether to update the parent Db instance to use the
	// new keyspace after creation. Defaults to false.
	UpdateDbKeyspace *bool
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions
}

// GetBlocking returns the Blocking option or true if not set.
func (o *CreateKeyspaceOptions) GetBlocking() bool {
	if o == nil || o.Blocking == nil {
		return true
	}
	return *o.Blocking
}

// GetPollInterval returns the PollInterval option or DefaultKeyspacePollInterval if not set.
func (o *CreateKeyspaceOptions) GetPollInterval() time.Duration {
	if o == nil || o.PollInterval == nil {
		return DefaultKeyspacePollInterval
	}
	return *o.PollInterval
}

// DropKeyspaceOptions represents options for the DropKeyspace operation.
type DropKeyspaceOptions struct {
	// Blocking controls whether to wait for the keyspace to be fully terminated.
	// Defaults to true.
	Blocking *bool
	// PollInterval is how often to check the keyspace status when blocking.
	// Defaults to DefaultKeyspacePollInterval (1 second).
	PollInterval *time.Duration
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Table→Command hierarchy.
	APIOptions *APIOptions
}

// GetBlocking returns the Blocking option or true if not set.
func (o *DropKeyspaceOptions) GetBlocking() bool {
	if o == nil || o.Blocking == nil {
		return true
	}
	return *o.Blocking
}

// GetPollInterval returns the PollInterval option or DefaultKeyspacePollInterval if not set.
func (o *DropKeyspaceOptions) GetPollInterval() time.Duration {
	if o == nil || o.PollInterval == nil {
		return DefaultKeyspacePollInterval
	}
	return *o.PollInterval
}

// DatabaseInfoOptions represents options for the DatabaseInfo (AstraAdmin) and Info (Db/AstraDatabaseAdmin) operations.
type DatabaseInfoOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→Admin hierarchy.
	APIOptions *APIOptions
}
