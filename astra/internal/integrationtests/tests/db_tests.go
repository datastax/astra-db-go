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

package tests

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/datastax/astra-db-go/astra/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
)

func init() {
	t := []harness.IntegrationTest{
		{Name: "DbFindEmbeddingProvidersDefault", Run: DbFindEmbeddingProvidersDefault},
		{Name: "DbFindEmbeddingProvidersFilterAll", Run: DbFindEmbeddingProvidersFilterAll},
		{Name: "DbFindEmbeddingProvidersFilterSupported", Run: DbFindEmbeddingProvidersFilterSupported},
		{Name: "DbFindEmbeddingProvidersFilterDeprecated", Run: DbFindEmbeddingProvidersFilterDeprecated},
		{Name: "DbFindEmbeddingProvidersOpenAI", Run: DbFindEmbeddingProvidersOpenAI},
		{Name: "DbFindEmbeddingProvidersNvidia", Run: DbFindEmbeddingProvidersNvidia},
		{Name: "DbFindEmbeddingProvidersHuggingFaceDedicated", Run: DbFindEmbeddingProvidersHuggingFaceDedicated},
	}
	harness.Register(t...)
}

// DbFindEmbeddingProvidersDefault calls FindEmbeddingProviders with no options and validates
// the response structure: at least one provider, each with a non-empty DisplayName/URL,
// at least one model with a non-empty name.
func DbFindEmbeddingProvidersDefault(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()
	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}

	result, err := dbAdmin.FindEmbeddingProviders(ctx)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders failed: %w", err)
	}

	if len(result.EmbeddingProviders) == 0 {
		return fmt.Errorf("expected at least one embedding provider, got none")
	}
	slog.Info("FindEmbeddingProviders", "providerCount", len(result.EmbeddingProviders))

	return validateEmbeddingProvidersResult(result)
}

// DbFindEmbeddingProvidersFilterAll calls FindEmbeddingProviders with ModelLifecycleStatusAll,
// which should include models of every lifecycle status. The result must be structurally valid
// and contain at least as many total models as the default (SUPPORTED-only) call.
func DbFindEmbeddingProvidersFilterAll(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()
	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}

	defaultResult, err := dbAdmin.FindEmbeddingProviders(ctx)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders (default) failed: %w", err)
	}

	allResult, err := dbAdmin.FindEmbeddingProviders(ctx,
		options.FindEmbeddingProviders().SetFilterModelStatus(options.ModelLifecycleStatusAll),
	)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders (all) failed: %w", err)
	}

	if err := validateEmbeddingProvidersResult(allResult); err != nil {
		return err
	}

	// The "all" result should have at least as many total models as the default (SUPPORTED).
	defaultTotal := countModels(defaultResult)
	allTotal := countModels(allResult)
	slog.Info("FindEmbeddingProviders model counts", "default(supported)", defaultTotal, "all", allTotal)
	if allTotal < defaultTotal {
		return fmt.Errorf("FilterAll returned fewer models (%d) than default (%d)", allTotal, defaultTotal)
	}

	return nil
}

// DbFindEmbeddingProvidersFilterSupported explicitly passes ModelLifecycleStatusSupported and
// verifies every returned model carries that status.
func DbFindEmbeddingProvidersFilterSupported(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()
	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}

	result, err := dbAdmin.FindEmbeddingProviders(ctx,
		options.FindEmbeddingProviders().SetFilterModelStatus(options.ModelLifecycleStatusSupported),
	)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders (supported) failed: %w", err)
	}

	if err := validateEmbeddingProvidersResult(result); err != nil {
		return err
	}

	for providerName, provider := range result.EmbeddingProviders {
		for _, model := range provider.Models {
			if model.ApiModelSupport.Status != results.ModelLifecycleStatusSupported {
				return fmt.Errorf(
					"provider %q model %q: expected status %q, got %q",
					providerName, model.Name,
					results.ModelLifecycleStatusSupported,
					model.ApiModelSupport.Status,
				)
			}
		}
	}

	slog.Info("All models have SUPPORTED status", "providerCount", len(result.EmbeddingProviders))
	return nil
}

// DbFindEmbeddingProvidersFilterDeprecated calls FindEmbeddingProviders filtering for
// DEPRECATED models. The call must succeed; any returned models must carry DEPRECATED status.
func DbFindEmbeddingProvidersFilterDeprecated(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()
	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}

	result, err := dbAdmin.FindEmbeddingProviders(ctx,
		options.FindEmbeddingProviders().SetFilterModelStatus(options.ModelLifecycleStatusDeprecated),
	)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders (deprecated) failed: %w", err)
	}

	totalModels := countModels(result)
	slog.Info("FindEmbeddingProviders (deprecated)", "providerCount", len(result.EmbeddingProviders), "modelCount", totalModels)

	// Any models that were returned must have DEPRECATED status.
	for providerName, provider := range result.EmbeddingProviders {
		for _, model := range provider.Models {
			if model.ApiModelSupport.Status != results.ModelLifecycleStatusDeprecated {
				return fmt.Errorf(
					"provider %q model %q: expected status %q, got %q",
					providerName, model.Name,
					results.ModelLifecycleStatusDeprecated,
					model.ApiModelSupport.Status,
				)
			}
		}
	}

	return nil
}

// validateEmbeddingProvidersResult checks the structural invariants of a
// FindEmbeddingProvidersResult that should hold for any non-error response.
func validateEmbeddingProvidersResult(result *results.FindEmbeddingProvidersResult) error {
	if result == nil {
		return fmt.Errorf("result is nil")
	}

	for name, provider := range result.EmbeddingProviders {
		if provider.DisplayName == "" {
			return fmt.Errorf("provider %q: DisplayName is empty", name)
		}
		if provider.URL != nil && *provider.URL == "" {
			return fmt.Errorf("provider %q: URL is empty string", name)
		}
		if provider.SupportedAuthentication == nil {
			return fmt.Errorf("provider %q: SupportedAuthentication is nil", name)
		}
		for _, model := range provider.Models {
			if model.Name == "" {
				return fmt.Errorf("provider %q: model has empty Name", name)
			}
		}
	}

	return nil
}

// countModels returns the total number of models across all providers in a result.
func countModels(result *results.FindEmbeddingProvidersResult) int {
	total := 0
	for _, provider := range result.EmbeddingProviders {
		total += len(provider.Models)
	}
	return total
}

// DbFindEmbeddingProvidersOpenAI verifies the concrete openai examples from the doc comments:
//   - EmbeddingProviders["openai"] => EmbeddingProviderInfo{DisplayName: "OpenAI", ...}
//   - openai.URL = "https://api.openai.com/v1/"
//   - openai models include text-embedding-3-small, text-embedding-3-large, text-embedding-ada-002
//   - openai.SupportedAuthentication["HEADER"]: Enabled=true, Tokens[0]={Accepted: "x-embedding-api-key", Forwarded: "Authorization"}
//   - openai.Parameters[0] (projectId): Type=STRING, Required=false, DisplayName="Organization ID", Hint="Add an (optional) organization ID"
//   - text-embedding-3-small.ApiModelSupport.Status = ModelLifecycleStatusSupported
//   - text-embedding-3-small.Parameters[vectorDimension]: Type="number", Required=true, DefaultValue="1536", Validation has "numericRange" key
func DbFindEmbeddingProvidersOpenAI(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()

	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}
	result, err := dbAdmin.FindEmbeddingProviders(ctx)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders failed: %w", err)
	}

	openai, ok := result.EmbeddingProviders["openai"]
	if !ok {
		return fmt.Errorf("openai provider not found in EmbeddingProviders map")
	}

	// DisplayName (EmbeddingProviderInfo.DisplayName example)
	if openai.DisplayName != "OpenAI" {
		return fmt.Errorf("openai.DisplayName: got %q, want %q", openai.DisplayName, "OpenAI")
	}

	// URL (EmbeddingProviderInfo.URL example)
	const wantURL = "https://api.openai.com/v1/"
	if openai.URL == nil || *openai.URL != wantURL {
		got := "<nil>"
		if openai.URL != nil {
			got = *openai.URL
		}
		return fmt.Errorf("openai.URL: got %q, want %q", got, wantURL)
	}

	// Model list (FindEmbeddingProvidersResult example)
	wantModels := []string{"text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"}
	modelSet := make(map[string]bool, len(openai.Models))
	for _, m := range openai.Models {
		modelSet[m.Name] = true
	}
	for _, name := range wantModels {
		if !modelSet[name] {
			return fmt.Errorf("openai: model %q not found", name)
		}
	}

	// HEADER auth (EmbeddingProviderAuthInfo and EmbeddingProviderTokenInfo examples)
	headerAuth, ok := openai.SupportedAuthentication["HEADER"]
	if !ok {
		return fmt.Errorf("openai.SupportedAuthentication[\"HEADER\"] not found")
	}
	if !headerAuth.Enabled {
		return fmt.Errorf("openai HEADER auth.Enabled: got false, want true")
	}
	if len(headerAuth.Tokens) == 0 {
		return fmt.Errorf("openai HEADER auth.Tokens: empty")
	}
	if headerAuth.Tokens[0].Accepted != "x-embedding-api-key" {
		return fmt.Errorf("openai HEADER auth.Tokens[0].Accepted: got %q, want %q", headerAuth.Tokens[0].Accepted, "x-embedding-api-key")
	}
	if headerAuth.Tokens[0].Forwarded != "Authorization" {
		return fmt.Errorf("openai HEADER auth.Tokens[0].Forwarded: got %q, want %q", headerAuth.Tokens[0].Forwarded, "Authorization")
	}

	// organizationId provider parameter (EmbeddingProviderProviderParameterInfo example)
	orgID := findProviderParam(openai.Parameters, "organizationId")
	if orgID == nil {
		return fmt.Errorf("openai.Parameters: %q not found", "organizationId")
	}
	if orgID.Type != "STRING" {
		return fmt.Errorf("openai.Parameters[organizationId].Type: got %q, want %q", orgID.Type, "STRING")
	}
	if orgID.Required {
		return fmt.Errorf("openai.Parameters[organizationId].Required: got true, want false")
	}
	if orgID.DefaultValue != "" {
		return fmt.Errorf("openai.Parameters[organizationId].DefaultValue: got %q, want %q", orgID.DefaultValue, "")
	}
	if orgID.DisplayName != "Organization ID" {
		return fmt.Errorf("openai.Parameters[organizationId].DisplayName: got %q, want %q", orgID.DisplayName, "Organization ID")
	}
	if orgID.Hint != "Add an (optional) organization ID" {
		return fmt.Errorf("openai.Parameters[organizationId].Hint: got %q, want %q", orgID.Hint, "Add an (optional) organization ID")
	}

	// text-embedding-3-small model details
	small := findModel(openai.Models, "text-embedding-3-small")
	if small == nil {
		return fmt.Errorf("openai: model %q not found", "text-embedding-3-small")
	}

	// ApiModelSupport.Status (EmbeddingProviderModelApiSupportInfo example)
	if small.ApiModelSupport.Status != results.ModelLifecycleStatusSupported {
		return fmt.Errorf("openai text-embedding-3-small.ApiModelSupport.Status: got %q, want %q",
			small.ApiModelSupport.Status, results.ModelLifecycleStatusSupported)
	}

	// vectorDimension model parameter (EmbeddingProviderModelParameterInfo example)
	dimParam := findModelParam(small.Parameters, "vectorDimension")
	if dimParam == nil {
		return fmt.Errorf("openai text-embedding-3-small.Parameters: %q not found", "vectorDimension")
	}
	if dimParam.Type != "number" {
		return fmt.Errorf("openai text-embedding-3-small.Parameters[vectorDimension].Type: got %q, want %q", dimParam.Type, "number")
	}
	if !dimParam.Required {
		return fmt.Errorf("openai text-embedding-3-small.Parameters[vectorDimension].Required: got false, want true")
	}
	if dimParam.DefaultValue != "1536" {
		return fmt.Errorf("openai text-embedding-3-small.Parameters[vectorDimension].DefaultValue: got %q, want %q", dimParam.DefaultValue, "1536")
	}
	if _, ok := dimParam.Validation["numericRange"]; !ok {
		return fmt.Errorf("openai text-embedding-3-small.Parameters[vectorDimension].Validation: missing %q key", "numericRange")
	}

	slog.Info("DbFindEmbeddingProvidersOpenAI passed", "modelCount", len(openai.Models))
	return nil
}

// DbFindEmbeddingProvidersNvidia verifies the concrete nvidia examples from the doc comments:
//   - nvidia model NV-Embed-QA: VectorDimension=1024, Parameters=nil (EmbeddingProviderModelInfo example)
//   - nvidia.SupportedAuthentication["NONE"] is enabled (SupportedAuthentication "NONE" description)
func DbFindEmbeddingProvidersNvidia(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()
	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}
	result, err := dbAdmin.FindEmbeddingProviders(ctx)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders failed: %w", err)
	}

	nvidia, ok := result.EmbeddingProviders["nvidia"]
	if !ok {
		return fmt.Errorf("nvidia provider not found in EmbeddingProviders map")
	}

	// NV-Embed-QA model (EmbeddingProviderModelInfo example)
	nvModel := findModel(nvidia.Models, "NV-Embed-QA")
	if nvModel == nil {
		return fmt.Errorf("nvidia: model %q not found", "NV-Embed-QA")
	}
	if nvModel.VectorDimension == nil {
		return fmt.Errorf("nvidia NV-Embed-QA.VectorDimension: got nil, want 1024")
	}
	if *nvModel.VectorDimension != 1024 {
		return fmt.Errorf("nvidia NV-Embed-QA.VectorDimension: got %d, want 1024", *nvModel.VectorDimension)
	}
	if len(nvModel.Parameters) != 0 {
		return fmt.Errorf("nvidia NV-Embed-QA.Parameters: got %d params, want none", len(nvModel.Parameters))
	}

	// NONE auth (SupportedAuthentication "NONE" description)
	noneAuth, ok := nvidia.SupportedAuthentication["NONE"]
	if !ok {
		return fmt.Errorf("nvidia.SupportedAuthentication[\"NONE\"] not found")
	}
	if !noneAuth.Enabled {
		return fmt.Errorf("nvidia NONE auth.Enabled: got false, want true")
	}

	slog.Info("DbFindEmbeddingProvidersNvidia passed", "vectorDimension", *nvModel.VectorDimension)
	return nil
}

// DbFindEmbeddingProvidersHuggingFaceDedicated verifies the huggingfaceDedicated examples from the doc comments:
//   - huggingfaceDedicated.URL uses the f-string template (EmbeddingProviderInfo.URL example)
//   - model "endpoint-defined-model": VectorDimension=nil (EmbeddingProviderModelInfo example)
//   - endpoint-defined-model parameter "vectorDimension": Type="number", Required=true, DefaultValue="" (EmbeddingProviderInfo.Models example)
func DbFindEmbeddingProvidersHuggingFaceDedicated(e *harness.TestEnv) error {
	ctx := context.Background()
	dbAdmin, err := e.DefaultDb().DatabaseAdmin()
	if err != nil {
		return fmt.Errorf("DatabaseAdmin() failed: %w", err)
	}
	result, err := dbAdmin.FindEmbeddingProviders(ctx)
	if err != nil {
		return fmt.Errorf("FindEmbeddingProviders failed: %w", err)
	}

	hf, ok := result.EmbeddingProviders["huggingfaceDedicated"]
	if !ok {
		return fmt.Errorf("huggingfaceDedicated provider not found in EmbeddingProviders map")
	}

	// URL (EmbeddingProviderInfo.URL example)
	const wantURL = "https://{endpointName}.{regionName}.{cloudName}.endpoints.huggingface.cloud/embeddings"
	if hf.URL == nil || *hf.URL != wantURL {
		got := "<nil>"
		if hf.URL != nil {
			got = *hf.URL
		}
		return fmt.Errorf("huggingfaceDedicated.URL: got %q, want %q", got, wantURL)
	}

	// "endpoint-defined-model" model (EmbeddingProviderModelInfo example)
	edModel := findModel(hf.Models, "endpoint-defined-model")
	if edModel == nil {
		return fmt.Errorf("huggingfaceDedicated: model %q not found", "endpoint-defined-model")
	}
	if edModel.VectorDimension != nil {
		return fmt.Errorf("huggingfaceDedicated endpoint-defined-model.VectorDimension: got %d, want nil", *edModel.VectorDimension)
	}

	// vectorDimension model parameter (EmbeddingProviderInfo.Models example)
	dimParam := findModelParam(edModel.Parameters, "vectorDimension")
	if dimParam == nil {
		return fmt.Errorf("huggingfaceDedicated endpoint-defined-model.Parameters: %q not found", "vectorDimension")
	}
	if dimParam.Type != "number" {
		return fmt.Errorf("huggingfaceDedicated endpoint-defined-model.Parameters[vectorDimension].Type: got %q, want %q", dimParam.Type, "number")
	}
	if !dimParam.Required {
		return fmt.Errorf("huggingfaceDedicated endpoint-defined-model.Parameters[vectorDimension].Required: got false, want true")
	}
	if dimParam.DefaultValue != "" {
		return fmt.Errorf("huggingfaceDedicated endpoint-defined-model.Parameters[vectorDimension].DefaultValue: got %q, want %q", dimParam.DefaultValue, "")
	}

	slog.Info("DbFindEmbeddingProvidersHuggingFaceDedicated passed")
	return nil
}

// findModel returns a pointer to the model with the given name, or nil if not found.
func findModel(models []results.EmbeddingProviderModelInfo, name string) *results.EmbeddingProviderModelInfo {
	for i := range models {
		if models[i].Name == name {
			return &models[i]
		}
	}
	return nil
}

// findProviderParam returns a pointer to the provider parameter with the given name, or nil if not found.
func findProviderParam(params []results.EmbeddingProviderProviderParameterInfo, name string) *results.EmbeddingProviderProviderParameterInfo {
	for i := range params {
		if params[i].Name == name {
			return &params[i]
		}
	}
	return nil
}

// findModelParam returns a pointer to the model parameter with the given name, or nil if not found.
func findModelParam(params []results.EmbeddingProviderModelParameterInfo, name string) *results.EmbeddingProviderModelParameterInfo {
	for i := range params {
		if params[i].Name == name {
			return &params[i]
		}
	}
	return nil
}
