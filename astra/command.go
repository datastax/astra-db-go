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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/update"
)

// command represents a command to be executed against the astra DB.
type command struct {
	db              *Db
	name            string
	payload         any
	keyspace        string
	apiVersion      string
	resourceName    string
	databaseAdmin   bool                // When true, URL skips keyspace and resource segments
	resourceOptions []options.APIOption // Cumulative options from Client -> DB -> Resource
	commandOptions  []options.APIOption // Options for this specific command
	commandAPIOpt   *options.APIOptions // Command-level APIOptions struct
	target          serdes.Target
}

// newCmd creates a new command from the given DB
func newCmd(d *Db, name string, payload any) command {
	return command{
		db:              d,
		name:            name,
		payload:         payload,
		resourceOptions: d.options,
	}
}

// newDatabaseAdminCmd creates a new command for database-level admin operations.
// The URL will be {endpoint}/api/json/{version} with no keyspace or resource segments.
func newDatabaseAdminCmd(db *Db, name string, payload any) command {
	return command{
		db:              db,
		name:            name,
		payload:         payload,
		databaseAdmin:   true,
		resourceOptions: db.options,
	}
}

// newCmdWithOptions creates a new command with resource and command-level options
func newCmdWithOptions(d *Db, resource, name string, payload any, resourceOpts []options.APIOption, target serdes.Target, cmdOpts ...options.APIOption) command {
	return command{
		db:              d,
		name:            name,
		resourceName:    resource,
		payload:         payload,
		resourceOptions: resourceOpts,
		commandOptions:  cmdOpts,
		target:          target,
	}
}

// newCmdWithMergedOptions creates a new command with resource and merged command-level options
func newCmdWithMergedOptions(d *Db, resource, name string, payload any, resourceOpts []options.APIOption, target serdes.Target, cmdOpts *options.APIOptions) command {
	return command{
		db:              d,
		name:            name,
		resourceName:    resource,
		payload:         payload,
		resourceOptions: resourceOpts,
		commandAPIOpt:   cmdOpts,
		target:          target,
	}
}

// resolveOptions merges all option layers and returns the final resolved options.
// Merge order: Cumulative Resource Options (Client -> DB -> Resource) -> Command Options
// Defaults are applied by options.Merge.
func (c *command) resolveOptions() *options.APIOptions {
	var opts []options.APIOption
	opts = append(opts, c.resourceOptions...)
	opts = append(opts, c.commandOptions...)

	if c.commandAPIOpt != nil {
		opts = append(opts, c.commandAPIOpt)
	}

	return options.Merge(opts...)
}

// Keyspace returns the keyspace to use for this command.
// If explicitly set on the command, that value is used.
// Otherwise, it falls back to the resolved options.
func (c *command) Keyspace() string {
	if len(c.keyspace) > 0 {
		return c.keyspace
	}
	return c.resolveOptions().GetKeyspace()
}

// ApiVersion returns the API version to use for this command.
// If explicitly set on the command, that value is used.
// Otherwise, it falls back to the resolved options.
func (c *command) ApiVersion() string {
	if len(c.apiVersion) > 0 {
		return c.apiVersion
	}
	return c.resolveOptions().GetAPIVersion()
}

func (c *command) url() (string, error) {
	if c.db == nil {
		return "", errors.New("nil Db")
	}
	if len(c.db.Endpoint()) == 0 {
		return "", errors.New("empty API endpoint")
	}
	basePath := c.resolveOptions().GetDataAPIBackend().DataAPIPath()
	if c.databaseAdmin {
		return url.JoinPath(c.db.Endpoint(), basePath, c.ApiVersion())
	}
	return url.JoinPath(c.db.Endpoint(), basePath, c.ApiVersion(), c.Keyspace(), c.resourceName)
}

// This is similar to the [.NET client]. If we have a command name we want to
// marshal into json such as:
//
//	{"createCollection":{"name":"COLLECTION_NAME","options":{}}}
//
// But if we don't have a command name, we just marshal the payload directly.
//
// [.NET client]: https://github.com/datastax/astra-db-csharp/blob/699ac093494b1a5adbb65c65be57af5b48eb8cc2/src/DataStax.AstraDB.DataApi/Core/Commands/Command.cs#L92
func (c command) MarshalAstraRaw(_ serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if len(c.name) > 0 {
		data := make(map[string]any)
		data[c.name] = c.payload
		return serdes.SerializeInto(data, c.target, dst)
	}
	return serdes.SerializeInto(c.payload, c.target, dst)
}

// Execute a command against the astra DB web API.
// Returns the response body, any warnings from the API, and any error that occurred.
func (c *command) Execute(ctx context.Context) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
	var body []byte
	if c.db == nil {
		return body, nil, nil, ErrCmdNilDb
	}

	// Resolve all options for this command
	opts := c.resolveOptions()

	b, err := serdes.Serialize(c, c.target)
	if err != nil {
		return body, nil, nil, err
	}
	cmdURL, err := c.url()
	if err != nil {
		return body, nil, nil, err
	}
	slog.Debug("Running cmd.Execute", "req.url", cmdURL, "req.body", string(b))

	req, err := http.NewRequestWithContext(ctx, "POST", cmdURL, bytes.NewReader(b))
	if err != nil {
		return body, nil, nil, err
	}

	// Set authentication token from resolved options
	token := opts.GetToken()
	if token != "" {
		req.Header.Set("Token", token)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add any custom headers from resolved options
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Use HTTP client from resolved options
	httpClient := opts.GetHTTPClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		return body, nil, nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	slog.Debug("cmd.Execute response", "resp.StatusCode", resp.StatusCode, "resp.Status", resp.Status, "resp.body", string(body))
	if err != nil {
		return body, nil, nil, err
	}
	return c.extractErrors(resp.StatusCode, body, opts)
}

// apiResponse captures both errors and warnings from API responses
type apiResponse struct {
	Errors DataAPIErrors `json:"errors"`
	Status struct {
		Warnings         results.Warnings `json:"warnings"`
		PrimaryKeySchema json.RawMessage  `json:"primaryKeySchema"`
		ProjectionSchema json.RawMessage  `json:"projectionSchema"`
	} `json:"status"`
}

type apiStatus struct {
	warnings results.Warnings
	schema   json.RawMessage
}

// extractErrors will extract errors and warnings from body. For example, it will
// turn this response into an error:
//
//	{"message":"Your database is resuming from hibernation and will be available in the next few minutes."}
//
// Will call WarningHandler if appropriate.
func (c *command) extractErrors(statusCode int, body []byte, opts *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
	if statusCode >= 400 {
		// We have a transport/server-level error so let's try to extract the message.
		var transportErr DataAPIError
		serdes.Deserialize(body, &transportErr, nil, serdes.TargetNone)
		if len(transportErr.Message) > 0 {
			return body, nil, nil, errors.New(transportErr.Message)
		}
		// We can't find a message; just return the body
		return body, nil, nil, errors.New(string(body))
	}

	// Parse the full response to get both errors and warnings
	var resp apiResponse
	serdes.Deserialize(body, &resp, nil, c.target)

	// Invoke warning handler for each warning if configured
	if opts != nil && opts.WarningHandler != nil && len(resp.Status.Warnings) > 0 {
		for _, w := range resp.Status.Warnings {
			opts.WarningHandler(w)
		}
	}

	status := apiStatus{
		warnings: resp.Status.Warnings,
		schema:   resp.Status.PrimaryKeySchema,
	}
	if resp.Status.ProjectionSchema != nil {
		status.schema = resp.Status.ProjectionSchema
	}

	var schema serdes.TargetDecodeCtx
	if c.target == serdes.TargetCollection {
		schema = documentCtx
	} else if status.schema != nil {
		schema = &lazySchema{AsRaw: status.schema}
	}

	// Return error if present
	if len(resp.Errors) > 0 {
		return body, resp.Status.Warnings, schema, &resp.Errors
	}

	return body, resp.Status.Warnings, schema, nil
}

// CollectionUpdate is implemented by [update.CollectionUpdateBuilder] and [update.U].
// See the [update package] for more details.
//
// [update package]: https://pkg.go.dev/github.com/datastax/astra-db-go/astra/update
type CollectionUpdate = update.CollectionUpdate

// TableUpdate is implemented by [update.TableUpdateBuilder] and [update.U].
// See the [update package] for more details.
//
// [update package]: https://pkg.go.dev/github.com/datastax/astra-db-go/astra/update
type TableUpdate = update.TableUpdate
