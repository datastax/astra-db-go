package datatypes_test

import (
	"math/big"
	"slices"
	"testing"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"
)

func TestSortedMap_Operations(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		m := datatypes.NewSortedMap[int, int]()
		shadow := make(map[int]int)

		type op struct {
			kind string
			key  int
			val  int
		}

		ops := rapid.SliceOf(rapid.Custom(func(t *rapid.T) op {
			kind := rapid.SampledFrom([]string{"set", "delete", "get", "clear"}).Draw(t, "kind")
			key := rapid.IntRange(-1000, 1000).Draw(t, "key")
			val := rapid.Int().Draw(t, "val")
			return op{kind, key, val}
		})).Draw(rt, "ops")

		for _, o := range ops {
			switch o.kind {
			case "set":
				m.Set(o.key, o.val)
				shadow[o.key] = o.val
			case "delete":
				m.Delete(o.key)
				delete(shadow, o.key)
			case "get":
				got, found := m.Get(o.key)
				want, wantFound := shadow[o.key]
				if found != wantFound {
					rt.Fatalf("Get(%d) found mismatch", o.key)
				}
				if found && got != want {
					rt.Fatalf("Get(%d) value mismatch", o.key)
				}
			case "clear":
				m.Clear()
				clear(shadow)
			}

			if m.Len() != len(shadow) {
				rt.Fatalf("Len mismatch")
			}
		}

		// Check sorted order
		var keys []int
		for k := range m.All() {
			keys = append(keys, k)
		}
		if !slices.IsSorted(keys) {
			rt.Fatal("All() not sorted")
		}

		shadowKeys := mapKeys(shadow)
		slices.Sort(shadowKeys)
		if diff := cmp.Diff(shadowKeys, keys, cmpopts.EquateEmpty()); diff != "" {
			rt.Fatalf("Keys mismatch: %s", diff)
		}
	})
}

func TestSortedMap_DeleteDuringIter(t *testing.T) {
	m := datatypes.NewSortedMap[int, int]()
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
}

func TestSortedMap_Ranges(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		m := datatypes.NewSortedMap[int, int]()
		for i := 0; i < 100; i++ {
			m.Set(i, i)
		}

		lo := rapid.IntRange(-10, 110).Draw(rt, "lo")
		hi := lo + rapid.IntRange(0, 50).Draw(rt, "diff")

		var keys []int
		for k := range m.All(lo, hi) {
			keys = append(keys, k)
			if k < lo || k > hi {
				rt.Fatalf("Key %d out of range [%d, %d]", k, lo, hi)
			}
		}
		if !slices.IsSorted(keys) {
			rt.Fatal("Range iterator not sorted")
		}

		// Reverse
		var revKeys []int
		for k := range m.AllRev(lo, hi) {
			revKeys = append(revKeys, k)
		}
		slices.Reverse(revKeys)
		if diff := cmp.Diff(keys, revKeys, cmpopts.EquateEmpty()); diff != "" {
			rt.Fatalf("Reverse range mismatch: %s", diff)
		}
	})
}

func TestSortedMap_BigTypes(t *testing.T) {
	// big.Int
	m := datatypes.NewSortedMap[big.Int, int]()
	for _, v := range []int64{10, -5, 100, 0} {
		var bi big.Int
		bi.SetInt64(v)
		m.Set(bi, int(v))
	}
	var got []int64
	for k := range m.All() {
		got = append(got, k.Int64())
	}
	if !slices.IsSorted(got) {
		t.Fatal("big.Int not sorted")
	}

	// big.Float
	mf := datatypes.NewSortedMap[big.Float, int]()
	for _, v := range []float64{10.5, -5.2, 100.1, 0.0} {
		var bf big.Float
		bf.SetFloat64(v)
		mf.Set(bf, int(v))
	}
	var gotf []float64
	for k := range mf.All() {
		f, _ := k.Float64()
		gotf = append(gotf, f)
	}
	if !slices.IsSorted(gotf) {
		t.Fatal("big.Float not sorted")
	}
}

func TestSortedMap_FirstLast(t *testing.T) {
	m := datatypes.NewSortedMap[int, int]()
	if m.First() != nil || m.Last() != nil {
		t.Fatal("Empty map should have nil First/Last")
	}
	m.Set(10, 1)
	m.Set(5, 1)
	m.Set(15, 1)
	if m.First().Key() != 5 || m.Last().Key() != 15 {
		t.Fatal("First/Last mismatch")
	}
}

func TestSortedMap_Nil(t *testing.T) {
	var m datatypes.SortedMap[string, int]

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
		} else if r != "assignment to entry in nil SortedMap" {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	m.Set("any", 1)
}
