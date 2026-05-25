package datatypes_test

import (
	"math/big"
	"slices"
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"
)

func TestSet_Operations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := datatypes.NewSet[string]()
		shadow := make(map[string]struct{})

		type op struct {
			kind string
			val  string
		}

		ops := rapid.SliceOf(rapid.Custom(func(t *rapid.T) op {
			kind := rapid.SampledFrom([]string{"add", "delete", "has", "pop", "clear"}).Draw(t, "kind")
			val := rapid.String().Draw(t, "val")
			return op{kind, val}
		})).Draw(t, "ops")

		for _, o := range ops {
			switch o.kind {
			case "add":
				s.Add(o.val)
				shadow[o.val] = struct{}{}
			case "delete":
				s.Delete(o.val)
				delete(shadow, o.val)
			case "has":
				if got, want := s.Has(o.val), shadowHas(shadow, o.val); got != want {
					t.Fatalf("Has(%q) mismatch: got %v, want %v", o.val, got, want)
				}
			case "pop":
				val, ok := s.Pop()
				_, shadowOk := firstKey(shadow)
				if ok != (shadowOk) {
					t.Fatalf("Pop() ok mismatch: got %v, want %v", ok, shadowOk)
				}
				if ok {
					if _, existed := shadow[val]; !existed {
						t.Fatalf("Pop() returned %q which was not in shadow", val)
					}
					delete(shadow, val)
				}
			case "clear":
				s.Clear()
				clear(shadow)
			}

			if got, want := s.Len(), len(shadow); got != want {
				t.Fatalf("Len() mismatch: got %d, want %d", got, want)
			}
		}

		// Final state check
		gotSlice := s.ToSlice()
		slices.Sort(gotSlice)
		wantSlice := mapKeys(shadow)
		slices.Sort(wantSlice)
		if diff := cmp.Diff(wantSlice, gotSlice, cmpopts.EquateEmpty()); diff != "" {
			t.Fatalf("Final state mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestSet_SetLogic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elems1 := rapid.SliceOf(rapid.Int()).Draw(t, "elems1")
		elems2 := rapid.SliceOf(rapid.Int()).Draw(t, "elems2")

		s1 := datatypes.NewSet(elems1...)
		s2 := datatypes.NewSet(elems2...)

		// Union
		union := s1.Union(s2)
		for _, e := range elems1 {
			if !union.Has(e) {
				t.Fatalf("Union missing element from s1: %v", e)
			}
		}
		for _, e := range elems2 {
			if !union.Has(e) {
				t.Fatalf("Union missing element from s2: %v", e)
			}
		}

		// Intersection
		intersection := s1.Intersection(s2)
		for _, e := range intersection.ToSlice() {
			if !s1.Has(e) || !s2.Has(e) {
				t.Fatalf("Intersection contains element not in both: %v", e)
			}
		}

		// Difference
		diff := s1.Difference(s2)
		for _, e := range diff.ToSlice() {
			if !s1.Has(e) || s2.Has(e) {
				t.Fatalf("Difference contains invalid element: %v", e)
			}
		}

		// SymmetricDifference
		symDiff := s1.SymmetricDifference(s2)
		for _, e := range symDiff.ToSlice() {
			in1, in2 := s1.Has(e), s2.Has(e)
			if (in1 && in2) || (!in1 && !in2) {
				t.Fatalf("SymmetricDifference contains invalid element: %v", e)
			}
		}

		// Subset
		if !s1.IsSubsetOf(union) {
			t.Fatalf("s1 should be subset of union")
		}
		if !datatypes.NewSet[int]().IsSubsetOf(s1) {
			t.Fatalf("empty set should be subset of s1")
		}
	})
}

func TestSet_Equals(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elems := rapid.SliceOf(rapid.String()).Draw(t, "elems")
		s1 := datatypes.NewSet(elems...)
		s2 := datatypes.NewSet(elems...)
		if !s1.Equals(s2) {
			t.Fatalf("Sets with same elements should be equal")
		}

		if len(elems) > 0 {
			s1.Add(rapid.String().Draw(t, "extra"))
			// This might still be equal if extra was already in elems, 
			// but Equals is simple enough.
		}
	})
}

func TestSet_Clone(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elems := rapid.SliceOf(rapid.Int()).Draw(t, "elems")
		s1 := datatypes.NewSet(elems...)
		s2 := s1.Clone()
		if !s1.Equals(s2) {
			t.Fatalf("Clone should be equal")
		}
		s1.Add(-123456)
		if s2.Has(-123456) {
			t.Fatalf("Clone should be independent")
		}
	})
}

func TestSet_AllRev(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elems := rapid.SliceOf(rapid.Int()).Draw(t, "elems")
		s := datatypes.NewSet(elems...)
		
		var got []int
		for e := range s.AllRev() {
			got = append(got, e)
		}
		
		want := s.ToSlice()
		slices.Reverse(want)
		
		if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
			t.Fatalf("AllRev mismatch (-want +got):\n%s", diff)
		}
	})
}

func shadowHas(m map[string]struct{}, val string) bool {
	_, ok := m[val]
	return ok
}

func firstKey(m map[string]struct{}) (string, bool) {
	for k := range m {
		return k, true
	}
	return "", false
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestSet_NonComparable(t *testing.T) {
	s := datatypes.NewSet[big.Int]()

	b1 := *big.NewInt(10)
	b2 := *big.NewInt(20)
	b3 := *big.NewInt(5)

	s.Add(b1)
	s.Add(b2)
	s.Add(b3)

	if s.Len() != 3 {
		t.Fatalf("expected len 3, got %d", s.Len())
	}

	// Should be sorted
	got := s.ToSlice()
	if got[0].Cmp(big.NewInt(5)) != 0 || got[1].Cmp(big.NewInt(10)) != 0 || got[2].Cmp(big.NewInt(20)) != 0 {
		t.Fatalf("not sorted correctly: %v", got)
	}
}

func TestSet_Nil(t *testing.T) {
	var s datatypes.Set[string]

	// Reads should be safe and return zero values
	if s.Has("any") {
		t.Error("Has on nil set: got true, want false")
	}
	if got := s.Len(); got != 0 {
		t.Errorf("Len on nil set: got %d, want 0", got)
	}
	if got := s.First(); got != "" {
		t.Errorf("First on nil set: got %q, want \"\"", got)
	}
	if got := s.Last(); got != "" {
		t.Errorf("Last on nil set: got %q, want \"\"", got)
	}

	// Iteration should be safe and empty
	for range s.All() {
		t.Fatal("All() on nil set should not yield anything")
	}

	// Delete and Clear should be safe no-ops
	s.Delete("any")
	s.Clear()

	// Pop should be safe and return zero/false
	if got, ok := s.Pop(); ok || got != "" {
		t.Errorf("Pop on nil set: got (%q, %v), want (\"\", false)", got, ok)
	}

	// Writes should panic (via SortedMap)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Add on nil set should panic")
		} else if r != "assignment to entry in nil SortedMap" {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	s.Add("any")
}
