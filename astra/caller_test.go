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
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/datastax/astra-db-go/astra/internal/command"
	"github.com/datastax/astra-db-go/astra/internal/constants"
	"github.com/datastax/astra-db-go/astra/options"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCommandUserAgent(t *testing.T) {
	var capturedUA string
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			capturedUA = req.Header.Get("User-Agent")
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, nil
		}),
	}

	// 1. Default User-Agent
	cmd := command.NewDataAPICommand("http://localhost", "", "", nil, 0, options.Join(nil, options.API().SetHTTPClient(httpClient)))
	_, _, _, _ = cmd.Execute(context.Background())
	expected := constants.LibName + "/" + constants.LibVersion
	if capturedUA != expected {
		t.Errorf("expected default User-Agent %q, got %q", expected, capturedUA)
	}

	// 2. With Callers
	opts := options.API().
		SetHTTPClient(httpClient).
		AddCaller("my-app", "1.2.3").
		AddCaller("my-framework", "")

	cmd = command.NewDataAPICommand("http://localhost", "", "", nil, 0, options.Join(nil, opts))
	_, _, _, _ = cmd.Execute(context.Background())
	expected = constants.LibName + "/" + constants.LibVersion + " my-app/1.2.3 my-framework"
	if capturedUA != expected {
		t.Errorf("expected User-Agent with callers %q, got %q", expected, capturedUA)
	}
}
