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

package options

import (
	"context"
	"encoding/base64"
)

// TokenProvider is an interface for providing authentication tokens dynamically.
type TokenProvider interface {
	// Token returns the token to be used for the current request.
	Token(ctx context.Context) (string, error)
}

type staticTokenProvider struct {
	Unwrap string
}

// NewStaticTokenProvider creates a new StaticTokenProvider with the given token.
func NewStaticTokenProvider(token string) TokenProvider {
	return &staticTokenProvider{token}
}

// Token returns the static token.
func (s staticTokenProvider) Token(_ context.Context) (string, error) {
	return s.Unwrap, nil
}

type usernamePasswordTokenProvider struct {
	staticTokenProvider
}

// NewUsernamePasswordTokenProvider creates a new TokenProvider that encodes the username and password in the format expected by DSE/HCD.
func NewUsernamePasswordTokenProvider(username, password string) TokenProvider {
	return usernamePasswordTokenProvider{
		staticTokenProvider{"Cassandra:" + encodeBase64(username) + ":" + encodeBase64(password)},
	}
}

func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
