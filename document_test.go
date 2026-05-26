package astradb

import (
	"reflect"
	"testing"

	"github.com/datastax/astra-db-go/serdes"
)

func TestDocument_DeferredDecoding(t *testing.T) {
	jsonData := `{"id": "123", "name": "Alice", "meta": {"score": 0.95}}`

	var doc Document
	err := serdes.Deserialize([]byte(jsonData), &doc, collectionCtx, serdes.TargetCollection)
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
	serdes.Deserialize([]byte(jsonData), &doc, collectionCtx, serdes.TargetCollection)

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
	serdes.Deserialize([]byte(jsonData), &doc, collectionCtx, serdes.TargetCollection)

	_, ok := doc.Get("meta", "missing")
	if ok {
		t.Error("expected ok=false for missing deep path")
	}

	_, ok = doc.Get("missing", "path")
	if ok {
		t.Error("expected ok=false for missing root path")
	}
}
