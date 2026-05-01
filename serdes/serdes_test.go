package serdes

import (
	"testing"
)

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

func (c CustomWithValue) ToAstraValue(_ targetKind) (any, error) {
	return map[string]any{"value": c.Value}, nil
}

func (c *CustomWithValue) FromAstraValue(_ targetKind, v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	if val, ok := m["value"].(string); ok {
		c.Value = val
	}
	return nil
}

func (c *CustomWithPointer) ToAstraRaw(_ targetKind, dst []byte) ([]byte, error) {
	m := map[string]any{"value": c.Value}
	return SerializeInto(m, TargetCollection, dst)
}

func (c *CustomWithPointer) FromAstraRaw(_ targetKind, val []byte) error {
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
	t.Logf("serialized CustomWithValue with value receiver: %s", str)

	src, err := Serialize(&CustomWithValue{Value: "hello"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithValue with pointer receiver: %v", err)
	}
	t.Logf("serialized CustomWithValue with pointer receiver: %s", src)

	str, err = Serialize(&CustomWithPointer{Value: "world"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithPointer with pointer receiver: %v", err)
	}
	t.Logf("serialized CustomWithPointer with pointer receiver: %s", str)

	str, err = Serialize(CustomWithPointer{Value: "world"}, TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithPointer with value receiver: %v", err)
	}
	t.Logf("serialized CustomWithPointer with value receiver: %s", str)

	var dstVal CustomWithValue
	if err := Deserialize([]byte(`{"value":"hello"}`), &dstVal, TargetCollection); err != nil {
		t.Fatalf("failed to deserialize into CustomWithValue: %v", err)
	}
	t.Logf("deserialized into CustomWithValue: %#v", dstVal)

	var dstPtr *CustomWithPointer
	if err := Deserialize([]byte(`{"value":"world"}`), &dstPtr, TargetCollection); err != nil {
		t.Fatalf("failed to deserialize into CustomWithPointer: %v", err)
	}
	t.Logf("deserialized into CustomWithPointer: %#v", dstPtr)
}
