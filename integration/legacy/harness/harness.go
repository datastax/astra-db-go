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

// Package harness defines a test harness for running integration tests.
package harness

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/DeanPDX/dotconfig"
	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
)

// TestEnv represents our test environment.
type TestEnv struct {
	// Our application API endpoint. Can be left blank if using non-astra backend.
	APIEndpoint string `env:"API_ENDPOINT,optional"`
	// The token to use for authentication. E.g. "AstraCS:myToken"
	ApplicationToken string `env:"APPLICATION_TOKEN,optional"`
	// If set, will only run tests with names that start with prefix.
	TestPrefix string `env:"TEST_PREFIX,optional"`
	// The backend we are running this against. "astra", "hcd", etc. Defaults to "astra".
	// See: https://pkg.go.dev/github.com/datastax/astra-db-go/v2/astra/options#Environment
	Backend string `env:"BACKEND" default:"astra"`
}

// Environment() retrieves a test environment with config based on environment variables.
func Environment() TestEnv {
	c, err := dotconfig.FromFileName[TestEnv](".env")
	if err != nil {
		// Should be impossible; But in the case of catastrophic failure...
		fmt.Println("Problem with config:", err)
		os.Exit(1)
	}
	// For non-astra DBs, default to the local HCD/DSE values.
	if c.Backend != "astra" {
		if c.APIEndpoint == "" {
			c.APIEndpoint = "http://127.0.0.1:8181"
		}
		if c.ApplicationToken == "" {
			c.ApplicationToken = "Cassandra:Y2Fzc2FuZHJh:Y2Fzc2FuZHJh"
		}
	}
	return c
}

func (e *TestEnv) DefaultClient() *astra.DataAPIClient {
	return astra.NewClient(
		options.API().
			SetToken(e.ApplicationToken).
			SetEnvironment(options.Environment(e.Backend)),
	)
}

// DefaultDb returns a Db handle configured with the test environment settings.
func (e *TestEnv) DefaultDb() *astra.Db {
	client := astra.NewClient(
		options.API().
			SetToken(e.ApplicationToken).
			SetEnvironment(options.Environment(e.Backend)).
			SetWarningHandler(func(w results.Warning) {
				// Add client handler just to make sure it is properly superseded by
				// DB level handler.
				slog.Error("Client handler called and should have been superseded by DB handler")
			}),
	)
	return client.Database(e.APIEndpoint, options.API().SetWarningHandler(func(w results.Warning) {
		// Warn and let logs know this came from DB handler. In our tests we will
		// make sure that collection/table/command level handlers supersede this.
		slog.Warn("API warning from DB handler", "code", w.ErrorCode, "message", w.Message)
	}))
}

// An integration test
type IntegrationTest struct {
	Name string
	Run  func(e *TestEnv) error
}

var (
	testsMu sync.RWMutex // Guards `tests`.
	tests   = make([]IntegrationTest, 0)
)

// Register adds a test(s) to our test runner. The approach is similar to
// how [database/sql] allows you to `Register` SQL drivers.
//
// [database/sql]: https://cs.opensource.google/go/go/+/refs/tags/go1.25.4:src/database/sql/sql.go;l=36
func Register(args ...IntegrationTest) {
	testsMu.Lock()
	defer testsMu.Unlock()
	tests = append(tests, args...)
}

// Tests returns all registered tests.
func Tests() []IntegrationTest {
	// By the time we are running the tests, nothing is adding to this map.
	// But - creating a copy and guarding with `testsMu` just to be extra safe
	// because it's not guaranteed future developers will adhere to this.
	testsMu.Lock()
	defer testsMu.Unlock()
	t := tests
	return t
}
