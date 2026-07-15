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

package untyped

import (
	"encoding/json"
	"math/big"
	"net"
	"reflect"
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/table"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
	"pgregory.net/rapid"
)

func TestRow_UnmarshalAstraRaw(t *testing.T) {
	tests := []struct {
		name     string
		schema   table.Columns
		jsonData string
		want     map[string]any
		validate func(t *testing.T, row Row)
	}{
		{
			name: "primitive fields",
			schema: table.Columns{
				{Name: "id", Column: table.Column{Type: table.TypeInt}},
				{Name: "name", Column: table.Column{Type: table.TypeText}},
				{Name: "active", Column: table.Column{Type: table.TypeBoolean}},
			},
			jsonData: `{"id": 42, "name": "test", "active": true}`,
			want: map[string]any{
				"id":     int32(42),
				"name":   "test",
				"active": true,
			},
		},
		{
			name: "numeric types",
			schema: table.Columns{
				{Name: "big", Column: table.Column{Type: table.TypeBigInt}},
				{Name: "small", Column: table.Column{Type: table.TypeSmallInt}},
				{Name: "tiny", Column: table.Column{Type: table.TypeTinyInt}},
				{Name: "f", Column: table.Column{Type: table.TypeFloat}},
				{Name: "d", Column: table.Column{Type: table.TypeDouble}},
			},
			jsonData: `{"big": 9223372036854775807, "small": 32767, "tiny": 127, "f": 3.14, "d": 3.141592653589793}`,
			want: map[string]any{
				"big":   int64(9223372036854775807),
				"small": int16(32767),
				"tiny":  int8(127),
				"f":     float32(3.14),
				"d":     3.141592653589793,
			},
		},
		{
			name: "collections",
			schema: table.Columns{
				{Name: "numbers", Column: table.Column{
					Type:      table.TypeList,
					ValueType: &table.Column{Type: table.TypeInt},
				}},
				{Name: "scores", Column: table.Column{
					Type:      table.TypeMap,
					KeyType:   func() *string { s := table.TypeText; return &s }(),
					ValueType: &table.Column{Type: table.TypeInt},
				}},
			},
			jsonData: `{"numbers": [1, 2, 3], "scores": {"alice": 100}}`,
			want: map[string]any{
				"numbers": []int32{1, 2, 3},
				"scores":  map[string]int32{"alice": 100},
			},
		},
		{
			name: "UDT",
			schema: table.Columns{
				{Name: "address", Column: table.Column{
					Type: table.TypeUDT,
					UDTDefinition: &table.UDTDefinition{
						Fields: table.Columns{
							{Name: "city", Column: table.Column{Type: table.TypeText}},
							{Name: "zip", Column: table.Column{Type: table.TypeInt}},
						},
					},
				}},
			},
			jsonData: `{"address": {"city": "Springfield", "zip": 12345}}`,
			want: map[string]any{
				"address": map[string]any{"city": "Springfield", "zip": int32(12345)},
			},
		},
		{
			name: "special types",
			schema: table.Columns{
				{Name: "uuid", Column: table.Column{Type: table.TypeUUID}},
				{Name: "ip", Column: table.Column{Type: table.TypeInet}},
				{Name: "blob", Column: table.Column{Type: table.TypeBlob}},
			},
			jsonData: `{"uuid": "550e8400-e29b-41d4-a716-446655440000", "ip": "192.168.1.1", "blob": {"$binary": "aGVsbG8="}}`,
			validate: func(t *testing.T, row Row) {
				data := row.ToMap()
				if _, ok := data["uuid"].(datatypes.UUID); !ok {
					t.Errorf("expected datatypes.UUID, got %T", data["uuid"])
				}
				if _, ok := data["ip"].(net.IP); !ok {
					t.Errorf("expected net.IP, got %T", data["ip"])
				}
				if b, ok := data["blob"].([]byte); !ok || string(b) != "hello" {
					t.Errorf("expected []byte 'hello', got %v", data["blob"])
				}
			},
		},
		{
			name: "duration",
			schema: table.Columns{
				{Name: "dur", Column: table.Column{Type: table.TypeDuration}},
			},
			jsonData: `{"dur": "1y2mo"}`,
			validate: func(t *testing.T, row Row) {
				data := row.ToMap()
				dur, ok := data["dur"].(datatypes.Duration)
				if !ok {
					t.Fatalf("expected datatypes.Duration, got %T", data["dur"])
				}
				want := datatypes.MustParseDuration("1y2mo")
				if !dur.Equals(want) {
					t.Errorf("got %v, want %v", dur, want)
				}
			},
		},
		{
			name: "null and missing handling",
			schema: table.Columns{
				{Name: "name", Column: table.Column{Type: table.TypeText}},
				{Name: "age", Column: table.Column{Type: table.TypeInt}},
				{Name: "udt_partial", Column: table.Column{Type: table.TypeUDT, UDTDefinition: &table.UDTDefinition{
					Fields: table.Columns{
						{Name: "field1", Column: table.Column{Type: table.TypeText}},
						{Name: "field2", Column: table.Column{Type: table.TypeBoolean}},
					},
				}}},
				{Name: "udt_missing", Column: table.Column{Type: table.TypeUDT, UDTDefinition: &table.UDTDefinition{
					Fields: table.Columns{
						{Name: "field1", Column: table.Column{Type: table.TypeText}},
						{Name: "field2", Column: table.Column{Type: table.TypeBoolean}},
					},
				}}},
				{Name: "map1", Column: table.Column{Type: table.TypeMap, KeyType: ptr.To("text"), ValueType: &table.Column{Type: table.TypeUUID}}},
				{Name: "map2", Column: table.Column{Type: table.TypeMap, KeyType: ptr.To("int"), ValueType: &table.Column{Type: table.TypeUUID}}},
				{Name: "map3", Column: table.Column{Type: table.TypeMap, KeyType: ptr.To("varint"), ValueType: &table.Column{Type: table.TypeUUID}}},
				{Name: "list", Column: table.Column{Type: table.TypeList, ValueType: &table.Column{Type: table.TypeDecimal}}},
				{Name: "set", Column: table.Column{Type: table.TypeSet, ValueType: &table.Column{Type: table.TypeAscii}}},
			},
			jsonData: `{"udt_partial":{"field2":true}}`,
			want: map[string]any{
				"name":        nil,
				"age":         nil,
				"udt_partial": map[string]any{"field1": nil, "field2": true},
				"udt_missing": map[string]any{"field1": nil, "field2": nil},
				"map1":        map[string]datatypes.UUID{},
				"map2":        map[int32]datatypes.UUID{},
				"map3":        datatypes.NewSortedMapWithComparator[any, any](datatypes.ComparatorFor(reflect.TypeFor[big.Int]())),
				"list":        []big.Float{},
				"set":         []string{},
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

			if tt.want != nil {
				if diff := testlib.Diff(t, tt.want, row.ToMap()); diff != "" {
					t.Errorf("ToMap() mismatch (-want +got):\n%s", diff)
				}
			}

			if tt.validate != nil {
				tt.validate(t, row)
			}
		})
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
	if !ok || zip != int32(12345) {
		t.Errorf("nestedRow.Get(zip): got %v, ok %v, want 12345", zip, ok)
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

func TestProperty_DeserializeWithTypeHint_Varint(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`-?[0-9]{1,50}`).Draw(t, "digits")

		if s[0] == '0' || s[0] == '-' && s[1] == '0' {
			return // invalid json
		}

		raw := json.RawMessage(s)
		col := table.Column{Type: table.TypeVarint}

		ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
		val, err := deserializeColumn(ctx, raw, col)
		if err != nil {
			t.Fatalf("untyped.deserializeColumn(%s) error = %v", s, err)
		}

		got, ok := val.(big.Int)
		if !ok {
			t.Fatalf("expected big.Int, got %T", val)
		}

		expected := new(big.Int)
		expected.SetString(s, 10)

		if got.Cmp(expected) != 0 {
			t.Errorf("got %s, want %s", got.String(), expected.String())
		}
	})
}

func TestProperty_DeserializeWithTypeHint_Decimal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`-?[0-9]{1,20}(\.[0-9]{1,20})?`).Draw(t, "digits")

		if s[0] == '0' || s[0] == '-' && s[1] == '0' {
			return // invalid json
		}

		raw := json.RawMessage(s)
		col := table.Column{Type: table.TypeDecimal}

		ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
		val, err := deserializeColumn(ctx, raw, col)
		if err != nil {
			t.Fatalf("untyped.deserializeColumn(%s) error = %v", s, err)
		}

		got, ok := val.(big.Float)
		if !ok {
			t.Fatalf("expected big.Float, got %T", val)
		}

		expected := new(big.Float)
		expected.SetString(s)

		if got.Cmp(expected) != 0 {
			t.Errorf("got %s, want %s", got.String(), expected.String())
		}
	})
}

func TestDeserializeWithTypeHint_UnknownType(t *testing.T) {
	raw := json.RawMessage(`"test"`)
	col := table.Column{Type: "unknown_type"}

	ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
	val, err := deserializeColumn(ctx, raw, col)
	if err != nil {
		t.Fatalf("untyped.deserializeColumn() error = %v", err)
	}

	// Should fall back to generic deserialization
	if val == nil {
		t.Error("expected non-nil value for unknown type")
	}
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

	id := row.MustGet("id").(int32)
	if id != 123 {
		t.Errorf("MustGet(id): got %v, want 123", id)
	}

	score := row.MustGet("meta", "score").(float32)
	if score != 0.95 {
		t.Errorf("MustGet(meta.score): got %v, want 0.95", score)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet(missing) did not panic")
		}
	}()

	row.MustGet("missing")
}

func TestNewRow_MustGet(t *testing.T) {
	row := NewRow{"id": 123, "meta": map[string]any{"score": 0.95}}

	id := row.MustGet("id").(int)
	if id != 123 {
		t.Errorf("MustGet(id): got %v, want 123", id)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet(missing) did not panic")
		}
	}()

	row.MustGet("missing")
}
