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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

type cachingRoundTripper struct {
	underlying http.RoundTripper
	cache      map[string]*cachedResponse
	mu         sync.RWMutex
}

type cachedResponse struct {
	statusCode int
	header     http.Header
	body       []byte
}

func newCachingRoundTripper() *cachingRoundTripper {
	return &cachingRoundTripper{
		underlying: http.DefaultTransport,
		cache:      make(map[string]*cachedResponse),
	}
}

func (c *cachingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	key, err := hashRequest(req)
	if err != nil {
		return c.underlying.RoundTrip(req)
	}

	c.mu.RLock()
	if cached, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		return reconstructResponse(req, cached), nil
	}
	c.mu.RUnlock()

	resp, err := c.underlying.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	cached, err := captureResponse(resp)
	if err != nil {
		return resp, nil
	}

	c.mu.Lock()
	c.cache[key] = cached
	c.mu.Unlock()

	return reconstructResponse(req, cached), nil
}

func hashRequest(req *http.Request) (string, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return "", err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	data := map[string]any{
		"url":    req.URL.String(),
		"method": req.Method,
		"body":   string(body),
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}

func captureResponse(resp *http.Response) (*cachedResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return &cachedResponse{
		statusCode: resp.StatusCode,
		header:     resp.Header.Clone(),
		body:       body,
	}, nil
}

func reconstructResponse(req *http.Request, cached *cachedResponse) *http.Response {
	return &http.Response{
		StatusCode: cached.statusCode,
		Header:     cached.header.Clone(),
		Body:       io.NopCloser(bytes.NewReader(cached.body)),
		Request:    req,
	}
}
