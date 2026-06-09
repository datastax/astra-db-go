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

package datatypes_test

import (
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

func TestVector_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		floats := rapid.SliceOf(rapid.Float32()).Draw(t, "floats")
		v := datatypes.NewVector(floats)

		// Base64 round-trip
		b64 := v.AsBase64()
		v2 := datatypes.NewVector(b64)
		floats2, err := v2.AsFloatArray()
		if err != nil {
			t.Fatalf("AsFloatArray failed for base64 %q: %v", b64, err)
		}
		if diff := cmp.Diff(floats, floats2); diff != "" {
			t.Fatalf("Base64 round-trip mismatch (-want +got):\n%s", diff)
		}

		// JSON round-trip (should produce {$binary: ...})
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var v3 datatypes.Vector
		if err := json.Unmarshal(data, &v3); err != nil {
			t.Fatalf("JSON unmarshal failed: %v", err)
		}
		floats3, err := v3.AsFloatArray()
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(floats, floats3); diff != "" {
			t.Fatalf("JSON round-trip mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestVector_Dimension(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		floats := rapid.SliceOf(rapid.Float32()).Draw(t, "floats")
		v := datatypes.NewVector(floats)
		if got, want := v.Dimension(), len(floats); got != want {
			t.Fatalf("Dimension mismatch: got %d, want %d", got, want)
		}

		// Test dimension from base64
		v2 := datatypes.NewVector(v.AsBase64())
		if got, want := v2.Dimension(), len(floats); got != want {
			t.Fatalf("Dimension mismatch (from base64): got %d, want %d", got, want)
		}
	})
}

func TestVector_UnmarshalDifferentFormats(t *testing.T) {
	// Raw array format
	input := `[0.1, -0.2, 0.3]`
	var v datatypes.Vector
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		t.Fatal(err)
	}
	floats, _ := v.AsFloatArray()
	if len(floats) != 3 {
		t.Fatal("Expected 3 floats")
	}

	// Binary format
	input2 := `{"$binary": "PczMzb5MzM0+mZma"}`
	var v2 datatypes.Vector
	if err := json.Unmarshal([]byte(input2), &v2); err != nil {
		t.Fatal(err)
	}
	if v2.Dimension() != 3 {
		t.Fatalf("Expected dimension 3, got %d", v2.Dimension())
	}
}

func TestVector_AppendBase64(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		floats := rapid.SliceOf(rapid.Float32()).Draw(t, "floats")
		v := datatypes.NewVector(floats)
		dst := []byte("prefix:")
		got := v.AppendBase64(dst)
		if string(got[:7]) != "prefix:" {
			t.Fatal("Prefix lost")
		}
		if string(got[7:]) != v.AsBase64() {
			t.Fatal("Base64 mismatch in AppendBase64")
		}
	})
}
