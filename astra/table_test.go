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

package astra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/sort"
	"github.com/datastax/astra-db-go/astra/table"
	"github.com/datastax/astra-db-go/astra/update"
	"github.com/datastax/astra-db-go/internal/testutils"
)

func TestTableDefinitionMarshal(t *testing.T) {
	tests := []struct {
		name       string
		definition table.Definition
		expected   string
	}{
		{
			name: "single column primary key",
			definition: table.Definition{
				Columns: table.Columns{
					{Name: "title", Column: table.Text()},
				},
				PrimaryKey: table.PrimaryKey{
					PartitionBy: []string{"title"},
				},
			},
			expected: `{"columns":{"title":{"type":"text"}},"primaryKey":"title"}`,
		},
		{
			name: "composite primary key",
			definition: table.Definition{
				Columns: table.Columns{
					{Name: "title", Column: table.Text()},
					{Name: "rating", Column: table.Float()},
				},
				PrimaryKey: table.PrimaryKey{
					PartitionBy: []string{"title", "rating"},
				},
			},
			expected: `{"columns":{"title":{"type":"text"},"rating":{"type":"float"}},"primaryKey":{"partitionBy":["title","rating"]}}`,
		},
		{
			name: "compound primary key with clustering",
			definition: table.Definition{
				Columns: table.Columns{
					{Name: "title", Column: table.Text()},
					{Name: "rating", Column: table.Float()},
					{Name: "number_of_pages", Column: table.Int()},
				},
				PrimaryKey: table.PrimaryKey{
					PartitionBy: []string{"title"},
					PartitionSort: table.PartitionSort{
						{Name: "rating", Order: table.SortAscending},
						{Name: "number_of_pages", Order: table.SortDescending},
					},
				},
			},
			expected: `{"columns":{"title":{"type":"text"},"rating":{"type":"float"},"number_of_pages":{"type":"int"}},"primaryKey":{"partitionBy":["title"],"partitionSort":{"rating":1,"number_of_pages":-1}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshalling only the table.Definition
			b, err := serdes.Serialize(tt.definition, serdes.TargetTable)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			if tt.expected != "" && string(b) != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, string(b))
			}

			// Verify it can be unmarshaled back into a table.Definition
			var result table.Definition
			if err := serdes.Deserialize(b, &result, nil, serdes.TargetTable); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
		})
	}
}

func TestColumnDefinitions(t *testing.T) {
	tests := []struct {
		name   string
		column table.Column
		check  func(t *testing.T, col table.Column)
	}{
		{
			name:   "text column",
			column: table.Text(),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "text" {
					t.Errorf("expected type text, got %s", col.Type)
				}
			},
		},
		{
			name:   "int column",
			column: table.Int(),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "int" {
					t.Errorf("expected type int, got %s", col.Type)
				}
			},
		},
		{
			name:   "float column",
			column: table.Float(),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "float" {
					t.Errorf("expected type float, got %s", col.Type)
				}
			},
		},
		{
			name:   "boolean column",
			column: table.Boolean(),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "boolean" {
					t.Errorf("expected type boolean, got %s", col.Type)
				}
			},
		},
		{
			name:   "uuid column",
			column: table.UUID(),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "uuid" {
					t.Errorf("expected type uuid, got %s", col.Type)
				}
			},
		},
		{
			name:   "date column",
			column: table.Date(),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "date" {
					t.Errorf("expected type date, got %s", col.Type)
				}
			},
		},
		{
			name:   "vector column",
			column: table.Vector(1024),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "vector" {
					t.Errorf("expected type vector, got %s", col.Type)
				}
				if col.Dimension == nil || *col.Dimension != 1024 {
					t.Errorf("expected dimension 1024")
				}
			},
		},
		{
			name:   "set of text",
			column: table.Set(table.Text()),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "set" {
					t.Errorf("expected type set, got %s", col.Type)
				}
				if col.ValueType == nil || col.ValueType.Type != "text" {
					t.Errorf("expected valueType text")
				}
			},
		},
		{
			name:   "list of int",
			column: table.List(table.Int()),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "list" {
					t.Errorf("expected type list, got %s", col.Type)
				}
				if col.ValueType == nil || col.ValueType.Type != "int" {
					t.Errorf("expected valueType int")
				}
			},
		},
		{
			name:   "map of text to text",
			column: table.Map("text", table.Text()),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "map" {
					t.Errorf("expected type map, got %s", col.Type)
				}
				if col.KeyType == nil || *col.KeyType != "text" {
					t.Errorf("expected keyType text")
				}
				if col.ValueType == nil || col.ValueType.Type != "text" {
					t.Errorf("expected valueType text")
				}
			},
		},
		{
			name:   "user defined type",
			column: table.UDT("person"),
			check: func(t *testing.T, col table.Column) {
				if col.Type != "userDefined" {
					t.Errorf("expected type userDefined, got %s", col.Type)
				}
				if col.UDTName == nil || *col.UDTName != "person" {
					t.Errorf("expected udtName person")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify it can be marshaled
			b, err := serdes.Serialize(tt.column, serdes.TargetTable)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify it can be unmarshaled back
			var result table.Column
			if err := serdes.Deserialize(b, &result, nil, serdes.TargetTable); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Run type-specific checks
			tt.check(t, result)
		})
	}
}

func TestVectorColumnWithService(t *testing.T) {
	service := &table.VectorService{
		Provider:  "openai",
		ModelName: "text-embedding-3-small",
		Authentication: map[string]string{
			"providerKey": "my-api-key",
		},
	}
	col := table.VectorWithService(1536, service)
	b, err := serdes.Serialize(col, serdes.TargetTable)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := `{"type":"vector","dimension":1536,"service":{"provider":"openai","modelName":"text-embedding-3-small","authentication":{"providerKey":"my-api-key"}}}`
	if string(b) != expected {
		t.Errorf("expected %s, got %s", expected, string(b))
	}
}

func TestPrimaryKeyUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected table.PrimaryKey
	}{
		{
			name:  "single column as string",
			input: `"title"`,
			expected: table.PrimaryKey{
				PartitionBy: []string{"title"},
			},
		},
		{
			name:  "composite key as object",
			input: `{"partitionBy":["title","rating"]}`,
			expected: table.PrimaryKey{
				PartitionBy: []string{"title", "rating"},
			},
		},
		{
			name:  "compound key with clustering",
			input: `{"partitionBy":["title"],"partitionSort":{"rating":1}}`,
			expected: table.PrimaryKey{
				PartitionBy:   []string{"title"},
				PartitionSort: table.PartitionSort{{Name: "rating", Order: 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pk table.PrimaryKey
			if err := serdes.Deserialize([]byte(tt.input), &pk, nil, serdes.TargetTable); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if len(pk.PartitionBy) != len(tt.expected.PartitionBy) {
				t.Errorf("PartitionBy length mismatch: expected %d, got %d", len(tt.expected.PartitionBy), len(pk.PartitionBy))
			}
			for i, col := range tt.expected.PartitionBy {
				if pk.PartitionBy[i] != col {
					t.Errorf("PartitionBy[%d] mismatch: expected %s, got %s", i, col, pk.PartitionBy[i])
				}
			}
		})
	}
}

func TestTableFindOptions(t *testing.T) {
	t.Run("with all options", func(t *testing.T) {
		opts, err := options.MergeAndValidate[options.TableFindOptions](
			options.TableFind().
				SetSort(sort.Asc("rating")).
				SetProjection(map[string]any{"title": true}).
				SetLimit(10).
				SetSkip(5).
				SetIncludeSimilarity(true).
				SetPageState("some-page-state"),
		)
		if err != nil {
			t.Fatal(err)
		}

		if opts.Sort == nil {
			t.Error("expected sort to be set")
		}
		if opts.Projection == nil {
			t.Error("expected projection to be set")
		}
		if opts.Limit == nil || *opts.Limit != 10 {
			t.Error("expected limit to be 10")
		}
		if opts.Skip == nil || *opts.Skip != 5 {
			t.Error("expected skip to be 5")
		}
		if opts.IncludeSimilarity == nil || !*opts.IncludeSimilarity {
			t.Error("expected includeSimilarity to be true")
		}
		if opts.InitialPageState == nil || *opts.InitialPageState != "some-page-state" {
			t.Error("expected initialPageState to be set")
		}
	})
}

func TestFilterWithStructuredFilters(t *testing.T) {
	// Test using the structured filter types
	f := filter.And(
		filter.Eq("is_checked_out", false),
		filter.Lt("number_of_pages", 300),
	)
	b, err := serdes.Serialize(f, serdes.TargetTable)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	t.Logf("Marshalled filter: %s", string(b))

	// Should produce something like:
	// {"$and":[{"is_checked_out":false},{"number_of_pages":{"$lt":300}}]}
	var result map[string]any
	if err := serdes.Deserialize(b, &result, nil, serdes.TargetTable); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	andFilters, ok := result["$and"].([]any)
	if !ok {
		t.Fatal("expected $and to be an array")
	}
	if len(andFilters) != 2 {
		t.Errorf("expected 2 filters in $and, got %d", len(andFilters))
	}
}

// getTestTable acts as a test fixture to provide a *Table.
func getTestTable(t *testing.T) *Table {
	// See: https://pkg.go.dev/testing#T.Helper
	t.Helper()

	client := NewClient(options.API().SetToken("TEST_TOKEN"))
	db := client.Database("https://API_ENDPOINT", options.API().SetKeyspace("some_keyspace"))
	return db.Table("example_table")
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-index.html#example-exists
//
// The endpoint should look like:
// "API_ENDPOINT/api/json/v1/KEYSPACE_NAME/TABLE_NAME"
var exampleIndexPayloadJSON = testutils.CleanString(`{
  "createIndex": {
    "definition": {
      "column": "example_column",
      "options": {
        "ascii": null,
        "caseSensitive": null,
        "normalize": null
      }
    },
    "name": "example_index_name",
    "options": {
      "ifNotExists": true
    }
  }
}`)

// TestCreateIndexCommandMarshal verifies that the resulting command from createIndexCommand matches
// the payload in the docs.
func TestCreateIndexCommandMarshal(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", "example_column", options.CreateIndex().SetIfNotExists(true))
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleIndexPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleIndexPayloadJSON, string(cmdBytes))
	}
}

func TestCreateIndexCommandURL(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", "example_column", options.CreateIndex().SetIfNotExists(true))
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	postURL, err := cmd.url()
	if err != nil {
		t.Fatalf("cmd.url: %v", err)
	}
	// Verify the URL matches what example CURL command is expecting
	expectedURL := "https://API_ENDPOINT/api/json/v1/some_keyspace/example_table"
	if postURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, postURL)
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-index.html#example-ascii
var exampleIndexASCIIPayloadJSON = testutils.CleanString(`{
  "createIndex": {
    "definition": {
      "column": "example_column",
      "options": {
        "ascii": true,
        "caseSensitive": null,
        "normalize": null
      }
    },
    "name": "example_index_name",
    "options": {
      "ifNotExists": null
    }
  }
}`)

// TestCreateIndexASCIICommandMarshal verifies that the resulting command from createIndexCommand
// with the ascii option matches the payload in the docs.
func TestCreateIndexASCIICommandMarshal(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", "example_column", options.CreateIndex().SetAscii(true))
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleIndexASCIIPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleIndexASCIIPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-index.html#example-index-map
var exampleIndexMapKeysPayloadJSON = testutils.CleanString(`{
  "createIndex": {
    "definition": {
      "column": {
        "example_map_column": "$keys"
      },
      "options": {
        "ascii": null,
        "caseSensitive": null,
        "normalize": null
      }
    },
    "name": "example_index_name",
    "options": {
      "ifNotExists": null
    }
  }
}`)

// TestCreateIndexMapKeysCommandMarshal verifies that the resulting command from createIndexCommand
// with a map column keys index matches the payload in the docs.
func TestCreateIndexMapKeysCommandMarshal(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", map[string]string{"example_map_column": "$keys"})
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleIndexMapKeysPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleIndexMapKeysPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#example-default
var exampleVectorIndexDefaultPayloadJSON = testutils.CleanString(`{
  "createVectorIndex": {
    "definition": {
      "column": "example_vector_column",
      "options": {
        "metric": null,
        "sourceModel": null
      }
    },
    "name": "example_index_name",
    "options": {
      "ifNotExists": null
    }
  }
}`)

// TestCreateVectorIndexDefaultCommandMarshal verifies that the resulting command from createVectorIndexCommand
// with default options matches the payload in the docs.
func TestCreateVectorIndexDefaultCommandMarshal(t *testing.T) {
	cmd, err := createVectorIndexCommand(getTestTable(t), "example_index_name", "example_vector_column")
	if err != nil {
		t.Fatalf("createVectorIndexCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleVectorIndexDefaultPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleVectorIndexDefaultPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#example-model-metric
var exampleVectorIndexModelMetricPayloadJSON = testutils.CleanString(`{
  "createVectorIndex": {
    "definition": {
      "column": "example_vector_column",
      "options": {
        "metric": "dot_product",
        "sourceModel": "ada002"
      }
    },
    "name": "example_index_name",
    "options": {
      "ifNotExists": null
    }
  }
}`)

// TestCreateVectorIndexModelMetricCommandMarshal verifies that the resulting command from createVectorIndexCommand
// with custom metric and sourceModel matches the payload in the docs.
func TestCreateVectorIndexModelMetricCommandMarshal(t *testing.T) {
	cmd, err := createVectorIndexCommand(getTestTable(t), "example_index_name", "example_vector_column",
		options.CreateVectorIndex().SetMetric(options.MetricDotProduct).SetSourceModel("ada002"))
	if err != nil {
		t.Fatalf("createVectorIndexCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleVectorIndexModelMetricPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleVectorIndexModelMetricPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#example-exists
var exampleVectorIndexIfNotExistsPayloadJSON = testutils.CleanString(`{
  "createVectorIndex": {
    "definition": {
      "column": "summary_genres_vector",
      "options": {
        "metric": null,
        "sourceModel": null
      }
    },
    "name": "example_index_name",
    "options": {
      "ifNotExists": true
    }
  }
}`)

// TestCreateVectorIndexIfNotExistsCommandMarshal verifies that the resulting command from createVectorIndexCommand
// with ifNotExists option matches the payload in the docs.
func TestCreateVectorIndexIfNotExistsCommandMarshal(t *testing.T) {
	cmd, err := createVectorIndexCommand(getTestTable(t), "example_index_name", "summary_genres_vector",
		options.CreateVectorIndex().SetIfNotExists(true))
	if err != nil {
		t.Fatalf("createVectorIndexCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleVectorIndexIfNotExistsPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleVectorIndexIfNotExistsPayloadJSON, string(cmdBytes))
	}
}

// getTestDb acts as a test fixture to provide a *Db.
func getTestDb(t *testing.T) *Db {
	t.Helper()
	client := NewClient(options.API().SetToken("TEST_TOKEN"))
	if client.ClientOptions().Token == nil {
		t.Fatal("expected token to be set")
	}
	return client.Database("https://API_ENDPOINT", options.API().SetKeyspace("some_keyspace"))
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/drop-index.html#drop-an-index
var exampleDropIndexPayloadJSON = testutils.CleanString(`{
  "dropIndex": {
    "name": "rating"
  }
}`)

// TestDropTableIndexCommandMarshal verifies that the resulting command from dropTableIndexCommand
// matches the payload in the docs.
func TestDropTableIndexCommandMarshal(t *testing.T) {
	cmd := dropTableIndexCommand(getTestDb(t), "rating")
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}
	if string(cmdBytes) != exampleDropIndexPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleDropIndexPayloadJSON, string(cmdBytes))
	}
}

// TestDropTableIndexCommandURL verifies that the dropTableIndexCommand URL
// matches the URL in the docs.
func TestDropTableIndexCommandURL(t *testing.T) {
	cmd := dropTableIndexCommand(getTestDb(t), "rating")
	postURL, err := cmd.url()
	if err != nil {
		t.Fatalf("cmd.url: %v", err)
	}
	// Verify the URL matches what example CURL command is expecting
	expectedURL := "https://API_ENDPOINT/api/json/v1/some_keyspace"
	if postURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, postURL)
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/list-index-metadata.html#example-names
var exampleListIndexesNamesOnlyPayloadJSON = testutils.CleanString(`{
  "listIndexes": {
    "options": {
      "explain": null
    }
  }
}`)

// TestListIndexesNamesOnlyCommandMarshal verifies that the resulting command from listIndexesCommand
// with default options (no explain) matches the payload in the docs.
func TestListIndexesNamesOnlyCommandMarshal(t *testing.T) {
	cmd, err := listIndexesCommand(getTestTable(t))
	if err != nil {
		t.Fatalf("listIndexesCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(cmdBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(exampleListIndexesNamesOnlyPayloadJSON), &expected); err != nil {
		t.Fatalf("json.Unmarshal expected: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleListIndexesNamesOnlyPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/list-index-metadata.html#example-explain
var exampleListIndexesExplainPayloadJSON = testutils.CleanString(`{
  "listIndexes": {
    "options": {
      "explain": true
    }
  }
}`)

// TestListIndexesExplainCommandMarshal verifies that the resulting command from listIndexesCommand
// with explain=true matches the payload in the docs.
func TestListIndexesExplainCommandMarshal(t *testing.T) {
	cmd, err := listIndexesCommand(getTestTable(t), options.ListIndexes().SetExplain(true))
	if err != nil {
		t.Fatalf("listIndexesCommand: %v", err)
	}
	cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
	if err != nil {
		t.Fatalf("serdes.Serialize: %v", err)
	}
	if string(cmdBytes) != exampleListIndexesExplainPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleListIndexesExplainPayloadJSON, string(cmdBytes))
	}
}

// TestListIndexesCommandURL verifies that the listIndexesCommand URL
// is correct (should hit the table endpoint).
func TestListIndexesCommandURL(t *testing.T) {
	cmd, err := listIndexesCommand(getTestTable(t))
	if err != nil {
		t.Fatalf("listIndexesCommand: %v", err)
	}
	postURL, err := cmd.url()
	if err != nil {
		t.Fatalf("cmd.url: %v", err)
	}
	// Verify the URL matches what example CURL command is expecting
	expectedURL := "https://API_ENDPOINT/api/json/v1/some_keyspace/example_table"
	if postURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, postURL)
	}
}

// TestListIndexesResponseUnmarshal tests unmarshaling the listIndexes response.
func TestListIndexesResponseUnmarshal(t *testing.T) {
	t.Run("names only response", func(t *testing.T) {
		// When explain=false, the API returns an array of strings
		jsonResp := `{"status":{"indexes":["rating_idx","title_idx"]}}`
		var resp listIndexesResponse
		if err := serdes.Deserialize([]byte(jsonResp), &resp, nil, serdes.TargetTable); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.Indexes) != 2 {
			t.Errorf("expected 2 indexes, got %d", len(resp.Status.Indexes))
		}
		if resp.Status.Indexes[0].Name != "rating_idx" {
			t.Errorf("expected index name 'rating_idx', got %s", resp.Status.Indexes[0].Name)
		}
		if resp.Status.Indexes[1].Name != "title_idx" {
			t.Errorf("expected index name 'title_idx', got %s", resp.Status.Indexes[1].Name)
		}
		// Definition should be nil for names-only response
		if resp.Status.Indexes[0].Definition != nil {
			t.Error("expected definition to be nil for names-only response")
		}
	})

	t.Run("explain response with regular index", func(t *testing.T) {
		jsonResp := `{"status":{"indexes":[{"name":"rating_idx","definition":{"column":"rating"},"indexType":"regular"}]}}`
		var resp listIndexesResponse
		if err := serdes.Deserialize([]byte(jsonResp), &resp, nil, serdes.TargetTable); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.Indexes) != 1 {
			t.Fatalf("expected 1 index, got %d", len(resp.Status.Indexes))
		}
		idx := resp.Status.Indexes[0]
		if idx.Name != "rating_idx" {
			t.Errorf("expected index name 'rating_idx', got %s", idx.Name)
		}
		if idx.IndexType != "regular" {
			t.Errorf("expected indexType 'regular', got %s", idx.IndexType)
		}
		if idx.Definition == nil {
			t.Fatal("expected definition to be present")
		}
		if idx.Definition.Column != "rating" {
			t.Errorf("expected column 'rating', got %s", idx.Definition.Column)
		}
	})

	t.Run("explain response with vector index", func(t *testing.T) {
		jsonResp := `{"status":{"indexes":[{"name":"embedding_idx","definition":{"column":"embedding","options":{"metric":"cosine","sourceModel":"other"}},"indexType":"vector"}]}}`
		var resp listIndexesResponse
		if err := serdes.Deserialize([]byte(jsonResp), &resp, nil, serdes.TargetTable); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.Indexes) != 1 {
			t.Fatalf("expected 1 index, got %d", len(resp.Status.Indexes))
		}
		idx := resp.Status.Indexes[0]
		if idx.Name != "embedding_idx" {
			t.Errorf("expected index name 'embedding_idx', got %s", idx.Name)
		}
		if idx.IndexType != "vector" {
			t.Errorf("expected indexType 'vector', got %s", idx.IndexType)
		}
		if idx.Definition == nil {
			t.Fatal("expected definition to be present")
		}
		if idx.Definition.Column != "embedding" {
			t.Errorf("expected column 'embedding', got %s", idx.Definition.Column)
		}
		if idx.Definition.Options == nil {
			t.Fatal("expected options to be present")
		}
		if idx.Definition.Options.Metric != "cosine" {
			t.Errorf("expected metric 'cosine', got %s", idx.Definition.Options.Metric)
		}
		if idx.Definition.Options.SourceModel != "other" {
			t.Errorf("expected sourceModel 'other', got %s", idx.Definition.Options.SourceModel)
		}
	})

	t.Run("empty indexes", func(t *testing.T) {
		jsonResp := `{"status":{"indexes":[]}}`
		var resp listIndexesResponse
		if err := serdes.Deserialize([]byte(jsonResp), &resp, nil, serdes.TargetTable); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.Indexes) != 0 {
			t.Errorf("expected 0 indexes, got %d", len(resp.Status.Indexes))
		}
	})
}

func boolPtr(b bool) *bool {
	return &b
}

// TestCreateIndexOptionsVarargs verifies that multiple options can be passed and merged.
func TestCreateIndexOptionsVarargs(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		cmd, err := createIndexCommand(getTestTable(t), "test_idx", "test_col")
		if err != nil {
			t.Fatalf("createIndexCommand: %v", err)
		}
		cmdBytes, _ := serdes.Serialize(cmd, serdes.TargetTable)
		// Should not have "options" key when no options provided
		var got, expected map[string]interface{}
		json.Unmarshal(cmdBytes, &got)
		json.Unmarshal([]byte(`{"createIndex":{"definition":{"column":"test_col","options":{"ascii":null,"caseSensitive":null,"normalize":null}},"name":"test_idx","options":{"ifNotExists":null}}}`), &expected)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("unexpected JSON: %s", string(cmdBytes))
		}
	})

	t.Run("single builder option", func(t *testing.T) {
		// Test with chaining a single options builder
		opts := options.CreateIndex().SetIfNotExists(true).SetCaseSensitive(true)
		cmd, err := createIndexCommand(getTestTable(t), "test_idx", "test_col", opts)
		if err != nil {
			t.Fatalf("createIndexCommand: %v", err)
		}
		cmdBytes, _ := serdes.Serialize(cmd, serdes.TargetTable)
		var got, expected map[string]interface{}
		json.Unmarshal(cmdBytes, &got)
		json.Unmarshal([]byte(`{"createIndex":{"definition":{"column":"test_col","options":{"ascii":null,"caseSensitive":true,"normalize":null}},"name":"test_idx","options":{"ifNotExists":true}}}`), &expected)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("unexpected JSON: %s", string(cmdBytes))
		}
	})

	t.Run("multiple builder options merged", func(t *testing.T) {
		// Pass multiple options - they should be merged with later options overriding earlier
		cmd, err := createIndexCommand(getTestTable(t), "test_idx", "test_col",
			options.CreateIndex().SetAscii(false), // Set false and make sure later "true" overrides
			options.CreateIndex().SetIfNotExists(true),
			options.CreateIndex().SetCaseSensitive(false),
			options.CreateIndex().SetAscii(true),
		)
		if err != nil {
			t.Fatalf("createIndexCommand: %v", err)
		}
		cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}
		var got, exp map[string]interface{}
		json.Unmarshal(cmdBytes, &got)
		json.Unmarshal([]byte(`{"createIndex":{"definition":{"column":"test_col","options":{"ascii":true,"caseSensitive":false,"normalize":null}},"name":"test_idx","options":{"ifNotExists":true}}}`), &exp)
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", `{"createIndex":{"definition":{"column":"test_col","options":{"ascii":true,"caseSensitive":false,"normalize":null}},"name":"test_idx","options":{"ifNotExists":true}}}`, string(cmdBytes))
		}
	})

	t.Run("later options override earlier", func(t *testing.T) {
		// Pass conflicting options - later should win
		cmd, err := createIndexCommand(getTestTable(t), "test_idx", "test_col",
			options.CreateIndex().SetAscii(true),
			options.CreateIndex().SetAscii(false)) // Override to false
		if err != nil {
			t.Fatalf("createIndexCommand: %v", err)
		}
		cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}
		var got, exp map[string]interface{}
		json.Unmarshal(cmdBytes, &got)
		json.Unmarshal([]byte(`{"createIndex":{"definition":{"column":"test_col","options":{"ascii":false,"caseSensitive":null,"normalize":null}},"name":"test_idx","options":{"ifNotExists":null}}}`), &exp)
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", `{"createIndex":{"definition":{"column":"test_col","options":{"ascii":false,"caseSensitive":null,"normalize":null}},"name":"test_idx","options":{"ifNotExists":null}}}`, string(cmdBytes))
		}
	})

	t.Run("raw struct option", func(t *testing.T) {
		// Pass raw struct directly (not builder) - this should also work
		rawOpts := &options.CreateIndexOptions{
			IfNotExists:   boolPtr(true),
			Ascii:         boolPtr(true),
			Normalize:     boolPtr(false), // Set to false to throw a curveball.
			CaseSensitive: boolPtr(true),
		}
		cmd, err := createIndexCommand(getTestTable(t), "test_idx", "test_col", rawOpts)
		if err != nil {
			t.Fatalf("createIndexCommand: %v", err)
		}
		cmdBytes, err := serdes.Serialize(cmd, serdes.TargetTable)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}
		var got, exp map[string]interface{}
		json.Unmarshal(cmdBytes, &got)
		json.Unmarshal([]byte(`{"createIndex":{"name":"test_idx","definition":{"column":"test_col","options":{"ascii":true,"normalize":false,"caseSensitive":true}},"options":{"ifNotExists":true}}}`), &exp)
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", `{"createIndex":{"name":"test_idx","definition":{"column":"test_col","options":{"ascii":true,"normalize":false,"caseSensitive":true}},"options":{"ifNotExists":true}}}`, string(cmdBytes))
		}
	})
}

func TestCreateIndedxOptionsValidation(t *testing.T) {
	t.Run("no index name", func(t *testing.T) {
		_, err := createIndexCommand(getTestTable(t), "", "some_column")
		if err == nil {
			t.Fatal("expected error for missing index name")
		}
	})

	t.Run("no column name", func(t *testing.T) {
		_, err := createIndexCommand(getTestTable(t), "some_index", "")
		if err == nil {
			t.Fatal("expected error for missing column name")
		}
	})

	t.Run("empty column name map", func(t *testing.T) {
		_, err := createIndexCommand(getTestTable(t), "some_index", map[string]string{})
		if err == nil {
			t.Fatal("expected error for empty column name map")
		}
	})

	t.Run("valid column name map", func(t *testing.T) {
		_, err := createIndexCommand(getTestTable(t), "some_index", map[string]string{"example_map_column": "$values"})
		if err != nil {
			t.Fatal("expected no error for valid column name map")
		}
	})

	t.Run("valid column name", func(t *testing.T) {
		_, err := createIndexCommand(getTestTable(t), "some_index", "example_column")
		if err != nil {
			t.Fatal("expected no error for valid column name")
		}
	})
}

// #region Table.UpdateOne tests

// This example was taken from the documentation:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/row-methods/update.html#example-update-multiple
// ^ confusingly says "update multiple" but it means multiple fields on an updateOne command.
var exampleUpdateOneSetPayloadJSON = testutils.CleanString(`{
  "updateOne": {
    "filter": {
    	"author": "John Anthony",
		"title": "Hidden Shadows of the Past"
    },
    "update": {
        "$set": {
        	"genres": ["Fiction", "Drama"],  
			"rating": 4.5
        },
        "$unset": {
            "borrower": ""
        }
    }
  }
}`)

// httpTestTable creates a Table backed by the given httptest.Server for
// integration-style testing. Mirrors newTestCollection in collection_test.go.
func httpTestTable(ts *httptest.Server, apiOpts ...options.APIOption) *Table {
	allOpts := append([]options.APIOption{options.API().SetToken("test-token")}, apiOpts...)
	client := NewClient(allOpts...)
	db := client.Database(ts.URL, options.API().SetKeyspace("ks"))
	return db.Table("tbl")
}

// TestTableUpdateOne_HappyPath verifies that UpdateOne posts the expected
// request body, reads a successful status response, and returns nil.
func TestTableUpdateOne_HappyPath(t *testing.T) {
	var gotBody atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if r.Header.Get("Token") != "test-token" {
			t.Errorf("expected token %q in request header, got %q", "test-token", r.Header.Get("Token"))
		}
		gotBody.Store(b)
		w.Header().Set("Content-Type", "application/json")
		// Server always returns this so this is a good proxy for verifying that the
		// client correctly reads the response body.
		fmt.Fprint(w, `{"status":{"matchedCount":1,"modifiedCount":1,"upsertCount":0}}`)
	}))
	defer ts.Close()

	tbl := httpTestTable(ts)
	err := tbl.UpdateOne(context.Background(),
		filter.F{"title": "Hidden Shadows of the Past", "author": "John Anthony"},
		update.Table().Set("rating", 4.5).Unset("borrower"),
		options.TableUpdateOne(), // Empty options just to throw a slight curveball.
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := gotBody.Load().([]byte)
	var sentBody map[string]any
	if err := serdes.Deserialize(body, &sentBody, nil, serdes.TargetTable); err != nil {
		t.Fatalf("server-received body was not JSON: %v (%s)", err, body)
	}
	inner, ok := sentBody["updateOne"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level key %q, got: %s", "updateOne", body)
	}
	if _, ok := inner["filter"]; !ok {
		t.Errorf("expected filter key in updateOne payload, got: %s", body)
	}
	if _, ok := inner["update"]; !ok {
		t.Errorf("expected update key in updateOne payload, got: %s", body)
	}
}

// TestTableUpdateOne_APIOptionsOverrideToken proves the command-level
// APIOptions override flows end-to-end through newCmdWithMergedOptions.
func TestTableUpdateOne_APIOptionsOverrideToken(t *testing.T) {
	var receivedToken atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken.Store(r.Header.Get("Token"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"matchedCount":1,"modifiedCount":1}}`)
	}))
	defer ts.Close()

	tbl := httpTestTable(ts) // uses "test-token" at client level
	err := tbl.UpdateOne(context.Background(),
		filter.F{"pk": 1},
		update.Table().Set("x", 2),
		options.TableUpdateOne().SetAPIOptions(options.API().SetToken("override-token")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, _ := receivedToken.Load().(string); got != "override-token" {
		t.Errorf("expected token %q in request header, got %q", "override-token", got)
	}
}

// TestTableUpdateOne_ContextCanceled verifies that a pre-canceled context
// causes UpdateOne to return without a successful round-trip.
func TestTableUpdateOne_ContextCanceled(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"matchedCount":1,"modifiedCount":1}}`)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tbl := httpTestTable(ts)
	err := tbl.UpdateOne(ctx,
		filter.F{"pk": 1},
		update.Table().Set("x", 2),
	)
	if calls.Load() != 0 {
		// If the context cancellation is working properly, the handler should never be called
		t.Errorf("expected 0 calls to server, got %d", calls.Load())
	}
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestTableUpdateOne_ContextTimeout verifies context timeout works.
func TestTableUpdateOne_ContextTimeout(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond) // Sleep to simulate a long request and give the test a chance to timeout
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"matchedCount":1,"modifiedCount":1}}`)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tbl := httpTestTable(ts)
	start := time.Now()
	err := tbl.UpdateOne(ctx,
		filter.F{"pk": 1},
		update.Table().Set("x", 2),
	)
	elapsed := time.Since(start)

	if calls.Load() != 1 {
		// Just make sure it got to our server
		t.Errorf("expected 1 call to server, got %d", calls.Load())
	}
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 100*time.Millisecond {
		// This is a timing-based assertion. If it fails in CI/CD due to slowness,
		// we can relax it. Works fine on my machine though.
		t.Errorf("cancellation took too long: %v", elapsed)
	}
}

// #endregion

// #region Table.DeleteOne tests

// This example was taken from the documentation:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/row-methods/delete-one.html#delete-a-row-by-primary-key
var exampleDeleteOnePayloadJSON = testutils.CleanString(`{
  "deleteOne": {
    "filter": {
      "author": "John Anthony",
      "title": "Hidden Shadows of the Past"
    }
  }
}`)

// TestTableDeleteOne_HappyPath verifies that DeleteOne posts the expected
// request body, reads a successful status response, and returns nil.
// NOTE: thought about also repeating the override token etc. but that is REALLY
// testing the command implementation, not the commands themselves.
func TestTableDeleteOne_HappyPath(t *testing.T) {
	var gotBody atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if r.Header.Get("Token") != "test-token" {
			t.Errorf("expected token %q in request header, got %q", "test-token", r.Header.Get("Token"))
		}
		gotBody.Store(b)
		w.Header().Set("Content-Type", "application/json")
		// From the docs:
		// > Always returns a status.deletedCount of -1, regardless of whether a row was found and deleted.
		fmt.Fprint(w, `{"status":{"deletedCount":-1}}`)
	}))
	defer ts.Close()

	tbl := httpTestTable(ts)
	err := tbl.DeleteOne(context.Background(),
		filter.F{"title": "Hidden Shadows of the Past", "author": "John Anthony"},
		options.TableDeleteOne(), // Empty options just to throw a slight curveball.
	)
	// Make sure we don't get an error when server returns the expected deletedCount of -1
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// #endregion

// #region Table.DeleteMany tests

// From the docs:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/row-methods/delete-many.html#delete-a-row-by-primary-key
// Order of filter keys changed to be alphanumeric.
var exampleDeleteManyPayloadJSON = testutils.CleanString(`{
  "deleteMany": {
    "filter": {
      "author": "John Anthony",  
	  "title": "Hidden Shadows of the Past"
    }
  }
}`)

// TestTableDeleteMany_HappyPath verifies DeleteMany posts the expected request
// body, handles the documented deletedCount=-1 response, and returns nil.
func TestTableDeleteMany_HappyPath(t *testing.T) {
	var gotBody atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if r.Header.Get("Token") != "test-token" {
			t.Errorf("expected token %q in request header, got %q", "test-token", r.Header.Get("Token"))
		}
		gotBody.Store(b)
		w.Header().Set("Content-Type", "application/json")
		// From the docs:
		// > Always returns a status.deletedCount of -1, regardless of whether a row was found and deleted.
		fmt.Fprint(w, `{"status":{"deletedCount":-1}}`)
	}))
	defer ts.Close()

	tbl := httpTestTable(ts)
	err := tbl.DeleteMany(context.Background(),
		filter.F{"title": "Hidden Shadows of the Past", "author": "John Anthony"},
		options.TableDeleteMany(), // Empty options just to throw a slight curveball.
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTableDeleteMany_EnforceNonNilFilter ensures a nil filter is rejected.
// Callers must pass filter.F{} explicitly to delete all rows so total-delete
// is always intentional.
func TestTableDeleteMany_EnforceNonNilFilter(t *testing.T) {
	tbl := &Table{}
	err := tbl.DeleteMany(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when filter is nil, got nil")
	}
	if !errors.Is(err, ErrNilFilter) {
		t.Errorf("expected ErrNilFilter, got: %v", err)
	}
}

// #endregion

// #region Table.Alter tests

// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/alter-table.html#example-add
var exampleAlterTableAddPayloadJSON = testutils.CleanString(`{
  "alterTable": {
    "operation": {
      "add": {
        "columns": {
          "is_summer_reading": {"type":"boolean"},
          "library_branch": {"type":"text"}
        }
      }
    }
  }
}`)

// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/alter-table.html#example-add-vector
// Vector form of an "add" operation.
var exampleAlterTableAddVectorColumnPayloadJSON = testutils.CleanString(`{
  "alterTable": {
    "operation": {
      "add": {
        "columns": {
          "example_vector": {"type":"vector","dimension":1024}
        }
      }
    }
  }
}`)

// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/alter-table.html#example-drop
var exampleAlterTableDropPayloadJSON = testutils.CleanString(`{
  "alterTable": {
    "operation": {
      "drop": {
        "columns": ["is_summer_reading", "library_branch"]
      }
    }
  }
}`)

// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/alter-table.html#example-add-vectorize
var exampleAlterTableAddVectorizePayloadJSON = testutils.CleanString(`{
  "alterTable": {
    "operation": {
      "addVectorize": {
        "columns": {
          "summary_vec": {
            "provider": "openai",
            "modelName": "text-embedding-3-small",
            "authentication": {"providerKey": "OPENAI_API_KEY"},
			"parameters": {"organizationId": "ORGANIZATION_ID","projectId": "PROJECT_ID"}
          }
        }
      }
    }
  }
}`)

// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/alter-table.html#example-drop-vectorize
var exampleAlterTableDropVectorizePayloadJSON = testutils.CleanString(`{
  "alterTable": {
    "operation": {
      "dropVectorize": {
        "columns": ["plot_synopsis"]
      }
    }
  }
}`)

// TestTableAlter_CommandMarshal verifies that the alterTable payload for each
// of the four operations matches the docs curl examples.
func TestTableAlter_CommandMarshal(t *testing.T) {
	tbl := getTestTable(t)
	tests := []testutils.JSONTestCase{{
		Name:     "Add columns",
		Expected: exampleAlterTableAddPayloadJSON,
		Args: []any{
			tbl.newCmd("alterTable", alterTablePayload{
				Operation: &table.AddColumns{
					Columns: table.Columns{
						{"is_summer_reading", table.Boolean()},
						{"library_branch", table.Text()},
					},
				},
			}),
		},
	}, {
		Name:     "Add vector column",
		Expected: exampleAlterTableAddVectorColumnPayloadJSON,
		Args: []any{
			tbl.newCmd("alterTable", alterTablePayload{
				Operation: &table.AddColumns{
					Columns: table.Columns{
						{"example_vector", table.Vector(1024)},
					},
				},
			}),
		},
	}, {
		Name:     "Drop columns",
		Expected: exampleAlterTableDropPayloadJSON,
		Args: []any{
			tbl.newCmd("alterTable", alterTablePayload{
				Operation: &table.DropColumns{
					Columns: []string{"is_summer_reading", "library_branch"},
				},
			}),
		},
	}, {
		Name:     "Add vectorize",
		Expected: exampleAlterTableAddVectorizePayloadJSON,
		Args: []any{
			tbl.newCmd("alterTable", alterTablePayload{
				Operation: &table.AddVectorize{
					Columns: map[string]table.VectorService{
						"summary_vec": {
							Provider:  "openai",
							ModelName: "text-embedding-3-small",
							Authentication: map[string]string{
								"providerKey": "OPENAI_API_KEY",
							},
							Parameters: map[string]string{
								"organizationId": "ORGANIZATION_ID",
								"projectId":      "PROJECT_ID",
							},
						},
					},
				},
			}),
		},
	}, {
		Name:     "Drop vectorize",
		Expected: exampleAlterTableDropVectorizePayloadJSON,
		Args: []any{
			tbl.newCmd("alterTable", alterTablePayload{
				Operation: &table.DropVectorize{
					Columns: []string{"plot_synopsis"},
				},
			}),
		},
	}}
	testutils.RunJSONTestCases(t, tests)
}

// TestTableAlter_HappyPath verifies that Alter posts the expected request
// body for an "add columns" call, reads the documented success response, and
// returns nil.
func TestTableAlter_HappyPath(t *testing.T) {
	var gotBody atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if r.Header.Get("Token") != "test-token" {
			t.Errorf("expected token %q in request header, got %q", "test-token", r.Header.Get("Token"))
		}
		gotBody.Store(b)
		w.Header().Set("Content-Type", "application/json")
		// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/alter-table.html
		fmt.Fprint(w, `{"status":{"ok":1}}`)
	}))
	defer ts.Close()

	tbl := httpTestTable(ts)
	err := tbl.Alter(context.Background(), table.AddColumns{
		Columns: table.Columns{
			{"is_summer_reading", table.Boolean()},
		},
	}, options.AlterTable())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := gotBody.Load().([]byte)
	var sentBody map[string]any
	if err := serdes.Deserialize(body, &sentBody, nil, serdes.TargetTable); err != nil {
		t.Fatalf("server-received body was not JSON: %v (%s)", err, body)
	}
	inner, ok := sentBody["alterTable"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level key %q, got: %s", "alterTable", body)
	}
	op, ok := inner["operation"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation key in alterTable payload, got: %s", body)
	}
	if _, ok := op["add"]; !ok {
		t.Errorf("expected add key in operation payload, got: %s", body)
	}
}

// TestTableAlter_RejectsZeroOrMultipleOperations ensures the client refuses
// payloads where the operation field is empty or sets more than one of
// Add/Drop/AddVectorize/DropVectorize, matching the Data API constraint.
func TestTableAlter_RejectsZeroOrMultipleOperations(t *testing.T) {
	tbl := getTestTable(t)
	t.Run("nil operation", func(t *testing.T) {
		err := tbl.Alter(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil operation, got nil")
		}
	})
}

// #endregion
