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

package structutil

import (
	"reflect"
	"testing"
)

type E1 struct {
	A string `json:"a"`
	B string `json:"b"`
}

type E2 struct {
	B string `json:"b"`
	C string `json:"c"`
}

type TopShadow struct {
	E1
	A string `json:"a"` // Shadows E1.A
}

type TopAmbiguous struct {
	E1
	E2 // Both E1 and E2 have a tagged "b" field -> unresolvable ambiguity
}

type PtrCycle struct {
	*PtrCycle
	Val string `json:"val"`
}

type Unexported struct {
	Exported   string
	unexported string `json:"unexported,allowunexported"`
	ignored    string `json:"-"`
}

func TestGetFields_Shadowing(t *testing.T) {
	fields, err := GetFields(reflect.TypeOf(TopShadow{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}

	if fields[0].Name != "b" || len(fields[0].Index) != 2 {
		t.Errorf("expected first field 'b' from embedded E1, got %s with index %v", fields[0].Name, fields[0].Index)
	}

	if fields[1].Name != "a" || len(fields[1].Index) != 1 {
		t.Errorf("expected second field 'a' from top level, got %s with index %v", fields[1].Name, fields[1].Index)
	}
}

func TestGetFields_Ambiguous(t *testing.T) {
	_, err := GetFields(reflect.TypeOf(TopAmbiguous{}))
	if err == nil {
		t.Fatal("expected error due to ambiguous field 'b', got nil")
	}
}

func TestGetFields_PtrCycle(t *testing.T) {
	fields, err := GetFields(reflect.TypeOf(PtrCycle{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	if fields[0].Name != "val" {
		t.Errorf("expected field 'val', got %s", fields[0].Name)
	}
}

func TestGetFields_Unexported(t *testing.T) {
	fields, err := GetFields(reflect.TypeOf(Unexported{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}

	if fields[0].Name != "Exported" {
		t.Errorf("expected first field 'Exported', got %s", fields[0].Name)
	}

	if fields[1].Name != "unexported" {
		t.Errorf("expected second field 'unexported', got %s", fields[1].Name)
	}
}
