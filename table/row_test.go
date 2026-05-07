package table

import (
	"encoding/json"
	"math/big"
	"net"
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/serdes"
)

func TestRow_UnmarshalAstraRaw_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   Columns
		jsonData string
		want     map[string]any
	}{
		{
			name: "int field",
			schema: Columns{
				{Name: "id", Column: Column{Type: TypeInt}},
			},
			jsonData: `{"id": 42}`,
			want:     map[string]any{"id": 42},
		},
		{
			name: "string field",
			schema: Columns{
				{Name: "name", Column: Column{Type: TypeText}},
			},
			jsonData: `{"name": "test"}`,
			want:     map[string]any{"name": "test"},
		},
		{
			name: "boolean field",
			schema: Columns{
				{Name: "active", Column: Column{Type: TypeBoolean}},
			},
			jsonData: `{"active": true}`,
			want:     map[string]any{"active": true},
		},
		{
			name: "multiple fields",
			schema: Columns{
				{Name: "id", Column: Column{Type: TypeInt}},
				{Name: "name", Column: Column{Type: TypeText}},
				{Name: "active", Column: Column{Type: TypeBoolean}},
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
			row := &Row{schema: tt.schema}
			err := serdes.Deserialize([]byte(tt.jsonData), row, serdes.TargetTable)
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
		schema   Columns
		jsonData string
		validate func(t *testing.T, data map[string]any)
	}{
		{
			name: "bigint",
			schema: Columns{
				{Name: "big", Column: Column{Type: TypeBigInt}},
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
			schema: Columns{
				{Name: "small", Column: Column{Type: TypeSmallInt}},
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
			schema: Columns{
				{Name: "tiny", Column: Column{Type: TypeTinyInt}},
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
			schema: Columns{
				{Name: "f", Column: Column{Type: TypeFloat}},
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
			schema: Columns{
				{Name: "d", Column: Column{Type: TypeDouble}},
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
			row := &Row{schema: tt.schema}
			err := serdes.Deserialize([]byte(tt.jsonData), row, serdes.TargetTable)
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
		schema   Columns
		jsonData string
	}{
		{
			name: "null value",
			schema: Columns{
				{Name: "name", Column: Column{Type: TypeText}},
			},
			jsonData: `{"name": null}`,
		},
		{
			name: "missing field",
			schema: Columns{
				{Name: "name", Column: Column{Type: TypeText}},
				{Name: "age", Column: Column{Type: TypeInt}},
			},
			jsonData: `{"name": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &Row{schema: tt.schema}
			err := serdes.Deserialize([]byte(tt.jsonData), row, serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Check that all schema fields exist in Data
			for _, nc := range tt.schema {
				val, ok := row.ToMap()[nc.Name]
				if !ok {
					t.Errorf("field %s missing from Data", nc.Name)
					continue
				}

				// For missing or null fields, value should be nil
				if tt.name == "null value" || (tt.name == "missing field" && nc.Name == "age") {
					if val != nil {
						t.Errorf("field %s: expected nil, got %v", nc.Name, val)
					}
				}
			}
		})
	}
}

func TestRow_UnmarshalAstraRaw_Collections(t *testing.T) {
	tests := []struct {
		name     string
		schema   Columns
		jsonData string
		validate func(t *testing.T, data map[string]any)
	}{
		{
			name: "list of ints",
			schema: Columns{
				{Name: "numbers", Column: Column{
					Type:      TypeList,
					ValueType: &Column{Type: TypeInt},
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
			schema: Columns{
				{Name: "tags", Column: Column{
					Type:      TypeSet,
					ValueType: &Column{Type: TypeText},
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
			schema: Columns{
				{Name: "scores", Column: Column{
					Type:      TypeMap,
					KeyType:   func() *string { s := TypeText; return &s }(),
					ValueType: &Column{Type: TypeInt},
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
			row := &Row{schema: tt.schema}
			err := serdes.Deserialize([]byte(tt.jsonData), row, serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}
			tt.validate(t, row.ToMap())
		})
	}
}

func TestRow_UnmarshalAstraRaw_UDT(t *testing.T) {
	// Define a simple UDT schema
	addressDef := &UDTDefinition{
		Fields: Columns{
			{Name: "street", Column: Column{Type: TypeText}},
			{Name: "city", Column: Column{Type: TypeText}},
			{Name: "zip", Column: Column{Type: TypeInt}},
		},
	}

	schema := Columns{
		{Name: "address", Column: Column{
			Type:       TypeUDT,
			definition: addressDef,
		}},
	}

	jsonData := `{"address": {"street": "123 Main St", "city": "Springfield", "zip": 12345}}`

	row := &Row{schema: schema}
	err := serdes.Deserialize([]byte(jsonData), row, serdes.TargetTable)
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

func TestRow_UnmarshalAstraRaw_SpecialTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   Columns
		jsonData string
		validate func(t *testing.T, data map[string]any)
	}{
		{
			name: "uuid",
			schema: Columns{
				{Name: "id", Column: Column{Type: TypeUUID}},
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
			schema: Columns{
				{Name: "data", Column: Column{Type: TypeBlob}},
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
			schema: Columns{
				{Name: "ip", Column: Column{Type: TypeInet}},
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
			schema: Columns{
				{Name: "embedding", Column: Column{Type: TypeVector}},
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
			row := &Row{schema: tt.schema}
			err := serdes.Deserialize([]byte(tt.jsonData), row, serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}
			tt.validate(t, row.ToMap())
		})
	}
}

func TestRow_UnmarshalAstraRaw_InvalidJSON(t *testing.T) {
	schema := Columns{
		{Name: "id", Column: Column{Type: TypeInt}},
	}

	row := &Row{schema: schema}
	err := serdes.Deserialize([]byte(`{invalid json}`), row, serdes.TargetTable)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestRow_UnmarshalAstraRaw_MissingCollectionType(t *testing.T) {
	tests := []struct {
		name     string
		schema   Columns
		jsonData string
	}{
		{
			name: "list without valueType",
			schema: Columns{
				{Name: "items", Column: Column{Type: TypeList}},
			},
			jsonData: `{"items": [1, 2, 3]}`,
		},
		{
			name: "set without valueType",
			schema: Columns{
				{Name: "items", Column: Column{Type: TypeSet}},
			},
			jsonData: `{"items": [1, 2, 3]}`,
		},
		{
			name: "map without keyType",
			schema: Columns{
				{Name: "items", Column: Column{Type: TypeMap, ValueType: &Column{Type: TypeInt}}},
			},
			jsonData: `{"items": {"a": 1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &Row{schema: tt.schema}
			err := serdes.Deserialize([]byte(tt.jsonData), row, serdes.TargetTable)
			if err == nil {
				t.Error("expected error for missing collection type, got nil")
			}
		})
	}
}

func TestDeserializeWithTypeHint_Varint(t *testing.T) {
	raw := json.RawMessage(`12345678901234567890`)
	col := Column{Type: TypeVarint}

	val, err := deserializeColumn(raw, col)
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
	col := Column{Type: TypeDecimal}

	val, err := deserializeColumn(raw, col)
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
	col := Column{Type: "unknown_type"}

	val, err := deserializeColumn(raw, col)
	if err != nil {
		t.Fatalf("deserializeColumn() error = %v", err)
	}

	// Should fall back to generic deserialization
	if val == nil {
		t.Error("expected non-nil value for unknown type")
	}
}
