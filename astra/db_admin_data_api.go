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

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/astra/results"
)

// DataAPIDatabaseAdmin provides keyspace operations via the Data API.
// Used for non-Astra environments (HCD, DSE, Cassandra).
type DataAPIDatabaseAdmin struct {
	db *Db
}

// ListKeyspaces returns the keyspace names for this database via the Data API.
func (d *DataAPIDatabaseAdmin) ListKeyspaces(ctx context.Context, opts ...options.ListKeyspacesOption) ([]string, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	cmd := d.db.newAdminCmdWithMergedOptions("findKeyspaces", struct{}{}, merged.APIOptions)
	body, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status struct {
			Keyspaces []string `json:"keyspaces"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse findKeyspaces response: %w", err)
	}

	return resp.Status.Keyspaces, nil
}

// CreateKeyspace creates a new keyspace via the Data API.
func (d *DataAPIDatabaseAdmin) CreateKeyspace(ctx context.Context, keyspace string, opts ...options.CreateKeyspaceOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	if ptr.From(merged.UpdateDbKeyspace) {
		d.db.UseKeyspace(keyspace)
	}

	type replicationOptions struct {
		Class             string `json:"class"`
		ReplicationFactor int    `json:"replication_factor"`
	}
	type createKeyspacePayloadOptions struct {
		Replication replicationOptions `json:"replication"`
	}
	type createKeyspacePayload struct {
		Name    string                        `json:"name"`
		Options *createKeyspacePayloadOptions `json:"options,omitempty"`
	}

	payload := createKeyspacePayload{Name: keyspace}
	if merged.ReplicationFactor != nil {
		payload.Options = &createKeyspacePayloadOptions{
			Replication: replicationOptions{
				Class:             "SimpleStrategy",
				ReplicationFactor: *merged.ReplicationFactor,
			},
		}
	}

	cmd := d.db.newAdminCmdWithMergedOptions("createKeyspace", payload, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// DropKeyspace drops a keyspace via the Data API.
func (d *DataAPIDatabaseAdmin) DropKeyspace(ctx context.Context, keyspace string, opts ...options.DropKeyspaceOption) error {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"name": keyspace,
	}
	cmd := d.db.newAdminCmdWithMergedOptions("dropKeyspace", payload, merged.APIOptions)
	_, _, _, err = cmd.Execute(ctx)
	return err
}

// FindEmbeddingProviders returns detailed information about the availability and usage of the
// vectorize embedding providers available on the current database (may vary based on cloud
// provider & region).
//
// Example:
//
//	result, err := dbAdmin.FindEmbeddingProviders(ctx)
//	if err != nil {
//	    return err
//	}
//	// ["text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"]
//	for _, model := range result.EmbeddingProviders["openai"].Models {
//	    fmt.Println(model.Name)
//	}
//
// Note: Warnings are accessible via the WarningHandler option callback only.
func (d *DataAPIDatabaseAdmin) FindEmbeddingProviders(ctx context.Context, opts ...options.FindEmbeddingProvidersOption) (*results.FindEmbeddingProvidersResult, error) {
	return findEmbeddingProviders(d.db, ctx, opts...)
}
