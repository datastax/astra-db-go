# Integration Tests

## Running Tests Against Astra DB

To get started with these integration tests, create an instance on [astra.datastax.com](https://astra.datastax.com/). Then create an Application Token with appropriate permissions. Next, set the following environment variables:

- `API_ENDPOINT` - the endpoint for the instance you created.
- `APPLICATION_TOKEN` - the token you just created.
- `BACKEND` - set to `"astra"` (or leave unset, as this is the default).

## Running Tests Against HCD/DSE

To run tests against a local HCD or DSE instance, start your HCD/DSE instance (e.g., using Docker). Then, set the following environment variables:

- `API_ENDPOINT` - the endpoint for your local instance (defaults to `http://127.0.0.1:8181` if not set)
- `APPLICATION_TOKEN` - authentication token (defaults to `Cassandra:Y2Fzc2FuZHJh:Y2Fzc2FuZHJh` which is `cassandra:cassandra` in base64)
- `BACKEND` - set to `"hcd"`, `"dse"`, `"cassandra"`, or `"other"`

**Note:** If `BACKEND` is set to a non-Astra value and `API_ENDPOINT` or `APPLICATION_TOKEN` are not provided, they will automatically default to the local development values shown above.

## Configuration

You can also use a `.env` file. Use [.env.example](./.env.example) as a template. Note the `TEST_PREFIX` property in there and what it does. For this reason, prefix test names with their domain/area:

```go
// Bad. There's no prefix we can use to run all collection-related integration tests.
func CreateCollection(e *harness.TestEnv) error { }
// Good. Setting TEST_PREFIX to "Collection" will match this test.
func CollectionCreate(e *harness.TestEnv) error { }
```

To run the tests:

```bash
# Will run these against your instance and print logs to stdout as well as
# a log file in ./logs.
go run github.com/datastax/astra-db-go/internal/integrationtests
```

Note that the files in [./tests](./tests) end with `_tests.go`, not `_test.go` because they aren't actually unit tests and we don't want them excluded from the `integrationtests` executable.