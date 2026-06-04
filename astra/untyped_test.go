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
	"encoding/json"
	"math/big"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/table"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"
)

func TestDocument_DeferredDecoding(t *testing.T) {
	jsonData := `{"id": "123", "name": "Alice", "meta": {"score": 0.95}}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, documentCtx, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Verify it's a serverDocument
	if reflect.TypeOf(doc).String() != "*astra.serverDocument" {
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
	expectedMeta := map[string]any{"score": 0.95}
	if diff := cmp.Diff(expectedMeta, m); diff != "" {
		t.Errorf("Decode(meta) mismatch (-want +got):\n%s", diff)
	}

	// Test ToMap()
	fullMap := doc.ToMap()
	expectedFullMap := map[string]any{
		"id":   "123",
		"name": "Alice",
		"meta": map[string]any{"score": 0.95},
	}
	if diff := cmp.Diff(expectedFullMap, fullMap); diff != "" {
		t.Errorf("ToMap() mismatch (-want +got):\n%s", diff)
	}
}

func TestNewDocument_Insertion(t *testing.T) {
	doc := NewDocument{
		"id":   "456",
		"tags": []string{"a", "b"},
	}

	encoded, err := serdes.Serialize(doc, serdes.TargetCollection, serdes.SortMapKeys)
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
				"id":     42,
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
			name: "null and missing handling",
			schema: table.Columns{
				{Name: "name", Column: table.Column{Type: table.TypeText}},
				{Name: "age", Column: table.Column{Type: table.TypeInt}},
			},
			jsonData: `{"name": null}`,
			want:     map[string]any{"name": nil, "age": nil},
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
				"numbers": []any{1, 2, 3},
				"scores":  map[string]any{"alice": 100},
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
				"address": map[string]any{"city": "Springfield", "zip": 12345},
			},
		},
		{
			name: "special types",
			schema: table.Columns{
				{Name: "uuid", Column: table.Column{Type: table.TypeUUID}},
				{Name: "ip", Column: table.Column{Type: table.TypeInet}},
				{Name: "blob", Column: table.Column{Type: table.TypeBlob}},
			},
			jsonData: `{"uuid": "550e8400-e29b-41d4-a716-446655440000", "ip": [192, 168, 1, 1], "blob": {"$binary": "aGVsbG8="}}`,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row Row
			err := serdes.Deserialize([]byte(tt.jsonData), &row, NewRowTargetCtx(tt.schema), serdes.TargetTable)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			if tt.want != nil {
				if diff := cmp.Diff(tt.want, row.ToMap()); diff != "" {
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
	if !ok || zip != 12345 {
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
		raw := json.RawMessage(s)
		col := table.Column{Type: table.TypeVarint}

		ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
		val, err := deserializeColumn(ctx, raw, col)
		if err != nil {
			t.Fatalf("deserializeColumn(%s) error = %v", s, err)
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
		raw := json.RawMessage(s)
		col := table.Column{Type: table.TypeDecimal}

		ctx := serdes.DecodeCtx{Target: serdes.TargetTable}
		val, err := deserializeColumn(ctx, raw, col)
		if err != nil {
			t.Fatalf("deserializeColumn(%s) error = %v", s, err)
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

func TestProperty_Document(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genJSONMap(0).Draw(t, "input")

		encoded, err := serdes.Serialize(input, serdes.TargetCollection)
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}

		var doc Document
		err = serdes.Deserialize(encoded, &doc, documentCtx, serdes.TargetCollection)
		if err != nil {
			t.Fatalf("Deserialize() error = %v", err)
		}

		got := doc.ToMap()

		cmpOpts := []cmp.Option{
			cmpopts.EquateEmpty(),
			cmp.AllowUnexported(big.Int{}, big.Float{}, datatypes.UUID{}, datatypes.ObjectId{}, datatypes.Duration{}, time.Time{}),
			cmp.Comparer(func(x, y big.Int) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y big.Float) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y datatypes.Duration) bool { return x.Equals(y) }),
			cmp.Comparer(func(x, y time.Time) bool { return x.Equal(y) }),
		}

		var expected map[string]any
		err = serdes.Deserialize(encoded, &expected, nil, serdes.TargetCollection)
		if err != nil {
			t.Fatalf("serdes.Deserialize() error = %v", err)
		}

		if diff := cmp.Diff(expected, got, cmpOpts...); diff != "" {
			t.Errorf("Document.ToMap() mismatch (-want +got):\n%s", diff)
		}

		// Test Get/MustGet/Decode for some paths
		testPaths(t, doc, expected, nil, cmpOpts)
	})
}

func testPaths(t *rapid.T, doc Document, expected map[string]any, path []string, cmpOpts []cmp.Option) {
	for k, v := range expected {
		fullPath := append(path, k)

		// Get
		gotVal, ok := doc.Get(fullPath...)
		if !ok {
			t.Errorf("Get(%v) failed", fullPath)
			continue
		}
		if diff := cmp.Diff(v, gotVal, cmpOpts...); diff != "" {
			t.Errorf("Get(%v) mismatch (-want +got):\n%s", fullPath, diff)
		}

		// MustGet
		mustGotVal := doc.MustGet(fullPath...)
		if diff := cmp.Diff(v, mustGotVal, cmpOpts...); diff != "" {
			t.Errorf("MustGet(%v) mismatch (-want +got):\n%s", fullPath, diff)
		}

		// Decode
		if v != nil {
			var decoded any
			if err := doc.Decode(&decoded, fullPath...); err != nil {
				t.Errorf("Decode(%v) error = %v", fullPath, err)
			} else {
				if diff := cmp.Diff(v, decoded, cmpOpts...); diff != "" {
					t.Errorf("Decode(%v) mismatch (-want +got):\n%s", fullPath, diff)
				}
			}
		}

		// Recurse into maps
		if nextMap, ok := v.(map[string]any); ok {
			testPaths(t, doc, nextMap, fullPath, cmpOpts)
		}
	}
}

func TestProperty_Row(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		schema := genTableColumns().Draw(t, "schema")
		input := genTableData(schema).Draw(t, "input")

		encoded, err := serdes.Serialize(input, serdes.TargetTable, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}

		var row Row
		err = serdes.Deserialize(encoded, &row, NewRowTargetCtx(schema), serdes.TargetTable, serdes.SparseRows)
		if err != nil {
			t.Fatalf("Deserialize() error = %v", err)
		}

		got := row.ToMap()

		// When comparing, we need to handle special types that Row decodes into
		// (like datatypes.UUID, net.IP, etc.)
		cmpOpts := []cmp.Option{
			cmpopts.EquateEmpty(),
			cmp.AllowUnexported(big.Int{}, big.Float{}, datatypes.UUID{}, datatypes.ObjectId{}, datatypes.Duration{}, time.Time{}),
			cmp.Comparer(func(x, y big.Int) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y big.Float) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y datatypes.Duration) bool { return x.Equals(y) }),
			cmp.Comparer(func(x, y time.Time) bool { return x.Equal(y) }),
		}

		if diff := cmp.Diff(input, got, cmpOpts...); diff != "" {
			t.Errorf("Row.ToMap() mismatch (-want +got):\n%s", diff)
		}

		// Test Get/MustGet/Decode for columns and nested UDT/Map fields
		testRowPaths(t, row, input, nil, cmpOpts)
	})
}

func testRowPaths(t *rapid.T, row Row, expected map[string]any, path []string, cmpOpts []cmp.Option) {
	for k, v := range expected {
		fullPath := append(path, k)

		// Get
		gotVal, ok := row.Get(fullPath...)
		if !ok {
			t.Errorf("Get(%v) failed", fullPath)
			continue
		}
		if diff := cmp.Diff(v, gotVal, cmpOpts...); diff != "" {
			t.Errorf("Get(%v) mismatch (-want +got):\n%s", fullPath, diff)
		}

		// MustGet
		mustGotVal := row.MustGet(fullPath...)
		if diff := cmp.Diff(v, mustGotVal, cmpOpts...); diff != "" {
			t.Errorf("MustGet(%v) mismatch (-want +got):\n%s", fullPath, diff)
		}

		// Decode
		if v != nil {
			var decoded any
			if err := row.Decode(&decoded, fullPath...); err != nil {
				t.Errorf("Decode(%v) error = %v", fullPath, err)
			} else {
				if diff := cmp.Diff(v, decoded, cmpOpts...); diff != "" {
					t.Errorf("Decode(%v) mismatch (-want +got):\n%s", fullPath, diff)
				}
			}
		}

		// Recurse into nested structures
		if nextMap, ok := v.(map[string]any); ok {
			testRowPaths(t, row, nextMap, fullPath, cmpOpts)
		}
	}
}

func TestProperty_NewDocument(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genJSONMap(0).Draw(t, "input")
		doc := NewDocument(input)

		cmpOpts := []cmp.Option{
			cmpopts.EquateEmpty(),
			cmp.AllowUnexported(big.Int{}, big.Float{}, datatypes.UUID{}, datatypes.ObjectId{}, datatypes.Duration{}, time.Time{}),
			cmp.Comparer(func(x, y big.Int) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y big.Float) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y datatypes.Duration) bool { return x.Equals(y) }),
			cmp.Comparer(func(x, y time.Time) bool { return x.Equal(y) }),
		}

		// Test ToMap
		if diff := cmp.Diff(map[string]any(input), doc.ToMap(), cmpOpts...); diff != "" {
			t.Errorf("ToMap() mismatch (-want +got):\n%s", diff)
		}

		// Test serialization consistency
		encodedDoc, err := serdes.Serialize(doc, serdes.TargetCollection, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize(doc) error = %v", err)
		}

		encodedMap, err := serdes.Serialize(input, serdes.TargetCollection, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize(input) error = %v", err)
		}

		if string(encodedDoc) != string(encodedMap) {
			t.Errorf("serialization mismatch\n  doc: %s\n  map: %s", string(encodedDoc), string(encodedMap))
		}
	})
}

func TestProperty_NewRow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		schema := genTableColumns().Draw(t, "schema")
		input := genTableData(schema).Draw(t, "input")
		row := NewRow(input)

		// When comparing, we need to handle special types
		cmpOpts := []cmp.Option{
			cmpopts.EquateEmpty(),
			cmp.AllowUnexported(big.Int{}, big.Float{}, datatypes.UUID{}, datatypes.ObjectId{}, datatypes.Duration{}, time.Time{}),
			cmp.Comparer(func(x, y big.Int) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y big.Float) bool { return x.Cmp(&y) == 0 }),
			cmp.Comparer(func(x, y datatypes.Duration) bool { return x.Equals(y) }),
			cmp.Comparer(func(x, y time.Time) bool { return x.Equal(y) }),
		}

		// Test ToMap
		if diff := cmp.Diff(map[string]any(input), row.ToMap(), cmpOpts...); diff != "" {
			t.Errorf("ToMap() mismatch (-want +got):\n%s", diff)
		}

		// Test serialization consistency
		encodedRow, err := serdes.Serialize(row, serdes.TargetTable, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize(row) error = %v", err)
		}

		encodedMap, err := serdes.Serialize(input, serdes.TargetTable, serdes.SortMapKeys)
		if err != nil {
			t.Fatalf("serdes.Serialize(input) error = %v", err)
		}

		if string(encodedRow) != string(encodedMap) {
			t.Errorf("serialization mismatch\n  row: %s\n  map: %s", string(encodedRow), string(encodedMap))
		}
	})
}

func genJSONValue(depth int) *rapid.Generator[any] {
	return rapid.Custom(func(t *rapid.T) any {
		gens := [](*rapid.Generator[any]){
			rapid.Map(rapid.String(), func(v string) any { return any(v) }),
			rapid.Map(rapid.Int(), func(v int) any { return float64(v) }),
			rapid.Map(rapid.Float64(), func(v float64) any { return any(v) }),
			rapid.Map(rapid.Bool(), func(v bool) any { return any(v) }),
			rapid.Just[any](nil),
			rapid.Map(rapid.StringMatching(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), func(v string) any {
				return any(datatypes.MustParseUUID(v))
			}),
			rapid.Map(rapid.StringMatching(`^[0-9a-f]{24}$`), func(v string) any {
				return any(datatypes.MustParseObjectId(v))
			}),
			rapid.Map(rapid.Int64Range(0, 4102444800000), func(v int64) any {
				return any(time.UnixMilli(v))
			}),
		}

		if depth < 3 {
			gens = append(gens,
				rapid.Map(rapid.SliceOf(rapid.Deferred(func() *rapid.Generator[any] { return genJSONValue(depth + 1) })), func(v []any) any { return any(v) }),
				rapid.Map(rapid.MapOf(rapid.String(), rapid.Deferred(func() *rapid.Generator[any] { return genJSONValue(depth + 1) })), func(v map[string]any) any { return any(v) }),
			)
		}

		return rapid.OneOf(gens...).Draw(t, "value")
	})
}

func genJSONMap(depth int) *rapid.Generator[map[string]any] {
	return rapid.MapOf(rapid.String(), genJSONValue(depth))
}

func genTableColumns() *rapid.Generator[table.Columns] {
	return rapid.Custom(func(t *rapid.T) table.Columns {
		count := rapid.IntRange(1, 5).Draw(t, "column_count")
		cols := make(table.Columns, count)
		names := make(map[string]bool)
		for i := 0; i < count; i++ {
			var name string
			for {
				name = rapid.StringMatching(`[a-z][a-z0-9_]*`).Draw(t, "name")
				if !names[name] {
					names[name] = true
					break
				}
			}
			cols[i] = table.NamedColumn{
				Name:   name,
				Column: genTableColumn(0).Draw(t, "column"),
			}
		}
		return cols
	})
}

func genTableColumn(depth int) *rapid.Generator[table.Column] {
	return rapid.Custom(func(t *rapid.T) table.Column {
		types := []string{
			table.TypeText, table.TypeInt, table.TypeBigInt, table.TypeSmallInt,
			table.TypeTinyInt, table.TypeFloat, table.TypeDouble, table.TypeBoolean,
			table.TypeUUID, table.TypeInet, table.TypeBlob, table.TypeVarint,
			table.TypeDecimal, table.TypeDate, table.TypeTime, table.TypeTimestamp,
		}
		if depth < 2 {
			types = append(types, table.TypeList, table.TypeSet, table.TypeMap, table.TypeUDT)
		}

		typ := rapid.SampledFrom(types).Draw(t, "type")
		col := table.Column{Type: typ}

		switch typ {
		case table.TypeList, table.TypeSet:
			vt := genTableColumn(depth+1).Draw(t, "valueType")
			col.ValueType = &vt
		case table.TypeMap:
			kt := table.TypeText
			col.KeyType = &kt
			vt := genTableColumn(depth+1).Draw(t, "valueType")
			col.ValueType = &vt
		case table.TypeUDT:
			col.UDTDefinition = &table.UDTDefinition{
				Fields: genTableColumnsAtDepth(depth+1).Draw(t, "fields"),
			}
		}
		return col
	})
}

func genTableColumnsAtDepth(depth int) *rapid.Generator[table.Columns] {
	return rapid.Custom(func(t *rapid.T) table.Columns {
		count := rapid.IntRange(1, 3).Draw(t, "column_count")
		cols := make(table.Columns, count)
		names := make(map[string]bool)
		for i := 0; i < count; i++ {
			var name string
			for {
				name = rapid.StringMatching(`[a-z][a-z0-9_]*`).Draw(t, "name")
				if !names[name] {
					names[name] = true
					break
				}
			}
			cols[i] = table.NamedColumn{
				Name:   name,
				Column: genTableColumn(depth).Draw(t, "column"),
			}
		}
		return cols
	})
}

func genTableData(cols table.Columns) *rapid.Generator[map[string]any] {
	return rapid.Custom(func(t *rapid.T) map[string]any {
		data := make(map[string]any)
		for _, nc := range cols {
			if rapid.Bool().Draw(t, "include") {
				data[nc.Name] = genValueForColumn(nc.Column).Draw(t, "value")
			}
		}
		return data
	})
}

func genValueForColumn(col table.Column) *rapid.Generator[any] {
	return rapid.Custom(func(t *rapid.T) any {
		if rapid.Float64Range(0, 1).Draw(t, "is_null") < 0.1 {
			return nil
		}

		switch col.Type {
		case table.TypeText, table.TypeAscii:
			return rapid.String().Draw(t, "val")
		case table.TypeInt:
			return rapid.Int().Draw(t, "val")
		case table.TypeBigInt:
			return rapid.Int64().Draw(t, "val")
		case table.TypeSmallInt:
			return rapid.Int16().Draw(t, "val")
		case table.TypeTinyInt:
			return rapid.Int8().Draw(t, "val")
		case table.TypeVarint:
			return rapid.Custom(func(t *rapid.T) any {
				var bi big.Int
				bi.SetBytes(rapid.SliceOfN(rapid.Byte(), 1, 8).Draw(t, "bytes"))
				if rapid.Bool().Draw(t, "neg") {
					bi.Neg(&bi)
				}
				return bi
			}).Draw(t, "val")
		case table.TypeFloat:
			return rapid.Float32().Draw(t, "val")
		case table.TypeDouble:
			return rapid.Float64().Draw(t, "val")
		case table.TypeBoolean:
			return rapid.Bool().Draw(t, "val")
		case table.TypeUUID, table.TypeTimeUUID:
			return datatypes.MustParseUUID(rapid.StringMatching(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).Draw(t, "uuid"))
		case table.TypeInet:
			return net.IP(rapid.SliceOfN(rapid.Byte(), 4, 16).Draw(t, "ip_bytes"))
		case table.TypeBlob:
			return rapid.SliceOf(rapid.Byte()).Draw(t, "blob")
		case table.TypeDecimal:
			return rapid.Custom(func(t *rapid.T) any {
				s := rapid.StringMatching(`-?[0-9]{1,10}\.[0-9]{1,10}`).Draw(t, "s")
				var bf big.Float
				bf.SetString(s)
				return bf
			}).Draw(t, "val")
		case table.TypeDate:
			t := time.UnixMilli(rapid.Int64Range(0, 4102444800000).Draw(t, "val"))
			return datatypes.DateOnlyFromTime(t)
		case table.TypeTime:
			t := time.UnixMilli(rapid.Int64Range(0, 4102444800000).Draw(t, "val"))
			return datatypes.TimeOnlyFromTime(t)
		case table.TypeTimestamp:
			return time.UnixMilli(rapid.Int64Range(0, 4102444800000).Draw(t, "val")) // Up to 2100-01-01
		case table.TypeVector:
			dim := 0
			if col.Dimension != nil {
				dim = *col.Dimension
			}
			if dim == 0 {
				dim = 3
			}
			return datatypes.NewVector(rapid.SliceOfN(rapid.Float32(), dim, dim).Draw(t, "vector"))
		case table.TypeDuration:
			return datatypes.MustParseDuration(rapid.StringMatching(`[0-9]+[smhdw]`).Draw(t, "duration"))
		case table.TypeList, table.TypeSet:
			slice := rapid.SliceOf(genValueForColumn(*col.ValueType)).Draw(t, "slice")
			if slice == nil {
				return []any{}
			}
			return slice
		case table.TypeMap:
			m := rapid.MapOf(rapid.String(), genValueForColumn(*col.ValueType)).Draw(t, "map")
			if m == nil {
				return map[string]any{}
			}
			res := make(map[string]any)
			for k, v := range m {
				res[k] = v
			}
			return res
		case table.TypeUDT:
			return genTableData(col.UDTDefinition.Fields).Draw(t, "udt")
		default:
			return nil
		}
	})
}
