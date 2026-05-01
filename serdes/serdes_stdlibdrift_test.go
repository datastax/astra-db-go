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
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/serdes"
)

// Hey Kavin, check this out. Did some poking around. I don't know if we NEED to address
// all of these, but we should add documentation where we diverge from stdlib potentially.

func TestTimeMarshaler_StdlibDivergence(t *testing.T) {
	type wrapper struct {
		T time.Time `json:"t"`
	}
	w := wrapper{T: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)}

	// What stdlib produces — a quoted RFC3339 string via time.Time.MarshalJSON
	stdlibOut, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("stdlib marshal failed: %v", err)
	}

	// What the custom serdes produces
	serdesOut, err := serdes.Serialize(w, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("serdes marshal failed: %v", err)
	}

	if string(stdlibOut) != string(serdesOut) {
		t.Errorf("divergence from stdlib:\nstdlib: %s\nserdes: %s", stdlibOut, serdesOut)
	}
}

// Same problem on the decode side — json.Unmarshaler is ignored.
// Decoding a valid RFC3339 string into a time.Time should work.
func TestTimeUnmarshaler_StdlibDivergence(t *testing.T) {
	type wrapper struct {
		T time.Time `json:"t"`
	}
	input := []byte(`{"t":"2025-01-15T10:30:00Z"}`)

	// stdlib handles this fine via time.Time.UnmarshalJSON
	var stdlibResult wrapper
	if err := json.Unmarshal(input, &stdlibResult); err != nil {
		t.Fatalf("stdlib unmarshal failed: %v", err)
	}

	var serdesResult wrapper
	err := serdes.Deserialize(input, &serdesResult, serdes.TargetCollection)
	if err != nil {
		t.Logf("serdes error (expected): %v", err)
	}

	if !stdlibResult.T.Equal(serdesResult.T) {
		t.Errorf("divergence from stdlib:\nstdlib: %v\nserdes: %v (err: %v)", stdlibResult, serdesResult, err)
	}
}

// This problem also affects user-defined types with custom MarshalJSON,
// not just stdlib types. Any type implementing json.Marshaler will hit this.
// Not sure if this is the end of the world as long as we document this.
type Dollars int64

func (d Dollars) MarshalJSON() ([]byte, error) {
	return []byte(`"$` + strconv.FormatInt(int64(d), 10) + `.00"`), nil
}

func (d *Dollars) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"$`)
	s = strings.TrimSuffix(s, ".00")
	n, err := strconv.ParseInt(s, 10, 64)
	*d = Dollars(n)
	return err
}

func TestCustomMarshaler_StdlibDivergence(t *testing.T) {
	val := Dollars(1999)

	stdlibOut, _ := json.Marshal(val)                              // "$1999.00"
	serdesOut, _ := serdes.Serialize(val, serdes.TargetCollection) // 1999 (raw int)

	if string(stdlibOut) != string(serdesOut) {
		t.Errorf("divergence:\nstdlib=%s\nserdes=%s", stdlibOut, serdesOut)
	}
}

// Escaping html is default behavior in stdlib:
// https://pkg.go.dev/encoding/json#HTMLEscape
func TestHTMLEscape_StdlibDivergence(t *testing.T) {
	// I don't think we need to reproduce this. I don't think it probably matters
	// much with how people will use Astra. But I think we should possibly document it.
	in := `<script>alert("you've been pwnd")&</script>`

	stdlibOut, _ := json.Marshal(in)
	serdesOut, _ := serdes.Serialize(in, serdes.TargetCollection)

	if string(stdlibOut) != string(serdesOut) {
		t.Errorf("divergence:\nstdlib=%s\nserdes=%s", stdlibOut, serdesOut)
	}
}

func TestStringTag_StdlibDivergence(t *testing.T) {
	type Wrapper struct {
		ID uint64 `json:"id,string"`
	}
	in := Wrapper{ID: 18446744073709551615}

	stdlibOut, _ := json.Marshal(in)
	serdesOut, _ := serdes.Serialize(in, serdes.TargetCollection)

	if string(stdlibOut) != string(serdesOut) {
		t.Errorf("divergence:\nstdlib=%s\nserdes=%s", stdlibOut, serdesOut)
	}
}

func TestOmitEmpty_StdlibDivergence(t *testing.T) {
	type Filter struct {
		Limit     int               `json:"limit,omitempty"`
		PageState string            `json:"pageState,omitempty"`
		Options   map[string]string `json:"options,omitempty"`
	}
	in := Filter{} // all zero — stdlib should produce {}

	stdlibOut, _ := json.Marshal(in)
	serdesOut, _ := serdes.Serialize(in, serdes.TargetCollection)

	if string(stdlibOut) != string(serdesOut) {
		t.Errorf("divergence:\nstdlib=%s\nserdes=%s", stdlibOut, serdesOut)
	}
}

// myAstraCodec implements AstraCodec with a pointer receiver.
type myAstraCodec struct{ n int }

func (m *myAstraCodec) ToAstraValue() any          { return m.n }
func (m *myAstraCodec) FromAstraValue(v any) error { m.n = v.(int); return nil }

func TestNilAstraCodecPointer_PanicAndDeadCode(t *testing.T) {
	var nilPtr *myAstraCodec // typed nil

	stdlibOut, err := json.Marshal(nilPtr)
	if err != nil {
		t.Fatalf("stdlib marshal failed: %v", err)
	}

	var serdesOut []byte
	var serdesErr error
	var panicVal any
	func() {
		defer func() { panicVal = recover() }()
		serdesOut, serdesErr = serdes.Serialize(nilPtr, serdes.TargetCollection)
	}()

	switch {
	case panicVal != nil:
		// encodeInlined replaces p with `&p`, the address of a local stack variable,
		// which is never nil. So encodeCustom skips the `if p == nil` branch.
		t.Errorf("serdes panicked on a nil AstraCodec pointer instead of emitting null: %v", panicVal)
	case serdesErr != nil:
		t.Errorf("serdes errored on a nil AstraCodec pointer: %v", serdesErr)
	case string(stdlibOut) != string(serdesOut):
		t.Errorf("divergence:\nstdlib=%q\nserdes=%q (raw bytes: % x)", stdlibOut, serdesOut, serdesOut)
	}

	// `0xc0` is invalid JSON, even though the branch is currently unreachable.
	if json.Valid([]byte{0xc0}) {
		t.Errorf("expected 0xc0 to be invalid JSON; if encode.go:76 ever becomes reachable it must not return this byte")
	}
}

func TestRawMessage_SliceHeaderCorruption(t *testing.T) {
	type W struct {
		R json.RawMessage `json:"r"`
	}
	// "null" is the only RawMessage payload safe to test in-process — every
	// other value (including the empty object `{}`) triggers SIGBUS
	in := W{R: json.RawMessage("null")}

	stdlibOut, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("stdlib marshal failed: %v", err)
	}
	serdesOut, err := serdes.Serialize(in, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("serdes serialize failed: %v", err)
	}

	t.Logf("stdlib: %s", stdlibOut)
	t.Logf("serdes: %s", serdesOut)

	if string(stdlibOut) != string(serdesOut) {
		t.Errorf("divergence:\nstdlib=%s\nserdes=%s", stdlibOut, serdesOut)
	}
	if !json.Valid(serdesOut) {
		t.Errorf("serdes produced invalid JSON: %s", serdesOut)
	}
}

type inner struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// Just a large payload to exercise marshal/serdes.
type testPayload struct {
	// Primitives
	BoolTrue  bool    `json:"bool_true"`
	BoolFalse bool    `json:"bool_false"`
	Int       int     `json:"int"`
	Int8      int8    `json:"int8"`
	Int16     int16   `json:"int16"`
	Int32     int32   `json:"int32"`
	Int64     int64   `json:"int64"`
	Uint      uint    `json:"uint"`
	Uint8     uint8   `json:"uint8"`
	Uint16    uint16  `json:"uint16"`
	Uint32    uint32  `json:"uint32"`
	Uint64    uint64  `json:"uint64"`
	Float32   float32 `json:"float32"`
	Float64   float64 `json:"float64"`

	// Strings
	EmptyString    string `json:"empty_string"`
	NonEmptyString string `json:"non_empty_string"`
	UnicodeString  string `json:"unicode_string"`

	// Byte slice (base64-encoded in JSON)
	Bytes []byte `json:"bytes"`

	// Time
	Timestamp time.Time `json:"timestamp"`

	// Pointers — one nil, one set
	PtrNil *string `json:"ptr_nil"`
	PtrSet *string `json:"ptr_set"`

	// Nested struct
	Nested inner `json:"nested"`

	// Pointer to struct
	NestedPtr *inner `json:"nested_ptr"`

	// Slices
	IntSlice    []int    `json:"int_slice"`
	StringSlice []string `json:"string_slice"`
	EmptySlice  []int    `json:"empty_slice"`
	NilSlice    []int    `json:"nil_slice"` // marshals as null

	// Maps
	StringMap map[string]string `json:"string_map"`
	IntMap    map[string]int    `json:"int_map"`
	NilMap    map[string]int    `json:"nil_map"` // marshals as null

	// Slice of structs
	InnerSlice []inner `json:"inner_slice"`

	// Interface (holds a concrete value)
	AnyField any `json:"any_field"`

	// Raw JSON passthrough
	Raw json.RawMessage `json:"raw"`
}

func newTestPayload() testPayload {
	ptrStr := "hello pointer"
	return testPayload{
		BoolTrue:       true,
		BoolFalse:      false,
		Int:            -42,
		Int8:           math.MaxInt8,
		Int16:          math.MaxInt16,
		Int32:          math.MaxInt32,
		Int64:          math.MaxInt64,
		Uint:           42,
		Uint8:          math.MaxUint8,
		Uint16:         math.MaxUint16,
		Uint32:         math.MaxUint32,
		Uint64:         math.MaxUint64,
		Float32:        3.14,
		Float64:        math.Pi,
		EmptyString:    "",
		NonEmptyString: "hello, world",
		UnicodeString:  "こんにちは 🌍",
		Bytes:          []byte("binary data"),
		Timestamp:      time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		PtrNil:         nil,
		PtrSet:         &ptrStr,
		Nested:         inner{Label: "nested", Value: 1},
		NestedPtr:      &inner{Label: "nested-ptr", Value: 2},
		IntSlice:       []int{1, 2, 3},
		StringSlice:    []string{"a", "b", "c"},
		EmptySlice:     []int{},
		NilSlice:       nil,
		StringMap:      map[string]string{"key": "value"},
		IntMap:         map[string]int{"one": 1, "two": 2},
		NilMap:         nil,
		InnerSlice:     []inner{{Label: "x", Value: 10}, {Label: "y", Value: 20}},
		AnyField:       map[string]any{"count": float64(3)},
		Raw:            json.RawMessage(`{"raw":true}`),
	}
}

func TestLargePayload(t *testing.T) {
	// This test won't work until you clear the errors in prior test.
	original := newTestPayload()
	stdlibJSON, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("stdlib marshal failed: %v", err)
	}
	serdesJSON, err := serdes.Serialize(original, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("serdes marshal failed: %v", err)
	}
	// I'm not 100% sure it matters if JSON matches as long as JSON is working/valid.
	// But this is an interesting test.
	if string(stdlibJSON) != string(serdesJSON) {
		t.Fatalf("divergence from stdlib:\nstdlib: %s\nserdes: %s", stdlibJSON, serdesJSON)
	}
	var stdlibRoundTripped testPayload
	err = json.Unmarshal(stdlibJSON, &stdlibRoundTripped)
	if err != nil {
		t.Fatalf("stdlib unmarshal failed: %v", err)
	}
	if !reflect.DeepEqual(original, stdlibRoundTripped) {
		t.Fatalf("stdlib round trip mismatch:\noriginal     : %+v\nround-tripped: %+v", original, stdlibRoundTripped)
	}
	var serdesRoundTripped testPayload
	err = serdes.Deserialize(serdesJSON, &serdesRoundTripped, serdes.TargetCollection)
	if err != nil {
		t.Fatalf("serdes unmarshal failed: %v", err)
	}
	if !reflect.DeepEqual(original, serdesRoundTripped) {
		t.Fatalf("serdes round trip mismatch:\noriginal     : %+v\nround-tripped: %+v", original, serdesRoundTripped)
	}
}
