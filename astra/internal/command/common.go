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
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-db-go/v2/astra/internal/constants"
	"github.com/datastax/astra-db-go/v2/astra/options"
)

func resolveToken(ctx context.Context, provider options.TokenProvider) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("no token provider configured")
	}
	token, err := provider.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token from provider: %w", err)
	}
	return token, nil
}

func setCommonHeaders(headers http.Header, callers []options.Caller) {
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/json")

	userAgent := constants.LibName + "/" + constants.LibVersion
	for _, caller := range callers {
		if caller.Version != "" {
			userAgent += " " + caller.Name + "/" + caller.Version
		} else {
			userAgent += " " + caller.Name
		}
	}
	headers.Set("User-Agent", userAgent)
}
