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
	// EmbeddingProviders is a map of embedding provider names (e.g. "openai") to information
	// about said provider (e.g. models/auth).
	//
	// Example:
	//
	//	result.EmbeddingProviders["openai"] // => EmbeddingProviderInfo{ DisplayName: "OpenAI", ... }
	RerankingProviders map[string]RerankingProviderInfo `json:"rerankingProviders"`
}

// RerankingProviderInfo contains info about a specific reranking provider.
type RerankingProviderInfo struct {
	// IsDefault Shows if the reranking provider is the default one. The current default is Nvidia
	IsDefault bool `json:"isDefault"`

	// DisplayName is the prettified name of the provider (as shown in the Astra portal).
	//
	// Example: "OpenAI"
	DisplayName string `json:"displayName"`

	// SupportedAuthentication maps auth method names to info about that auth method for this provider.
	//
	// Possible methods include "HEADER", "SHARED_SECRET", and "NONE".
	//
	//   - "HEADER": Authentication using direct API keys passed through headers on every Data API call.
	//
	//	    coll := db.Collection("my_coll",
	//	        options.API().SetHeaders(map[string]string{
	//	            // Not tied to the collection; can be different every time.
	//	            "nvidia/llama-3.2-nv-rerankqa-1b-v2": "sk-...",
	//	        }),
	//	    )
	//
	//   - "SHARED_SECRET": Authentication tied to a collection at collection creation time using the Astra KMS.
	//
	//	    _, err = db.CreateCollection(ctx, "my_coll",
	//	        options.CreateCollection().UpdateVector(
	//	            options.Vector().UpdateService(
	//	                options.VectorService().
	//	                    SetProvider("nvidia").
	//	                    SetModelName("nvidia/llama-3.2-nv-rerankqa-1b-v2").
	//	                    // Name of the key in Astra portal's OpenAI integration (KMS):
	//	                    SetAuthentication(map[string]any{"providerKey": "*KEY_NAME*"}),
	//	            ),
	//	        ),
	//	    )
	//
	//
	//   - "NONE": For providers that do not require authentication (e.g. nvidia).
	//     No key or credential is needed when creating or using the collection.
	//
	//	    _, err = db.CreateCollection(ctx, "my_coll",
	//	        options.CreateCollection().UpdateVector(
	//	            options.Vector().UpdateService(
	//	                options.VectorService().
	//	                    SetProvider("nvidia").
	//	                    SetModelName("NV-Embed-QA"),
	//	            ),
	//	        ),
	//	    )
	SupportedAuthentication map[string]RerankingProviderAuthInfo `json:"supportedAuthentication"`

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

// RerankingProviderAuthInfo contains information about a specific auth method
// (such as "HEADER", "SHARED_SECRET", or "NONE") for a specific provider.
//
// Example:
//
//	// openai.SupportedAuthentication["HEADER"]:
//	RerankingProviderAuthInfo{
//	    Enabled: true,
//	    Tokens: []RerankingProviderTokenInfo{{
//	        Accepted:  "x-embedding-api-key",
//	        Forwarded: "Authorization",
//	    }},
//	}
type RerankingProviderTokenInfo struct {
	// Accepted is the accepted token.
	// Most often "providerKey" for SHARED_SECRET, or "x-reranking-api-key" for HEADER.
	Accepted string `json:"accepted"`

	// Forwarded is how the token is forwarded to the reranking provider.
	Forwarded string `json:"forwarded"`
}

// RerankingProviderModelParameterInfo contains info about any additional, arbitrary parameter
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

// RerankingProviderProviderParameterInfo contains info about any additional, arbitrary parameter
// a provider may take in. May or may not be required. Extends RerankingProviderModelParameterInfo
// with display metadata.
type RerankingProviderProviderParameterInfo struct {
	RerankingProviderModelParameterInfo

	// DisplayName is the UI display name for the parameter.
	DisplayName string `json:"displayName"`

	// Hint is a short usage hint for the parameter.
	Hint string `json:"hint"`
}

// RerankingProviderModelInfo describes a specific model offered by an reranking provider.
//
// May include an "endpoint-defined-model" for some providers, such as huggingfaceDedicated,
// where the model may be truly arbitrary.
type RerankingProviderModelInfo struct {
	// Name is the name of the model to use.
	// May be "endpoint-defined-model" for providers like huggingfaceDedicated.
	Name string `json:"name"`

	// ApiModelSupport describes the API support status and lifecycle of the model.
	ApiModelSupport RerankingProviderModelApiSupportInfo `json:"apiModelSupport"`

	IsDefault bool `json:"isDefault"`

	// URL is the rerankings endpoint used for the provider. May be nil for some providers.
	//
	// May use a Python f-string-style interpolation pattern for certain providers which take
	// in additional parameters
	URL *string `json:"url"`

	// Properties is a free-form dictionary with string keys, describing the model.
	Properties map[string]any `json:"properties"`
}

// RerankingProviderModelApiSupportInfo describes the API support status and lifecycle of a model.
type RerankingProviderModelApiSupportInfo struct {
	// Status is the current lifecycle status of the model.
	Status ModelLifecycleStatus `json:"status"`
}
