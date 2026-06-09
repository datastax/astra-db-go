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

package options_test

import (
	"context"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
	"pgregory.net/rapid"
)

func TestAdditiveMerging(t *testing.T) {
	// Test that multiple SetAPIOptions calls merge additively
	opts := options.ListCollections().
		SetAPIOptions(options.API().SetToken("token1")).
		SetAPIOptions(options.API().SetKeyspace("ks1"))

	merged := options.Merge(opts)

	if merged.APIOptions == nil {
		t.Fatal("expected APIOptions to be non-nil")
	}
	if gotToken, _ := merged.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != "token1" {
		t.Errorf("expected token1, got %q", gotToken)
	}
	if merged.APIOptions.GetKeyspace() != "ks1" {
		t.Errorf("expected ks1, got %q", merged.APIOptions.GetKeyspace())
	}
}

func TestExplicitReplace(t *testing.T) {
	// Test that Replace overwrites instead of merging
	opts := options.ListCollections().
		SetAPIOptions(options.API().SetToken("token1").SetKeyspace("ks1")).
		SetAPIOptions(options.Replace(options.API().SetToken("token2")))

	merged := options.Merge(opts)

	if merged.APIOptions == nil {
		t.Fatal("expected APIOptions to be non-nil")
	}
	if gotToken, _ := merged.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != "token2" {
		t.Errorf("expected token2, got %q", gotToken)
	}
	// Keyspace should have been wiped and then set to default by Merge initializing new APIOptions
	if merged.APIOptions.GetKeyspace() != "default_keyspace" {
		t.Errorf("expected default keyspace after replace, got %q", merged.APIOptions.GetKeyspace())
	}
}

func TestAdditiveHeaderMerging(t *testing.T) {
	// Proves that map merging works across multiple builder applications
	// when using the manual SetHeader override.
	opts := options.API().
		SetHeader("X-A", "1").
		SetHeader("X-B", "2")

	moreOpts := options.API().
		SetHeader("X-C", "3").
		SetHeader("X-A", "overridden")

	resolved := options.Merge(opts, moreOpts)

	if len(resolved.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(resolved.Headers))
	}
	if resolved.Headers["X-A"] != "overridden" {
		t.Errorf("expected X-A to be overridden, got %q", resolved.Headers["X-A"])
	}
}

func TestMapOverwriteRule(t *testing.T) {
	// Proves that SetHeaders (plural) overwrites the entire map.
	opts := options.API().SetHeader("X-A", "1")
	overwrite := options.API().SetHeaders(map[string]string{"X-B": "2"})

	resolved := options.Merge(opts, overwrite)

	if len(resolved.Headers) != 1 {
		t.Errorf("expected map to be overwritten, got length %d", len(resolved.Headers))
	}
	if resolved.Headers["X-B"] != "2" {
		t.Error("expected only X-B to be present")
	}
}

func TestNestedInitialization(t *testing.T) {
	// Verifies that "lifted" fields deep in a command struct correctly
	// initialize their container structs (APIOptions, TimeoutOptions).
	opts := options.ListCollections().SetAPIOptions(options.API().SetRequestTimeout(45 * time.Second))

	resolved := options.Merge(opts)

	if resolved.APIOptions == nil {
		t.Fatal("APIOptions should have been auto-initialized")
	}
	if resolved.APIOptions.Timeout == nil {
		t.Fatal("APIOptions.Timeout should have been auto-initialized")
	}
	if resolved.APIOptions.GetRequestTimeout() != 45*time.Second {
		t.Errorf("expected 45s, got %v", resolved.APIOptions.GetRequestTimeout())
	}
}

func TestHierarchyInheritance(t *testing.T) {
	// 1. Client sets a token and a global header
	client := astra.NewClient(
		options.API().SetToken("client-token"),
		options.API().SetHeader("X-Global", "true"),
	)

	// 2. Database overrides keyspace and adds a header
	db := client.Database("https://db.astra.com",
		options.API().SetKeyspace("db-keyspace"),
		options.API().SetHeader("X-DB", "true"),
	)

	// 3. Collection adds a timeout and another header
	coll := db.Collection("my-coll", options.API().
		SetTimeout(options.Timeout().SetRequest(10*time.Second)).
		SetHeader("X-Coll", "true"),
	)

	// 4. Resolve at the final level
	resolved := coll.ClientOptions()

	// ASSERTIONS:
	if gotToken, _ := resolved.GetTokenProvider().Token(context.Background()); gotToken != "client-token" {
		t.Errorf("Client token lost: got %q", gotToken)
	}
	if resolved.GetKeyspace() != "db-keyspace" {
		t.Errorf("DB keyspace lost: got %q", resolved.GetKeyspace())
	}
	if resolved.GetRequestTimeout() != 10*time.Second {
		t.Errorf("Coll timeout lost: got %v", resolved.GetRequestTimeout())
	}
	if len(resolved.Headers) != 3 {
		t.Errorf("Headers not merged correctly: expected 3, got %d", len(resolved.Headers))
	}
}

// Property-based tests

// genAPIOption generates a single options.APIOption builder.
func genAPIOption(t *rapid.T) options.APIOption {
	return rapid.OneOf[options.APIOption](
		rapid.Custom(func(t *rapid.T) options.APIOption {
			return options.API().SetToken(rapid.String().Draw(t, "token"))
		}),
		rapid.Custom(func(t *rapid.T) options.APIOption {
			return options.API().SetKeyspace(rapid.String().Draw(t, "keyspace"))
		}),
		rapid.Custom(func(t *rapid.T) options.APIOption {
			return options.API().SetAPIVersion(rapid.String().Draw(t, "version"))
		}),
		rapid.Custom(func(t *rapid.T) options.APIOption {
			k := rapid.StringMatching(`^[a-zA-Z0-9-]+$`).Draw(t, "headerKey")
			v := rapid.String().Draw(t, "headerVal")
			return options.API().SetHeader(k, v)
		}),
	).Draw(t, "apiOption")
}

func TestProperty_MergeResultIsNeverNil(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		opts := rapid.SliceOf(rapid.Custom(genAPIOption)).Draw(t, "opts")
		res := options.Merge(opts...)
		if res == nil {
			t.Fatal("Merge returned nil")
		}
	})
}

func TestProperty_MergePrecedence(t *testing.T) {
	// Property: If we set a scalar value in the last builder,
	// it MUST be the value in the resolved struct.
	rapid.Check(t, func(t *rapid.T) {
		token := rapid.String().Draw(t, "finalToken")
		leading := rapid.SliceOf(rapid.Custom(genAPIOption)).Draw(t, "leading")
		final := options.API().SetToken(token)

		resolved := options.Merge(append(leading, final)...)

		if gotToken, _ := resolved.GetTokenProvider().Token(context.Background()); gotToken != token {
			t.Errorf("Precedence failed: expected %q, got %q", token, gotToken)
		}
	})
}

func TestProperty_AdditiveHeaderMerging(t *testing.T) {
	// Property: Headers from multiple layers should accumulate.
	rapid.Check(t, func(t *rapid.T) {
		h1_k := rapid.StringMatching(`^[a-zA-Z0-9-]+$`).Draw(t, "h1_k")
		h1_v := rapid.String().Draw(t, "h1_v")
		h2_k := rapid.StringMatching(`^[a-zA-Z0-9-]+$`).Filter(func(s string) bool { return s != h1_k }).Draw(t, "h2_k")
		h2_v := rapid.String().Draw(t, "h2_v")

		opt1 := options.API().SetHeader(h1_k, h1_v)
		opt2 := options.API().SetHeader(h2_k, h2_v)

		resolved := options.Merge(opt1, opt2)

		if resolved.Headers[h1_k] != h1_v {
			t.Errorf("header 1 lost: expected %q, got %q", h1_v, resolved.Headers[h1_k])
		}
		if resolved.Headers[h2_k] != h2_v {
			t.Errorf("header 2 lost: expected %q, got %q", h2_v, resolved.Headers[h2_k])
		}
	})
}

func TestProperty_ReplaceLaw(t *testing.T) {
	// Property: Merge(A, Replace(B)) == Merge(B)
	rapid.Check(t, func(t *rapid.T) {
		optA := rapid.SliceOf(rapid.Custom(genAPIOption)).Draw(t, "optA")
		optB := rapid.Custom(genAPIOption).Draw(t, "optB")

		res1 := options.Merge(append(optA, options.Replace(optB))...)
		res2 := options.Merge(optB)

		if diff := testlib.Diff(t, res1, res2); diff != "" {
			t.Errorf("Replace law violated (-merged_with_replace +merged_directly):\n%s", diff)
		}
	})
}

func TestProperty_DefaultsAreStable(t *testing.T) {
	// Property: Merge() is idempotent with respect to defaults.
	rapid.Check(t, func(t *rapid.T) {
		res1 := options.Merge[options.APIOptions]()
		res2 := options.Merge[options.APIOptions]()

		if diff := testlib.Diff(t, res1, res2); diff != "" {
			t.Errorf("Defaults not stable:\n%s", diff)
		}

		if res1.GetAPIVersion() != "v1" {
			t.Errorf("Default APIVersion missing: got %q", res1.GetAPIVersion())
		}
	})
}
