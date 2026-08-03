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

	"github.com/datastax/astra-db-go/v2/astra/internal/timeout"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
)

// DatabaseAdmin provides keyspace management operations for a database.
// The concrete implementation depends on the environment:
//   - Astra environments use [AstraDatabaseAdmin] (DevOps API)
//   - Non-Astra environments use [DataAPIDatabaseAdmin] (Data API)
type DatabaseAdmin interface {
	ListKeyspaces(ctx context.Context, opts ...options.ListKeyspacesOption) ([]string, error)
	CreateKeyspace(ctx context.Context, keyspace string, opts ...options.CreateKeyspaceOption) error
	DropKeyspace(ctx context.Context, keyspace string, opts ...options.DropKeyspaceOption) error
	FindEmbeddingProviders(ctx context.Context, opts ...options.FindEmbeddingProvidersOption) (*results.FindEmbeddingProvidersResult, error)
	FindRerankingProviders(ctx context.Context, opts ...options.FindRerankingProvidersOption) (*results.FindRerankingProvidersResult, error)
}

// Compile-time interface checks
var _ DatabaseAdmin = (*AstraDatabaseAdmin)(nil)
var _ DatabaseAdmin = (*DataAPIDatabaseAdmin)(nil)

func findEmbeddingProviders(db *Db, ctx context.Context, opts ...options.FindEmbeddingProvidersOption) (*results.FindEmbeddingProvidersResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	payload := map[string]any{
		"options": merged,
	}

	cmd := db.newAdminCmd("findEmbeddingProviders", payload, merged.APIOptions)
	b, _, _, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status results.FindEmbeddingProvidersResult `json:"status"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse findEmbeddingProviders response: %w", err)
	}

	return &resp.Status, nil
}

func findRerankingProviders(db *Db, ctx context.Context, opts ...options.FindRerankingProvidersOption) (*results.FindRerankingProvidersResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	payload := map[string]any{
		"options": merged,
	}

	cmd := db.newAdminCmd("findRerankingProviders", payload, merged.APIOptions)
	b, _, _, err := cmd.ExecuteSingle(ctx, timeout.DatabaseAdmin)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status results.FindRerankingProvidersResult `json:"status"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse findRerankingProviders response: %w", err)
	}

	return &resp.Status, nil
}