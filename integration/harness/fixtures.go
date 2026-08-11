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

package harness

import (
	"context"
	"net/http"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/options"
)

var Ctx = context.Background()

var TestKeyspaces = []string{"default_keyspace", "other_keyspace"}

type TestObjects struct {
	Client      *astra.DataAPIClient
	Db          *astra.Db
	Collection  *astra.Collection
	Collection_ *astra.Collection
	DbAdmin     astra.DatabaseAdmin
	Table       *astra.Table
	Table_      *astra.Table
	Admin       *astra.AstraAdmin
}

var GlobalFixtures *TestObjects

func NewTestObjects() *TestObjects {
	return newTestObjects(false)
}

func NewMemoizedTestObjects() *TestObjects {
	return newTestObjects(true)
}

func newTestObjects(memoize bool) *TestObjects {
	var httpClient *http.Client
	if memoize {
		cachingTransport := newCachingRoundTripper()
		httpClient = &http.Client{Transport: cachingTransport}
	}

	client := astra.NewClient(
		options.API().
			SetToken(ApplicationToken()).
			SetEnvironment(Backend()).
			SetHTTPClient(httpClient).
			SetRequestTimeout(90 * time.Second).
			SetDatabaseAdminTimeout(90 * time.Second).
			SetCollectionAdminTimeout(90 * time.Second).
			SetTableAdminTimeout(90 * time.Second).
			SetKeyspaceAdminTimeout(90 * time.Second).
			SetGeneralMethodTimeout(90 * time.Second),
	)
	db := client.Database(APIEndpoint())

	dbAdmin, err := db.DatabaseAdmin()
	if err != nil {
		panic("Failed to get DatabaseAdmin: " + err.Error())
	}

	admin, _ := client.Admin()

	return &TestObjects{
		Client:      client,
		Db:          db,
		Collection:  db.Collection(DefaultCollectionName, options.GetCollection().SetEmbeddingAPIKey(EmbeddingApiKey())),
		Collection_: db.Collection(DefaultCollectionName, options.GetCollection().SetEmbeddingAPIKey(EmbeddingApiKey()).SetKeyspace("other_keyspace")),
		DbAdmin:     dbAdmin,
		Table:       db.Table(DefaultTableName, options.GetTable().SetEmbeddingAPIKey(EmbeddingApiKey())),
		Table_:      db.Table(DefaultTableName, options.GetTable().SetEmbeddingAPIKey(EmbeddingApiKey()).SetKeyspace("other_keyspace")),
		Admin:       admin,
	}
}
