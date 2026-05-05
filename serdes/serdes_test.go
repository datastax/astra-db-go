package serdes

import (
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
)

func TestSerdesOrderedMap(t *testing.T) {
	Serialize(datatypes.NewOrderedMap[int, any](), TargetCollection)
}

func TestSerdesAny_Collection(t *testing.T) {
	str := `{"$vector":[0.1, 0.2, 0.3],"nested":{"$uuid":"123e4567-e89b-12d3-a456-426614174000"}}}`
	var dst any

	if err := Deserialize([]byte(str), &dst, TargetCollection); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Prints:
	// {
	//   $vector: DataAPIVector{...},
	//   nested: datatypes.UUID{...},
	// }
	t.Logf("deserialized: %#v", dst)
}

type CustomWithValue struct {
	Value string
}

type CustomWithPointer struct {
	Value string
}

func (c CustomWithValue) MarshalAstra(_ Target) (any, error) {
	return map[string]any{"value": c.Value}, nil
}

func (c *CustomWithValue) UnmarshalAstra(_ Target, v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	if val, ok := m["value"].(string); ok {
		c.Value = val
	}
	return nil
}

func (c *CustomWithPointer) MarshalAstraRaw(_ Target, dst []byte) ([]byte, error) {
	m := map[string]any{"value": c.Value}
	return SerializeInto(m, TargetCollection, dst)
}

func (c *CustomWithPointer) UnmarshalAstraRaw(_ Target, val []byte) error {
	var m map[string]any
	if err := Deserialize(val, &m, TargetCollection); err != nil {
		return err
	}
	if val, ok := m["value"].(string); ok {
		c.Value = val
	}
	return nil
}

func TestSerdesCustom(t *testing.T) {
	str, err := Serialize(CustomWithValue{Value: "hello"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithValue with value receiver: %v", err)
	}
	if string(str) != `{"value":"hello"}` {
		t.Errorf("CustomWithValue value receiver: expected %s, got %s", `{"value":"hello"}`, str)
	}

	src, err := Serialize(&CustomWithValue{Value: "hello"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithValue with pointer receiver: %v", err)
	}
	if string(src) != `{"value":"hello"}` {
		t.Errorf("CustomWithValue pointer receiver: expected %s, got %s", `{"value":"hello"}`, src)
	}

	str, err = Serialize(&CustomWithPointer{Value: "world"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithPointer with pointer receiver: %v", err)
	}
	if string(str) != `{"value":"world"}` {
		t.Errorf("CustomWithPointer pointer receiver: expected %s, got %s", `{"value":"world"}`, str)
	}

	str, err = Serialize(CustomWithPointer{Value: "world"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithPointer with value receiver: %v", err)
	}
	if string(str) != `{"Value":"world"}` {
		t.Errorf("CustomWithPointer value receiver: expected %s, got %s", `{"Value":"world"}`, str)
	}

	var dstVal CustomWithValue
	if err := Deserialize([]byte(`{"value":"hello"}`), &dstVal, TargetCollection); err != nil {
		t.Fatalf("failed to deserialize into CustomWithValue: %v", err)
	}
	if dstVal.Value != "hello" {
		t.Errorf("CustomWithValue deserialization: expected Value='hello', got Value='%s'", dstVal.Value)
	}

	var dstPtr *CustomWithPointer
	if err := Deserialize([]byte(`{"value":"world"}`), &dstPtr, TargetCollection); err != nil {
		t.Fatalf("failed to deserialize into CustomWithPointer: %v", err)
	}
	if dstPtr == nil || dstPtr.Value != "world" {
		t.Errorf("CustomWithPointer deserialization: expected Value='world', got %#v", dstPtr)
	}
}
