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
	// can be used by the caller’s organization.
	FilterByOrg *bool
}

// Validate implements the Validator interface for FindAvailableRegionsOptions.
func (o FindAvailableRegionsOptions) Validate() error {
	// No required fields, always valid
	return nil
}

// List implements Builder[FindAvailableRegionsOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[FindAvailableRegionsOptions].
func (o *FindAvailableRegionsOptions) List() []func(*FindAvailableRegionsOptions) {
	return NoopBuilder(o)
}

// FindAvailableRegionsOptionsBuilder is a builder for FindAvailableRegionsOptions that implements
// Builder[FindAvailableRegionsOptions] following the MongoDB Go driver pattern.
type FindAvailableRegionsOptionsBuilder struct {
	Opts []func(*FindAvailableRegionsOptions)
}

// FindAvailableRegions creates a new FindAvailableRegionsOptionsBuilder.
func FindAvailableRegions() *FindAvailableRegionsOptionsBuilder {
	return &FindAvailableRegionsOptionsBuilder{}
}

// List implements Builder[FindAvailableRegionsOptions].
func (b *FindAvailableRegionsOptionsBuilder) List() []func(*FindAvailableRegionsOptions) {
	return b.Opts
}

// SetFilterByOrg sets the filter-by-org query parameter.
// Valid values: FilterByOrgEnabled, FilterByOrgDisabled, or empty string.
func (b *FindAvailableRegionsOptionsBuilder) SetFilterByOrg(v bool) *FindAvailableRegionsOptionsBuilder {
	b.Opts = append(b.Opts, func(o *FindAvailableRegionsOptions) {
		o.FilterByOrg = &v
	})
	return b
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
	// Include filters databases by status. Defaults to DatabaseStatusNonTerminated.
	Include *DatabaseStatus
	// Provider filters databases by cloud provider. Defaults to "ALL".
	Provider *CloudProviderFilter
	// Limit is the maximum number of databases to return (1-100). Defaults to 25.
	Limit *int
	// StartingAfter is a database ID to use with pagination. Pass the DB ID of the
	// last item on the previous page to get the next page.
	StartingAfter *string
}

// Validate implements the Validator interface for ListDatabasesOptions.
func (o ListDatabasesOptions) Validate() error {
	return nil
}

// List implements Builder[ListDatabasesOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[ListDatabasesOptions].
func (o *ListDatabasesOptions) List() []func(*ListDatabasesOptions) {
	return NoopBuilder(o)
}

// ListDatabasesOptionsBuilder is a builder for ListDatabasesOptions.
type ListDatabasesOptionsBuilder struct {
	Opts []func(*ListDatabasesOptions)
}

// ListDatabases creates a new ListDatabasesOptionsBuilder.
func ListDatabases() *ListDatabasesOptionsBuilder {
	return &ListDatabasesOptionsBuilder{}
}

// List implements Builder[ListDatabasesOptions].
func (b *ListDatabasesOptionsBuilder) List() []func(*ListDatabasesOptions) {
	return b.Opts
}

// SetInclude filters databases by status.
func (b *ListDatabasesOptionsBuilder) SetInclude(v DatabaseStatus) *ListDatabasesOptionsBuilder {
	b.Opts = append(b.Opts, func(o *ListDatabasesOptions) {
		o.Include = &v
	})
	return b
}

// SetProvider filters databases by cloud provider.
func (b *ListDatabasesOptionsBuilder) SetProvider(v CloudProviderFilter) *ListDatabasesOptionsBuilder {
	b.Opts = append(b.Opts, func(o *ListDatabasesOptions) {
		o.Provider = &v
	})
	return b
}

// SetLimit sets the maximum number of databases to return (1-100).
func (b *ListDatabasesOptionsBuilder) SetLimit(v int) *ListDatabasesOptionsBuilder {
	b.Opts = append(b.Opts, func(o *ListDatabasesOptions) {
		o.Limit = &v
	})
	return b
}

// SetStartingAfter sets the pagination cursor. Results will start after this database ID.
func (b *ListDatabasesOptionsBuilder) SetStartingAfter(v string) *ListDatabasesOptionsBuilder {
	b.Opts = append(b.Opts, func(o *ListDatabasesOptions) {
		o.StartingAfter = &v
	})
	return b
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

// Validate implements the Validator interface for CreateDatabaseOptions.
func (o CreateDatabaseOptions) Validate() error {
	return nil
}

// List implements Builder[CreateDatabaseOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[CreateDatabaseOptions].
func (o *CreateDatabaseOptions) List() []func(*CreateDatabaseOptions) {
	return NoopBuilder(o)
}

// CreateDatabaseOptionsBuilder is a builder for CreateDatabaseOptions.
type CreateDatabaseOptionsBuilder struct {
	Opts []func(*CreateDatabaseOptions)
}

// CreateDatabase creates a new CreateDatabaseOptionsBuilder.
func CreateDatabase() *CreateDatabaseOptionsBuilder {
	return &CreateDatabaseOptionsBuilder{}
}

// List implements Builder[CreateDatabaseOptions].
func (b *CreateDatabaseOptionsBuilder) List() []func(*CreateDatabaseOptions) {
	return b.Opts
}

// SetKeyspace sets the initial keyspace name for the database.
func (b *CreateDatabaseOptionsBuilder) SetKeyspace(v string) *CreateDatabaseOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateDatabaseOptions) {
		o.Keyspace = &v
	})
	return b
}

// SetBlocking controls whether to wait for the database to become ACTIVE.
// Defaults to true if not specified.
func (b *CreateDatabaseOptionsBuilder) SetBlocking(v bool) *CreateDatabaseOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateDatabaseOptions) {
		o.Blocking = &v
	})
	return b
}

// SetPollInterval sets how often to check the database status when blocking.
func (b *CreateDatabaseOptionsBuilder) SetPollInterval(v time.Duration) *CreateDatabaseOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateDatabaseOptions) {
		o.PollInterval = &v
	})
	return b
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

// Validate implements the Validator interface for DropDatabaseOptions.
func (o DropDatabaseOptions) Validate() error {
	return nil
}

// List implements Builder[DropDatabaseOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[DropDatabaseOptions].
func (o *DropDatabaseOptions) List() []func(*DropDatabaseOptions) {
	return NoopBuilder(o)
}

// DropDatabaseOptionsBuilder is a builder for DropDatabaseOptions.
type DropDatabaseOptionsBuilder struct {
	Opts []func(*DropDatabaseOptions)
}

// DropDatabase creates a new DropDatabaseOptionsBuilder.
func DropDatabase() *DropDatabaseOptionsBuilder {
	return &DropDatabaseOptionsBuilder{}
}

// List implements Builder[DropDatabaseOptions].
func (b *DropDatabaseOptionsBuilder) List() []func(*DropDatabaseOptions) {
	return b.Opts
}

// SetBlocking controls whether to wait for the database to be fully terminated.
// Defaults to true if not specified.
func (b *DropDatabaseOptionsBuilder) SetBlocking(v bool) *DropDatabaseOptionsBuilder {
	b.Opts = append(b.Opts, func(o *DropDatabaseOptions) {
		o.Blocking = &v
	})
	return b
}

// SetPollInterval sets how often to check the database status when blocking.
func (b *DropDatabaseOptionsBuilder) SetPollInterval(v time.Duration) *DropDatabaseOptionsBuilder {
	b.Opts = append(b.Opts, func(o *DropDatabaseOptions) {
		o.PollInterval = &v
	})
	return b
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

// Validate implements the Validator interface for CreateKeyspaceOptions.
func (o CreateKeyspaceOptions) Validate() error {
	return nil
}

// List implements Builder[CreateKeyspaceOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[CreateKeyspaceOptions].
func (o *CreateKeyspaceOptions) List() []func(*CreateKeyspaceOptions) {
	return NoopBuilder(o)
}

// CreateKeyspaceOptionsBuilder is a builder for CreateKeyspaceOptions.
type CreateKeyspaceOptionsBuilder struct {
	Opts []func(*CreateKeyspaceOptions)
}

// CreateKeyspace creates a new CreateKeyspaceOptionsBuilder.
func CreateKeyspace() *CreateKeyspaceOptionsBuilder {
	return &CreateKeyspaceOptionsBuilder{}
}

// List implements Builder[CreateKeyspaceOptions].
func (b *CreateKeyspaceOptionsBuilder) List() []func(*CreateKeyspaceOptions) {
	return b.Opts
}

// SetBlocking controls whether to wait for the keyspace to become visible.
// Defaults to true if not specified.
func (b *CreateKeyspaceOptionsBuilder) SetBlocking(v bool) *CreateKeyspaceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateKeyspaceOptions) {
		o.Blocking = &v
	})
	return b
}

// SetPollInterval sets how often to check whether the keyspace exists when blocking.
func (b *CreateKeyspaceOptionsBuilder) SetPollInterval(v time.Duration) *CreateKeyspaceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateKeyspaceOptions) {
		o.PollInterval = &v
	})
	return b
}

// SetReplicationFactor sets the replication factor for the keyspace.
// Only used by the Data API path (non-Astra environments).
func (b *CreateKeyspaceOptionsBuilder) SetReplicationFactor(v int) *CreateKeyspaceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *CreateKeyspaceOptions) {
		o.ReplicationFactor = &v
	})
	return b
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

// Validate implements the Validator interface for DropKeyspaceOptions.
func (o DropKeyspaceOptions) Validate() error {
	return nil
}

// List implements Builder[DropKeyspaceOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[DropKeyspaceOptions].
func (o *DropKeyspaceOptions) List() []func(*DropKeyspaceOptions) {
	return NoopBuilder(o)
}

// DropKeyspaceOptionsBuilder is a builder for DropKeyspaceOptions.
type DropKeyspaceOptionsBuilder struct {
	Opts []func(*DropKeyspaceOptions)
}

// DropKeyspace creates a new DropKeyspaceOptionsBuilder.
func DropKeyspace() *DropKeyspaceOptionsBuilder {
	return &DropKeyspaceOptionsBuilder{}
}

// List implements Builder[DropKeyspaceOptions].
func (b *DropKeyspaceOptionsBuilder) List() []func(*DropKeyspaceOptions) {
	return b.Opts
}

// SetBlocking controls whether to wait for the keyspace to be fully terminated.
// Defaults to true if not specified.
func (b *DropKeyspaceOptionsBuilder) SetBlocking(v bool) *DropKeyspaceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *DropKeyspaceOptions) {
		o.Blocking = &v
	})
	return b
}

// SetPollInterval sets how often to check the keyspace status when blocking.
func (b *DropKeyspaceOptionsBuilder) SetPollInterval(v time.Duration) *DropKeyspaceOptionsBuilder {
	b.Opts = append(b.Opts, func(o *DropKeyspaceOptions) {
		o.PollInterval = &v
	})
	return b
}
