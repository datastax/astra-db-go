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

// FindRerankingProvidersResult is the overarching result containing the rerankingProviders map.
//
// Example:
//
//	result, err := dbAdmin.FindRerankingProviders(ctx)
//	if err != nil {
//	    return err
//	}
//	// ["nvidia/llama-3.2-nv-rerankqa-1b-v2"]
//	for _, model := range result.rerankingProviders["nvidia"].Models {
//	    fmt.Println(model.Name)
//	}
type FindRerankingProvidersResult struct {
	RerankingProviders map[string]RerankingProviderInfo `json:"rerankingProviders"`
}

// RerankingProviderInfo contains info about a specific reranking provider.
type RerankingProviderInfo struct {
	// DisplayName is the prettified name of the provider (as shown in the Astra portal).
	DisplayName string `json:"displayName"`

	// URL is the rerankings endpoint used for the provider. May be nil for some providers.
	//
	// May use a Python f-string-style interpolation pattern for certain providers which take
	// in additional parameters
	URL *string `json:"url"`

	// SupportedAuthentication maps auth method names to info about that auth method for this provider.
	//
	// Possible methods include "HEADER", "SHARED_SECRET", and "NONE".
	SupportedAuthentication map[string]RerankingProviderAuthInfo `json:"supportedAuthentication"`

	// Parameters are any additional, arbitrary parameters the provider may take in. May or may not be required.
	//
	// Passed into the parameters block when creating a vectorize-enabled collection or table
	// (except for vectorDimension, which belongs in the vector dimension field).
	Parameters []RerankingProviderProviderParameterInfo `json:"parameters"`

	// Models are the specific models that the provider supports.
	//
	// May include an "endpoint-defined-model" for some providers, such as huggingfaceDedicated,
	// where the model may be truly arbitrary.
	Models []RerankingProviderModelInfo `json:"models"`
}

type RerankingProviderAuthInfo struct {
	// Enabled indicates whether this auth method is supported for the provider.
	Enabled bool `json:"enabled"`

	// Tokens contains additional info on how exactly this auth method is supposed to be used.
	// Will be empty if Enabled is false.
	Tokens []RerankingProviderTokenInfo `json:"tokens"`
}

// rerankingProviderTokenInfo contains info on how exactly a method of auth may be used.
type RerankingProviderTokenInfo struct {
	// Accepted is the accepted token.
	// Most often "providerKey" for SHARED_SECRET, or "x-reranking-api-key" for HEADER.
	Accepted string `json:"accepted"`

	// Forwarded is how the token is forwarded to the reranking provider.
	Forwarded string `json:"forwarded"`
}

// rerankingProviderModelParameterInfo contains info about any additional, arbitrary parameter
// a model may take in. May or may not be required.
//
// Passed into the parameters block when creating a vectorize-enabled collection or table
// (except for vectorDimension, which should be set in the vector dimension field instead).

type RerankingProviderModelParameterInfo struct {
	// Name is the name of the parameter to be passed in.
	//
	// The one exception is the vectorDimension parameter, which should be passed into the
	// dimension field of the vector block instead.
	Name string `json:"name"`

	// Type is the datatype of the parameter. Commonly "number" or "STRING".
	Type string `json:"type"`

	// Required indicates whether the parameter is required to be passed in.
	Required bool `json:"required"`

	// DefaultValue is the default value of the parameter, or an empty string if there is none.
	// Will always be in string form even if Type is "number".
	DefaultValue string `json:"defaultValue"`

	// Validation holds validations that may be done on the inputted value.
	// Commonly either nil, or contains a "numericRange" key with [min, max].
	Validation map[string]any `json:"validation"`

	// Help is any additional help text/information about the parameter.
	Help string `json:"help"`
}

// rerankingProviderProviderParameterInfo contains info about any additional, arbitrary parameter
// a provider may take in. May or may not be required. Extends rerankingProviderModelParameterInfo
// with display metadata.
type RerankingProviderProviderParameterInfo struct {
	RerankingProviderModelParameterInfo

	// DisplayName is the UI display name for the parameter.
	DisplayName string `json:"displayName"`

	// Hint is a short usage hint for the parameter.
	Hint string `json:"hint"`
}

// rerankingProviderModelInfo describes a specific model offered by an reranking provider.
//
// May include an "endpoint-defined-model" for some providers, such as huggingfaceDedicated,
// where the model may be truly arbitrary.
type RerankingProviderModelInfo struct {
	// Name is the name of the model to use.
	// May be "endpoint-defined-model" for providers like huggingfaceDedicated.
	Name string `json:"name"`

	// VectorDimension is the preset, exact vector dimension to be used (if applicable).
	// If nil, a vectorDimension parameter will be present in Parameters instead.
	VectorDimension *int `json:"vectorDimension"`

	// Parameters are any additional, arbitrary parameters the model may take in.
	Parameters []RerankingProviderModelParameterInfo `json:"parameters"`

	// ApiModelSupport describes the API support status and lifecycle of the model.
	ApiModelSupport RerankingProviderModelApiSupportInfo `json:"apiModelSupport"`
}

// RerankingProviderModelApiSupportInfo describes the API support status and lifecycle of a model.
type RerankingProviderModelApiSupportInfo struct {
	// Status is the current lifecycle status of the model.
	Status ModelLifecycleStatus `json:"status"`
}
