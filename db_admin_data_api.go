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
	"context"
	"encoding/json"
	"fmt"

	"github.com/datastax/astra-db-go/options"
)

// DataAPIDatabaseAdmin provides keyspace operations via the Data API.
// Used for non-Astra environments (HCD, DSE, Cassandra).
type DataAPIDatabaseAdmin struct {
	db *Db
}

// ListKeyspaces returns the keyspace names for this database via the Data API.
func (d *DataAPIDatabaseAdmin) ListKeyspaces(ctx context.Context) ([]string, error) {
	cmd := newDatabaseAdminCmd(d.db, "findKeyspaces", struct{}{})
	body, _, err := cmd.Execute(ctx)
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

	cmd := newDatabaseAdminCmd(d.db, "createKeyspace", payload)
	_, _, err = cmd.Execute(ctx)
	return err
}

// DropKeyspace drops a keyspace via the Data API.
func (d *DataAPIDatabaseAdmin) DropKeyspace(ctx context.Context, keyspace string, opts ...options.DropKeyspaceOption) error {
	payload := struct {
		Name string `json:"name"`
	}{Name: keyspace}
	cmd := newDatabaseAdminCmd(d.db, "dropKeyspace", payload)
	_, _, err := cmd.Execute(ctx)
	return err
}
