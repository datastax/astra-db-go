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

package results

// FindRerankingProvidersResult is the overarching result containing the RerankingProviders map.
type FindRerankingProvidersResult struct {
	// RerankingProviders is a map of reranking provider names to information about said provider.
	RerankingProviders map[string]RerankingProviderInfo `json:"rerankingProviders"`
}

// RerankingProviderInfo contains info about a specific reranking provider.
type RerankingProviderInfo struct {
	// DisplayName is the prettified name of the provider.
	DisplayName string `json:"displayName"`

	// Models are the specific models that the provider supports.
	Models []RerankingProviderModelInfo `json:"models"`
}

// RerankingProviderModelInfo describes a specific model offered by a reranking provider.
type RerankingProviderModelInfo struct {
	// Name is the name of the model to use.
	Name string `json:"name"`

	// ApiModelSupport describes the API support status and lifecycle of the model.
	ApiModelSupport EmbeddingProviderModelApiSupportInfo `json:"apiModelSupport"`
}
