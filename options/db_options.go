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

// ListCollectionsOptions represents options for listing collections in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListCollectionsOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// ListCollectionNamesOptions represents options for listing collection names in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListCollectionNamesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// ListTablesOptions represents options for listing tables in a database with full metadata.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListTablesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}

// ListTableNamesOptions represents options for listing table names in a database.
// Right now this is empty except for APIOptions, but leaving it here for future-proofing.
type ListTableNamesOptions struct {
	// APIOptions overrides API-level settings (token, timeout, headers, etc.)
	// for this command. These are merged into the Client→DB→Command hierarchy.
	APIOptions *APIOptions `json:"-"`
}
