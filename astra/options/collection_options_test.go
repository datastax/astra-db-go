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
	"encoding/json"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/options"
)

func TestIndexingOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    options.CreateCollectionOption
		wantErr bool
	}{
		{
			name: "allow only",
			opts: options.CreateCollection().SetIndexing(&options.IndexingOptions{
				Allow: []string{"field1", "field2"},
			}),
			wantErr: false,
		},
		{
			name: "deny only",
			opts: options.CreateCollection().SetIndexing(&options.IndexingOptions{
				Deny: []string{"field3", "field4"},
			}),
			wantErr: false,
		},
		{
			name: "allow and deny",
			opts: options.CreateCollection().SetIndexing(&options.IndexingOptions{
				Allow: []string{"field1"},
				Deny:  []string{"field2"},
			}),
			wantErr: true,
		},
		{
			name:    "fluent version with allow",
			opts:    options.CreateCollection().SetIndexingAllow("field1", "field2"),
			wantErr: false,
		},
		{
			name:    "fluent version with allow and deny",
			opts:    options.CreateCollection().SetIndexingAllow("field1", "field2").SetIndexingDeny("field3", "field4"),
			wantErr: true,
		},
		{
			name:    "empty",
			opts:    options.CreateCollection(),
			wantErr: false,
		},
		{
			name: "no methods",
			opts: &options.CreateCollectionOptions{
				DefaultId: &options.CollectionDefaultIdOptions{},
				Vector:    &options.VectorOptions{},
				Indexing: &options.IndexingOptions{
					Allow: []string{},
					Deny:  []string{},
				},
				Lexical: &options.LexicalOptions{},
				Rerank:  &options.RerankOptions{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := options.MergeAndValidate(tt.opts)
			if tt.wantErr && err == nil {
				// We expected an error but got nil
				t.Errorf("options.MergeAndValidate(): was expecting error. Got: %v", err)
			} else if !tt.wantErr && err != nil {
				// We weren't expecting an error but got one
				t.Errorf("options.MergeAndValidate(): wasn't expecting error. Got: %v", err)
			}
		})
	}
}

func TestVectorServiceOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    options.CreateCollectionOption
		wantErr bool
	}{
		{
			name: "both provider and modelName set",
			opts: options.CreateCollection().SetVector(
				options.Vector().SetDimension(1024).SetService(
					options.VectorService().SetProvider("openai").SetModelName("text-embedding-3-small"),
				),
			),
			wantErr: false,
		},
		{
			name: "neither provider nor modelName set",
			opts: options.CreateCollection().SetVector(
				options.Vector().SetDimension(1024).SetService(
					options.VectorService(),
				),
			),
			wantErr: false,
		},
		{
			name: "provider only",
			opts: options.CreateCollection().SetVector(
				options.Vector().SetDimension(1024).SetService(
					options.VectorService().SetProvider("openai"),
				),
			),
			wantErr: true,
		},
		{
			name: "modelName only",
			opts: options.CreateCollection().SetVector(
				options.Vector().SetDimension(1024).SetService(
					options.VectorService().SetModelName("text-embedding-3-small"),
				),
			),
			wantErr: true,
		},
		{
			name: "no service at all",
			opts: options.CreateCollection().SetVector(
				options.Vector().SetDimension(1024),
			),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := options.MergeAndValidate(tt.opts)
			if tt.wantErr && err == nil {
				t.Errorf("options.MergeAndValidate(): was expecting error. Got: %v", err)
			} else if !tt.wantErr && err != nil {
				t.Errorf("options.MergeAndValidate(): wasn't expecting error. Got: %v", err)
			}
		})
	}
}

func TestDeleteManyOptionsTimeout(t *testing.T) {
	timeout := 3 * time.Minute

	// Builder style
	opts, err := options.MergeAndValidate(options.CollectionDeleteMany().SetTimeout(timeout))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Timeout == nil || *opts.Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, opts.Timeout)
	}

	// Raw struct style
	opts2, err := options.MergeAndValidate(&options.CollectionDeleteManyOptions{Timeout: &timeout})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts2.Timeout == nil || *opts2.Timeout != timeout {
		t.Errorf("expected timeout %v via raw struct, got %v", timeout, opts2.Timeout)
	}
}

func TestUpdateManyOptionsTimeout(t *testing.T) {
	timeout := 3 * time.Minute

	// Builder style
	opts, err := options.MergeAndValidate(options.CollectionUpdateMany().SetTimeout(timeout))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Timeout == nil || *opts.Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, opts.Timeout)
	}

	// Raw struct style
	opts2, err := options.MergeAndValidate(&options.CollectionUpdateManyOptions{Timeout: &timeout})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts2.Timeout == nil || *opts2.Timeout != timeout {
		t.Errorf("expected timeout %v via raw struct, got %v", timeout, opts2.Timeout)
	}
}

func TestDeleteManyOptionsTimeoutNotSerialized(t *testing.T) {
	timeout := 3 * time.Minute
	opts := options.CollectionDeleteManyOptions{Timeout: &timeout}
	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	// json:"-" means Timeout should not appear in the JSON
	if string(b) != "{}" {
		t.Errorf("expected empty JSON object, got %s", string(b))
	}
}

func TestSetAPIOptionsBuilder(t *testing.T) {
	// Builder style
	opts, err := options.MergeAndValidate(
		options.CollectionDeleteMany().
			SetAPIOptions(options.API().SetToken("override-token")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.APIOptions == nil {
		t.Fatal("expected APIOptions to be set")
	}
	if opts.APIOptions.GetToken() != "override-token" {
		t.Errorf("expected token 'override-token', got %q", opts.APIOptions.GetToken())
	}
}

func TestSetAPIOptionsRawStruct(t *testing.T) {
	token := "raw-token"
	opts, err := options.MergeAndValidate(
		options.CollectionDeleteMany().
			SetAPIOptions(&options.APIOptions{Token: &token}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.APIOptions == nil {
		t.Fatal("expected APIOptions to be set")
	}
	if opts.APIOptions.GetToken() != "raw-token" {
		t.Errorf("expected token 'raw-token', got %q", opts.APIOptions.GetToken())
	}
}

func TestAPIOptionsNotSerialized(t *testing.T) {
	// Verify APIOptions (json:"-") does not leak into JSON for any of the 6 structs
	token := "secret"
	structs := []any{
		options.CollectionFindOptions{APIOptions: &options.APIOptions{Token: &token}},
		options.CollectionUpdateOneOptions{APIOptions: &options.APIOptions{Token: &token}},
		options.CollectionUpdateManyOptions{APIOptions: &options.APIOptions{Token: &token}},
		options.CollectionDeleteOneOptions{APIOptions: &options.APIOptions{Token: &token}},
		options.CollectionDeleteManyOptions{APIOptions: &options.APIOptions{Token: &token}},
		options.CollectionFindOneAndUpdateOptions{APIOptions: &options.APIOptions{Token: &token}},
	}
	for _, s := range structs {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("json.Marshal error: %v", err)
		}
		if string(b) != "{}" {
			t.Errorf("expected empty JSON, got %s for %T", string(b), s)
		}
	}
}

func TestSetAPIOptionsAllBuilders(t *testing.T) {
	// Verify SetAPIOptions works on all 6 collection builders
	token := "t"
	apiOpt := options.API().SetToken(token)

	// CollectionFind
	f, err := options.MergeAndValidate(options.CollectionFind().SetAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionFind: %v", err)
	}
	if f.APIOptions == nil || f.APIOptions.GetToken() != token {
		t.Error("CollectionFind: APIOptions not set")
	}

	// CollectionUpdateOne
	u1, err := options.MergeAndValidate(options.CollectionUpdateOne().SetAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionUpdateOne: %v", err)
	}
	if u1.APIOptions == nil || u1.APIOptions.GetToken() != token {
		t.Error("CollectionUpdateOne: APIOptions not set")
	}

	// CollectionUpdateMany
	um, err := options.MergeAndValidate(options.CollectionUpdateMany().SetAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionUpdateMany: %v", err)
	}
	if um.APIOptions == nil || um.APIOptions.GetToken() != token {
		t.Error("CollectionUpdateMany: APIOptions not set")
	}

	// CollectionDeleteOne
	d1, err := options.MergeAndValidate(options.CollectionDeleteOne().SetAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionDeleteOne: %v", err)
	}
	if d1.APIOptions == nil || d1.APIOptions.GetToken() != token {
		t.Error("CollectionDeleteOne: APIOptions not set")
	}

	// CollectionDeleteMany
	dm, err := options.MergeAndValidate(options.CollectionDeleteMany().SetAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionDeleteMany: %v", err)
	}
	if dm.APIOptions == nil || dm.APIOptions.GetToken() != token {
		t.Error("CollectionDeleteMany: APIOptions not set")
	}

	// CollectionFindOneAndUpdate
	fu, err := options.MergeAndValidate(options.CollectionFindOneAndUpdate().SetAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionFindOneAndUpdate: %v", err)
	}
	if fu.APIOptions == nil || fu.APIOptions.GetToken() != token {
		t.Error("CollectionFindOneAndUpdate: APIOptions not set")
	}
}

func TestCollectionOptionsEmptySort(t *testing.T) {
	// This test is just to validate that, even though we aren't using pointers,
	// the Sort field is nil by default.
	builder := options.CollectionFind().SetIncludeSimilarity(true)
	opts, err := options.MergeAndValidate(builder)
	if err != nil {
		t.Errorf("options.MergeAndValidate(): wasn't expecting error. Got: %v", err)
	}
	if opts.Sort != nil {
		t.Errorf("Expected nil Sort. Got %v.", opts.Sort)
	}
	b, err := json.Marshal(opts)
	if err != nil {
		t.Errorf("json.Marshal(): wasn't expecting error. Got: %v", err)
	}
	expected := `{"includeSimilarity":true}`
	if string(b) != expected {
		t.Errorf("Expected JSON %s. Got: %s", expected, string(b))
	}
}
