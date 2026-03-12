package options_test

import (
	"testing"

	"github.com/datastax/astra-db-go/options"
)

func TestIndexingOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    options.Builder[options.CreateCollectionOptions]
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
			_, err := options.MergeOptions(tt.opts)
			if tt.wantErr && err == nil {
				// We expected an error but got nil
				t.Errorf("options.MergeOptions(): was expecting error. Got: %v", err)
			} else if !tt.wantErr && err != nil {
				// We weren't expecting an error but got one
				t.Errorf("options.MergeOptions(): wasn't expecting error. Got: %v", err)
			}
		})
	}
}
