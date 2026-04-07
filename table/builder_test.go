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

package table_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/datastax/astra-db-go/table"
)

func TestDefinitionBuilder_Basic(t *testing.T) {
	def := table.NewDefinition().
		AddColumn("title", table.Text()).
		AddColumn("rating", table.Float()).
		SetPartitionBy("title").
		Build()

	if len(def.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(def.Columns))
	}
	if def.Columns["title"].Type != table.TypeText {
		t.Errorf("expected title to be text, got %s", def.Columns["title"].Type)
	}
	if def.Columns["rating"].Type != table.TypeFloat {
		t.Errorf("expected rating to be float, got %s", def.Columns["rating"].Type)
	}
	if len(def.PrimaryKey.PartitionBy) != 1 || def.PrimaryKey.PartitionBy[0] != "title" {
		t.Errorf("expected partition by [title], got %v", def.PrimaryKey.PartitionBy)
	}
}

func TestDefinitionBuilder_TypedColumns(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("name").
		AddIntColumn("age").
		AddBigIntColumn("big_num").
		AddFloatColumn("score").
		AddDoubleColumn("precise_score").
		AddBooleanColumn("active").
		AddDateColumn("birth_date").
		AddTimestampColumn("created_at").
		AddUUIDColumn("id").
		AddBlobColumn("data").
		SetPartitionBy("id").
		Build()

	expectedTypes := map[string]string{
		"name":          table.TypeText,
		"age":           table.TypeInt,
		"big_num":       table.TypeBigInt,
		"score":         table.TypeFloat,
		"precise_score": table.TypeDouble,
		"active":        table.TypeBoolean,
		"birth_date":    table.TypeDate,
		"created_at":    table.TypeTimestamp,
		"id":            table.TypeUUID,
		"data":          table.TypeBlob,
	}

	for name, expectedType := range expectedTypes {
		col, ok := def.Columns[name]
		if !ok {
			t.Errorf("expected column %s to exist", name)
			continue
		}
		if col.Type != expectedType {
			t.Errorf("expected %s to be %s, got %s", name, expectedType, col.Type)
		}
	}
}

func TestDefinitionBuilder_VectorColumn(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("id").
		AddVectorColumn("embeddings", 1536).
		SetPartitionBy("id").
		Build()

	col, ok := def.Columns["embeddings"]
	if !ok {
		t.Fatal("expected embeddings column to exist")
	}
	if col.Type != table.TypeVector {
		t.Errorf("expected vector type, got %s", col.Type)
	}
	if col.Dimension == nil || *col.Dimension != 1536 {
		t.Errorf("expected dimension 1536")
	}
}

func TestDefinitionBuilder_VectorColumnWithService(t *testing.T) {
	service := &table.VectorService{
		Provider:  "openai",
		ModelName: "text-embedding-3-small",
		Authentication: map[string]string{
			"providerKey": "my-key",
		},
	}

	def := table.NewDefinition().
		AddTextColumn("id").
		AddVectorColumnWithService("embeddings", 1536, service).
		SetPartitionBy("id").
		Build()

	col := def.Columns["embeddings"]
	if col.Service == nil {
		t.Fatal("expected service to be set")
	}
	if col.Service.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", col.Service.Provider)
	}
}

func TestDefinitionBuilder_CollectionColumns(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("id").
		AddSetColumn("tags", table.Text()).
		AddListColumn("scores", table.Int()).
		AddMapColumn("metadata", "text", table.Text()).
		SetPartitionBy("id").
		Build()

	// Check set column
	setCol := def.Columns["tags"]
	if setCol.Type != table.TypeSet {
		t.Errorf("expected set type, got %s", setCol.Type)
	}
	if setCol.ValueType == nil || setCol.ValueType.Type != table.TypeText {
		t.Error("expected set value type to be text")
	}

	// Check list column
	listCol := def.Columns["scores"]
	if listCol.Type != table.TypeList {
		t.Errorf("expected list type, got %s", listCol.Type)
	}
	if listCol.ValueType == nil || listCol.ValueType.Type != table.TypeInt {
		t.Error("expected list value type to be int")
	}

	// Check map column
	mapCol := def.Columns["metadata"]
	if mapCol.Type != table.TypeMap {
		t.Errorf("expected map type, got %s", mapCol.Type)
	}
	if mapCol.KeyType == nil || *mapCol.KeyType != "text" {
		t.Error("expected map key type to be text")
	}
	if mapCol.ValueType == nil || mapCol.ValueType.Type != table.TypeText {
		t.Error("expected map value type to be text")
	}
}

func TestDefinitionBuilder_UDTColumn(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("id").
		AddUDTColumn("address", "address_type").
		SetPartitionBy("id").
		Build()

	col := def.Columns["address"]
	if col.Type != table.TypeUDT {
		t.Errorf("expected userDefined type, got %s", col.Type)
	}
	if col.UDTName == nil || *col.UDTName != "address_type" {
		t.Error("expected UDT name to be address_type")
	}
}

func TestDefinitionBuilder_CompositePartitionKey(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("tenant_id").
		AddTextColumn("user_id").
		AddTextColumn("name").
		SetPartitionBy("tenant_id", "user_id").
		Build()

	if len(def.PrimaryKey.PartitionBy) != 2 {
		t.Errorf("expected 2 partition columns, got %d", len(def.PrimaryKey.PartitionBy))
	}
	if def.PrimaryKey.PartitionBy[0] != "tenant_id" || def.PrimaryKey.PartitionBy[1] != "user_id" {
		t.Errorf("expected [tenant_id, user_id], got %v", def.PrimaryKey.PartitionBy)
	}
}

func TestDefinitionBuilder_AddPartitionBy(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("a").
		AddTextColumn("b").
		AddTextColumn("c").
		AddPartitionBy("a").
		AddPartitionBy("b").
		Build()

	if len(def.PrimaryKey.PartitionBy) != 2 {
		t.Errorf("expected 2 partition columns, got %d", len(def.PrimaryKey.PartitionBy))
	}
}

func TestDefinitionBuilder_ClusteringColumns(t *testing.T) {
	def := table.NewDefinition().
		AddTextColumn("id").
		AddTimestampColumn("created_at").
		AddIntColumn("priority").
		SetPartitionBy("id").
		AddClusteringColumnDesc("created_at").
		AddClusteringColumnAsc("priority").
		Build()

	if len(def.PrimaryKey.PartitionSort) != 2 {
		t.Errorf("expected 2 clustering columns, got %d", len(def.PrimaryKey.PartitionSort))
	}
	if def.PrimaryKey.PartitionSort["created_at"] != table.SortDescending {
		t.Errorf("expected created_at to be descending")
	}
	if def.PrimaryKey.PartitionSort["priority"] != table.SortAscending {
		t.Errorf("expected priority to be ascending")
	}
}

func TestDefinitionBuilder_JSONMarshal(t *testing.T) {
	// Build with fluent API
	defBuilder := table.NewDefinition().
		AddTextColumn("title").
		AddFloatColumn("rating").
		SetPartitionBy("title").
		Build()

	// Build with struct
	defStruct := table.Definition{
		Columns: map[string]table.Column{
			"title":  table.Text(),
			"rating": table.Float(),
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy: []string{"title"},
		},
	}

	// Marshal both
	jsonBuilder, err := json.Marshal(defBuilder)
	if err != nil {
		t.Fatalf("failed to marshal builder definition: %v", err)
	}

	jsonStruct, err := json.Marshal(defStruct)
	if err != nil {
		t.Fatalf("failed to marshal struct definition: %v", err)
	}

	// They should produce equivalent JSON (unmarshal and compare)
	var resultBuilder, resultStruct table.Definition
	if err := json.Unmarshal(jsonBuilder, &resultBuilder); err != nil {
		t.Fatalf("failed to unmarshal builder JSON: %v", err)
	}
	if err := json.Unmarshal(jsonStruct, &resultStruct); err != nil {
		t.Fatalf("failed to unmarshal struct JSON: %v", err)
	}

	// Compare key fields
	if len(resultBuilder.Columns) != len(resultStruct.Columns) {
		t.Error("column counts don't match")
	}
	if resultBuilder.Columns["title"].Type != resultStruct.Columns["title"].Type {
		t.Error("title column types don't match")
	}
	if len(resultBuilder.PrimaryKey.PartitionBy) != len(resultStruct.PrimaryKey.PartitionBy) {
		t.Error("partition key lengths don't match")
	}
}

func TestDefinitionBuilder_FullExample(t *testing.T) {
	// This replicates the AstraPy example from the README
	def := table.NewDefinition().
		AddIntColumn("dream_id").
		AddTextColumn("summary").
		AddSetColumn("tags", table.Text()).
		AddVectorColumn("dream_vector", 3).
		SetPartitionBy("dream_id").
		Build()

	// Verify the structure
	if len(def.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(def.Columns))
	}

	// Check dream_id
	if def.Columns["dream_id"].Type != table.TypeInt {
		t.Errorf("expected dream_id to be int")
	}

	// Check tags is a set of text
	tagsCol := def.Columns["tags"]
	if tagsCol.Type != table.TypeSet {
		t.Errorf("expected tags to be set")
	}
	if tagsCol.ValueType == nil || tagsCol.ValueType.Type != table.TypeText {
		t.Error("expected tags value type to be text")
	}

	// Check vector
	vecCol := def.Columns["dream_vector"]
	if vecCol.Type != table.TypeVector || vecCol.Dimension == nil || *vecCol.Dimension != 3 {
		t.Error("expected dream_vector to be vector with dimension 3")
	}

	// Check primary key
	if len(def.PrimaryKey.PartitionBy) != 1 || def.PrimaryKey.PartitionBy[0] != "dream_id" {
		t.Error("expected primary key to be dream_id")
	}
}

// TestDefinitionBuilder_DeepEqual verifies that building a definition using
// the fluent builder API produces an identical struct to the struct-based approach.
func TestDefinitionBuilder_DeepEqual(t *testing.T) {
	t.Run("simple definition", func(t *testing.T) {
		// Build with fluent API
		builderDef := table.NewDefinition().
			AddColumn("title", table.Text()).
			AddColumn("author", table.Text()).
			AddColumn("rating", table.Float()).
			SetPartitionBy("title").
			Build()

		// Build with struct
		structDef := table.Definition{
			Columns: map[string]table.Column{
				"title":  table.Text(),
				"author": table.Text(),
				"rating": table.Float(),
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"title"},
			},
		}

		if !reflect.DeepEqual(builderDef, structDef) {
			t.Errorf("definitions not equal\nbuilder: %+v\nstruct:  %+v", builderDef, structDef)
		}
	})

	t.Run("complex definition with collections and clustering", func(t *testing.T) {
		// Build with fluent API
		builderDef := table.NewDefinition().
			AddUUIDColumn("id").
			AddTextColumn("tenant").
			AddTimestampColumn("created_at").
			AddSetColumn("tags", table.Text()).
			AddListColumn("scores", table.Int()).
			AddMapColumn("metadata", "text", table.Text()).
			AddVectorColumn("embeddings", 1536).
			SetPartitionBy("tenant", "id").
			AddClusteringColumnDesc("created_at").
			Build()

		// Build with struct
		structDef := table.Definition{
			Columns: map[string]table.Column{
				"id":         table.UUID(),
				"tenant":     table.Text(),
				"created_at": table.Timestamp(),
				"tags":       table.Set(table.Text()),
				"scores":     table.List(table.Int()),
				"metadata":   table.Map("text", table.Text()),
				"embeddings": table.Vector(1536),
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"tenant", "id"},
				PartitionSort: map[string]int{
					"created_at": table.SortDescending,
				},
			},
		}

		if !reflect.DeepEqual(builderDef, structDef) {
			t.Errorf("definitions not equal\nbuilder: %+v\nstruct:  %+v", builderDef, structDef)
		}
	})

	t.Run("definition with vector service", func(t *testing.T) {
		service := &table.VectorService{
			Provider:  "openai",
			ModelName: "text-embedding-3-small",
			Authentication: map[string]string{
				"providerKey": "OPENAI_API_KEY",
			},
			Parameters: map[string]string{
				"projectId": "my-project",
			},
		}

		// Build with fluent API
		builderDef := table.NewDefinition().
			AddTextColumn("id").
			AddTextColumn("content").
			AddVectorColumnWithService("embeddings", 1536, service).
			SetPartitionBy("id").
			Build()

		// Build with struct
		structDef := table.Definition{
			Columns: map[string]table.Column{
				"id":         table.Text(),
				"content":    table.Text(),
				"embeddings": table.VectorWithService(1536, service),
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"id"},
			},
		}

		if !reflect.DeepEqual(builderDef, structDef) {
			t.Errorf("definitions not equal\nbuilder: %+v\nstruct:  %+v", builderDef, structDef)
		}
	})

	t.Run("definition with UDT column", func(t *testing.T) {
		// Build with fluent API
		builderDef := table.NewDefinition().
			AddUUIDColumn("user_id").
			AddTextColumn("name").
			AddUDTColumn("address", "address_type").
			SetPartitionBy("user_id").
			Build()

		// Build with struct
		structDef := table.Definition{
			Columns: map[string]table.Column{
				"user_id": table.UUID(),
				"name":    table.Text(),
				"address": table.UDT("address_type"),
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"user_id"},
			},
		}

		if !reflect.DeepEqual(builderDef, structDef) {
			t.Errorf("definitions not equal\nbuilder: %+v\nstruct:  %+v", builderDef, structDef)
		}
	})

	t.Run("definition with multiple clustering columns", func(t *testing.T) {
		// Build with fluent API
		builderDef := table.NewDefinition().
			AddTextColumn("partition_key").
			AddTimestampColumn("event_time").
			AddIntColumn("priority").
			AddTextColumn("event_id").
			SetPartitionBy("partition_key").
			AddClusteringColumnDesc("event_time").
			AddClusteringColumnAsc("priority").
			Build()

		// Build with struct
		structDef := table.Definition{
			Columns: map[string]table.Column{
				"partition_key": table.Text(),
				"event_time":    table.Timestamp(),
				"priority":      table.Int(),
				"event_id":      table.Text(),
			},
			PrimaryKey: table.PrimaryKey{
				PartitionBy: []string{"partition_key"},
				PartitionSort: map[string]int{
					"event_time": table.SortDescending,
					"priority":   table.SortAscending,
				},
			},
		}

		if !reflect.DeepEqual(builderDef, structDef) {
			t.Errorf("definitions not equal\nbuilder: %+v\nstruct:  %+v", builderDef, structDef)
		}
	})
}
