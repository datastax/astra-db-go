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
	"strings"
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

func TestFlags_SortMapKeys(t *testing.T) {
	m := map[string]any{
		"z": 1,
		"a": 2,
		"m": 3,
	}

	// Default: unsorted (random order in Go)
	// We can't easily test for "unsorted" because it might accidentally be sorted,
	// but we can test that it IS sorted when we ask for it.

	got, err := serdes.Serialize(m, serdes.TargetNone, serdes.SortMapKeys)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	expected := `{"a":2,"m":3,"z":1}`
	if string(got) != expected {
		t.Errorf("SortMapKeys: expected %s, got %s", expected, string(got))
	}
}

func TestFlags_SortMapKeys_Nested(t *testing.T) {
	m := map[string]any{
		"z": map[string]any{
			"b": 1,
			"a": 2,
		},
		"a": 3,
	}

	got, err := serdes.Serialize(m, serdes.TargetNone, serdes.SortMapKeys)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	expected := `{"a":3,"z":{"a":2,"b":1}}`
	if string(got) != expected {
		t.Errorf("SortMapKeys Nested: expected %s, got %s", expected, string(got))
	}
}

func TestFlags_TrustRawMessage(t *testing.T) {
	raw := json.RawMessage(`{"invalid": }`) // invalid JSON

	// Without TrustRawMessage: should error
	_, err := serdes.Serialize(raw, serdes.TargetNone)
	if err == nil {
		t.Error("expected error for invalid RawMessage without TrustRawMessage")
	}

	// With TrustRawMessage: should NOT error
	got, err := serdes.Serialize(raw, serdes.TargetNone, serdes.TrustRawMessage)
	if err != nil {
		t.Fatalf("Serialize failed with TrustRawMessage: %v", err)
	}

	if string(got) != string(raw) {
		t.Errorf("TrustRawMessage: expected %s, got %s", string(raw), string(got))
	}
}

func TestFlags_UseNumber(t *testing.T) {
	data := []byte(`{"large": 12345678901234567890}`)

	// Without UseNumber: float64 (might lose precision)
	var res1 any
	err := serdes.Deserialize(data, &res1, nil, serdes.TargetNone)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}
	m1 := res1.(map[string]any)
	if _, ok := m1["large"].(float64); !ok {
		t.Errorf("expected float64, got %T", m1["large"])
	}

	// With UseNumber: json.Number
	var res2 any
	err = serdes.Deserialize(data, &res2, nil, serdes.TargetNone, serdes.UseNumber)
	if err != nil {
		t.Fatalf("Deserialize failed with UseNumber: %v", err)
	}
	m2 := res2.(map[string]any)
	if num, ok := m2["large"].(json.Number); ok {
		if num.String() != "12345678901234567890" {
			t.Errorf("UseNumber: expected string '12345678901234567890', got %s", num.String())
		}
	} else {
		t.Errorf("expected json.Number, got %T", m2["large"])
	}
}

type JSONCustom struct {
	Value string
}

func (j JSONCustom) MarshalJSON() ([]byte, error) {
	return []byte(`"json:` + j.Value + `"`), nil
}

func (j *JSONCustom) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if strings.HasPrefix(s, "json:") {
		j.Value = s[5:]
	} else {
		j.Value = s
	}
	return nil
}

func TestFlags_RecognizeJSONMarshaler(t *testing.T) {
	c := JSONCustom{Value: "hello"}

	// Default: should use default struct marshaler
	got1, err := serdes.Serialize(c, serdes.TargetNone)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	if string(got1) != `{"Value":"hello"}` {
		t.Errorf("Default: expected %s, got %s", `{"Value":"hello"}`, string(got1))
	}

	// With UseJSONMarshal: should use MarshalJSON
	got2, err := serdes.Serialize(c, serdes.TargetNone, serdes.UseJSONMarshal)
	if err != nil {
		t.Fatalf("Serialize with UseJSONMarshal failed: %v", err)
	}
	if string(got2) != `"json:hello"` {
		t.Errorf("UseJSONMarshal: expected %s, got %s", `"json:hello"`, string(got2))
	}
}

func TestFlags_RecognizeJSONUnmarshaler(t *testing.T) {
	data := []byte(`"json:world"`)

	// Default: should fail to unmarshal string into struct
	var res1 JSONCustom
	err := serdes.Deserialize(data, &res1, nil, serdes.TargetNone)
	if err == nil {
		t.Error("expected error for Default")
	}

	// With UseJSONUnmarshal: should use UnmarshalJSON
	var res2 JSONCustom
	err = serdes.Deserialize(data, &res2, nil, serdes.TargetNone, serdes.UseJSONUnmarshal)
	if err != nil {
		t.Fatalf("Deserialize with UseJSONUnmarshal failed: %v", err)
	}
	if res2.Value != "world" {
		t.Errorf("UseJSONUnmarshal: expected Value='world', got %s", res2.Value)
	}
}

type BothCustom struct {
	Value string
}

func (b BothCustom) MarshalAstra(ctx serdes.EncodeCtx) (any, error) {
	return "astra:" + b.Value, nil
}

func (b BothCustom) MarshalJSON() ([]byte, error) {
	return []byte(`"json:` + b.Value + `"`), nil
}

func TestFlags_JSONPrecedence(t *testing.T) {
	c := BothCustom{Value: "both"}

	// Astra should take precedence even if UseJSONMarshal is set
	got, err := serdes.Serialize(c, serdes.TargetNone, serdes.UseJSONMarshal)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	if string(got) != `"astra:both"` {
		t.Errorf("Precedence: expected %s, got %s", `"astra:both"`, string(got))
	}
}

func TestFlags_ExtendedErrorSnippet(t *testing.T) {
	// Unmarshal a long string into an int to trigger UnmarshalTypeError
	longStr := "x" + strings.Repeat("x", 58) + "x" // 60 chars
	data := []byte(`{"a": "` + longStr + `"}`)
	type Target struct {
		A int `json:"a"`
	}

	// Default: 16 chars snippet
	var res1 Target
	err := serdes.Deserialize(data, &res1, nil, serdes.TargetNone)
	if err == nil {
		t.Fatal("expected error")
	}
	msg1 := err.Error()
	expectedSnippet1 := `{"a": "xxxxxxxxx...`
	if !strings.Contains(msg1, expectedSnippet1) {
		t.Errorf("Default snippet: expected to contain '%s', got '%s'", expectedSnippet1, msg1)
	}

	// ExtendedErrorSnippet: 64 chars snippet
	err = serdes.Deserialize(data, &res1, nil, serdes.TargetNone, serdes.ExtendedErrorSnippet)
	if err == nil {
		t.Fatal("expected error")
	}
	msg2 := err.Error()
	expectedSnippet2 := `{"a": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx...`
	if !strings.Contains(msg2, expectedSnippet2) {
		t.Errorf("ExtendedErrorSnippet snippet: expected to contain '%s', got '%s'", expectedSnippet2, msg2)
	}
}

func TestFlags_NoCache(t *testing.T) {
	type DynamicStruct struct {
		A int
	}

	// Serialize with cache
	_, _ = serdes.Serialize(DynamicStruct{A: 1}, serdes.TargetNone)

	// This is hard to test but we can at least verify it doesn't crash
	_, err := serdes.Serialize(DynamicStruct{A: 1}, serdes.TargetNone, serdes.SerNoCache)
	if err != nil {
		t.Errorf("Serialize with SerNoCache failed: %v", err)
	}

	var res DynamicStruct
	err = serdes.Deserialize([]byte(`{"A":1}`), &res, nil, serdes.TargetNone, serdes.DesNoCache)
	if err != nil {
		t.Errorf("Deserialize with DesNoCache failed: %v", err)
	}
}
