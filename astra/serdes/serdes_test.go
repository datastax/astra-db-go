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

package serdes_test

import (
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func TestSerdesLinkedMap(t *testing.T) {
	var om datatypes.LinkedMap[int, any]
	b, err := serdes.Serialize(om, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize nil ordered map")
	t.Logf("serialized nil ordered map: %s", b)

	b, err = serdes.Serialize(datatypes.NewLinkedMap[int, any](), serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize empty ordered map")
	t.Logf("serialized empty ordered map: %s", b)

	om = datatypes.NewLinkedMap[int, any]()
	om.Set(1, "one")
	om.Set(3, datatypes.NewUUID())
	om.Set(2, 3)

	b, err = serdes.Serialize(om, serdes.TargetTable)
	testlib.FailIfErr(t, err, "failed to serialize ordered map")
	t.Logf("serialized ordered map: %s", b)

	var dst datatypes.LinkedMap[int, any]
	if err := serdes.Deserialize(b, &dst, nil, serdes.TargetTable); err != nil {
		t.Fatalf("failed to deserialize ordered map: %v", err)
	}
	t.Logf("deserialized ordered map: %#v", dst.String())
}

func BenchmarkSerdesLinkedMap(t *testing.B) {
	var b []byte
	var dst datatypes.LinkedMap[int, string]
	_ = b

	var om = datatypes.NewLinkedMap[int, string]()
	om.Set(1, "one")
	om.Set(3, "two")
	om.Set(2, "three")

	t.Run("serialize", func(t *testing.B) {
		t.ReportAllocs()
		for i := 0; i < t.N; i++ {
			b, _ = serdes.Serialize(om, serdes.TargetTable)
		}
	})

	src, _ := serdes.Serialize(om, serdes.TargetTable)

	t.Run("deserialize", func(t *testing.B) {
		t.ReportAllocs()
		for i := 0; i < t.N; i++ {
			if err := serdes.Deserialize(src, &dst, nil, serdes.TargetTable); err != nil {
				t.Fatalf("failed to deserialize ordered map: %v", err)
			}
		}
	})
}

func TestSerdesSet(t *testing.T) {
	s := datatypes.NewSet[string]()
	s.Add("one")
	s.Add("two")
	s.Add("three")

	b, err := serdes.Serialize(s, serdes.TargetCollection)
	testlib.FailIfErr(t, err, "failed to serialize set")
	t.Logf("serialized set: %s", b)

	var dst datatypes.Set[string]
	if err := serdes.Deserialize(b, &dst, nil, serdes.TargetCollection); err != nil {
		t.Fatalf("failed to deserialize set: %v", err)
	}
	t.Logf("deserialized set: %#v", dst.String())
}

func BenchmarkSerdesSet(t *testing.B) {
	var b []byte
	var dst datatypes.Set[string]
	_ = b

	s := datatypes.NewSet[string]()
	s.Add("one")
	s.Add("two")
	s.Add("three")

	t.Run("serialize", func(t *testing.B) {
		t.ReportAllocs()
		for i := 0; i < t.N; i++ {
			b, _ = serdes.Serialize(s, serdes.TargetCollection)
		}
	})

	src, _ := serdes.Serialize(s, serdes.TargetCollection)

	t.Run("deserialize", func(t *testing.B) {
		t.ReportAllocs()
		for i := 0; i < t.N; i++ {
			if err := serdes.Deserialize(src, &dst, nil, serdes.TargetCollection); err != nil {
				t.Fatalf("failed to deserialize set: %v", err)
			}
		}
	})
}

func TestSerdesAny_Collection(t *testing.T) {
	str := `{"$vector":[0.1, 0.2, 0.3],"nested":{"$uuid":"123e4567-e89b-12d3-a456-426614174000"}}`
	var dst any

	if err := serdes.Deserialize([]byte(str), &dst, nil, serdes.TargetCollection); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Prints:
	// {
	//   $vector: Vector{...},
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

func (c CustomWithValue) MarshalAstra(_ serdes.EncodeCtx) (any, error) {
	return map[string]any{"value": c.Value}, nil
}

func (c *CustomWithValue) UnmarshalAstra(_ serdes.DecodeCtx, v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	if val, ok := m["value"].(string); ok {
		c.Value = val
	}
	return nil
}

func (c *CustomWithPointer) MarshalAstraRaw(_ serdes.EncodeCtx, dst []byte) ([]byte, error) {
	m := map[string]any{"value": c.Value}
	return serdes.SerializeInto(m, serdes.TargetCollection, dst)
}

func (c *CustomWithPointer) UnmarshalAstraRaw(_ serdes.DecodeCtx, val []byte) error {
	var m map[string]any
	if err := serdes.Deserialize(val, &m, nil, serdes.TargetCollection); err != nil {
		return err
	}
	if val, ok := m["value"].(string); ok {
		c.Value = val
	}
	return nil
}

func TestSerdesCustom(t *testing.T) {
	str, err := serdes.Serialize(CustomWithValue{Value: "hello"}, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithValue with value receiver: %v", err)
	}
	if string(str) != `{"value":"hello"}` {
		t.Errorf("CustomWithValue value receiver: expected %s, got %s", `{"value":"hello"}`, str)
	}

	src, err := serdes.Serialize(&CustomWithValue{Value: "hello"}, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithValue with pointer receiver: %v", err)
	}
	if string(src) != `{"value":"hello"}` {
		t.Errorf("CustomWithValue pointer receiver: expected %s, got %s", `{"value":"hello"}`, src)
	}

	str, err = serdes.Serialize(&CustomWithPointer{Value: "world"}, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithPointer with pointer receiver: %v", err)
	}
	if string(str) != `{"value":"world"}` {
		t.Errorf("CustomWithPointer pointer receiver: expected %s, got %s", `{"value":"world"}`, str)
	}

	str, err = serdes.Serialize(CustomWithPointer{Value: "world"}, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("failed to serialize CustomWithPointer with value receiver: %v", err)
	}
	if string(str) != `{"Value":"world"}` {
		t.Errorf("CustomWithPointer value receiver: expected %s, got %s", `{"Value":"world"}`, str)
	}

	var dstVal CustomWithValue
	if err := serdes.Deserialize([]byte(`{"value":"hello"}`), &dstVal, nil, serdes.TargetCollection); err != nil {
		t.Fatalf("failed to deserialize into CustomWithValue: %v", err)
	}
	if dstVal.Value != "hello" {
		t.Errorf("CustomWithValue deserialization: expected Value='hello', got Value='%s'", dstVal.Value)
	}

	var dstPtr *CustomWithPointer
	if err := serdes.Deserialize([]byte(`{"value":"world"}`), &dstPtr, nil, serdes.TargetCollection); err != nil {
		t.Fatalf("failed to deserialize into CustomWithPointer: %v", err)
	}
	if dstPtr == nil || dstPtr.Value != "world" {
		t.Errorf("CustomWithPointer deserialization: expected Value='world', got %#v", dstPtr)
	}
}
