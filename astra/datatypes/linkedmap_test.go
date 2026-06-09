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
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"
)

func TestLinkedMap_Operations(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		m := datatypes.NewLinkedMap[string, int]()
		shadow := make(map[string]int)
		var order []string

		type op struct {
			kind string
			key  string
			val  int
		}

		ops := rapid.SliceOf(rapid.Custom(func(t *rapid.T) op {
			kind := rapid.SampledFrom([]string{"set", "delete", "get", "get_default", "clear"}).Draw(t, "kind")
			key := rapid.String().Draw(t, "key")
			val := rapid.Int().Draw(t, "val")
			return op{kind, key, val}
		})).Draw(rt, "ops")

		for _, o := range ops {
			switch o.kind {
			case "set":
				oldVal, existed := m.Set(o.key, o.val)
				shadowOldVal, shadowExisted := shadow[o.key]
				if existed != shadowExisted {
					rt.Fatalf("Set(%q) existed mismatch: got %v, want %v", o.key, existed, shadowExisted)
				}
				if existed && oldVal != shadowOldVal {
					rt.Fatalf("Set(%q) old value mismatch: got %v, want %v", o.key, oldVal, shadowOldVal)
				}
				shadow[o.key] = o.val
				if !shadowExisted {
					order = append(order, o.key)
				}
			case "delete":
				val, found := m.Delete(o.key)
				shadowVal, shadowFound := shadow[o.key]
				if found != shadowFound {
					rt.Fatalf("Delete(%q) found mismatch: got %v, want %v", o.key, found, shadowFound)
				}
				if found && val != shadowVal {
					rt.Fatalf("Delete(%q) value mismatch: got %v, want %v", o.key, val, shadowVal)
				}
				delete(shadow, o.key)
				for i, k := range order {
					if k == o.key {
						order = append(order[:i], order[i+1:]...)
						break
					}
				}
			case "get":
				val, found := m.Get(o.key)
				shadowVal, shadowFound := shadow[o.key]
				if found != shadowFound {
					rt.Fatalf("Get(%q) found mismatch", o.key)
				}
				if found && val != shadowVal {
					rt.Fatalf("Get(%q) value mismatch", o.key)
				}
			case "get_default":
				def := rapid.Int().Draw(rt, "def")
				got := m.GetOrDefault(o.key, def)
				want := def
				if v, ok := shadow[o.key]; ok {
					want = v
				}
				if got != want {
					rt.Fatalf("GetOrDefault(%q) mismatch", o.key)
				}
			case "clear":
				m.Clear()
				clear(shadow)
				order = nil
			}

			if m.Len() != len(shadow) {
				rt.Fatalf("Len mismatch")
			}

			// Check order
			var currentOrder []string
			for k := range m.All() {
				currentOrder = append(currentOrder, k)
			}
			if diff := cmp.Diff(order, currentOrder, cmpopts.EquateEmpty()); diff != "" {
				rt.Fatalf("Order mismatch after %s: %s", o.kind, diff)
			}

			// Check reverse order
			var currentOrderRev []string
			for k := range m.AllRev() {
				currentOrderRev = append(currentOrderRev, k)
			}
			expectedRev := make([]string, len(order))
			for i, k := range order {
				expectedRev[len(order)-1-i] = k
			}
			if diff := cmp.Diff(expectedRev, currentOrderRev, cmpopts.EquateEmpty()); diff != "" {
				rt.Fatalf("Reverse order mismatch after %s: %s", o.kind, diff)
			}
		}
	})
}

func TestLinkedMap_DeleteDuringIter(t *testing.T) {
	m := datatypes.NewLinkedMap[int, int]()
	m.Set(1, 1)
	m.Set(2, 2)
	m.Set(3, 3)

	count := 0
	for k := range m.All() {
		count++
		if k == 2 {
			m.Delete(k)
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 iterations, got %d", count)
	}

	// Reverse
	m = datatypes.NewLinkedMap[int, int]()
	m.Set(1, 1)
	m.Set(2, 2)
	m.Set(3, 3)

	count = 0
	for k := range m.AllRev() {
		count++
		if k == 2 {
			m.Delete(k)
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 iterations (rev), got %d", count)
	}
}

func TestLinkedMap_Clone(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		m := datatypes.NewLinkedMap[int, int]()
		keys := rapid.SliceOf(rapid.Int()).Draw(rt, "keys")
		for _, k := range keys {
			m.Set(k, k)
		}
		clone := m.Clone()
		if clone.Len() != m.Len() {
			rt.Fatal("Clone len mismatch")
		}

		// Find a key not in the map
		uniqueKey := rapid.Int().Filter(func(k int) bool { return !m.Has(k) }).Draw(rt, "uniqueKey")
		m.Set(uniqueKey, uniqueKey)
		if clone.Has(uniqueKey) {
			rt.Fatal("Clone should be independent")
		}
	})
}

func TestLinkedMap_FirstLast(t *testing.T) {
	m := datatypes.NewLinkedMap[string, string]()
	if m.First() != nil || m.Last() != nil {
		t.Fatal("Empty map should have nil First/Last")
	}
	m.Set("a", "1")
	if m.First().Key() != "a" || m.Last().Key() != "a" {
		t.Fatal("Single element First/Last mismatch")
	}
	m.Set("b", "2")
	if m.First().Key() != "a" || m.Last().Key() != "b" {
		t.Fatal("Two element First/Last mismatch")
	}
}

func TestLinkedMap_Nil(t *testing.T) {
	var m datatypes.LinkedMap[string, int]

	// Reads should be safe and return zero values
	if got, found := m.Get("any"); found || got != 0 {
		t.Errorf("Get on nil map: got (%v, %v), want (0, false)", got, found)
	}
	if m.Has("any") {
		t.Error("Has on nil map: got true, want false")
	}
	if got := m.Len(); got != 0 {
		t.Errorf("Len on nil map: got %d, want 0", got)
	}
	if m.First() != nil || m.Last() != nil {
		t.Error("First/Last on nil map should be nil")
	}
	if m.GetOrDefault("any", 42) != 42 {
		t.Error("GetOrDefault on nil map failed")
	}

	// Iteration should be safe and empty
	for range m.All() {
		t.Fatal("All() on nil map should not yield anything")
	}
	for range m.AllRev() {
		t.Fatal("AllRev() on nil map should not yield anything")
	}

	// Delete and Clear should be safe no-ops
	m.Delete("any")
	m.Clear()

	// Writes should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Set on nil map should panic")
		} else if r != "assignment to entry in nil LinkedMap" {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	m.Set("any", 1)
}
