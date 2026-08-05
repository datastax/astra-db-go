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
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/options"
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
			opts: options.CreateCollection().UpdateVector(
				options.Vector().SetDimension(1024).UpdateService(
					options.VectorService().SetProvider("openai").SetModelName("text-embedding-3-small"),
				),
			),
			wantErr: false,
		},
		{
			name: "neither provider nor modelName set",
			opts: options.CreateCollection().UpdateVector(
				options.Vector().SetDimension(1024).UpdateService(
					options.VectorService(),
				),
			),
			wantErr: false,
		},
		{
			name: "provider only",
			opts: options.CreateCollection().UpdateVector(
				options.Vector().SetDimension(1024).UpdateService(
					options.VectorService().SetProvider("openai"),
				),
			),
			wantErr: true,
		},
		{
			name: "modelName only",
			opts: options.CreateCollection().UpdateVector(
				options.Vector().SetDimension(1024).UpdateService(
					options.VectorService().SetModelName("text-embedding-3-small"),
				),
			),
			wantErr: true,
		},
		{
			name: "no service at all",
			opts: options.CreateCollection().UpdateVector(
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

func TestUpdateAPIOptionsBuilder(t *testing.T) {
	// Builder style
	opts, err := options.MergeAndValidate(
		options.CollectionDeleteMany().
			UpdateAPIOptions(options.API().SetToken("override-token")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.APIOptions == nil {
		t.Fatal("expected APIOptions to be set")
	}
	if token, _ := opts.APIOptions.GetTokenProvider().Token(context.Background()); token != "override-token" {
		t.Errorf("expected token 'override-token', got %q", token)
	}
}

func TestUpdateAPIOptionsRawStruct(t *testing.T) {
	token := "raw-token"
	opts, err := options.MergeAndValidate(
		options.CollectionDeleteMany().
			UpdateAPIOptions(&options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.APIOptions == nil {
		t.Fatal("expected APIOptions to be set")
	}
	if gotToken, _ := opts.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != "raw-token" {
		t.Errorf("expected token 'raw-token', got %q", gotToken)
	}
}

func TestAPIOptionsNotSerialized(t *testing.T) {
	// Verify APIOptions (json:"-") does not leak into JSON for any of the 6 structs
	token := "secret"
	structs := []any{
		options.CollectionFindOptions{APIOptions: &options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}},
		options.CollectionUpdateOneOptions{APIOptions: &options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}},
		options.CollectionUpdateManyOptions{APIOptions: &options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}},
		options.CollectionDeleteOneOptions{APIOptions: &options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}},
		options.CollectionDeleteManyOptions{APIOptions: &options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}},
		options.CollectionFindOneAndUpdateOptions{APIOptions: &options.APIOptions{TokenProvider: options.NewStaticTokenProvider(token)}},
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

func TestUpdateAPIOptionsAllBuilders(t *testing.T) {
	// Verify UpdateAPIOptions works on all 6 collection builders
	token := "t"
	apiOpt := options.API().SetToken(token)

	// CollectionFind
	f, err := options.MergeAndValidate(options.CollectionFind().UpdateAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionFind: %v", err)
	}
	if f.APIOptions == nil {
		t.Fatal("CollectionFind: APIOptions not set")
	}
	if gotToken, _ := f.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != token {
		t.Errorf("CollectionFind: expected token %q, got %q", token, gotToken)
	}

	// CollectionUpdateOne
	u1, err := options.MergeAndValidate(options.CollectionUpdateOne().UpdateAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionUpdateOne: %v", err)
	}
	if u1.APIOptions == nil {
		t.Fatal("CollectionUpdateOne: APIOptions not set")
	}
	if gotToken, _ := u1.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != token {
		t.Errorf("CollectionUpdateOne: expected token %q, got %q", token, gotToken)
	}

	// CollectionUpdateMany
	um, err := options.MergeAndValidate(options.CollectionUpdateMany().UpdateAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionUpdateMany: %v", err)
	}
	if um.APIOptions == nil {
		t.Fatal("CollectionUpdateMany: APIOptions not set")
	}
	if gotToken, _ := um.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != token {
		t.Errorf("CollectionUpdateMany: expected token %q, got %q", token, gotToken)
	}

	// CollectionDeleteOne
	d1, err := options.MergeAndValidate(options.CollectionDeleteOne().UpdateAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionDeleteOne: %v", err)
	}
	if d1.APIOptions == nil {
		t.Fatal("CollectionDeleteOne: APIOptions not set")
	}
	if gotToken, _ := d1.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != token {
		t.Errorf("CollectionDeleteOne: expected token %q, got %q", token, gotToken)
	}

	// CollectionDeleteMany
	dm, err := options.MergeAndValidate(options.CollectionDeleteMany().UpdateAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionDeleteMany: %v", err)
	}
	if dm.APIOptions == nil {
		t.Fatal("CollectionDeleteMany: APIOptions not set")
	}
	if gotToken, _ := dm.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != token {
		t.Errorf("CollectionDeleteMany: expected token %q, got %q", token, gotToken)
	}

	// CollectionFindOneAndUpdate
	fu, err := options.MergeAndValidate(options.CollectionFindOneAndUpdate().UpdateAPIOptions(apiOpt))
	if err != nil {
		t.Fatalf("CollectionFindOneAndUpdate: %v", err)
	}
	if fu.APIOptions == nil {
		t.Fatal("CollectionFindOneAndUpdate: APIOptions not set")
	}
	if gotToken, _ := fu.APIOptions.GetTokenProvider().Token(context.Background()); gotToken != token {
		t.Errorf("CollectionFindOneAndUpdate: expected token %q, got %q", token, gotToken)
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
