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

package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/datastax/astra-db-go/astra/internal/constants"
	"github.com/datastax/astra-db-go/astra/internal/untyped"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
)

var (
	ErrCmdNilDb = errors.New("command cannot execute with nil Db")
)

// DataAPI represents a command to be executed against the astra DB.
type DataAPI struct {
	url     string
	name    string
	payload any
	options *options.APIOptions
	target  serdes.Target
}

func newDataAPICommand(endpoint, resourceName, name string, payload any, target serdes.Target, joined options.Joined[options.APIOptions], admin bool) DataAPI {
	opts := options.Merge(joined...)

	var u string
	if endpoint != "" {
		basePath := opts.GetDataAPIBackend().DataAPIPath()
		u, _ = url.JoinPath(endpoint, basePath, opts.GetAPIVersion())
		if !admin {
			u, _ = url.JoinPath(u, opts.GetKeyspace(), resourceName)
		}
	}

	return DataAPI{
		url:     u,
		name:    name,
		payload: payload,
		options: opts,
		target:  target,
	}
}

func NewDataAPICommand(endpoint, resourceName, name string, payload any, target serdes.Target, opts options.Joined[options.APIOptions]) DataAPI {
	return newDataAPICommand(endpoint, resourceName, name, payload, target, opts, false)
}

func NewDataAPIAdminCommand(endpoint, resourceName, name string, payload any, target serdes.Target, opts options.Joined[options.APIOptions]) DataAPI {
	return newDataAPICommand(endpoint, resourceName, name, payload, target, opts, true)
}

// ResolveOptions returns the final resolved options.
func (c *DataAPI) ResolveOptions() *options.APIOptions {
	return c.options
}

// Keyspace returns the keyspace to use for this command.
func (c *DataAPI) Keyspace() string {
	return c.options.GetKeyspace()
}

// ApiVersion returns the Data API version to use for this command.
func (c *DataAPI) ApiVersion() string {
	return c.options.GetAPIVersion()
}

func (c *DataAPI) URL() string {
	return c.url
}

func (c DataAPI) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if len(c.name) > 0 {
		dst = append(dst, `{"`...)
		dst = append(dst, c.name...)
		dst = append(dst, `":`...)

		var err error
		dst, err = serdes.SerializeInto(c.payload, c.target, dst, ctx.Flags)
		if err != nil {
			return nil, err
		}
		return append(dst, '}'), nil
	}
	return serdes.SerializeInto(c.payload, c.target, dst, ctx.Flags)
}

// Execute a command against the astra DB web API.
func (c *DataAPI) Execute(ctx context.Context) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
	var body []byte
	if c.url == "" {
		return body, nil, nil, ErrCmdNilDb
	}

	// Resolve all options for this command
	opts := c.options

	b, err := serdes.Serialize(c, c.target)
	if err != nil {
		return body, nil, nil, err
	}
	cmdURL := c.URL()
	slog.Debug("Running cmd.Execute", "req.URL", cmdURL, "req.body", string(b))

	req, err := http.NewRequestWithContext(ctx, "POST", cmdURL, bytes.NewReader(b))
	if err != nil {
		return body, nil, nil, err
	}

	// Set authentication token from resolved options
	if opts.TokenProvider != nil {
		token, err := opts.TokenProvider.Token(ctx)
		if err != nil {
			return body, nil, nil, fmt.Errorf("failed to get token from provider: %w", err)
		}
		if token != "" {
			req.Header.Set("Token", token)
		}
	}

	req.Header.Set("Content-Type", "application/json")

	userAgent := constants.LibName + "/" + constants.LibVersion
	for _, caller := range opts.Callers {
		if caller.Version != "" {
			userAgent += " " + caller.Name + "/" + caller.Version
		} else {
			userAgent += " " + caller.Name
		}
	}
	req.Header.Set("User-Agent", userAgent)

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
	return c.ExtractErrors(resp.StatusCode, body, opts)
}

// apiResponse captures both errors and warnings from API responses
type apiResponse struct {
	Errors results.DataAPIErrors `json:"errors"`
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

func (c *DataAPI) ExtractErrors(statusCode int, body []byte, opts *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
	if statusCode >= 400 {
		var transportErr results.DataAPIError
		serdes.Deserialize(body, &transportErr, nil, serdes.TargetNone, opts.GetDesFlags())
		if len(transportErr.Message) > 0 {
			return body, nil, nil, errors.New(transportErr.Message)
		}
		return body, nil, nil, errors.New(string(body))
	}

	var resp apiResponse
	serdes.Deserialize(body, &resp, nil, c.target, opts.GetDesFlags())

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
		schema = untyped.GlobalDocumentCtx
	} else if status.schema != nil {
		schema = &untyped.LazySchema{AsRaw: status.schema}
	}

	if len(resp.Errors) > 0 {
		return body, resp.Status.Warnings, schema, &resp.Errors
	}

	return body, resp.Status.Warnings, schema, nil
}
