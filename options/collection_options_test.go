package options_test

import (
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/options"
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
