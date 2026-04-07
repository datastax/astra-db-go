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

package astradb

import (
	"context"

	"github.com/datastax/astra-db-go/options"
)

// DatabaseAdmin provides keyspace management operations for a database.
// The concrete implementation depends on the environment:
//   - Astra environments use [AstraDbAdmin] (DevOps API)
//   - Non-Astra environments use [DataAPIDatabaseAdmin] (Data API)
type DatabaseAdmin interface {
	ListKeyspaces(ctx context.Context) ([]string, error)
	CreateKeyspace(ctx context.Context, keyspace string, opts ...options.CreateKeyspaceOption) error
	DropKeyspace(ctx context.Context, keyspace string, opts ...options.DropKeyspaceOption) error
}

// Compile-time interface checks
var _ DatabaseAdmin = (*AstraDbAdmin)(nil)
var _ DatabaseAdmin = (*DataAPIDatabaseAdmin)(nil)
