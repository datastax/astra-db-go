package serdes

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
)

var targets = []Target{TargetUnknown, TargetCollection, TargetTable}

var mkUUIDs = []func() datatypes.UUID{datatypes.NewUUID, datatypes.NewUUIDv1, datatypes.NewUUIDv4, datatypes.NewUUIDv6, datatypes.NewUUIDv7}

func TestSerdesUUID(t *testing.T) {
	for _, target := range targets {
		t.Run(target.String(), func(t *testing.T) {
			for _, mkUUID := range mkUUIDs {
				uuid := mkUUID()

				encoded, err := Serialize(uuid, target)
				if err != nil {
					t.Fatalf("failed to serialize UUID: %v", err)
				}

				expected := `"` + uuid.String() + `"`
				if target.kind == collectionKind {
					expected = `{"$uuid":` + expected + `}`
				}

				if string(encoded) != expected {
					t.Fatalf("unexpected serialized form: got %s, expected %s", encoded, expected)
				}

				var decoded datatypes.UUID
				err = Deserialize(encoded, &decoded, target)
				if err != nil {
					t.Fatalf("failed to deserialize UUID: %v", err)
				}

				if uuid != decoded {
					t.Fatalf("mismatch after serdes: original %s, decoded %s", uuid, decoded)
				}
			}
		})
	}
}

func TestSerdesMaps(t *testing.T) {
	uuid1, uuid2 := datatypes.NewUUID(), datatypes.NewUUID()

	tests := []struct {
		name  string
		input any
		ptr   any
		coll  string
		table string
	}{
		{
			name:  "nil",
			input: nil,
			ptr:   new(map[string]struct{}),
			coll:  `null`,
			table: `null`,
		},
		{
			name:  "empty",
			input: map[string]struct{}{},
			ptr:   new(map[string]struct{}),
			coll:  `{}`,
			table: `{}`,
		},
		{
			name:  "map[string]int",
			input: map[string]int{"a": 1, "b": 2},
			ptr:   new(map[string]int),
			coll:  `{"a":1,"b":2}`,
			table: `{"a":1,"b":2}`,
		},
		{
			name:  "map[int]string",
			input: map[int]string{1: "a", 2: "b"},
			ptr:   new(map[int]string),
			table: `[[1,"a"],[2,"b"]]`,
		},
		{
			name:  "map[string]UUID",
			input: map[string]datatypes.UUID{"id1": uuid1, "id2": uuid2},
			ptr:   new(map[string]datatypes.UUID),
			coll:  `{"id1":{"$uuid":"` + uuid1.String() + `"},"id2":{"$uuid":"` + uuid2.String() + `"}}`,
			table: `{"id1":"` + uuid1.String() + `","id2":"` + uuid2.String() + `"}`,
		},
		{
			name:  "map[string]map[string]map[string]UUID",
			input: map[string]map[string]map[string]datatypes.UUID{"outer": {"inner": {"id": uuid1}}},
			ptr:   new(map[string]map[string]map[string]datatypes.UUID),
			coll:  `{"outer":{"inner":{"id":{"$uuid":"` + uuid1.String() + `"}}}}`,
			table: `{"outer":{"inner":{"id":"` + uuid1.String() + `"}}}`,
		},
	}

	compare := func(a []byte, b string) bool {
		var m1 any
		_ = json.Unmarshal(a, &m1)
		var m2 any
		_ = json.Unmarshal([]byte(b), &m2)
		return reflect.DeepEqual(m1, m2)
	}

	for _, target := range targets {
		for _, tt := range tests {
			t.Run(target.String()+"-"+tt.name, func(t *testing.T) {
				testShouldFail := tt.coll == "" && target.kind != tableKind

				encoded, err := Serialize(tt.input, target)

				if testShouldFail == (err == nil) {
					t.Fatalf("improper serialization")
				}

				err = Deserialize(encoded, tt.ptr, target)

				if testShouldFail == (err == nil) {
					t.Fatalf("improper deserialization")
				}

				if !testShouldFail {
					switch target.kind {
					case collectionKind:
						if !compare(encoded, tt.coll) {
							t.Fatalf("unexpected serialized form: got %s, expected %s", encoded, tt.coll)
						}
					case tableKind:
						if !compare(encoded, tt.table) {
							t.Fatalf("unexpected serialized form: got %s, expected %s", encoded, tt.table)
						}
					}

					decoded := reflect.ValueOf(tt.ptr).Elem().Interface()

					if tt.input != nil && !reflect.DeepEqual(tt.input, decoded) {
						t.Fatalf("mismatch after serdes: original %v, decoded %v", tt.input, decoded)
					}
				}
			})
		}
	}
}
