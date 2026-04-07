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

package astradb

import (
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/sort"
	"github.com/datastax/astra-db-go/table"
)

func TestCreateTablePayloadMarshal(t *testing.T) {
	tests := []struct {
		name     string
		payload  createTablePayload
		expected string
	}{
		{
			name: "single column primary key",
			payload: createTablePayload{
				Name: "test_table",
				Definition: table.Definition{
					Columns: map[string]table.Column{
						"title": table.Text(),
					},
					PrimaryKey: table.PrimaryKey{
						PartitionBy: []string{"title"},
					},
				},
			},
			expected: `{"name":"test_table","definition":{"columns":{"title":{"type":"text"}},"primaryKey":"title"}}`,
		},
		{
			name: "composite primary key",
			payload: createTablePayload{
				Name: "test_table",
				Definition: table.Definition{
					Columns: map[string]table.Column{
						"title":  table.Text(),
						"rating": table.Float(),
					},
					PrimaryKey: table.PrimaryKey{
						PartitionBy: []string{"title", "rating"},
					},
				},
			},
			expected: `{"name":"test_table","definition":{"columns":{"rating":{"type":"float"},"title":{"type":"text"}},"primaryKey":{"partitionBy":["title","rating"]}}}`,
		},
		{
			name: "compound primary key with clustering",
			payload: createTablePayload{
				Name: "test_table",
				Definition: table.Definition{
					Columns: map[string]table.Column{
						"title":           table.Text(),
						"rating":          table.Float(),
						"number_of_pages": table.Int(),
					},
					PrimaryKey: table.PrimaryKey{
						PartitionBy:   []string{"title"},
						PartitionSort: map[string]int{"rating": table.SortAscending, "number_of_pages": table.SortDescending},
					},
				},
			},
		},
		{
			name: "with ifNotExists",
			payload: createTablePayload{
				Name: "test_table",
				Definition: table.Definition{
					Columns: map[string]table.Column{
						"id": table.UUID(),
					},
					PrimaryKey: table.PrimaryKey{
						PartitionBy: []string{"id"},
					},
				},
				Options: &createTableOpts{IfNotExists: true},
			},
			expected: `{"name":"test_table","definition":{"columns":{"id":{"type":"uuid"}},"primaryKey":"id"},"options":{"ifNotExists":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			if tt.expected != "" && string(b) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(b))
			}
			// Verify it can be unmarshaled back
			var result createTablePayload
			if err := json.Unmarshal(b, &result); err != nil {
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
			b, err := json.Marshal(tt.column)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify it can be unmarshaled back
			var result table.Column
			if err := json.Unmarshal(b, &result); err != nil {
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

	b, err := json.Marshal(col)
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
				PartitionSort: map[string]int{"rating": 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pk table.PrimaryKey
			if err := json.Unmarshal([]byte(tt.input), &pk); err != nil {
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

func TestTableFindPayloadMarshal(t *testing.T) {
	tests := []struct {
		name    string
		payload tableFindPayload
		check   func(t *testing.T, result map[string]any)
	}{
		{
			name: "empty filter",
			payload: tableFindPayload{
				Filter: filter.F{},
			},
			check: func(t *testing.T, result map[string]any) {
				if result["filter"] == nil {
					t.Error("expected filter to be present")
				}
			},
		},
		{
			name: "with filter",
			payload: tableFindPayload{
				Filter: filter.F{"is_checked_out": false},
			},
			check: func(t *testing.T, result map[string]any) {
				f, ok := result["filter"].(map[string]any)
				if !ok {
					t.Fatal("expected filter to be a map")
				}
				if f["is_checked_out"] != false {
					t.Error("expected is_checked_out to be false")
				}
			},
		},
		{
			name: "with sort ascending",
			payload: tableFindPayload{
				Filter: filter.F{},
				Sort:   sort.Asc("rating"),
			},
			check: func(t *testing.T, result map[string]any) {
				s, ok := result["sort"].(map[string]any)
				if !ok {
					t.Fatal("expected sort to be a map")
				}
				// JSON numbers unmarshal as float64
				if s["rating"] != float64(1) {
					t.Errorf("expected rating sort to be 1, got %v", s["rating"])
				}
			},
		},
		{
			name: "with sort descending",
			payload: tableFindPayload{
				Filter: filter.F{},
				Sort:   sort.Desc("title"),
			},
			check: func(t *testing.T, result map[string]any) {
				s, ok := result["sort"].(map[string]any)
				if !ok {
					t.Fatal("expected sort to be a map")
				}
				if s["title"] != float64(-1) {
					t.Errorf("expected title sort to be -1, got %v", s["title"])
				}
			},
		},
		{
			name: "with vector search",
			payload: tableFindPayload{
				Filter: filter.F{},
				Sort:   sort.Vector([]float32{0.1, 0.2, 0.3}),
			},
			check: func(t *testing.T, result map[string]any) {
				s, ok := result["sort"].(map[string]any)
				if !ok {
					t.Fatal("expected sort to be a map")
				}
				vec, ok := s["$vector"].([]any)
				if !ok {
					t.Fatal("expected $vector to be a slice")
				}
				if len(vec) != 3 {
					t.Errorf("expected vector length 3, got %d", len(vec))
				}
			},
		},
		{
			name: "with projection include",
			payload: tableFindPayload{
				Filter:     filter.F{},
				Projection: map[string]bool{"title": true, "rating": true},
			},
			check: func(t *testing.T, result map[string]any) {
				proj, ok := result["projection"].(map[string]any)
				if !ok {
					t.Fatal("expected projection to be a map")
				}
				if proj["title"] != true {
					t.Error("expected title to be included")
				}
				if proj["rating"] != true {
					t.Error("expected rating to be included")
				}
			},
		},
		{
			name: "with limit and skip",
			payload: tableFindPayload{
				Filter: filter.F{},
				Options: &tableFindOpts{
					Limit: intPtr(10),
					Skip:  intPtr(5),
				},
			},
			check: func(t *testing.T, result map[string]any) {
				opts, ok := result["options"].(map[string]any)
				if !ok {
					t.Fatal("expected options to be a map")
				}
				if opts["limit"] != float64(10) {
					t.Errorf("expected limit 10, got %v", opts["limit"])
				}
				if opts["skip"] != float64(5) {
					t.Errorf("expected skip 5, got %v", opts["skip"])
				}
			},
		},
		{
			name: "with includeSimilarity",
			payload: tableFindPayload{
				Filter: filter.F{},
				Sort:   sort.Vector([]float32{0.1, 0.2}),
				Options: &tableFindOpts{
					IncludeSimilarity: boolPtr(true),
				},
			},
			check: func(t *testing.T, result map[string]any) {
				opts, ok := result["options"].(map[string]any)
				if !ok {
					t.Fatal("expected options to be a map")
				}
				if opts["includeSimilarity"] != true {
					t.Error("expected includeSimilarity to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(b, &result); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			tt.check(t, result)
		})
	}
}

func TestTableFindOptions(t *testing.T) {
	t.Run("with all options", func(t *testing.T) {
		opts, err := options.MergeAndValidate(
			options.TableFind().
				SetSort(sort.Asc("rating")).
				SetProjection(map[string]bool{"title": true}).
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

	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Should produce something like:
	// {"$and":[{"is_checked_out":false},{"number_of_pages":{"$lt":300}}]}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
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

func TestTableInsertOnePayloadMarshal(t *testing.T) {
	type TestRow struct {
		Title  string  `json:"title"`
		Author string  `json:"author"`
		Rating float32 `json:"rating"`
	}

	payload := tableInsertOnePayload{
		Document: TestRow{
			Title:  "The Great Gatsby",
			Author: "F. Scott Fitzgerald",
			Rating: 4.5,
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	doc, ok := result["document"].(map[string]any)
	if !ok {
		t.Fatal("expected document to be a map")
	}

	if doc["title"] != "The Great Gatsby" {
		t.Errorf("expected title 'The Great Gatsby', got %v", doc["title"])
	}
	if doc["author"] != "F. Scott Fitzgerald" {
		t.Errorf("expected author 'F. Scott Fitzgerald', got %v", doc["author"])
	}
}

func TestTableInsertManyPayloadMarshal(t *testing.T) {
	type TestRow struct {
		Title  string  `json:"title"`
		Rating float32 `json:"rating"`
	}

	rows := []TestRow{
		{Title: "Book 1", Rating: 4.0},
		{Title: "Book 2", Rating: 4.5},
		{Title: "Book 3", Rating: 5.0},
	}

	payload := tableInsertManyPayload{
		Documents: rows,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	docs, ok := result["documents"].([]any)
	if !ok {
		t.Fatal("expected documents to be a slice")
	}

	if len(docs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(docs))
	}
}

func TestTableInsertResponseUnmarshal(t *testing.T) {
	// Test single-column primary key response
	// The API returns insertedIds as an array of arrays: [["value1"], ["value2"]]
	t.Run("single column primary key", func(t *testing.T) {
		jsonResp := `{"status":{"insertedIds":[["The Great Gatsby"]],"primaryKeySchema":{"title":{"type":"text"}}}}`
		var resp TableInsertResponse
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.InsertedIds) != 1 {
			t.Errorf("expected 1 inserted ID, got %d", len(resp.Status.InsertedIds))
		}

		// Each inserted ID is an array of primary key values
		pkValues, ok := resp.Status.InsertedIds[0].([]any)
		if !ok {
			t.Fatalf("expected inserted ID to be []any, got %T", resp.Status.InsertedIds[0])
		}
		if len(pkValues) != 1 {
			t.Errorf("expected 1 pk value, got %d", len(pkValues))
		}
		if pkValues[0] != "The Great Gatsby" {
			t.Errorf("expected 'The Great Gatsby', got %v", pkValues[0])
		}
	})

	// Test composite primary key response
	t.Run("composite primary key", func(t *testing.T) {
		// For composite keys, each inserted ID is still an array with multiple values
		jsonResp := `{"status":{"insertedIds":[["Book 1","Author 1"]],"primaryKeySchema":{"title":{"type":"text"},"author":{"type":"text"}}}}`
		var resp TableInsertResponse
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.InsertedIds) != 1 {
			t.Errorf("expected 1 inserted ID, got %d", len(resp.Status.InsertedIds))
		}

		pkValues, ok := resp.Status.InsertedIds[0].([]any)
		if !ok {
			t.Fatalf("expected inserted ID to be []any, got %T", resp.Status.InsertedIds[0])
		}
		if len(pkValues) != 2 {
			t.Errorf("expected 2 pk values, got %d", len(pkValues))
		}
		if pkValues[0] != "Book 1" {
			t.Errorf("expected 'Book 1', got %v", pkValues[0])
		}
	})

	// Test multiple inserts
	t.Run("multiple inserts", func(t *testing.T) {
		jsonResp := `{"status":{"insertedIds":[["Book 1"],["Book 2"],["Book 3"]]}}`
		var resp TableInsertResponse
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.InsertedIds) != 3 {
			t.Errorf("expected 3 inserted IDs, got %d", len(resp.Status.InsertedIds))
		}
	})
}

// getTestTable acts as a test fixture to provide a *Table.
func getTestTable(t *testing.T) *Table {
	// See: https://pkg.go.dev/testing#T.Helper
	t.Helper()

	client := NewClient(options.WithToken("TEST_TOKEN"))
	db := client.Database("https://API_ENDPOINT", options.WithKeyspace("some_keyspace"))
	return db.Table("example_table")
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-index.html#example-exists
//
// The endpoint should look like:
// "API_ENDPOINT/api/json/v1/KEYSPACE_NAME/TABLE_NAME"
const exampleIndexPayloadJSON = `{
  "createIndex": {
    "name": "example_index_name",
    "definition": {
      "column": "example_column"
    },
    "options": {
      "ifNotExists": true
    }
  }
}`

// TestCreateIndexCommandMarshal verifies that the resulting command from createIndexCommand matches
// the payload in the docs.
func TestCreateIndexCommandMarshal(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", "example_column", options.CreateIndex().SetIfNotExists(true))
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleIndexPayloadJSON {
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
const exampleIndexASCIIPayloadJSON = `{
  "createIndex": {
    "name": "example_index_name",
    "definition": {
      "column": "example_column",
      "options": {
        "ascii": true
      }
    }
  }
}`

// TestCreateIndexASCIICommandMarshal verifies that the resulting command from createIndexCommand
// with the ascii option matches the payload in the docs.
func TestCreateIndexASCIICommandMarshal(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", "example_column", options.CreateIndex().SetAscii(true))
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleIndexASCIIPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleIndexASCIIPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-index.html#example-index-map
const exampleIndexMapKeysPayloadJSON = `{
  "createIndex": {
    "name": "example_index_name",
    "definition": {
      "column": {
        "example_map_column": "$keys"
      }
    }
  }
}`

// TestCreateIndexMapKeysCommandMarshal verifies that the resulting command from createIndexCommand
// with a map column keys index matches the payload in the docs.
func TestCreateIndexMapKeysCommandMarshal(t *testing.T) {
	cmd, err := createIndexCommand(getTestTable(t), "example_index_name", map[string]string{"example_map_column": "$keys"})
	if err != nil {
		t.Fatalf("createIndexCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleIndexMapKeysPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleIndexMapKeysPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#example-default
const exampleVectorIndexDefaultPayloadJSON = `{
  "createVectorIndex": {
    "name": "example_index_name",
    "definition": {
      "column": "example_vector_column"
    }
  }
}`

// TestCreateVectorIndexDefaultCommandMarshal verifies that the resulting command from createVectorIndexCommand
// with default options matches the payload in the docs.
func TestCreateVectorIndexDefaultCommandMarshal(t *testing.T) {
	cmd, err := createVectorIndexCommand(getTestTable(t), "example_index_name", "example_vector_column")
	if err != nil {
		t.Fatalf("createVectorIndexCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleVectorIndexDefaultPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleVectorIndexDefaultPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#example-model-metric
const exampleVectorIndexModelMetricPayloadJSON = `{
  "createVectorIndex": {
    "name": "example_index_name",
    "definition": {
      "column": "example_vector_column",
      "options": {
        "metric": "dot_product",
        "sourceModel": "ada002"
      }
    }
  }
}`

// TestCreateVectorIndexModelMetricCommandMarshal verifies that the resulting command from createVectorIndexCommand
// with custom metric and sourceModel matches the payload in the docs.
func TestCreateVectorIndexModelMetricCommandMarshal(t *testing.T) {
	cmd, err := createVectorIndexCommand(getTestTable(t), "example_index_name", "example_vector_column",
		options.CreateVectorIndex().SetMetric(options.MetricDotProduct).SetSourceModel("ada002"))
	if err != nil {
		t.Fatalf("createVectorIndexCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleVectorIndexModelMetricPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleVectorIndexModelMetricPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/create-vector-index.html#example-exists
const exampleVectorIndexIfNotExistsPayloadJSON = `{
  "createVectorIndex": {
    "name": "example_index_name",
    "definition": {
      "column": "summary_genres_vector"
    },
    "options": {
      "ifNotExists": true
    }
  }
}`

// TestCreateVectorIndexIfNotExistsCommandMarshal verifies that the resulting command from createVectorIndexCommand
// with ifNotExists option matches the payload in the docs.
func TestCreateVectorIndexIfNotExistsCommandMarshal(t *testing.T) {
	cmd, err := createVectorIndexCommand(getTestTable(t), "example_index_name", "summary_genres_vector",
		options.CreateVectorIndex().SetIfNotExists(true))
	if err != nil {
		t.Fatalf("createVectorIndexCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleVectorIndexIfNotExistsPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleVectorIndexIfNotExistsPayloadJSON, string(cmdBytes))
	}
}

// getTestDb acts as a test fixture to provide a *Db.
func getTestDb(t *testing.T) *Db {
	t.Helper()
	client := NewClient(options.WithToken("TEST_TOKEN"))
	if client.Options().Token == nil {
		t.Fatal("expected token to be set")
	}
	return client.Database("https://API_ENDPOINT", options.WithKeyspace("some_keyspace"))
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/drop-index.html#drop-an-index
const exampleDropIndexPayloadJSON = `{
  "dropIndex": {
    "name": "rating"
  }
}`

// TestDropTableIndexCommandMarshal verifies that the resulting command from dropTableIndexCommand
// matches the payload in the docs.
func TestDropTableIndexCommandMarshal(t *testing.T) {
	cmd := dropTableIndexCommand(getTestDb(t), "rating")
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
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
const exampleListIndexesNamesOnlyPayloadJSON = `{
  "listIndexes": {}
}`

// TestListIndexesNamesOnlyCommandMarshal verifies that the resulting command from listIndexesCommand
// with default options (no explain) matches the payload in the docs.
func TestListIndexesNamesOnlyCommandMarshal(t *testing.T) {
	cmd, err := listIndexesCommand(getTestTable(t))
	if err != nil {
		t.Fatalf("listIndexesCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if string(cmdBytes) != exampleListIndexesNamesOnlyPayloadJSON {
		t.Errorf("expected JSON:\n%s\nGot:\n%s", exampleListIndexesNamesOnlyPayloadJSON, string(cmdBytes))
	}
}

// This example was taken from the documentation here:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-index-methods/list-index-metadata.html#example-explain
const exampleListIndexesExplainPayloadJSON = `{
  "listIndexes": {
    "options": {
      "explain": true
    }
  }
}`

// TestListIndexesExplainCommandMarshal verifies that the resulting command from listIndexesCommand
// with explain=true matches the payload in the docs.
func TestListIndexesExplainCommandMarshal(t *testing.T) {
	cmd, err := listIndexesCommand(getTestTable(t), options.ListIndexes().SetExplain(true))
	if err != nil {
		t.Fatalf("listIndexesCommand: %v", err)
	}
	// MarshalIndent and match the indentation of the example JSON
	cmdBytes, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
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
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
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
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
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
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
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
		if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Status.Indexes) != 0 {
			t.Errorf("expected 0 indexes, got %d", len(resp.Status.Indexes))
		}
	})
}

// Helper functions for creating pointers
func intPtr(i int) *int {
	return &i
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
		cmdBytes, _ := json.Marshal(cmd)
		// Should not have "options" key when no options provided
		if string(cmdBytes) != `{"createIndex":{"name":"test_idx","definition":{"column":"test_col"}}}` {
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
		cmdBytes, _ := json.Marshal(cmd)
		if string(cmdBytes) != `{"createIndex":{"name":"test_idx","definition":{"column":"test_col","options":{"caseSensitive":true}},"options":{"ifNotExists":true}}}` {
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
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		expected := `{"createIndex":{"name":"test_idx","definition":{"column":"test_col","options":{"ascii":true,"caseSensitive":false}},"options":{"ifNotExists":true}}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
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
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		expected := `{"createIndex":{"name":"test_idx","definition":{"column":"test_col","options":{"ascii":false}}}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
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
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		expected := `{"createIndex":{"name":"test_idx","definition":{"column":"test_col","options":{"ascii":true,"normalize":false,"caseSensitive":true}},"options":{"ifNotExists":true}}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
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
