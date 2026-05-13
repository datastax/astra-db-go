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

	"github.com/datastax/astra-db-go/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/results"
)

func init() {
	t := []harness.IntegrationTest{
		{Name: "DbFindEmbeddingProvidersDefault", Run: DbFindEmbeddingProvidersDefault},
		{Name: "DbFindEmbeddingProvidersFilterAll", Run: DbFindEmbeddingProvidersFilterAll},
		{Name: "DbFindEmbeddingProvidersFilterSupported", Run: DbFindEmbeddingProvidersFilterSupported},
		{Name: "DbFindEmbeddingProvidersFilterDeprecated", Run: DbFindEmbeddingProvidersFilterDeprecated},
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
