package serdes

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzStdlibDrift(f *testing.F) {
	// The fuzzer will now automatically pick up the extensive JSON corpus
	// located in testdata/fuzz/FuzzStdlibDrift/

	f.Fuzz(func(t *testing.T, data []byte) {
		if !utf8.Valid(data) {
			t.Skip()
		}

		if bytes.Contains(data, []byte("\u2028")) || bytes.Contains(data, []byte("\\u2028")) {
			t.Skip()
		}

		if bytes.Contains(data, []byte("\u2029")) || bytes.Contains(data, []byte("\\u2029")) {
			t.Skip()
		}

		for _, ctor := range []func() any{
			func() any { return nil },
			func() any { return new([]any) },
			func() any { m := map[string]string{}; return &m },
			func() any { m := map[string]any{}; return &m },
			func() any { return new(S) },
		} {
			v1 := ctor()
			v2 := ctor()

			err1 := json.Unmarshal(data, v1)
			err2 := Deserialize(data, v2, nil, TargetNone, CaseInsensitiveFieldMatching)

			if err1 != nil {
				if err2 != nil {
					continue
				} else {
					if err1 != nil && err2 == nil {
						msg := err1.Error()

						// serdes intentionally skips control char validation in strings
						if strings.Contains(msg, "in string literal") {
							continue
						}

						// serdes loosely skips invalid primitive values on unmapped fields
						// (e.g. {"unknown": A} or {"unknown": 00})
						if strings.Contains(msg, "invalid character") {
							continue
						}
					}
					t.Fatalf("input: %s\nencoding/json.Unmarshal(%T): %T: %s\nserdes.Deserialize(%T): <nil>\nerror values mismatch", string(data), v1, err1, err1, v2)
				}
			} else {
				if err2 != nil {
					t.Fatalf("input: %s\nencoding/json.Unmarshal(%T): <nil>\nserdes.Deserialize(%T): %T: %s\nerror values mismatch", string(data), v1, v2, err2, err2)
				} else {
					// both implementations pass
				}
			}

			fixS(v1)
			fixS(v2)

			if !reflect.DeepEqual(v1, v2) {
				t.Fatalf("input: %s\nencoding/json: %#v\nserdes:        %#v\nnot equal", string(data), v1, v2)
			}

			data1, err := jsonMarshal(v1)
			if err != nil {
				t.Fatal(err)
			}

			data2, err := Serialize(v2, TargetNone, SortMapKeys)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(data1, data2) {
				t.Fatalf("input: %s\nencoding/json: %s\nserdes:        %s\nnot equal", string(data), string(data1), string(data2))
			}
		}
	})
}

func jsonMarshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), err
}

func fixS(v any) {
	if s, ok := v.(*S); ok {
		if len(s.P) == 0 {
			s.P = []byte(`""`)
		} else {
			var buf bytes.Buffer
			if err := json.Compact(&buf, s.P); err == nil {
				s.P = buf.Bytes()
			}
		}
	}
}

type S struct {
	A int    `json:",omitempty"`
	B string `json:"B1,omitempty"`
	C float64
	D bool
	E uint8
	F []byte
	G any
	H map[string]any
	I map[string]string
	J []any
	K []string
	L S1
	M *S1
	N *int
	O **int
	P json.RawMessage
	Q Marshaller
	R int `json:"-"`
}

type S1 struct {
	A int
	B string
}

type Marshaller struct {
	v string
}

var (
	_ AstraRawMarshaler   = (*Marshaller)(nil)
	_ AstraRawUnmarshaler = (*Marshaller)(nil)
)

func (m *Marshaller) MarshalJSON() ([]byte, error) {
	return jsonMarshal(m.v)
}

func (m *Marshaller) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.v)
}

func (m *Marshaller) MarshalAstraRaw(ctx EncodeCtx, dst []byte) ([]byte, error) {
	return SerializeInto(m.v, ctx.Target, dst)
}

func (m *Marshaller) UnmarshalAstraRaw(ctx DecodeCtx, data []byte) error {
	return Deserialize(data, &m.v, ctx.TargetCtx, ctx.Target)
}
