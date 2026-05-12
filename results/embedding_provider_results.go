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

import "github.com/datastax/astra-db-go/internal/constants"

// ModelLifecycleStatus is the lifecycle status of an embedding provider model.
type ModelLifecycleStatus string

const (
	// ModelLifecycleStatusSupported indicates the model is actively supported and recommended for use.
	ModelLifecycleStatusSupported ModelLifecycleStatus = ModelLifecycleStatus(constants.ModelLifecycleStatusSupported)
	// ModelLifecycleStatusDeprecated indicates the model is still functional but deprecated and may be removed in the future.
	ModelLifecycleStatusDeprecated ModelLifecycleStatus = ModelLifecycleStatus(constants.ModelLifecycleStatusDeprecated)
	// ModelLifecycleStatusEndOfLife indicates the model is no longer supported and should not be used.
	ModelLifecycleStatusEndOfLife ModelLifecycleStatus = ModelLifecycleStatus(constants.ModelLifecycleStatusEndOfLife)
)

// FindEmbeddingProvidersResult is the overarching result containing the EmbeddingProviders map.
//
// Example:
//
//	result, err := dbAdmin.FindEmbeddingProviders(ctx)
//	if err != nil {
//	    return err
//	}
//	// ["text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"]
//	for _, model := range result.EmbeddingProviders["openai"].Models {
//	    fmt.Println(model.Name)
//	}
type FindEmbeddingProvidersResult struct {
	// EmbeddingProviders is a map of embedding provider names (e.g. "openai") to information
	// about said provider (e.g. models/auth).
	//
	// Example:
	//
	//	result.EmbeddingProviders["openai"] // => EmbeddingProviderInfo{ DisplayName: "OpenAI", ... }
	EmbeddingProviders map[string]EmbeddingProviderInfo `json:"embeddingProviders"`
}

// EmbeddingProviderInfo contains info about a specific embedding provider.
type EmbeddingProviderInfo struct {
	// DisplayName is the prettified name of the provider (as shown in the Astra portal).
	//
	// Example: "OpenAI"
	DisplayName string `json:"displayName"`

	// URL is the embeddings endpoint used for the provider.
	//
	// May use a Python f-string-style interpolation pattern for certain providers which take
	// in additional parameters (such as huggingfaceDedicated or azureOpenAI).
	//
	// Example:
	//
	//	// openai.URL:
	//	"https://api.openai.com/v1/"
	//
	//	// huggingfaceDedicated.URL:
	//	"https://{endpointName}.{regionName}.{cloudName}.endpoints.huggingface.cloud/embeddings"
	URL string `json:"url"`

	// SupportedAuthentication maps auth method names to info about that auth method for this provider.
	//
	// Possible methods include "HEADER", "SHARED_SECRET", and "NONE".
	//
	//   - "HEADER": Authentication using direct API keys passed through headers on every Data API call.
	//
	//	    coll := db.Collection("my_coll",
	//	        options.API().SetHeaders(map[string]string{
	//	            // Not tied to the collection; can be different every time.
	//	            "x-embedding-api-key": "sk-...",
	//	        }),
	//	    )
	//
	//   - "SHARED_SECRET": Authentication tied to a collection at collections creation time using the Astra KMS.
	//TODO: confirm this example and comment
	//
	//	    _, err = db.CreateCollection(ctx, "my_coll",
	//	        options.CreateCollection().SetVector(
	//	            options.Vector().SetService(
	//	                options.VectorService().
	//	                    SetProvider("openai").
	//	                    SetModelName("text-embedding-3-small").
	//	                    // Name of the key in Astra portal's OpenAI integration (KMS):
	//	                    SetProviderKey("*KEY_NAME*"),
	//	            ),
	//	        ),
	//	    )
	//
	//
	//   - "NONE": For providers that do not require authentication (e.g. nvidia).
	//     No key or credential is needed when creating or using the collection.
	//
	//	    _, err = db.CreateCollection(ctx, "my_coll",
	//	        options.CreateCollection().SetVector(
	//	            options.Vector().SetService(
	//	                options.VectorService().
	//	                    SetProvider("nvidia").
	//	                    SetModelName("NV-Embed-QA"),
	//	            ),
	//	        ),
	//	    )
	//
	// Example:
	//
	//	// openai.SupportedAuthentication["HEADER"]:
	//	EmbeddingProviderAuthInfo{
	//	    Enabled: true,
	//	    Tokens: []EmbeddingProviderTokenInfo{{
	//	        Accepted:  "x-embedding-api-key",
	//	        Forwarded: "Authorization",
	//	    }},
	//	}
	SupportedAuthentication map[string]EmbeddingProviderAuthInfo `json:"supportedAuthentication"`

	// Parameters are any additional, arbitrary parameters the provider may take in. May or may not be required.
	//
	// Passed into the parameters block when creating a vectorize-enabled collection or table
	// (except for vectorDimension, which belongs in the vector dimension field).
	//
	// Example:
	//
	//	// openai.Parameters[0]:
	//	EmbeddingProviderProviderParameterInfo{
	//	    EmbeddingProviderModelParameterInfo: EmbeddingProviderModelParameterInfo{
	//	        Name:         "projectId",
	//	        Type:         "STRING",
	//	        Required:     false,
	//	        DefaultValue: "",
	//	        Validation:   nil,
	//	        Help:         "Optional, OpenAI Project ID. If provided passed as `OpenAI-Project` header.",
	//	    },
	//	    DisplayName: "Organization ID",
	//	    Hint:        "Add an (optional) organization ID",
	//	}
	Parameters []EmbeddingProviderProviderParameterInfo `json:"parameters"`

	// Models are the specific models that the provider supports.
	//
	// May include an "endpoint-defined-model" for some providers, such as huggingfaceDedicated,
	// where the model may be truly arbitrary.
	//
	// Example:
	//
	//	// nvidia.Models[0]:
	//	EmbeddingProviderModelInfo{
	//	    Name:            "NV-Embed-QA",
	//	    VectorDimension: ptr.To(1024),
	//	    Parameters:      nil,
	//	}
	//
	//	// huggingfaceDedicated.Models[0]:
	//	EmbeddingProviderModelInfo{
	//	    Name:            "endpoint-defined-model",
	//	    VectorDimension: nil,
	//	    Parameters: []EmbeddingProviderModelParameterInfo{{
	//	        Name:         "vectorDimension",
	//	        Type:         "number",
	//	        Required:     true,
	//	        DefaultValue: "",
	//	        Validation:   []map[string]any{{"numericRange": []int{2, 3072}}},
	//	        Help:         "Vector dimension to use in the database, should be the same as ...",
	//	    }},
	//	}
	Models []EmbeddingProviderModelInfo `json:"models"`
}

// EmbeddingProviderAuthInfo contains information about a specific auth method
// (such as "HEADER", "SHARED_SECRET", or "NONE") for a specific provider.
//
// Example:
//
//	// openai.SupportedAuthentication["HEADER"]:
//	EmbeddingProviderAuthInfo{
//	    Enabled: true,
//	    Tokens: []EmbeddingProviderTokenInfo{{
//	        Accepted:  "x-embedding-api-key",
//	        Forwarded: "Authorization",
//	    }},
//	}
type EmbeddingProviderAuthInfo struct {
	// Enabled indicates whether this auth method is supported for the provider.
	Enabled bool `json:"enabled"`

	// Tokens contains additional info on how exactly this auth method is supposed to be used.
	// Will be empty if Enabled is false.
	Tokens []EmbeddingProviderTokenInfo `json:"tokens"`
}

// EmbeddingProviderTokenInfo contains info on how exactly a method of auth may be used.
//
// Example:
//
//	// openai.SupportedAuthentication["HEADER"].Tokens[0]:
//	EmbeddingProviderTokenInfo{
//	    Accepted:  "x-embedding-api-key",
//	    Forwarded: "Authorization",
//	}
type EmbeddingProviderTokenInfo struct {
	// Accepted is the accepted token.
	// Most often "providerKey" for SHARED_SECRET, or "x-embedding-api-key" for HEADER.
	Accepted string `json:"accepted"`

	// Forwarded is how the token is forwarded to the embedding provider.
	Forwarded string `json:"forwarded"`
}

// EmbeddingProviderModelParameterInfo contains info about any additional, arbitrary parameter
// a model may take in. May or may not be required.
//
// Passed into the parameters block when creating a vectorize-enabled collection or table
// (except for vectorDimension, which should be set in the vector dimension field instead).
//
// Example:
//
//	// openai.Models[0].Parameters[0] (text-embedding-3-small):
//	EmbeddingProviderModelParameterInfo{
//	    Name:         "vectorDimension",
//	    Type:         "number",
//	    Required:     true,
//	    DefaultValue: "1536",
//	    Validation:   []map[string]any{{"numericRange": []int{2, 1536}}},
//	    Help:         "Vector dimension to use in the database and when calling OpenAI.",
//	}
type EmbeddingProviderModelParameterInfo struct {
	// Name is the name of the parameter to be passed in.
	//
	// The one exception is the vectorDimension parameter, which should be passed into the
	// dimension field of the vector block instead.
	//
	// Example: "endpointName" (huggingface.Parameters[0].Name)
	Name string `json:"name"`

	// Type is the datatype of the parameter. Commonly "number" or "STRING".
	//
	// Example: "STRING" (huggingface.Parameters[0].Type)
	Type string `json:"type"`

	// Required indicates whether the parameter is required to be passed in.
	//
	// Example: true (huggingface.Parameters[0].Required)
	Required bool `json:"required"`

	// DefaultValue is the default value of the parameter, or an empty string if there is none.
	// Will always be in string form even if Type is "number".
	//
	// Example: "" (huggingface.Parameters[0].DefaultValue)
	DefaultValue string `json:"defaultValue"`

	// Validation holds validations that may be done on the inputted value.
	// Commonly either nil, or contains a "numericRange" key with [min, max].
	//
	// Example: nil (huggingface.Parameters[0].Validation — no validation)
	Validation []map[string]any `json:"validation"`

	// Help is any additional help text/information about the parameter.
	//
	// Example: "The name of your Hugging Face dedicated endpoint, the first part of the Endpoint URL."
	//   (huggingface.Parameters[0].Help)
	Help string `json:"help"`
}

// EmbeddingProviderProviderParameterInfo contains info about any additional, arbitrary parameter
// a provider may take in. May or may not be required. Extends EmbeddingProviderModelParameterInfo
// with display metadata.
//
// Example:
//
//	// openai.Parameters[0]:
//	EmbeddingProviderProviderParameterInfo{
//	    EmbeddingProviderModelParameterInfo: EmbeddingProviderModelParameterInfo{
//	        Name:         "projectId",
//	        Type:         "STRING",
//	        Required:     false,
//	        DefaultValue: "",
//	        Validation:   nil,
//	        Help:         "Optional, OpenAI Project ID. If provided passed as `OpenAI-Project` header.",
//	    },
//	    DisplayName: "Organization ID",
//	    Hint:        "Add an (optional) organization ID",
//	}
type EmbeddingProviderProviderParameterInfo struct {
	EmbeddingProviderModelParameterInfo

	// DisplayName is the UI display name for the parameter.
	//
	// Example: "Organization ID" (openai.Parameters[0].DisplayName)
	DisplayName string `json:"displayName"`

	// Hint is a short usage hint for the parameter.
	//
	// Example: "Add an (optional) organization ID" (openai.Parameters[0].Hint)
	Hint string `json:"hint"`
}

// EmbeddingProviderModelInfo describes a specific model offered by an embedding provider.
//
// May include an "endpoint-defined-model" for some providers, such as huggingfaceDedicated,
// where the model may be truly arbitrary.
//
// Example:
//
//	// nvidia.Models[0]:
//	EmbeddingProviderModelInfo{Name: "NV-Embed-QA", VectorDimension: ptr.To(1024), Parameters: nil}
//
//	// huggingfaceDedicated.Models[0]:
//	EmbeddingProviderModelInfo{Name: "endpoint-defined-model", VectorDimension: nil, Parameters: [...]}
type EmbeddingProviderModelInfo struct {
	// Name is the name of the model to use.
	// May be "endpoint-defined-model" for providers like huggingfaceDedicated.
	//
	// Example:
	//
	//	// openai.Models[0].Name
	//	"text-embedding-3-small"
	//
	//	// huggingfaceDedicated.Models[0].Name
	//	"endpoint-defined-model"
	Name string `json:"name"`

	// VectorDimension is the preset, exact vector dimension to be used (if applicable).
	// If nil, a vectorDimension parameter will be present in Parameters instead.
	//
	// Example:
	//
	//	// openai.Models[3].VectorDimension (text-embedding-ada-002)
	//	ptr.To(1536)
	//
	//	// huggingfaceDedicated.Models[0].VectorDimension (endpoint-defined-model)
	//	nil
	VectorDimension *int `json:"vectorDimension"`

	// Parameters are any additional, arbitrary parameters the model may take in.
	//
	// Example:
	//
	//	// openai.Models[0].Parameters[0] (text-embedding-3-small):
	//	EmbeddingProviderModelParameterInfo{
	//	    Name:         "vectorDimension",
	//	    Type:         "number",
	//	    Required:     true,
	//	    DefaultValue: "1536",
	//	    Validation:   []map[string]any{{"numericRange": []int{2, 1536}}},
	//	    Help:         "Vector dimension to use in the database and when calling OpenAI.",
	//	}
	Parameters []EmbeddingProviderModelParameterInfo `json:"parameters"`

	// ApiModelSupport describes the API support status and lifecycle of the model.
	//
	// Example:
	//
	//	// openai.Models[0].ApiModelSupport
	//	EmbeddingProviderModelApiSupportInfo{Status: ModelLifecycleStatusSupported}
	ApiModelSupport EmbeddingProviderModelApiSupportInfo `json:"apiModelSupport"`
}

// EmbeddingProviderModelApiSupportInfo describes the API support status and lifecycle of a model.
//
// Example:
//
//	// openai.Models[0].ApiModelSupport:
//	EmbeddingProviderModelApiSupportInfo{Status: ModelLifecycleStatusSupported}
type EmbeddingProviderModelApiSupportInfo struct {
	// Status is the current lifecycle status of the model.
	//
	// Example: ModelLifecycleStatusSupported (openai.Models[0].ApiModelSupport.Status)
	Status ModelLifecycleStatus `json:"status"`
}
