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

// Package astradb implements the astra database client.
package astradb

import (
	"context"
	"fmt"
	"net/url"

	"github.com/datastax/astra-db-go/options"
)

// Db represents a connection to a specific Astra DB database.
//
// Options set on the database are inherited by all collections, tables,
// and commands created from it, unless overridden at a lower level.
type Db struct {
	endpoint string
	client   *DataAPIClient
	options  *options.APIOptions
}

func (d *Db) newCmd(name string, payload any) command {
	return newCmd(d, name, payload)
}

// Endpoint returns the database API endpoint.
func (d *Db) Endpoint() string {
	return d.endpoint
}

// Options returns the database's options (or empty options if nil).
func (d *Db) Options() *options.APIOptions {
	if d.options == nil {
		return &options.APIOptions{}
	}
	return d.options
}

// Client returns the parent DataAPIClient.
func (d *Db) Client() *DataAPIClient {
	return d.client
}

// Collection returns a handle for the named collection.
//
// Options set here override those set on the database.
//
// Example:
//
//	coll := db.Collection("my_collection",
//	    options.WithTimeout(60 * time.Second),
//	)
func (d *Db) Collection(name string, opts ...options.APIOption) *Collection {
	return &Collection{
		db:      d,
		name:    name,
		options: options.NewAPIOptions(opts...),
	}
}

// CreateCollection creates a collection in the database.
//
// Options can be passed using the builder pattern or as raw structs:
//
//	// No options (simple collection)
//	coll, err := db.CreateCollection(ctx, "my_collection")
//
//	// With vector options
//	coll, err := db.CreateCollection(ctx, "my_collection",
//	    options.CreateCollection().SetVector(&options.VectorOptions{
//	        Dimension: 1024,
//	        Metric:    "cosine",
//	    })
//	)
//
//	// Passing in options as raw struct
//	opts := &options.CreateCollectionOptions{
//		DefaultId: &options.CollectionDefaultIdOptions{
//			Type: options.DefaultIdTypeUUIDv7,
//		},
//	}
//	coll, err := db.CreateCollection(ctx, "my_collection", options.WithCreateCollectionOptions(opts))
//
// Note: Warnings are accessible via the WarningHandler option callback only.
func (d *Db) CreateCollection(ctx context.Context, name string, opts ...options.Builder[options.CreateCollectionOptions]) (*Collection, error) {
	cmd, err := createCollectionCommand(d, name, opts...)
	if err != nil {
		return nil, err
	}
	_, _, err = cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return &Collection{
		db:   d,
		name: name,
	}, nil
}

// createCollectionPayload is the payload for the createCollection command
type createCollectionPayload struct {
	Name    string                           `json:"name"`
	Options *options.CreateCollectionOptions `json:"options,omitempty"`
}

// createCollectionCommand builds the createCollection command for the database
func createCollectionCommand(d *Db, name string, opts ...options.Builder[options.CreateCollectionOptions]) (command, error) {
	payload := createCollectionPayload{
		Name: name,
	}

	merged, err := options.MergeOptions(opts...)
	if err != nil {
		return command{}, err
	}

	// Only set Options if at least one field is non-nil (prevents "options":{} in JSON)
	if merged != nil && (merged.DefaultId != nil || merged.Vector != nil || merged.Indexing != nil || merged.Lexical != nil || merged.Rerank != nil) {
		payload.Options = merged
	}

	return d.newCmd("createCollection", payload), nil
}

// DropCollection drops a collection from the database.
// Note: Warnings are accessible via the WarningHandler option callback only.
func (d *Db) DropCollection(ctx context.Context, name string) error {
	payload := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}
	cmd := newCmd(d, "deleteCollection", payload)
	_, _, err := cmd.Execute(ctx)
	return err
}

// DatabaseAdmin returns a DatabaseAdmin for managing keyspaces on this database.
// The concrete implementation depends on the environment:
//   - Astra environments return an [AstraDbAdmin] (DevOps API)
//   - Non-Astra environments return a [DataAPIDatabaseAdmin] (Data API)
func (d *Db) DatabaseAdmin() (DatabaseAdmin, error) {
	// Astra backends use the DevOps API.
	if d.client.dataAPIBackend.IsAstra() {
		env := options.ParseAstraEnvironmentFromEndpoint(d.endpoint)
		admin, err := d.client.Admin(options.WithAstraEnvironment(env))
		if err != nil {
			return nil, err
		}
		id, err := parseAstraEndpoint(d.endpoint)
		if err != nil {
			return nil, fmt.Errorf("cannot parse database ID from endpoint %q: %w", d.endpoint, err)
		}
		return admin.DbAdmin(id), nil
	}
	// Non-Astra backends use the Data API.
	return &DataAPIDatabaseAdmin{db: d}, nil
}

// parseAstraEndpoint extracts the database UUID from an Astra Data API endpoint.
// The expected format is: https://{uuid}-{region}.apps.astra.datastax.com
// The UUID is always 36 characters (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func parseAstraEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint URL: %w", err)
	}
	host := u.Hostname()
	if len(host) < 36 {
		return "", fmt.Errorf("hostname too short to contain a UUID: %s", host)
	}
	return host[:36], nil
}
