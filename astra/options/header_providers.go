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

import "context"

// EmbeddingHeadersProvider is an interface for providing headers for embedding services (e.g. $vectorize).
type EmbeddingHeadersProvider interface {
	GetEmbeddingHeaders(ctx context.Context) (map[string]string, error)
}

// RerankingHeadersProvider is an interface for providing headers for reranking services.
type RerankingHeadersProvider interface {
	GetRerankingHeaders(ctx context.Context) (map[string]string, error)
}

type staticHeadersProvider struct {
	headers map[string]string
}

func (p *staticHeadersProvider) GetEmbeddingHeaders(_ context.Context) (map[string]string, error) {
	return p.headers, nil
}

func (p *staticHeadersProvider) GetRerankingHeaders(_ context.Context) (map[string]string, error) {
	return p.headers, nil
}

// NewEmbeddingAPIKeyHeadersProvider creates an EmbeddingHeadersProvider that returns the x-embedding-api-key header.
func NewEmbeddingAPIKeyHeadersProvider(apiKey string) EmbeddingHeadersProvider {
	return &staticHeadersProvider{
		headers: map[string]string{"x-embedding-api-key": apiKey},
	}
}

// NewAWSEmbeddingHeadersProvider creates an EmbeddingHeadersProvider for AWS-based embedding providers.
// It sets the x-embedding-access-id and x-embedding-secret-id headers.
func NewAWSEmbeddingHeadersProvider(accessKeyID, secretAccessKey string) EmbeddingHeadersProvider {
	return &staticHeadersProvider{
		headers: map[string]string{
			"x-embedding-access-id": accessKeyID,
			"x-embedding-secret-id": secretAccessKey,
		},
	}
}

// NewRerankingAPIKeyHeadersProvider creates a RerankingHeadersProvider that returns the x-rerank-api-key header.
func NewRerankingAPIKeyHeadersProvider(apiKey string) RerankingHeadersProvider {
	return &staticHeadersProvider{
		headers: map[string]string{"x-rerank-api-key": apiKey},
	}
}
