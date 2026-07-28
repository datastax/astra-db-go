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
	"reflect"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
	"pgregory.net/rapid"
)

func TestDocument_DeferredDecoding(t *testing.T) {
	jsonData := `{"id": "123", "name": "Alice", "meta": {"score": 0.95}}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, GlobalDocumentCtx, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reflect.TypeOf(doc).String() != "*untyped.serverDocument" {
		t.Errorf("expected *untyped.serverDocument, got %T", doc)
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
	if diff := testlib.Diff(t, expectedMeta, m); diff != "" {
		t.Errorf("Decode(meta) mismatch (-want +got):\n%s", diff)
	}

	// Test ToMap()
	fullMap := doc.ToMap()
	expectedFullMap := map[string]any{
		"id":   "123",
		"name": "Alice",
		"meta": map[string]any{"score": 0.95},
	}
	if diff := testlib.Diff(t, expectedFullMap, fullMap); diff != "" {
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
	_ = serdes.Deserialize([]byte(jsonData), &doc, GlobalDocumentCtx, serdes.TargetCollection)

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
	_ = serdes.Deserialize([]byte(jsonData), &doc, GlobalDocumentCtx, serdes.TargetCollection)

	_, ok := doc.Get("meta", "missing")
	if ok {
		t.Error("expected ok=false for missing deep path")
	}

	_, ok = doc.Get("missing", "path")
	if ok {
		t.Error("expected ok=false for missing root path")
	}
}

func TestDocument_MustGet(t *testing.T) {
	jsonData := `{"id": "123", "meta": {"score": 0.95}}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, GlobalDocumentCtx, serdes.TargetCollection)
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

func TestDocument_VectorFieldHint(t *testing.T) {
	jsonData := `{"id": "123", "$vector": [0.1, 0.2, 0.3]}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, GlobalDocumentCtx, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	vector := doc.MustGet("$vector").(datatypes.Vector)
	expectedVector := datatypes.NewVector([]float32{0.1, 0.2, 0.3})
	if diff := testlib.Diff(t, expectedVector, vector); diff != "" {
		t.Errorf("MustGet($vector) mismatch (-want +got):\n%s", diff)
	}
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

func TestProperty_Document(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genJSONMap(0).Draw(t, "input")

		encoded, err := serdes.Serialize(input, serdes.TargetCollection)
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}

		var doc Document
		err = serdes.Deserialize(encoded, &doc, GlobalDocumentCtx, serdes.TargetCollection)
		if err != nil {
			t.Fatalf("Deserialize() error = %v", err)
		}

		got := doc.ToMap()

		var expected map[string]any
		err = serdes.Deserialize(encoded, &expected, nil, serdes.TargetCollection)
		if err != nil {
			t.Fatalf("serdes.Deserialize() error = %v", err)
		}

		if diff := testlib.Diff(t, expected, got); diff != "" {
			t.Errorf("Document.ToMap() mismatch (-want +got):\n%s", diff)
		}

		// Test Get/MustGet/Decode for some paths
		testPaths(t, doc, expected, nil)
	})
}

func testPaths(t *rapid.T, doc Document, expected map[string]any, path []string) {
	for k, v := range expected {
		fullPath := append(path, k)

		// Get
		gotVal, ok := doc.Get(fullPath...)
		if !ok {
			t.Errorf("Get(%v) failed", fullPath)
			continue
		}
		if diff := testlib.Diff(t, v, gotVal); diff != "" {
			t.Errorf("Get(%v) mismatch (-want +got):\n%s", fullPath, diff)
		}

		// MustGet
		mustGotVal := doc.MustGet(fullPath...)
		if diff := testlib.Diff(t, v, mustGotVal); diff != "" {
			t.Errorf("MustGet(%v) mismatch (-want +got):\n%s", fullPath, diff)
		}

		// Decode
		if v != nil {
			var decoded any
			if err := doc.Decode(&decoded, fullPath...); err != nil {
				t.Errorf("Decode(%v) error = %v", fullPath, err)
			} else {
				if diff := testlib.Diff(t, v, decoded); diff != "" {
					t.Errorf("Decode(%v) mismatch (-want +got):\n%s", fullPath, diff)
				}
			}
		}

		// Recurse into maps
		if nextMap, ok := v.(map[string]any); ok {
			testPaths(t, doc, nextMap, fullPath)
		}
	}
}

func TestProperty_NewDocument(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genJSONMap(0).Draw(t, "input")
		doc := NewDocument(input)

		// Test ToMap
		if diff := testlib.Diff(t, input, doc.ToMap()); diff != "" {
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
