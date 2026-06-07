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
	"testing"

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
)

func TestDbUseKeyspace(t *testing.T) {
	client := NewClient(options.API().SetToken("test-token"))

	db := client.Database("https://id-region.apps.astra.datastax.com")

	// Check default keyspace
	if ks := db.ClientOptions().GetKeyspace(); ks != "default_keyspace" {
		t.Errorf("expected default keyspace, got %s", ks)
	}

	// UseKeyspace
	db.UseKeyspace("new_ks")
	if ks := db.ClientOptions().GetKeyspace(); ks != "new_ks" {
		t.Errorf("expected new_ks, got %s", ks)
	}

	// Check that child collection inherits it
	coll := db.Collection("test_coll")
	if ks := coll.ClientOptions().GetKeyspace(); ks != "new_ks" {
		t.Errorf("expected collection to inherit new_ks, got %s", ks)
	}

	// Double UseKeyspace
	db.UseKeyspace("newer_ks")
	if ks := db.ClientOptions().GetKeyspace(); ks != "newer_ks" {
		t.Errorf("expected newer_ks, got %s", ks)
	}

	// New collection should have newer_ks
	coll2 := db.Collection("test_coll_2")
	if ks := coll2.ClientOptions().GetKeyspace(); ks != "newer_ks" {
		t.Errorf("expected coll2 to inherit newer_ks, got %s", ks)
	}

	// Old collection should not be affected by new UseKeyspace call
	if ks := coll.ClientOptions().GetKeyspace(); ks != "new_ks" {
		t.Errorf("expected old collection to still have new_ks, got %s", ks)
	}
}

func TestCollectionOptionsMarshal(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		var opts *options.CreateCollectionOptions
		cmdBytes, err := serdes.Serialize(options.Merge[options.CreateCollectionOptions](opts), serdes.TargetNone, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}

		expected := `{}`
		if string(cmdBytes) != expected {
			t.Errorf("expected %s, got %s", expected, string(cmdBytes))
		}
	})

	t.Run("with vector", func(t *testing.T) {
		opts := options.CreateCollection().SetVector(&options.VectorOptions{
			Dimension: ptr.To(1024),
			Metric:    ptr.To("cosine"),
		})

		cmdBytes, err := serdes.Serialize(options.Merge[options.CreateCollectionOptions](opts), serdes.TargetNone, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}

		expected := `{"vector":{"dimension":1024,"metric":"cosine"}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected %s, got %s", expected, string(cmdBytes))
		}
	})

	t.Run("multiple builders merged", func(t *testing.T) {
		opts := options.CreateCollection().SetVector(&options.VectorOptions{
			Dimension: ptr.To(512),
			Metric:    ptr.To("euclidean"),
		})
		opts.SetVector(&options.VectorOptions{
			Dimension: ptr.To(1024),
			Metric:    ptr.To("cosine"),
		})

		cmdBytes, err := serdes.Serialize(options.Merge[options.CreateCollectionOptions](opts), serdes.TargetNone, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}

		expected := `{"vector":{"dimension":1024,"metric":"cosine"}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected %s, got %s", expected, string(cmdBytes))
		}
	})

	t.Run("raw struct passed directly", func(t *testing.T) {
		rawOpts := &options.CreateCollectionOptions{
			DefaultId: &options.CollectionDefaultIdOptions{
				Type: ptr.To(options.DefaultIdTypeUUIDv7),
			},
		}

		cmdBytes, err := serdes.Serialize(rawOpts, serdes.TargetNone, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize: %v", err)
		}

		expected := `{"defaultId":{"type":"uuidv7"}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected %s, got %s", expected, string(cmdBytes))
		}
	})
}

// NOTE: These were both pulled from log files from integration tests. Hence
// the somewhat odd formatting (escaped quotes).

// This is with explain = true.
const listCollectionsExplainTrueResp = "{\"status\":{\"collections\":[{\"name\":\"GoTest\",\"options\":{\"lexical\":{\"enabled\":true,\"analyzer\":\"standard\"},\"rerank\":{\"enabled\":true,\"service\":{\"provider\":\"nvidia\",\"modelName\":\"nvidia/llama-3.2-nv-rerankqa-1b-v2\"}}}},{\"name\":\"quickstart_collection\",\"options\":{\"vector\":{\"dimension\":1024,\"metric\":\"cosine\",\"sourceModel\":\"other\",\"service\":{\"provider\":\"nvidia\",\"modelName\":\"nvidia/nv-embedqa-e5-v5\"}},\"lexical\":{\"enabled\":true,\"analyzer\":\"standard\"},\"rerank\":{\"enabled\":true,\"service\":{\"provider\":\"nvidia\",\"modelName\":\"nvidia/llama-3.2-nv-rerankqa-1b-v2\"}}}}]}}"

// This is with explain = false.
const listCollectionsExplainFalseResp = "{\"status\":{\"collections\":[\"GoTest\",\"quickstart_collection\"]}}"

// TestListCollectionsUnmarshal verifies that both types of listCollection responses can
// be properly json.Unmarshal'd into the internal listCollectionsResponse struct.
func TestListCollectionsUnmarshal_ExplainTrue(t *testing.T) {
	var resp listCollectionsResponse[[]results.CollectionDescriptor]
	if err := serdes.Deserialize([]byte(listCollectionsExplainTrueResp), &resp, nil, serdes.TargetNone); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Status.Collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(resp.Status.Collections))
	}
	if resp.Status.Collections[0].Name != "GoTest" {
		t.Errorf("expected first collection name 'GoTest', got '%s'", resp.Status.Collections[0].Name)
	}
	if resp.Status.Collections[1].Name != "quickstart_collection" {
		t.Errorf("expected second collection name 'quickstart_collection', got '%s'", resp.Status.Collections[1].Name)
	}
}

func TestListCollectionsUnmarshal_ExplainFalse(t *testing.T) {
	var resp listCollectionsResponse[[]string]
	if err := serdes.Deserialize([]byte(listCollectionsExplainFalseResp), &resp, nil, serdes.TargetNone); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Status.Collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(resp.Status.Collections))
	}
	if resp.Status.Collections[0] != "GoTest" {
		t.Errorf("expected first collection name 'GoTest', got '%s'", resp.Status.Collections[0])
	}
	if resp.Status.Collections[1] != "quickstart_collection" {
		t.Errorf("expected second collection name 'quickstart_collection', got '%s'", resp.Status.Collections[1])
	}
}

// Example json from docs:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/list-table-metadata.html#result
const listTablesExplainTrueResp = `{
  "status": {
    "tables": [
      {
        "name": "customers",
        "definition": {
          "columns": {
            "order_date": { "type": "timestamp" },
            "preferences": {
              "type": "map",
              "keyType": "text",
              "valueType": { "type": "text" }
            },
            "is_active": { "type": "boolean" },
            "user_id": { "type": "uuid" },
            "name": { "type": "text" },
            "login_attempts": {
              "type": "set",
              "valueType": { "type": "int" }
            },
            "photo": { "type": "blob" },
            "salary": { "type": "decimal" },
            "order_id": { "type": "uuid" },
            "age": { "type": "int" },
            "tags": {
              "type": "list",
              "valueType": { "type": "text" }
            }
          },
          "primaryKey": {
            "partitionBy": ["user_id"],
            "partitionSort": {
              "order_id": 1,
              "order_date": -1
            }
          }
        }
      }
    ]
  }
}`

func TestListTablesResponseUnmarshal_ExplainTrue(t *testing.T) {
	var resp listTablesResponse[[]results.TableDescriptor]
	if err := serdes.Deserialize([]byte(listTablesExplainTrueResp), &resp, nil, serdes.TargetNone); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Status.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(resp.Status.Tables))
	}
	td := resp.Status.Tables[0]
	if td.Name != "customers" {
		t.Errorf("expected name 'customers', got %q", td.Name)
	}
	if len(td.Definition.Columns) == 0 {
		t.Error("expected non-empty Definition.Columns")
	}
	if col, ok := td.Definition.Columns.Get("user_id"); !ok {
		t.Error("expected user_id column to exist")
	} else if got := col.Type; got != "uuid" {
		t.Errorf("expected user_id type 'uuid', got %q", got)
	}
	if got := td.Definition.PrimaryKey.PartitionBy; len(got) != 1 || got[0] != "user_id" {
		t.Errorf("expected PartitionBy=[\"user_id\"], got %v", got)
	}
	if sort, ok := td.Definition.PrimaryKey.PartitionSort.Get("order_date"); !ok {
		t.Error("expected order_date in PartitionSort")
	} else if sort != -1 {
		t.Errorf("expected order_date sort -1, got %d", sort)
	}
}

// Example json from docs:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/list-table-names.html#result
const listTablesExplainFalseResp = `{"status":{"tables":["quickstart_table","another_table"]}}`

func TestListTablesResponseUnmarshal_ExplainFalse(t *testing.T) {
	var resp listTablesResponse[[]string]
	if err := serdes.Deserialize([]byte(listTablesExplainFalseResp), &resp, nil, serdes.TargetNone); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Status.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(resp.Status.Tables))
	}
	if resp.Status.Tables[0] != "quickstart_table" {
		t.Errorf("expected first name 'quickstart_table', got %q", resp.Status.Tables[0])
	}
	if resp.Status.Tables[1] != "another_table" {
		t.Errorf("expected second name 'another_table', got %q", resp.Status.Tables[1])
	}
}

// This response is a real response that broke in integration tests.
// Also this looks slightly strange because it was pasted from a structured log file.
// So, escaped quotes instead of string literal.
const listTablesBrokenResp = "{\"status\":{\"tables\":[{\"name\":\"go_test_books\",\"definition\":{\"columns\":{\"title\":{\"type\":\"text\"},\"author\":{\"type\":\"text\"},\"genres\":{\"type\":\"list\",\"valueType\":\"text\"},\"is_checked_out\":{\"type\":\"boolean\"},\"number_of_pages\":{\"type\":\"int\"},\"rating\":{\"type\":\"float\"}},\"primaryKey\":{\"partitionBy\":[\"title\"],\"partitionSort\":{}}}},{\"name\":\"test_table_go_integration_tests\",\"definition\":{\"columns\":{\"title\":{\"type\":\"text\"},\"is_checked_out\":{\"type\":\"boolean\"},\"number_of_pages\":{\"type\":\"int\"},\"rating\":{\"type\":\"float\"}},\"primaryKey\":{\"partitionBy\":[\"title\"],\"partitionSort\":{}}}}]}}"

func TestListTablesResponseUnmarshal_RealworldExample(t *testing.T) {
	var resp listTablesResponse[[]results.TableDescriptor]
	if err := serdes.Deserialize([]byte(listTablesBrokenResp), &resp, nil, serdes.TargetNone); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Status.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(resp.Status.Tables))
	}
	td := resp.Status.Tables[0]
	if td.Name != "go_test_books" {
		t.Errorf("expected name 'go_test_books', got %q", td.Name)
	}
	if got := resp.Status.Tables[1].Name; got != "test_table_go_integration_tests" {
		t.Errorf("expected second table 'test_table_go_integration_tests', got %q", got)
	}
	if col, ok := td.Definition.Columns.Get("title"); !ok {
		t.Error("expected title column to exist")
	} else if got := col.Type; got != "text" {
		t.Errorf("expected title type 'text', got %q", got)
	}
	// genres is a list column whose valueType is the string form ("text")
	// rather than an object — this is the case that broke in integration tests.
	genres, ok := td.Definition.Columns.Get("genres")
	if !ok {
		t.Error("expected genres column to exist")
	}
	if genres.Type != "list" {
		t.Errorf("expected genres type 'list', got %q", genres.Type)
	}
	if genres.ValueType == nil || genres.ValueType.Type != "text" {
		t.Errorf("expected genres valueType 'text', got %+v", genres.ValueType)
	}
	if got := td.Definition.PrimaryKey.PartitionBy; len(got) != 1 || got[0] != "title" {
		t.Errorf("expected PartitionBy=[\"title\"], got %v", got)
	}
	if got := td.Definition.PrimaryKey.PartitionSort; len(got) != 0 {
		t.Errorf("expected empty PartitionSort, got %v", got)
	}
}
