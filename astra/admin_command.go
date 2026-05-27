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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type adminCommand struct {
	admin       *AstraAdmin
	method      string
	path        string
	payload     any
	queryParams url.Values
}

func (ac *adminCommand) url() (string, error) {
	baseURL, err := url.JoinPath(ac.admin.astraEnvironment.DevOpsURL(), ac.admin.apiVersion, ac.path)
	if err != nil {
		return "", err
	}
	if len(ac.queryParams) > 0 {
		return baseURL + "?" + ac.queryParams.Encode(), nil
	}
	return baseURL, nil
}

func (ac *adminCommand) withQueryParam(key, value string) *adminCommand {
	ac.queryParams.Set(key, value)
	return ac
}

// adminResponse holds the response from an admin command execution.
type adminResponse struct {
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

func (ac *adminCommand) execute(ctx context.Context) (*adminResponse, error) {
	// Build URL with query params
	reqURL, err := ac.url()
	if err != nil {
		return nil, err
	}

	// Marshal payload to JSON if present
	var bodyReader io.Reader
	var payloadBytes []byte
	if ac.payload != nil {
		payloadBytes, err = json.Marshal(ac.payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		bodyReader = bytes.NewReader(payloadBytes)
	}

	slog.Debug("Running adminCommand.execute", "req.method", ac.method, "req.url", reqURL, "req.body", string(payloadBytes))

	// Create request
	req, err := http.NewRequestWithContext(ctx, ac.method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	resolvedOpts := ac.admin.resolveOptions()
	token := resolvedOpts.GetToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

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
		slog.Debug("adminCommand.execute response headers", extractHeaders(resp.Header))
	}

	slog.Debug("adminCommand.execute response",
		"resp.StatusCode", resp.StatusCode,
		"resp.Status", resp.Status,
		"resp.body", string(body))

	// Handle error responses
	if resp.StatusCode >= 400 {
		return &adminResponse{
			Body:       body,
			Headers:    resp.Header,
			StatusCode: resp.StatusCode,
		}, extractDevOpsError(resp.StatusCode, body)
	}

	return &adminResponse{
		Body:       body,
		Headers:    resp.Header,
		StatusCode: resp.StatusCode,
	}, nil
}

// extractDevOpsError handles error responses from the DevOps API.
func extractDevOpsError(statusCode int, body []byte) error {
	// Try to parse as a structured error
	var resp apiResponse
	// Ignoring errors here because we want to fallback to the raw body if we can't parse
	json.Unmarshal(body, &resp)
	if len(resp.Errors) > 0 {
		return resp.Errors
	}
	// Fallback to raw body
	return fmt.Errorf("DevOps API error (status %d): %s", statusCode, string(body))
}
