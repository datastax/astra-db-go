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

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/options"
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
	client := astra.NewClient(
		options.API().
			SetToken(ApplicationToken()).
			SetDataAPIBackend(Backend()), // need to check if we need any more options here
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
		Collection:  db.Collection(DefaultCollectionName),
		Collection_: db.Collection(DefaultCollectionName, options.API().SetKeyspace("slania")),
		DbAdmin:     dbAdmin,
		Table:       db.Table(DefaultTableName),
		Table_:      db.Table(DefaultTableName, options.API().SetKeyspace("slania")),
		Admin:       admin,
	}
}
