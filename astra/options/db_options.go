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

import "github.com/datastax/astra-db-go/astra/internal/constants"

// ModelLifecycleStatus is the lifecycle status of an embedding provider model,
// used to filter models returned by FindEmbeddingProviders.
//
// Use ModelLifecycleStatusAll ("") to include models of every status.
type ModelLifecycleStatus string

const (
	// ModelLifecycleStatusAll includes models of every lifecycle status.
	// Equivalent to passing an empty string to the API.
	ModelLifecycleStatusAll ModelLifecycleStatus = ""
	// ModelLifecycleStatusSupported includes only actively supported models (the default).
	ModelLifecycleStatusSupported ModelLifecycleStatus = ModelLifecycleStatus(constants.ModelLifecycleStatusSupported)
	// ModelLifecycleStatusDeprecated includes only deprecated models.
	ModelLifecycleStatusDeprecated ModelLifecycleStatus = ModelLifecycleStatus(constants.ModelLifecycleStatusDeprecated)
	// ModelLifecycleStatusEndOfLife includes only end-of-life models.
	ModelLifecycleStatusEndOfLife ModelLifecycleStatus = ModelLifecycleStatus(constants.ModelLifecycleStatusEndOfLife)
)

// FindEmbeddingProvidersOptions represents options for the FindEmbeddingProviders operation.
type FindEmbeddingProvidersOptions struct {
	// FilterModelStatus filters models by their lifecycle status.
	//
	//   - If not provided: defaults to SUPPORTED models only.
	//   - If set to ModelLifecycleStatusAll (""): includes all statuses (SUPPORTED, DEPRECATED, END_OF_LIFE).
	//   - If set to a specific status: includes only models with that status.
	//
	// Example:
	//
	//	// Only supported models (default behavior)
	//	options.FindEmbeddingProviders().SetFilterModelStatus(options.ModelLifecycleStatusSupported)
	//
	//	// All models regardless of status
	//	options.FindEmbeddingProviders().SetFilterModelStatus(options.ModelLifecycleStatusAll)
	//
	//	// Only deprecated models
	//	options.FindEmbeddingProviders().SetFilterModelStatus(options.ModelLifecycleStatusDeprecated)
	FilterModelStatus *ModelLifecycleStatus `json:"filterModelStatus,omitempty"`

	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// ListCollectionsOptions represents options for listing collections in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListCollectionsOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// ListCollectionNamesOptions represents options for listing collection names in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListCollectionNamesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// ListTablesOptions represents options for listing tables in a database with full metadata.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListTablesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// ListTableNamesOptions represents options for listing table names in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListTableNamesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// DropCollectionOptions represents options for dropping a collection.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type DropCollectionOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// DropTableOptions represents options for dropping a table.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type DropTableOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}

// DropTableIndexOptions represents options for dropping a table index.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type DropTableIndexOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `optlift:"Keyspace"`
}
