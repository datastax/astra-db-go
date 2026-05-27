// Copyright IBM Corp.

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

package options

import (
	"testing"

	"github.com/datastax/astra-db-go/astra/ptr"
)

// testStruct is a simple struct with pointer fields for testing copyNonNilFields.
type testStruct struct {
	Name    *string
	Age     *int
	Enabled *bool
}

func TestCopyNonNilFields_AllNil(t *testing.T) {
	src := &testStruct{}
	dst := &testStruct{}

	copyNonNilFields(src, dst)

	if dst.Name != nil {
		t.Errorf("expected Name to remain nil, got %v", *dst.Name)
	}
	if dst.Age != nil {
		t.Errorf("expected Age to remain nil, got %v", *dst.Age)
	}
	if dst.Enabled != nil {
		t.Errorf("expected Enabled to remain nil, got %v", *dst.Enabled)
	}
}

func TestCopyNonNilFields_AllSet(t *testing.T) {
	src := &testStruct{
		Name:    ptr.To("alice"),
		Age:     ptr.To(30),
		Enabled: ptr.To(true),
	}
	dst := &testStruct{}

	copyNonNilFields(src, dst)

	if dst.Name == nil || *dst.Name != "alice" {
		t.Errorf("expected Name 'alice', got %v", dst.Name)
	}
	if dst.Age == nil || *dst.Age != 30 {
		t.Errorf("expected Age 30, got %v", dst.Age)
	}
	if dst.Enabled == nil || *dst.Enabled != true {
		t.Errorf("expected Enabled true, got %v", dst.Enabled)
	}
}

func TestCopyNonNilFields_PartialOverwrite(t *testing.T) {
	src := &testStruct{
		Name: ptr.To("bob"),
		// Age and Enabled left nil
	}
	dst := &testStruct{
		Name:    ptr.To("original"),
		Age:     ptr.To(25),
		Enabled: ptr.To(false),
	}

	copyNonNilFields(src, dst)

	// Name should be overwritten
	if dst.Name == nil || *dst.Name != "bob" {
		t.Errorf("expected Name 'bob', got %v", dst.Name)
	}
	// Age and Enabled should be preserved
	if dst.Age == nil || *dst.Age != 25 {
		t.Errorf("expected Age to remain 25, got %v", dst.Age)
	}
	if dst.Enabled == nil || *dst.Enabled != false {
		t.Errorf("expected Enabled to remain false, got %v", dst.Enabled)
	}
}

func TestCopyNonNilFields_NilSrcPreservesDst(t *testing.T) {
	src := &testStruct{}
	dst := &testStruct{
		Name:    ptr.To("keep"),
		Age:     ptr.To(42),
		Enabled: ptr.To(true),
	}

	copyNonNilFields(src, dst)

	if dst.Name == nil || *dst.Name != "keep" {
		t.Errorf("expected Name to remain 'keep', got %v", dst.Name)
	}
	if dst.Age == nil || *dst.Age != 42 {
		t.Errorf("expected Age to remain 42, got %v", dst.Age)
	}
	if dst.Enabled == nil || *dst.Enabled != true {
		t.Errorf("expected Enabled to remain true, got %v", dst.Enabled)
	}
}

func TestCopyNonNilFields_PointersAreShared(t *testing.T) {
	name := ptr.To("shared")
	src := &testStruct{Name: name}
	dst := &testStruct{}

	copyNonNilFields(src, dst)

	// dst.Name should point to the same value as src.Name
	if dst.Name != src.Name {
		t.Error("expected dst.Name to share the same pointer as src.Name")
	}
}

// testStructWithNonPointer has a mix of pointer and non-pointer fields.
type testStructWithNonPointer struct {
	Label   string
	Count   int
	Enabled *bool
	Name    *string
}

func TestCopyNonNilFields_IgnoresNonPointerFields(t *testing.T) {
	src := &testStructWithNonPointer{
		Label:   "src-label",
		Count:   99,
		Enabled: ptr.To(true),
		Name:    ptr.To("alice"),
	}
	dst := &testStructWithNonPointer{
		Label:   "dst-label",
		Count:   1,
		Enabled: ptr.To(false),
		Name:    ptr.To("bob"),
	}

	copyNonNilFields(src, dst)

	// Non-pointer fields should NOT be copied
	if dst.Label != "dst-label" {
		t.Errorf("expected Label to remain 'dst-label', got %q", dst.Label)
	}
	if dst.Count != 1 {
		t.Errorf("expected Count to remain 1, got %d", dst.Count)
	}
	// Pointer fields should be copied
	if dst.Enabled == nil || *dst.Enabled != true {
		t.Errorf("expected Enabled true, got %v", dst.Enabled)
	}
	if dst.Name == nil || *dst.Name != "alice" {
		t.Errorf("expected Name 'alice', got %v", dst.Name)
	}
}
