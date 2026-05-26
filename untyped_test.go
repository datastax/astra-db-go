package astradb

import (
	"encoding/json"
	"math/big"
	"net"
	"reflect"
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/serdes"
	"github.com/datastax/astra-db-go/table"
)

func TestDocument_DeferredDecoding(t *testing.T) {
	jsonData := `{"id": "123", "name": "Alice", "meta": {"score": 0.95}}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, documentCtx, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Verify it's a serverDocument
	if reflect.TypeOf(doc).String() != "*astradb.serverDocument" {
		t.Errorf("expected *serverDocument, got %T", doc)
	}

	// Test Get()
	id, ok := doc.Get("id")
	if !ok || id != "123" {
		t.Errorf("Get(id): got %v, ok %v, want 123", id, ok)
	}

	score, ok := doc.Get("meta", "score")
	if !ok || score != 0.95 {
		t.Errorf("Get(meta.score): got %v, ok %v, want 0.95", score, ok)
	}

	// Test Decode()
	var name string
	if err := doc.Decode(&name, "name"); err != nil {
		t.Fatalf("Decode(name) error = %v", err)
	}
	if name != "Alice" {
		t.Errorf("Decode(name): got %s, want Alice", name)
	}

	var m map[string]any
	if err := doc.Decode(&m, "meta"); err != nil {
		t.Fatalf("Decode(meta) error = %v", err)
	}
	if m["score"] != 0.95 {
		t.Errorf("Decode(meta) score: got %v, want 0.95", m["score"])
	}

	// Test ToMap()
	fullMap := doc.ToMap()
	if fullMap["id"] != "123" || fullMap["name"] != "Alice" {
		t.Errorf("ToMap() mismatch: %v", fullMap)
	}
	meta := fullMap["meta"].(map[string]any)
	if meta["score"] != 0.95 {
		t.Errorf("ToMap() nested mismatch: %v", meta)
	}
}

func TestNewDocument_Insertion(t *testing.T) {
	doc := NewDocument{
		"id":   "456",
		"tags": []string{"a", "b"},
	}

	encoded, err := serdes.Serialize(doc, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := `{"id":"456","tags":["a","b"]}`
	if string(encoded) != expected {
		t.Errorf("expected %s, got %s", expected, string(encoded))
	}

	// Verify it cannot be used for results
	var res NewDocument
	err = serdes.Deserialize([]byte(expected), &res, nil, serdes.TargetCollection)
	if err == nil {
		t.Error("expected error when deserializing into NewDocument, got nil")
	}
}

func TestDocument_NullHandling(t *testing.T) {
	jsonData := `{"id": "123", "optional": null}`

	var doc Document
	serdes.Deserialize([]byte(jsonData), &doc, documentCtx, serdes.TargetCollection)

	val, ok := doc.Get("optional")
	if !ok || val != nil {
		t.Errorf("Get(optional): got %v, ok %v, want nil, true", val, ok)
	}

	fullMap := doc.ToMap()
	if v, ok := fullMap["optional"]; !ok || v != nil {
		t.Errorf("ToMap(optional): got %v, ok %v, want nil, true", v, ok)
	}
}

func TestDocument_DeepPathNotFound(t *testing.T) {
	jsonData := `{"id": "123", "meta": {"score": 0.95}}`

	var doc Document
	serdes.Deserialize([]byte(jsonData), &doc, documentCtx, serdes.TargetCollection)

	_, ok := doc.Get("meta", "missing")
	if ok {
		t.Error("expected ok=false for missing deep path")
	}

	_, ok = doc.Get("missing", "path")
	if ok {
		t.Error("expected ok=false for missing root path")
	}
}

func TestRow_UnmarshalAstraRaw_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   table.Columns
		jsonData string
		want     map[string]any
	}{
		{
			name: "int field",
			schema: table.Columns{
				{Name: "id", Column: table.Column{Type: table.TypeInt}},
			},
			jsonData: `{"id": 42}`,
			want:     map[string]any{"id": 42},
		},
		{
			name: "string field",
			schema: table.Columns{
				{Name: "name", Column: table.Column{Type: table.TypeText}},
			},
			jsonData: `{"name": "test"}`,
			want:     map[string]any{"name": "test"},
		},
		{
			name: "boolean field",
			schema: table.Columns{
				{Name: "active", Column: table.Column{Type: table.TypeBoolean}},
			},
			jsonData: `{"active": true}`,
			want:     map[string]any{"active": true},
		},
		{
			name: "multiple fields",
			schema: table.Columns{
				{Name: "id", Column: table.Column{Type: table.TypeInt}},
				{Name: "name", Column: table.Column{Type: table.TypeText}},
				{Name: "active", Column: table.Column{Type: table.TypeBoolean}},
			},
			jsonData: `{"id": 123, "name": "alice", "active": false}`,
			want: map[string]any{
				"id":     123,
				"name":   "alice",
				"active": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row Row
			err := serdes.Deserialize([]byte(tt.jsonData), &row, NewRowTargetCtx(tt.schema), serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			for key, wantVal := range tt.want {
				gotVal, ok := row.ToMap()[key]
				if !ok {
					t.Errorf("missing field %s in result", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("field %s: got %v (%T), want %v (%T)", key, gotVal, gotVal, wantVal, wantVal)
				}
			}
		})
	}
}

func TestRow_UnmarshalAstraRaw_NumericTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   table.Columns
		jsonData string
		validate func(t *testing.T, data map[string]any)
	}{
		{
			name: "bigint",
			schema: table.Columns{
				{Name: "big", Column: table.Column{Type: table.TypeBigInt}},
			},
			jsonData: `{"big": 9223372036854775807}`,
			validate: func(t *testing.T, data map[string]any) {
				val, ok := data["big"].(int64)
				if !ok {
					t.Errorf("expected int64, got %T", data["big"])
					return
				}
				if val != 9223372036854775807 {
					t.Errorf("got %d, want 9223372036854775807", val)
				}
			},
		},
		{
			name: "smallint",
			schema: table.Columns{
				{Name: "small", Column: table.Column{Type: table.TypeSmallInt}},
			},
			jsonData: `{"small": 32767}`,
			validate: func(t *testing.T, data map[string]any) {
				val, ok := data["small"].(int16)
				if !ok {
					t.Errorf("expected int16, got %T", data["small"])
					return
				}
				if val != 32767 {
					t.Errorf("got %d, want 32767", val)
				}
			},
		},
		{
			name: "tinyint",
			schema: table.Columns{
				{Name: "tiny", Column: table.Column{Type: table.TypeTinyInt}},
			},
			jsonData: `{"tiny": 127}`,
			validate: func(t *testing.T, data map[string]any) {
				val, ok := data["tiny"].(int8)
				if !ok {
					t.Errorf("expected int8, got %T", data["tiny"])
					return
				}
				if val != 127 {
					t.Errorf("got %d, want 127", val)
				}
			},
		},
		{
			name: "float",
			schema: table.Columns{
				{Name: "f", Column: table.Column{Type: table.TypeFloat}},
			},
			jsonData: `{"f": 3.14}`,
			validate: func(t *testing.T, data map[string]any) {
				val, ok := data["f"].(float32)
				if !ok {
					t.Errorf("expected float32, got %T", data["f"])
					return
				}
				if val < 3.13 || val > 3.15 {
					t.Errorf("got %f, want ~3.14", val)
				}
			},
		},
		{
			name: "double",
			schema: table.Columns{
				{Name: "d", Column: table.Column{Type: table.TypeDouble}},
			},
			jsonData: `{"d": 3.141592653589793}`,
			validate: func(t *testing.T, data map[string]any) {
				val, ok := data["d"].(float64)
				if !ok {
					t.Errorf("expected float64, got %T", data["d"])
					return
				}
				if val < 3.14 || val > 3.15 {
					t.Errorf("got %f, want ~3.141592653589793", val)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row Row
			err := serdes.Deserialize([]byte(tt.jsonData), &row, NewRowTargetCtx(tt.schema), serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}
			tt.validate(t, row.ToMap())
		})
	}
}

func TestRow_UnmarshalAstraRaw_NullHandling(t *testing.T) {
	tests := []struct {
		name     string
		schema   table.Columns
		jsonData string
	}{
		{
			name: "null value",
			schema: table.Columns{
				{Name: "name", Column: table.Column{Type: table.TypeText}},
			},
			jsonData: `{"name": null}`,
		},
		{
			name: "missing field",
			schema: table.Columns{
				{Name: "name", Column: table.Column{Type: table.TypeText}},
				{Name: "age", Column: table.Column{Type: table.TypeInt}},
			},
			jsonData: `{"name": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row Row
			err := serdes.Deserialize([]byte(tt.jsonData), &row, NewRowTargetCtx(tt.schema), serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Check that all data fields exist in the result
			dataMap := row.ToMap()
			for _, nc := range tt.schema {
				val, ok := dataMap[nc.Name]

				if tt.name == "null value" {
					if !ok {
						t.Errorf("field %s missing from Data", nc.Name)
					} else if val != nil {
						t.Errorf("field %s: expected nil, got %v", nc.Name, val)
					}
				}

				if tt.name == "missing field" {
					if nc.Name == "age" {
						if ok {
							t.Errorf("field age should be missing from Data")
						}
					} else {
						if !ok {
							t.Errorf("field %s missing from Data", nc.Name)
						}
					}
				}
			}
		})
	}
}

func TestRow_UnmarshalAstraRaw_Collections(t *testing.T) {
	tests := []struct {
		name     string
		schema   table.Columns
		jsonData string
		validate func(t *testing.T, data map[string]any)
	}{
		{
			name: "list of ints",
			schema: table.Columns{
				{Name: "numbers", Column: table.Column{
					Type:      table.TypeList,
					ValueType: &table.Column{Type: table.TypeInt},
				}},
			},
			jsonData: `{"numbers": [1, 2, 3]}`,
			validate: func(t *testing.T, data map[string]any) {
				list, ok := data["numbers"].([]any)
				if !ok {
					t.Errorf("expected []any, got %T", data["numbers"])
					return
				}
				if len(list) != 3 {
					t.Errorf("expected 3 elements, got %d", len(list))
					return
				}
				for i, expected := range []int{1, 2, 3} {
					if list[i] != expected {
						t.Errorf("element %d: got %v, want %d", i, list[i], expected)
					}
				}
			},
		},
		{
			name: "set of strings",
			schema: table.Columns{
				{Name: "tags", Column: table.Column{
					Type:      table.TypeSet,
					ValueType: &table.Column{Type: table.TypeText},
				}},
			},
			jsonData: `{"tags": ["go", "test", "db"]}`,
			validate: func(t *testing.T, data map[string]any) {
				set, ok := data["tags"].([]any)
				if !ok {
					t.Errorf("expected []any, got %T", data["tags"])
					return
				}
				if len(set) != 3 {
					t.Errorf("expected 3 elements, got %d", len(set))
				}
			},
		},
		{
			name: "map string to int",
			schema: table.Columns{
				{Name: "scores", Column: table.Column{
					Type:      table.TypeMap,
					KeyType:   func() *string { s := table.TypeText; return &s }(),
					ValueType: &table.Column{Type: table.TypeInt},
				}},
			},
			jsonData: `{"scores": {"alice": 100, "bob": 95}}`,
			validate: func(t *testing.T, data map[string]any) {
				m, ok := data["scores"].(map[string]any)
				if !ok {
					t.Errorf("expected map[any]any, got %T", data["scores"])
					return
				}
				if len(m) != 2 {
					t.Errorf("expected 2 entries, got %d", len(m))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row Row
			err := serdes.Deserialize([]byte(tt.jsonData), &row, NewRowTargetCtx(tt.schema), serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}
			tt.validate(t, row.ToMap())
		})
	}
}

func TestRow_UnmarshalAstraRaw_UDT(t *testing.T) {
	// Define a simple UDT Schema
	addressDef := &table.UDTDefinition{
		Fields: table.Columns{
			{Name: "street", Column: table.Column{Type: table.TypeText}},
			{Name: "city", Column: table.Column{Type: table.TypeText}},
			{Name: "zip", Column: table.Column{Type: table.TypeInt}},
		},
	}

	schema := table.Columns{
		{Name: "address", Column: table.Column{
			Type:          table.TypeUDT,
			UDTDefinition: addressDef,
		}},
	}

	jsonData := `{"address": {"street": "123 Main St", "city": "Springfield", "zip": 12345}}`

	var row Row
	err := serdes.Deserialize([]byte(jsonData), &row, NewRowTargetCtx(schema), serdes.TargetTable)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	address, ok := row.ToMap()["address"].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", row.ToMap()["address"])
	}

	if address["street"] != "123 Main St" {
		t.Errorf("street: got %v, want '123 Main St'", address["street"])
	}
	if address["city"] != "Springfield" {
		t.Errorf("city: got %v, want 'Springfield'", address["city"])
	}
	if address["zip"] != 12345 {
		t.Errorf("zip: got %v, want 12345", address["zip"])
	}
}

func TestRow_Decode_NestedUDT(t *testing.T) {
	// Define a simple UDT Schema
	addressDef := &table.UDTDefinition{
		Fields: table.Columns{
			{Name: "street", Column: table.Column{Type: table.TypeText}},
			{Name: "city", Column: table.Column{Type: table.TypeText}},
			{Name: "zip", Column: table.Column{Type: table.TypeInt}},
		},
	}

	schema := table.Columns{
		{Name: "address", Column: table.Column{
			Type:          table.TypeUDT,
			UDTDefinition: addressDef,
		}},
	}

	jsonData := `{"address": {"street": "123 Main St", "city": "Springfield", "zip": 12345}}`

	var row Row
	err := serdes.Deserialize([]byte(jsonData), &row, NewRowTargetCtx(schema), serdes.TargetTable)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Test Decode into a Row (nested)
	var nestedRow Row
	if err := row.Decode(&nestedRow, "address"); err != nil {
		t.Fatalf("Decode(address) error = %v", err)
	}

	street, ok := nestedRow.Get("street")
	if !ok || street != "123 Main St" {
		t.Errorf("nestedRow.Get(street): got %v, ok %v, want '123 Main St'", street, ok)
	}

	zip, ok := nestedRow.Get("zip")
	if !ok || zip != 42 { // Wait, zip is 12345 in jsonData
		if !ok || zip != 12345 {
			t.Errorf("nestedRow.Get(zip): got %v, ok %v, want 12345", zip, ok)
		}
	}
}

func TestRow_UnmarshalAstraRaw_SpecialTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   table.Columns
		jsonData string
		validate func(t *testing.T, data map[string]any)
	}{
		{
			name: "uuid",
			schema: table.Columns{
				{Name: "id", Column: table.Column{Type: table.TypeUUID}},
			},
			jsonData: `{"id": "550e8400-e29b-41d4-a716-446655440000"}`,
			validate: func(t *testing.T, data map[string]any) {
				_, ok := data["id"].(datatypes.UUID)
				if !ok {
					t.Errorf("expected datatypes.UUID, got %T", data["id"])
				}
			},
		},
		{
			name: "blob",
			schema: table.Columns{
				{Name: "data", Column: table.Column{Type: table.TypeBlob}},
			},
			jsonData: `{"data": {"$binary": "aGVsbG8="}}`,
			validate: func(t *testing.T, data map[string]any) {
				blob, ok := data["data"].([]byte)
				if !ok {
					t.Errorf("expected []byte, got %T", data["data"])
					return
				}
				if string(blob) != "hello" {
					t.Errorf("got %s, want 'hello'", string(blob))
				}
			},
		},
		{
			name: "inet",
			schema: table.Columns{
				{Name: "ip", Column: table.Column{Type: table.TypeInet}},
			},
			jsonData: `{"ip": [192, 168, 1, 1]}`,
			validate: func(t *testing.T, data map[string]any) {
				ip, ok := data["ip"].(net.IP)
				if !ok {
					t.Errorf("expected net.IP, got %T", data["ip"])
					return
				}
				expected := net.IPv4(192, 168, 1, 1)
				if !ip.Equal(expected) {
					t.Errorf("got %v, want %v", ip, expected)
				}
			},
		},
		{
			name: "vector",
			schema: table.Columns{
				{Name: "embedding", Column: table.Column{Type: table.TypeVector}},
			},
			jsonData: `{"embedding": [0.1, 0.2, 0.3]}`,
			validate: func(t *testing.T, data map[string]any) {
				_, ok := data["embedding"].(datatypes.DataAPIVector)
				if !ok {
					t.Errorf("expected datatypes.DataAPIVector, got %T", data["embedding"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row Row
			err := serdes.Deserialize([]byte(tt.jsonData), &row, NewRowTargetCtx(tt.schema), serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}
			tt.validate(t, row.ToMap())
		})
	}
}

func TestRow_UnmarshalAstraRaw_InvalidJSON(t *testing.T) {
	schema := table.Columns{
		{Name: "id", Column: table.Column{Type: table.TypeInt}},
	}

	var row Row
	err := serdes.Deserialize([]byte(`{invalid json}`), &row, NewRowTargetCtx(schema), serdes.TargetTable)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestDeserializeWithTypeHint_Varint(t *testing.T) {
	raw := json.RawMessage(`12345678901234567890`)
	col := table.Column{Type: table.TypeVarint}

	ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
	val, err := deserializeColumn(ctx, raw, col)
	if err != nil {
		t.Fatalf("deserializeColumn() error = %v", err)
	}

	bigInt, ok := val.(big.Int)
	if !ok {
		t.Fatalf("expected big.Int, got %T", val)
	}

	expected := new(big.Int)
	expected.SetString("12345678901234567890", 10)

	if bigInt.Cmp(expected) != 0 {
		t.Errorf("got %s, want %s", bigInt.String(), expected.String())
	}
}

func TestDeserializeWithTypeHint_Decimal(t *testing.T) {
	raw := json.RawMessage(`123.456`)
	col := table.Column{Type: table.TypeDecimal}

	ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
	val, err := deserializeColumn(ctx, raw, col)
	if err != nil {
		t.Fatalf("deserializeColumn() error = %v", err)
	}

	_, ok := val.(big.Float)
	if !ok {
		t.Fatalf("expected big.Float, got %T", val)
	}
}

func TestDeserializeWithTypeHint_UnknownType(t *testing.T) {
	raw := json.RawMessage(`"test"`)
	col := table.Column{Type: "unknown_type"}

	ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
	val, err := deserializeColumn(ctx, raw, col)
	if err != nil {
		t.Fatalf("deserializeColumn() error = %v", err)
	}

	// Should fall back to generic deserialization
	if val == nil {
		t.Error("expected non-nil value for unknown type")
	}
}
func TestDocument_MustGet(t *testing.T) {
	jsonData := `{"id": "123", "meta": {"score": 0.95}}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, documentCtx, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Test success
	id := doc.MustGet("id").(string)
	if id != "123" {
		t.Errorf("MustGet(id): got %v, want 123", id)
	}

	score := doc.MustGet("meta", "score").(float64)
	if score != 0.95 {
		t.Errorf("MustGet(meta.score): got %v, want 0.95", score)
	}

	// Test panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet(missing) did not panic")
		}
	}()

	doc.MustGet("missing")
}

func TestRow_MustGet(t *testing.T) {
	schema := table.Columns{
		{Name: "id", Column: table.Column{Type: table.TypeInt}},
		{Name: "meta", Column: table.Column{
			Type:      table.TypeMap,
			ValueType: &table.Column{Type: table.TypeFloat},
		}},
	}
	jsonData := `{"id": 123, "meta": {"score": 0.95}}`

	var row Row
	err := serdes.Deserialize([]byte(jsonData), &row, NewRowTargetCtx(schema), serdes.TargetTable)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Test success
	id := row.MustGet("id").(int)
	if id != 123 {
		t.Errorf("MustGet(id): got %v, want 123", id)
	}

	score := row.MustGet("meta", "score").(float32)
	if score != 0.95 {
		t.Errorf("MustGet(meta.score): got %v, want 0.95", score)
	}

	// Test panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet(missing) did not panic")
		}
	}()

	row.MustGet("missing")
}

func TestNewDocument_MustGet(t *testing.T) {
	doc := NewDocument{"id": "123", "meta": map[string]any{"score": 0.95}}

	// Test success
	id := doc.MustGet("id").(string)
	if id != "123" {
		t.Errorf("MustGet(id): got %v, want 123", id)
	}

	// Test panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet(missing) did not panic")
		}
	}()

	doc.MustGet("missing")
}

func TestNewRow_MustGet(t *testing.T) {
	row := NewRow{"id": 123, "meta": map[string]any{"score": 0.95}}

	// Test success
	id := row.MustGet("id").(int)
	if id != 123 {
		t.Errorf("MustGet(id): got %v, want 123", id)
	}

	// Test panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet(missing) did not panic")
		}
	}()

	row.MustGet("missing")
}
