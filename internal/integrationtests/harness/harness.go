// Package harness defines a test harness for running integration tests.
package harness

import (
	"log/slog"
	"sync"

	"github.com/DeanPDX/dotconfig"
	astradb "github.com/datastax/astra-db-go"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/results"
)

// TestEnv represents our test environment.
type TestEnv struct {
	APIEndpoint      string `env:"API_ENDPOINT"`
	ApplicationToken string `env:"APPLICATION_TOKEN"`
	TestPrefix       string `env:"TEST_PREFIX"`
	Backend          string `env:"BACKEND"`
}

// Environment() retrieves a test environment with config based on environment variables.
func Environment() TestEnv {
	c, err := dotconfig.FromFileName[TestEnv](".env")
	if err != nil {
		slog.Error("dotconfig.FromFileName failed", "error", err)
	}

	if c.Backend == "" {
		c.Backend = "astra"
	}

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

func (e *TestEnv) DefaultClient() *astradb.DataAPIClient {
	return astradb.NewClient(
		options.WithToken(e.ApplicationToken),
		options.WithDataAPIBackend(options.DataAPIBackend(e.Backend)),
	)
}

// DefaultDb returns a Db handle configured with the test environment settings.
func (e *TestEnv) DefaultDb() *astradb.Db {
	client := astradb.NewClient(
		options.WithToken(e.ApplicationToken),
		options.WithDataAPIBackend(options.DataAPIBackend(e.Backend)),
		options.WithWarningHandler(func(w results.Warning) {
			// Add client handler just to make sure it is properly superseded by
			// DB level handler.
			slog.Error("Client handler called and should have been superseded by DB handler")
		}),
	)
	return client.Database(e.APIEndpoint, options.WithWarningHandler(func(w results.Warning) {
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
