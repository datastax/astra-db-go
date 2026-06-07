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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/datastax/astra-db-go/astra/internal/constants"
	"github.com/datastax/astra-db-go/astra/options"
)

type DevOpsAPI struct {
	endpoint    string
	method      string
	payload     any
	queryParams url.Values
	apiOptions  *options.APIOptions
}

func NewDevOpsAPICommand(endpoint, apiVersion, path, method string, payload any, params url.Values, opts *options.APIOptions) *DevOpsAPI {
	return &DevOpsAPI{
		endpoint:    strings.TrimRight(endpoint, "/") + "/" + apiVersion + "/" + strings.Trim(path, "/"),
		method:      method,
		payload:     payload,
		queryParams: params,
		apiOptions:  opts,
	}
}

func (ac *DevOpsAPI) URL() string {
	if len(ac.queryParams) > 0 {
		return ac.endpoint + "?" + ac.queryParams.Encode()
	}
	return ac.endpoint
}

func (ac *DevOpsAPI) WithQueryParam(key, value string) *DevOpsAPI {
	ac.queryParams.Set(key, value)
	return ac
}

// DevOpsResponse holds the response from an admin command execution.
type DevOpsResponse struct {
	Body       []byte
	Headers    http.Header
	StatusCode int
}

func extractHeaders(h http.Header) slog.Attr {
	pairs := make([]slog.Attr, 0, len(h))
	for k, v := range h {
		// Redact sensitive info
		if k == "Authorization" || k == "Cookie" {
			pairs = append(pairs, slog.String(k, "[REDACTED]"))
			continue
		}
		// Join slices to keep the output clean
		pairs = append(pairs, slog.String(k, strings.Join(v, ", ")))
	}
	return slog.Attr{
		Key:   "headers",
		Value: slog.GroupValue(pairs...),
	}
}

func (ac *DevOpsAPI) Execute(ctx context.Context) (*DevOpsResponse, error) {
	// Build URL with query params
	reqURL := ac.URL()

	// Marshal payload to JSON if present
	var bodyReader io.Reader
	var payloadBytes []byte
	if ac.payload != nil {
		payloadBytes, err := json.Marshal(ac.payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		bodyReader = bytes.NewReader(payloadBytes)
	}

	slog.Debug("Running admincmd.Execute", "req.method", ac.method, "req.URL", reqURL, "req.body", string(payloadBytes))

	// Create request
	req, err := http.NewRequestWithContext(ctx, ac.method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	resolvedOpts := ac.apiOptions

	// Set authentication token from resolved options
	if resolvedOpts.TokenProvider != nil {
		token, err := resolvedOpts.TokenProvider.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get token from provider: %w", err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	userAgent := constants.LibName + "/" + constants.LibVersion
	for _, caller := range resolvedOpts.Callers {
		if caller.Version != "" {
			userAgent += " " + caller.Name + "/" + caller.Version
		} else {
			userAgent += " " + caller.Name
		}
	}
	req.Header.Set("User-Agent", userAgent)

	// Add custom headers from options
	for key, value := range resolvedOpts.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	httpClient := resolvedOpts.GetHTTPClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Only do the work to extract headers if we need to.
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		slog.Debug("admincmd.Execute response headers", extractHeaders(resp.Header))
	}

	slog.Debug("admincmd.Execute response",
		"resp.StatusCode", resp.StatusCode,
		"resp.Status", resp.Status,
		"resp.body", string(body))

	// Handle error responses
	if resp.StatusCode >= 400 {
		return &DevOpsResponse{
			Body:       body,
			Headers:    resp.Header,
			StatusCode: resp.StatusCode,
		}, ExtractDevOpsError(resp.StatusCode, body)
	}

	return &DevOpsResponse{
		Body:       body,
		Headers:    resp.Header,
		StatusCode: resp.StatusCode,
	}, nil
}

// ExtractDevOpsError handles error responses from the DevOps API.
func ExtractDevOpsError(statusCode int, body []byte) error {
	// Try to parse as a structured error
	var resp apiResponse
	// Ignoring errors here because we want to fallback to the raw body if we can't parse
	json.Unmarshal(body, &resp)
	if len(resp.Errors) > 0 {
		return resp.Errors
	}
	// Fallback to raw body
	return fmt.Errorf("DevOps API error (status %d): %s", statusCode, body)
}
