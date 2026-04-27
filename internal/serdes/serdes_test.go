package serdes

import (
	"testing"
)

// Test 1: Basic Serialization
func TestSerializationBasic(t *testing.T) {
	type TestStruct struct {
		Name    string
		Age     int
		Tags    []string
		Scores  map[string]int
		Data    interface{}
		Numbers [3]int
	}

	input := TestStruct{
		Name:    "Alice",
		Age:     30,
		Tags:    []string{"go", "rust", "python"},
		Scores:  map[string]int{"math": 95, "english": 88},
		Data:    "dynamic value",
		Numbers: [3]int{1, 2, 3},
	}

	data, err := Serialize(input)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	t.Logf("Serialized: %s", string(data))

	// Verify it's valid JSON structure
	if len(data) == 0 {
		t.Fatal("Serialized data is empty")
	}
}

// Test 2: Basic Deserialization
func TestDeserializationBasic(t *testing.T) {
	type TestStruct struct {
		Name    string
		Age     int
		Tags    []string
		Scores  map[string]int
		Data    interface{}
		Numbers [3]int
	}

	jsonData := []byte(`{
		"Name": "Bob",
		"Age": 25,
		"Tags": ["java", "kotlin"],
		"Scores": {"science": 92, "history": 85},
		"Data": 42,
		"Numbers": [10, 20, 30]
	}`)

	var result TestStruct
	err := Deserialize(jsonData, &result)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify basic fields
	if result.Name != "Bob" {
		t.Errorf("Expected Name=Bob, got %s", result.Name)
	}
	if result.Age != 25 {
		t.Errorf("Expected Age=25, got %d", result.Age)
	}
	if len(result.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(result.Tags))
	}
	if len(result.Scores) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(result.Scores))
	}

	t.Logf("Deserialized: %+v", result)
}