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

package table

import (
	"encoding/json"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/internal/testutils"
)

func TestGoTypeToColumn(t *testing.T) {
	// These tests verify that we get the datatypes we are expecting. We are passing the tagInfo as a struct
	// so this doesn't verify tag parsing. Trying to split out concerns.
	tests := []struct {
		name    string
		goType  reflect.Type
		tag     tagInfo
		want    Column
		wantErr bool
	}{
		{"string = text", reflect.TypeFor[string](), tagInfo{}, Text(), false},
		{"*string = text", reflect.TypeFor[*string](), tagInfo{}, Text(), false},
		{"int = int", reflect.TypeFor[int](), tagInfo{}, Int(), false},
		{"int32 = int", reflect.TypeFor[int32](), tagInfo{}, Int(), false},
		{"int64 = bigint", reflect.TypeFor[int64](), tagInfo{}, BigInt(), false},
		{"int16 = smallint", reflect.TypeFor[int16](), tagInfo{}, SmallInt(), false},
		{"int8 = tinyint", reflect.TypeFor[int8](), tagInfo{}, TinyInt(), false},
		{"uint8 = tinyint", reflect.TypeFor[uint8](), tagInfo{}, TinyInt(), false},
		{"float32 = float", reflect.TypeFor[float32](), tagInfo{}, Float(), false},
		{"float64 = double", reflect.TypeFor[float64](), tagInfo{}, Double(), false},
		{"bool = boolean", reflect.TypeFor[bool](), tagInfo{}, Boolean(), false},
		{"time.Time = timestamp", reflect.TypeFor[time.Time](), tagInfo{}, Timestamp(), false},
		{"time.Time+type = date", reflect.TypeFor[time.Time](), tagInfo{typeOverride: "date"}, Date(), false},
		{"time.Time+type = time", reflect.TypeFor[time.Time](), tagInfo{typeOverride: "time"}, Time(), false},
		{"UUID = uuid", reflect.TypeFor[datatypes.UUID](), tagInfo{}, UUID(), false},
		{"[]byte = blob", reflect.TypeFor[[]byte](), tagInfo{}, Blob(), false},
		{"net.IP = inet", reflect.TypeFor[net.IP](), tagInfo{}, Inet(), false},
		{"[]string = list", reflect.TypeFor[[]string](), tagInfo{}, List(Text()), false},
		{"[]string+type = set", reflect.TypeFor[[]string](), tagInfo{typeOverride: "set"}, Set(Text()), false},
		{"[]int = list<int>", reflect.TypeFor[[]int](), tagInfo{}, List(Int()), false},
		{"map[string]int = map<text,int>", reflect.TypeFor[map[string]int](), tagInfo{}, Map(TypeText, Int()), false},
		{"map[string]float64 = map<text,double>", reflect.TypeFor[map[string]float64](), tagInfo{}, Map(TypeText, Double()), false},
		{"float64+type = decimal", reflect.TypeFor[float64](), tagInfo{typeOverride: "decimal"}, Decimal(), false},
		{"string+type = ascii", reflect.TypeFor[string](), tagInfo{typeOverride: "ascii"}, Ascii(), false},
		{"string+type = uuid", reflect.TypeFor[string](), tagInfo{typeOverride: "uuid"}, UUID(), false},
		{"[]float32+vector+dim = 3", reflect.TypeFor[[]float32](), tagInfo{isVector: true, dimension: 3}, Vector(3), false},
		{"DataAPIVector+dim = 4", reflect.TypeFor[datatypes.DataAPIVector](), tagInfo{dimension: 4}, Vector(4), false},
		{
			"vectorize",
			reflect.TypeFor[any](),
			tagInfo{hasVectorize: true, provider: "openai", model: "text-embedding-3-small"},
			VectorWithService(0, &VectorService{Provider: "openai", ModelName: "text-embedding-3-small"}),
			false,
		},

		// Error cases
		{"DataAPIVector no dim", reflect.TypeFor[datatypes.DataAPIVector](), tagInfo{}, Column{}, true},
		{"vector no dim", reflect.TypeFor[[]float32](), tagInfo{isVector: true}, Column{}, true},
		{"interface no modifier", reflect.TypeFor[any](), tagInfo{}, Column{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goTypeToColumn(tt.goType, tt.tag)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestInfer_CompoundPrimaryKey(t *testing.T) {
	// Matches CompoundPrimaryKey from models.go
	type Row struct {
		KeyTwo            string `json:"keyTwo" astra:"pk,2"`
		KeyOne            string `json:"keyOne" astra:"pk,1"`
		SortTwoDescending string `json:"sortTwoDescending" astra:"ck,2,desc"`
		SortOneAscending  string `json:"sortOneAscending" astra:"ck,1,asc"`
	}
	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}
	wantPK := []string{"keyOne", "keyTwo"}
	if !reflect.DeepEqual(def.PrimaryKey.PartitionBy, wantPK) {
		t.Errorf("PartitionBy = %v, want %v", def.PrimaryKey.PartitionBy, wantPK)
	}
	wantSort := PartitionSort{
		{Name: "sortOneAscending", Order: SortAscending},
		{Name: "sortTwoDescending", Order: SortDescending},
	}
	if !reflect.DeepEqual(def.PrimaryKey.PartitionSort, wantSort) {
		t.Errorf("PartitionSort = %v, want %v", def.PrimaryKey.PartitionSort, wantSort)
	}
}

func TestInfer_PartitionKeyStructOrder(t *testing.T) {
	// Without ordinals, struct declaration order is used
	type Row struct {
		A string `json:"a" astra:"pk"`
		B string `json:"b" astra:"pk"`
	}
	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}
	wantPK := []string{"a", "b"}
	if !reflect.DeepEqual(def.PrimaryKey.PartitionBy, wantPK) {
		t.Errorf("PartitionBy = %v, want %v", def.PrimaryKey.PartitionBy, wantPK)
	}
}

// Create new table command to test JSON marshal.
func newCreateTableCmd(tableName string, definition any) any {
	type createTable struct {
		Name       string `json:"name"`
		Definition any    `json:"definition"`
	}
	return testutils.NewTestCmd("createTable", createTable{
		Name:       tableName,
		Definition: definition,
	})
}

// Example JSON from docs:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#create-a-table-with-a-single-column-primary-key
const singleColumnPKJSON = `{
  "createTable": {
    "name": "example_table",
    "definition": {
      "columns": {
        "title": {
          "type": "text"
        },
        "number_of_pages": {
          "type": "int"
        },
        "rating": {
          "type": "float"
        },
        "metadata": {
          "type": "map",
          "keyType": "text",
          "valueType": "text"
        },
        "genres": {
          "type": "set",
          "valueType": "text"
        },
        "is_checked_out": {
          "type": "boolean"
        },
        "due_date": {
          "type": "date"
        }
      },
      "primaryKey": "title"
    }
  }
}`

type bookSingleKey struct {
	Title         string            `json:"title" astra:"pk"`
	NumberOfPages int               `json:"number_of_pages"`
	Rating        float32           `json:"rating"`
	Metadata      map[string]string `json:"metadata"`
	Genres        []string          `json:"genres" astra:"type=set"`
	IsCheckedOut  bool              `json:"is_checked_out"`
	DueDate       time.Time         `json:"due_date" astra:"type=date"`
}

func TestInferDocsSingleColumnExample(t *testing.T) {
	// This text makes sure the docs example can properly Infer.
	def, err := Infer[bookSingleKey]()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure it also matches the struct-based definition.
	structBased := Definition{
		Columns: Columns{
			{"title", Text()},
			{"number_of_pages", Int()},
			{"rating", Float()},
			{"metadata", Map(TypeText, Text())},
			{"genres", Set(Text())},
			{"is_checked_out", Boolean()},
			{"due_date", Date()},
		},
		PrimaryKey: PrimaryKey{PartitionBy: []string{"title"}},
	}

	// And throw in a builder for extra fun.
	builder := NewDefinition().
		AddTextColumn("title").
		AddIntColumn("number_of_pages").
		AddFloatColumn("rating").
		AddMapColumn("metadata", TypeText, Text()).
		AddSetColumn("genres", Text()).
		AddBooleanColumn("is_checked_out").
		AddDateColumn("due_date").
		SetPartitionBy("title").
		Build()

	// Wrap them in the same command structure used for JSON testing.
	builderCmd := newCreateTableCmd("example_table", builder)
	inferCmd := newCreateTableCmd("example_table", def)
	structBasedCmd := newCreateTableCmd("example_table", structBased)

	testutils.AssertJSONEqual(t, singleColumnPKJSON, structBasedCmd, inferCmd, builderCmd)
}

// Docs example for composite key:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#create-a-table-with-a-composite-primary-key
const compositeKeyJSON = `{
  "createTable": {
    "name": "example_table",
    "definition": {
      "columns": {
        "title": {
          "type": "text"
        },
        "number_of_pages": {
          "type": "int"
        },
        "rating": {
          "type": "float"
        },
        "metadata": {
          "type": "map",
          "keyType": "text",
          "valueType": "text"
        },
        "genres": {
          "type": "set",
          "valueType": "text"
        },
        "is_checked_out": {
          "type": "boolean"
        },
        "due_date": {
          "type": "date"
        }
      },
      "primaryKey": {
        "partitionBy": [
          "title", "rating"
        ]
      }
    }
  }
}`

func TestInferDocsCompositeKeyExample(t *testing.T) {
	type bookCompositeKey struct {
		Title         string            `json:"title" astra:"pk"`
		NumberOfPages int               `json:"number_of_pages"`
		Rating        float32           `json:"rating" astra:"pk"`
		Metadata      map[string]string `json:"metadata"`
		Genres        []string          `json:"genres" astra:"type=set"`
		IsCheckedOut  bool              `json:"is_checked_out"`
		DueDate       time.Time         `json:"due_date" astra:"type=date"`
	}
	// This text makes sure the docs example can properly Infer.
	def, err := Infer[bookCompositeKey]()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure it also matches the struct-based definition.
	structBased := Definition{
		Columns: Columns{
			{"title", Text()},
			{"number_of_pages", Int()},
			{"rating", Float()},
			{"metadata", Map(TypeText, Text())},
			{"genres", Set(Text())},
			{"is_checked_out", Boolean()},
			{"due_date", Date()},
		},
		PrimaryKey: PrimaryKey{PartitionBy: []string{"title", "rating"}},
	}

	// And throw in a builder for extra fun.
	builder := NewDefinition().
		AddTextColumn("title").
		AddIntColumn("number_of_pages").
		AddFloatColumn("rating").
		AddMapColumn("metadata", TypeText, Text()).
		AddSetColumn("genres", Text()).
		AddBooleanColumn("is_checked_out").
		AddDateColumn("due_date").
		SetPartitionBy("title", "rating").
		Build()

	// Wrap them in the same command structure used for JSON testing.
	builderCmd := newCreateTableCmd("example_table", builder)
	inferCmd := newCreateTableCmd("example_table", def)
	structBasedCmd := newCreateTableCmd("example_table", structBased)

	testutils.AssertJSONEqual(t, compositeKeyJSON, structBasedCmd, inferCmd, builderCmd)
}

// Docs example for compound key:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#create-a-table-with-a-compound-primary-key
const compoundKeyJSON = `{
  "createTable": {
    "name": "example_table",
    "definition": {
      "columns": {
        "title": {
          "type": "text"
        },
        "number_of_pages": {
          "type": "int"
        },
        "rating": {
          "type": "float"
        },
        "metadata": {
          "type": "map",
          "keyType": "text",
          "valueType": "text"
        },
        "genres": {
          "type": "set",
          "valueType": "text"
        },
        "is_checked_out": {
          "type": "boolean"
        },
        "due_date": {
          "type": "date"
        }
      },
      "primaryKey": {
        "partitionBy": [
          "title",
          "rating"
        ],
        "partitionSort": {
          "number_of_pages": 1,
          "is_checked_out": -1
        }
      }
    }
  }
}`

func TestInferDocsCompoundKeyExample(t *testing.T) {
	// Just testing infer from here on out. I think the other tests tested the builder/raw
	// struct against infer well enough.
	type bookCompoundKey struct {
		Title         string            `json:"title" astra:"pk"`
		NumberOfPages int               `json:"number_of_pages" astra:"ck,1,asc"`
		Rating        float32           `json:"rating" astra:"pk"`
		Metadata      map[string]string `json:"metadata"`
		Genres        []string          `json:"genres" astra:"type=set"`
		IsCheckedOut  bool              `json:"is_checked_out" astra:"ck,2,desc"`
		DueDate       time.Time         `json:"due_date" astra:"type=date"`
	}
	def, err := Infer[bookCompoundKey]()
	if err != nil {
		t.Fatal(err)
	}
	inferCmd := newCreateTableCmd("example_table", def)
	testutils.AssertJSONEqual(t, compoundKeyJSON, inferCmd)
}

// Docs example for vector column:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#example-vector
const vectorColumnJSON = `{
  "createTable": {
    "name": "example_table",
    "definition": {
      "columns": {
        "example_vector": {
          "type": "vector",
          "dimension": 1024
        },
        "example_non_vector": {
          "type": "text"
        }
      },
      "primaryKey": "example_non_vector"
    }
  }
}`

func TestInferDocsVectorColumnExample(t *testing.T) {
	type vectorExample struct {
		ExampleVector    []float32 `json:"example_vector" astra:"type=vector,dim=1024"`
		ExampleNonVector string    `json:"example_non_vector" astra:"pk"`
	}
	def, err := Infer[vectorExample]()
	if err != nil {
		t.Fatal(err)
	}
	inferCmd := newCreateTableCmd("example_table", def)
	testutils.AssertJSONEqual(t, vectorColumnJSON, inferCmd)
}

// Docs example:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#example-vectorize
// Removed comments from this JSON because that is not valid JSON.
// Also changed TEXT_COLUMN_NAME to have {"type": "text"}. Not sure what the shorthand version
// was; it doesn't seem to show up in the docs anywhere else.
const vectorizeExampleJSON = `{
  "createTable": {
    "name": "TABLE_NAME",
    "definition": {
      "columns": {
        "VECTOR_COLUMN_NAME": {
          "type": "vector",
          "service": {
            "provider": "nvidia",
            "modelName": "nvidia/nv-embedqa-e5-v5"
          }
        },
        "TEXT_COLUMN_NAME": { "type": "text" }
      },
      "primaryKey": "TEXT_COLUMN_NAME"
    }
  }
}`

func TestInferDocsVectorizeExample(t *testing.T) {
	type vectorizeExample struct {
		VectorColumnName []float32 `json:"VECTOR_COLUMN_NAME" astra:"vectorize,provider=nvidia,model=nvidia/nv-embedqa-e5-v5"`
		TextColumnName   string    `json:"TEXT_COLUMN_NAME" astra:"pk"`
	}
	def, err := Infer[vectorizeExample]()
	if err != nil {
		t.Fatal(err)
	}
	inferCmd := newCreateTableCmd("TABLE_NAME", def)
	testutils.AssertJSONEqual(t, vectorizeExampleJSON, inferCmd)
}

// From docs:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#example-create-table-udt
const udtExampleJSON = `{
  "createTable": {
    "name": "example_table",
    "definition": {
      "columns": {
        "id": {
          "type": "uuid"
        },
        "group_leader": {
          "type": "userDefined",
          "udtName": "person"
        },
        "group_members": {
          "type": "set",
          "valueType": {
            "type": "userDefined",
            "udtName": "person"
          }
        },
        "group_roles": {
          "type": "map",
          "keyType": "text",
          "valueType": {
            "type": "userDefined",
            "udtName": "person"
          }
        }
      },
      "primaryKey": "id"
    }
  }
}`

func TestInferDocsUDTExample(t *testing.T) {
	// Skip for now. See note below.
	t.Skip()
	type Person struct {
		Name       string `json:"name" astra:"type=text"`
		IsActive   bool   `json:"is_active" astra:"type=boolean"`
		DateJoined string `json:"date_joined" astra:"type=date"`
	}
	// TODO: these tags are aspirational. I went down this road of `Infer` and I'm
	// not sure if we should just say "if you want UDTs, use a Definition" or if we
	// want to keep improving the astra tag.
	type Group struct {
		ID           datatypes.UUID `json:"id" astra:"pk"`
		GroupLeader  Person         `json:"group_leader" astra:"udt=person"`
		GroupMembers []Person       `json:"group_members" astra:"type=set,udt=person"`
		GroupRoles   []Person       `json:"group_roles" astra:"type=map,keyType=text,udt=person"`
	}
	def, err := Infer[Group]()
	if err != nil {
		t.Fatal(err)
	}
	inferCmd := newCreateTableCmd("example_table", def)
	testutils.AssertJSONEqual(t, udtExampleJSON, inferCmd)
}

func TestInfer_AllColumnTypes(t *testing.T) {
	type Row struct {
		PK             string                  `json:"pk" astra:"pk"`
		TextField      string                  `json:"text_field"`
		AsciiField     string                  `json:"ascii_field" astra:"type=ascii"`
		IntField       int                     `json:"int_field"`
		BigIntField    int64                   `json:"bigint_field"`
		SmallIntField  int16                   `json:"smallint_field"`
		TinyIntField   int8                    `json:"tinyint_field"`
		FloatField     float32                 `json:"float_field"`
		DoubleField    float64                 `json:"double_field"`
		DecimalField   float64                 `json:"decimal_field" astra:"type=decimal"`
		BoolField      bool                    `json:"bool_field"`
		TimestampField time.Time               `json:"timestamp_field"`
		DateField      time.Time               `json:"date_field" astra:"type=date"`
		TimeField      time.Time               `json:"time_field" astra:"type=time"`
		UUIDField      datatypes.UUID          `json:"uuid_field"`
		BlobField      []byte                  `json:"blob_field"`
		InetField      net.IP                  `json:"inet_field"`
		ListField      []string                `json:"list_field"`
		SetField       []string                `json:"set_field" astra:"type=set"`
		MapField       map[string]int          `json:"map_field"`
		VectorField    datatypes.DataAPIVector `json:"vector_field" astra:"dim=3"`
		PtrField       *string                 `json:"ptr_field"`
	}

	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}

	expect := map[string]Column{
		"pk":              Text(),
		"text_field":      Text(),
		"ascii_field":     Ascii(),
		"int_field":       Int(),
		"bigint_field":    BigInt(),
		"smallint_field":  SmallInt(),
		"tinyint_field":   TinyInt(),
		"float_field":     Float(),
		"double_field":    Double(),
		"decimal_field":   Decimal(),
		"bool_field":      Boolean(),
		"timestamp_field": Timestamp(),
		"date_field":      Date(),
		"time_field":      Time(),
		"uuid_field":      UUID(),
		"blob_field":      Blob(),
		"inet_field":      Inet(),
		"list_field":      List(Text()),
		"set_field":       Set(Text()),
		"map_field":       Map(TypeText, Int()),
		"vector_field":    Vector(3),
		"ptr_field":       Text(),
	}

	for name, wantCol := range expect {
		gotCol, ok := def.Columns.Get(name)
		if !ok {
			t.Errorf("missing column %q", name)
			continue
		}
		if !reflect.DeepEqual(gotCol, wantCol) {
			t.Errorf("column %q:\n got %+v\nwant %+v", name, gotCol, wantCol)
		}
	}
	if len(def.Columns) != len(expect) {
		t.Errorf("column count: got %d, want %d", len(def.Columns), len(expect))
	}
}

// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/tables.html#createTable
func TestInfer_JSONEquivalence_CompoundKey(t *testing.T) {
	type EventByDay struct {
		EventDate string `json:"event_date" astra:"pk,1"`
		ID        string `json:"id" astra:"pk,2"`
		Title     string `json:"title"`
		Location  string `json:"location"`
	}

	inferred, err := Infer[EventByDay]()
	if err != nil {
		t.Fatal(err)
	}

	manual := Definition{
		Columns: Columns{
			{"event_date", Text()},
			{"id", Text()},
			{"title", Text()},
			{"location", Text()},
		},
		PrimaryKey: PrimaryKey{PartitionBy: []string{"event_date", "id"}},
	}

	inferredJSON, _ := json.Marshal(inferred)
	manualJSON, _ := json.Marshal(manual)

	var inferredMap, manualMap map[string]any
	json.Unmarshal(inferredJSON, &inferredMap)
	json.Unmarshal(manualJSON, &manualMap)

	if !reflect.DeepEqual(inferredMap, manualMap) {
		t.Errorf("JSON mismatch:\n  inferred: %s\n  manual:   %s", inferredJSON, manualJSON)
	}
}

func TestInfer_EmbeddedStruct(t *testing.T) {
	type Base struct {
		Title  string  `json:"title" astra:"pk"`
		Rating float32 `json:"rating"`
	}
	type Extended struct {
		Base
		Similarity float64 `json:"$similarity"` // API metadata — should be skipped
		Extra      string  `json:"extra"`
	}

	def, err := Infer[Extended]()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := def.Columns.Get("$similarity"); ok {
		t.Error("$similarity should be skipped")
	}
	if _, ok := def.Columns.Get("title"); !ok {
		t.Error("embedded field 'title' missing")
	}
	if _, ok := def.Columns.Get("rating"); !ok {
		t.Error("embedded field 'rating' missing")
	}
	if _, ok := def.Columns.Get("extra"); !ok {
		t.Error("field 'extra' missing")
	}
	if len(def.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(def.Columns))
	}
}

func TestInfer_NestedEmbeddedStruct(t *testing.T) {
	type A struct {
		X string `json:"x" astra:"pk"`
		Y int    `json:"y"`
	}
	type B struct{ A }
	type C struct{ B }

	def, err := Infer[C]()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := def.Columns.Get("x"); !ok {
		t.Error("grand-embedded field 'x' missing")
	}
	if _, ok := def.Columns.Get("y"); !ok {
		t.Error("grand-embedded field 'y' missing")
	}
	if got, want := def.PrimaryKey.PartitionBy, []string{"x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("PartitionBy = %v, want %v", got, want)
	}
	if len(def.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(def.Columns))
	}
}

func TestInfer_DuplicateColumnName(t *testing.T) {
	// Built via reflect.StructOf so `go vet` doesn't flag the duplicate tag.
	typ := reflect.StructOf([]reflect.StructField{
		{Name: "A", Type: reflect.TypeFor[string](), Tag: reflect.StructTag(`json:"same" astra:"pk"`)},
		{Name: "B", Type: reflect.TypeFor[int](), Tag: reflect.StructTag(`json:"same" astra:"pk"`)},
	})
	_, err := collectFields(typ)
	if err == nil {
		t.Fatal("expected error for duplicate column name")
	}
	if !strings.Contains(err.Error(), "duplicate column name") {
		t.Errorf("error %q should mention duplicate column name", err)
	}
}

func TestInfer_FieldShadowing(t *testing.T) {
	type Base struct {
		Name string `json:"name" astra:"pk"`
		Foo  int    `json:"foo"`
	}
	type Outer struct {
		Base
		Foo string `json:"foo"` // shadows Base.Foo — should be text, not int
	}

	def, err := Infer[Outer]()
	if err != nil {
		t.Fatal(err)
	}

	col, _ := def.Columns.Get("foo")
	if col.Type != TypeText {
		t.Errorf("shadowed column type = %q, want %q", col.Type, TypeText)
	}
}

func TestInfer_Vectorize(t *testing.T) {
	type Row struct {
		ID    string `json:"id" astra:"pk"`
		Embed any    `json:"embed" astra:"vectorize,provider=openai,model=text-embedding-3-small"`
	}

	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}

	col, _ := def.Columns.Get("embed")
	if col.Type != TypeVector {
		t.Errorf("type = %q, want %q", col.Type, TypeVector)
	}
	if col.Service == nil {
		t.Fatal("expected vectorize service")
	}
	if col.Service.Provider != "openai" {
		t.Errorf("provider = %q, want openai", col.Service.Provider)
	}
	if col.Service.ModelName != "text-embedding-3-small" {
		t.Errorf("model = %q, want text-embedding-3-small", col.Service.ModelName)
	}
}

func TestInfer_VectorizeWithDimension(t *testing.T) {
	type Row struct {
		ID    string `json:"id" astra:"pk"`
		Embed any    `json:"embed" astra:"vectorize,provider=nvidia,model=NV-Embed-QA,dim=1536"`
	}

	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}

	col, _ := def.Columns.Get("embed")
	if col.Dimension == nil || *col.Dimension != 1536 {
		t.Errorf("dimension = %v, want 1536", col.Dimension)
	}
}

func TestInfer_JSONString(t *testing.T) {
	type Inner struct{ Foo string }
	type Row struct {
		ID   string  `json:"id" astra:"pk"`
		Data []Inner `json:"data" astra:"jsonString"`
	}

	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}

	col, _ := def.Columns.Get("data")
	if col.Type != TypeText {
		t.Errorf("jsonString column type = %q, want %q", col.Type, TypeText)
	}
}

func TestInfer_SkipFields(t *testing.T) {
	type Row struct {
		ID      string `json:"id" astra:"pk"`
		Visible string `json:"visible"`
		Hidden  string `json:"-"`
		Skipped string `json:"skipped" astra:"-"`
	}

	def, err := Infer[Row]()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := def.Columns.Get("Hidden"); ok {
		t.Error("json:\"-\" field should be skipped")
	}
	if _, ok := def.Columns.Get("skipped"); ok {
		t.Error("astra:\"-\" field should be skipped")
	}
	if len(def.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(def.Columns))
	}
}

func TestInfer_NoPrimaryKey(t *testing.T) {
	type Row struct {
		Name string `json:"name"`
	}
	_, err := Infer[Row]()
	if err == nil {
		t.Fatal("expected error for missing primary key")
	}
}

func TestInfer_BrokenCompositeKey_OrdinalGap(t *testing.T) {
	// Matches BrokenCompositePrimaryKey from models.go
	type Row struct {
		KeyTwo string `json:"keyTwo" astra:"pk,3"`
		KeyOne string `json:"keyOne" astra:"pk,1"`
	}
	_, err := Infer[Row]()
	if err == nil {
		t.Fatal("expected error for ordinal gap (1, 3)")
	}
}

func TestInfer_BrokenCompoundKey_OrdinalZero(t *testing.T) {
	// Matches BrokenCompoundPrimaryKey from models.go
	type Row struct {
		KeyTwo            string `json:"keyTwo" astra:"pk,2"`
		KeyOne            string `json:"keyOne" astra:"pk,1"`
		SortTwoDescending string `json:"sortTwoDescending" astra:"ck,2,desc"`
		SortOneAscending  string `json:"sortOneAscending" astra:"ck,0,asc"`
	}
	_, err := Infer[Row]()
	if err == nil {
		t.Fatal("expected error for ck ordinal 0")
	}
}

func TestInfer_DuplicateOrdinals(t *testing.T) {
	type Row struct {
		A string `json:"a" astra:"pk,1"`
		B string `json:"b" astra:"pk,1"`
	}
	_, err := Infer[Row]()
	if err == nil {
		t.Fatal("expected error for duplicate ordinals")
	}
}

func TestInfer_MixedOrdinals(t *testing.T) {
	type Row struct {
		A string `json:"a" astra:"pk,1"`
		B string `json:"b" astra:"pk"` // no ordinal — mixed with ordinal
	}
	_, err := Infer[Row]()
	if err == nil {
		t.Fatal("expected error for mixed ordinals")
	}
}

func TestInfer_AllFieldsSkipped(t *testing.T) {
	type Row struct {
		A string `json:"-"`
		B string `json:"-"`
	}
	_, err := Infer[Row]()
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

// TestInfer_BracketTypeOverride covers the parameterized type= forms:
// set[T], list[T], map[K]V, udt[<name>], and the infer keyword. Bare
// containers (type=set / type=list / type=map) are still covered by
// TestGoTypeToColumn and the TestInferDocs* suite.
func TestInfer_BracketTypeOverride(t *testing.T) {
	t.Run("set[ascii] on []string", func(t *testing.T) {
		type Row struct {
			ID   string   `json:"id" astra:"pk"`
			Tags []string `json:"tags" astra:"type=set[ascii]"`
		}
		def, err := Infer[Row]()
		if err != nil {
			t.Fatalf("Infer errored: %v", err)
		}
		col, _ := def.Columns.Get("tags")
		if col.Type != TypeSet {
			t.Fatalf("outer type = %q, want %q", col.Type, TypeSet)
		}
		if col.ValueType == nil || col.ValueType.Type != TypeAscii {
			t.Errorf("value type = %+v, want ascii", col.ValueType)
		}
	})

	t.Run("list[blob] on []string", func(t *testing.T) {
		type Row struct {
			ID   string   `json:"id" astra:"pk"`
			Data []string `json:"data" astra:"type=list[blob]"`
		}
		def, err := Infer[Row]()
		if err != nil {
			t.Fatalf("Infer errored: %v", err)
		}
		col, _ := def.Columns.Get("data")
		if col.Type != TypeList {
			t.Fatalf("outer = %q, want %q", col.Type, TypeList)
		}
		if col.ValueType == nil || col.ValueType.Type != TypeBlob {
			t.Errorf("value = %+v, want blob", col.ValueType)
		}
	})

	t.Run("map[uuid]blob on map[string]string", func(t *testing.T) {
		type Row struct {
			ID string            `json:"id" astra:"pk"`
			M  map[string]string `json:"m" astra:"type=map[uuid]blob"`
		}
		def, err := Infer[Row]()
		if err != nil {
			t.Fatalf("Infer errored: %v", err)
		}
		col, _ := def.Columns.Get("m")
		if col.Type != TypeMap {
			t.Fatalf("outer = %q, want %q", col.Type, TypeMap)
		}
		if col.KeyType == nil || *col.KeyType != TypeUUID {
			t.Errorf("key = %v, want %q", col.KeyType, TypeUUID)
		}
		if col.ValueType == nil || col.ValueType.Type != TypeBlob {
			t.Errorf("value = %+v, want blob", col.ValueType)
		}
	})

	t.Run("map[infer]infer on map[UUID][]byte", func(t *testing.T) {
		type Row struct {
			ID string                    `json:"id" astra:"pk"`
			M  map[datatypes.UUID][]byte `json:"m" astra:"type=map[infer]infer"`
		}
		def, err := Infer[Row]()
		if err != nil {
			t.Fatalf("Infer errored: %v", err)
		}
		col, _ := def.Columns.Get("m")
		if col.KeyType == nil || *col.KeyType != TypeUUID {
			t.Errorf("key = %v, want %q", col.KeyType, TypeUUID)
		}
		if col.ValueType == nil || col.ValueType.Type != TypeBlob {
			t.Errorf("value = %+v, want blob", col.ValueType)
		}
	})

	t.Run("udt[person]", func(t *testing.T) {
		type Person struct{ Name string }
		type Row struct {
			ID string  `json:"id" astra:"pk"`
			P  *Person `json:"p" astra:"type=udt[person]"`
		}
		def, err := Infer[Row]()
		if err != nil {
			t.Fatalf("Infer errored: %v", err)
		}
		col, _ := def.Columns.Get("p")
		if col.Type != TypeUDT {
			t.Fatalf("type = %q, want %q", col.Type, TypeUDT)
		}
		if col.UDTName == nil || *col.UDTName != "person" {
			t.Errorf("UDT name = %v, want person", col.UDTName)
		}
	})

	t.Run("map[text]udt[person]", func(t *testing.T) {
		type Person struct{ Name string }
		type Row struct {
			ID string             `json:"id" astra:"pk"`
			M  map[string]*Person `json:"m" astra:"type=map[text]udt[person]"`
		}
		def, err := Infer[Row]()
		if err != nil {
			t.Fatalf("Infer errored: %v", err)
		}
		col, _ := def.Columns.Get("m")
		if col.KeyType == nil || *col.KeyType != TypeText {
			t.Errorf("key = %v, want %q", col.KeyType, TypeText)
		}
		if col.ValueType == nil || col.ValueType.Type != TypeUDT {
			t.Fatalf("value type = %+v, want udt", col.ValueType)
		}
		if col.ValueType.UDTName == nil || *col.ValueType.UDTName != "person" {
			t.Errorf("value UDT name = %v, want person", col.ValueType.UDTName)
		}
	})

	t.Run("set[ascii] on non-slice errors", func(t *testing.T) {
		type Row struct {
			ID string `json:"id" astra:"pk"`
			S  string `json:"s" astra:"type=set[ascii]"`
		}
		_, err := Infer[Row]()
		if err == nil {
			t.Fatal("expected error for type=set on non-slice")
		}
		if !strings.Contains(err.Error(), "type=set requires slice") {
			t.Errorf("error = %q, want containing %q", err.Error(), "type=set requires slice")
		}
	})
}
