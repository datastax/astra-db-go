// Copyright DataStax, Inc.

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
	"time"

	"github.com/datastax/astra-db-go/ptr"
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
	// CloudProviderAll returns databases from all cloud providers (default).
	CloudProviderAll CloudProviderFilter = "ALL"
	// CloudProviderGCP returns only GCP databases.
	CloudProviderGCP CloudProviderFilter = "GCP"
	// CloudProviderAWS returns only AWS databases.
	CloudProviderAWS CloudProviderFilter = "AWS"
	// CloudProviderAzure returns only Azure databases.
	CloudProviderAzure CloudProviderFilter = "AZURE"
)

// ListDatabasesOptions represents options for the ListDatabases operation.
type ListDatabasesOptions struct {
	// Include filters databases by status. Defaults to [DatabaseStatusNonTerminated].
	Include *DatabaseStatus
	// Provider filters databases by cloud provider. Defaults to [CloudProviderAll].
	Provider *CloudProviderFilter
	// Limit is the maximum number of databases to return (1-100). Defaults to 25.
	Limit *int
	// StartingAfter is a database ID to use with pagination. Pass the DB ID of the
	// last item on the previous page to get the next page.
	StartingAfter *string
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
}

// SetDefaults implements the Defaulter interface for CreateDatabaseOptions.
func (o *CreateDatabaseOptions) SetDefaults() {
	o.Blocking = ptr.To(true)
	o.PollInterval = ptr.To(DefaultDatabasePollInterval)
}

// DropDatabaseOptions represents options for the DropDatabase operation.
type DropDatabaseOptions struct {
	// Blocking controls whether to wait for the database to be fully terminated.
	// Defaults to true.
	Blocking *bool
	// PollInterval is how often to check the database status when blocking.
	// Defaults to DefaultDatabasePollInterval (10 seconds).
	PollInterval *time.Duration
}

// SetDefaults implements the Defaulter interface for DropDatabaseOptions.
func (o *DropDatabaseOptions) SetDefaults() {
	o.Blocking = ptr.To(true)
	o.PollInterval = ptr.To(DefaultDatabasePollInterval)
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
}

// SetDefaults implements the Defaulter interface for CreateKeyspaceOptions.
func (o *CreateKeyspaceOptions) SetDefaults() {
	o.Blocking = ptr.To(true)
	o.PollInterval = ptr.To(DefaultKeyspacePollInterval)
}

// DropKeyspaceOptions represents options for the DropKeyspace operation.
type DropKeyspaceOptions struct {
	// Blocking controls whether to wait for the keyspace to be fully terminated.
	// Defaults to true.
	Blocking *bool
	// PollInterval is how often to check the keyspace status when blocking.
	// Defaults to DefaultKeyspacePollInterval (1 second).
	PollInterval *time.Duration
}

// SetDefaults implements the Defaulter interface for DropKeyspaceOptions.
func (o *DropKeyspaceOptions) SetDefaults() {
	o.Blocking = ptr.To(true)
	o.PollInterval = ptr.To(DefaultKeyspacePollInterval)
}
