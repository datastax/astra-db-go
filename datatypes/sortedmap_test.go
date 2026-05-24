package datatypes_test

import (
	"cmp"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/quick"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/internal/testutils"
)

// Helper to construct a SortedMap for strings using the implementation's comparator factory.
func newStringSortedMap[V any]() datatypes.SortedMap[string, V] {
	return datatypes.NewSortedMap[string, V](datatypes.ComparatorFor(reflect.TypeOf("")))
}

// Helper to construct a SortedMap for ints using the implementation's comparator factory.
func newIntSortedMap[V any]() datatypes.SortedMap[int, V] {
	return datatypes.NewSortedMap[int, V](datatypes.ComparatorFor(reflect.TypeOf(0)))
}

//goland:noinspection GoMaybeNil
func TestSortedMapConstructors_NewSortedMap(t *testing.T) {
	m := newStringSortedMap[int]()
	testutils.FailIf(t, m.Len() != 0, "expected new map to be empty")
}

func TestSortedMapSetAndGet(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()

		for i := range keys {
			val := values[i%len(values)]
			oldVal, existed := m.Set(keys[i], val)

			if !existed {
				testutils.FailIf(t, oldVal != 0, "expected zero value for new key")
			}

			gotVal, found := m.Get(keys[i])
			testutils.FailIf(t, !found, "expected to find key after setting")
			testutils.FailIf(t, gotVal != val, "expected retrieved value to match set value")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapSetUpdate(t *testing.T) {
	f := func(key string, val1 int, val2 int) bool {
		m := newStringSortedMap[int]()

		oldVal1, existed1 := m.Set(key, val1)
		testutils.FailIf(t, existed1, "expected first set to be new")
		testutils.FailIf(t, oldVal1 != 0, "expected zero value for new key")

		oldVal2, existed2 := m.Set(key, val2)
		testutils.FailIf(t, !existed2, "expected second set to be update")
		testutils.FailIf(t, oldVal2 != val1, "expected old value to be returned on update")

		gotVal, found := m.Get(key)
		testutils.FailIf(t, !found, "expected to find key after update")
		testutils.FailIf(t, gotVal != val2, "expected updated value")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapGetOrDefault(t *testing.T) {
	f := func(key string, value int, defaultValue int) bool {
		m := newStringSortedMap[int]()

		gotDefault := m.GetOrDefault(key, defaultValue)
		testutils.FailIf(t, gotDefault != defaultValue, "expected default value for missing key")

		m.Set(key, value)
		gotValue := m.GetOrDefault(key, defaultValue)
		testutils.FailIf(t, gotValue != value, "expected actual value for existing key")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapDelete(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()
		expected := make(map[string]int)

		for i := range keys {
			val := values[i%len(values)]
			m.Set(keys[i], val)
			expected[keys[i]] = val
		}

		for _, key := range keys {
			if !m.Has(key) {
				continue
			}

			val, found := m.Delete(key)
			testutils.FailIf(t, !found, "expected to find key on first delete")
			testutils.FailIf(t, val != expected[key], "expected correct value on delete")

			_, found2 := m.Delete(key)
			testutils.FailIf(t, found2, "expected not to find key on second delete")

			testutils.FailIf(t, m.Has(key), "expected key to not exist after delete")
		}

		testutils.FailIf(t, m.Len() != 0, "expected map to be empty after deleting all keys")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapHas(t *testing.T) {
	f := func(key string, value int) bool {
		m := newStringSortedMap[int]()

		testutils.FailIf(t, m.Has(key), "expected key to not exist in empty map")

		m.Set(key, value)
		testutils.FailIf(t, !m.Has(key), "expected key to exist after setting")

		m.Delete(key)
		testutils.FailIf(t, m.Has(key), "expected key to not exist after deleting")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapLen(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()
		testutils.FailIf(t, m.Len() != 0, "expected empty map to have length 0")

		unique := make(map[string]struct{})
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
			unique[keys[i]] = struct{}{}
			testutils.FailIf(t, m.Len() != len(unique), "expected length to match unique keys")
		}

		for _, key := range keys {
			if m.Has(key) {
				m.Delete(key)
				delete(unique, key)
				testutils.FailIf(t, m.Len() != len(unique), "expected length to decrease after delete")
			}
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapClear(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		if m.Len() > 0 {
			m.Clear()
			testutils.FailIf(t, m.Len() != 0, "expected map to be empty after clear")

			for _, key := range keys {
				testutils.FailIf(t, m.Has(key), "expected no keys to exist after clear")
			}
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapFirstLast(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()

		firstNode := m.First()
		testutils.FailIf(t, firstNode != nil, "expected First to return nil for empty map")

		lastNode := m.Last()
		testutils.FailIf(t, lastNode != nil, "expected Last to return nil for empty map")

		uniqueKeys := make(map[string]bool)
		var sortedKeys []string

		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
			if !uniqueKeys[keys[i]] {
				uniqueKeys[keys[i]] = true
				sortedKeys = append(sortedKeys, keys[i])
			}
		}
		slices.SortFunc(sortedKeys, strings.Compare)

		firstNode = m.First()
		testutils.FailIf(t, firstNode == nil, "expected First to return non-nil for non-empty map")
		testutils.FailIf(t, firstNode.Key() != sortedKeys[0], "expected First to return the lowest sorted key")

		lastNode = m.Last()
		testutils.FailIf(t, lastNode == nil, "expected Last to return non-nil for non-empty map")
		testutils.FailIf(t, lastNode.Key() != sortedKeys[len(sortedKeys)-1], "expected Last to return the highest sorted key")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapAll(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		count := 0
		seen := make(map[string]struct{})
		var yieldedKeys []string

		for key, val := range m.All() {
			count++
			_, alreadySeen := seen[key]
			testutils.FailIf(t, alreadySeen, "expected iterator to not yield duplicate keys")

			gotVal, found := m.Get(key)
			testutils.FailIf(t, !found, "expected iterator to only yield keys in map")
			testutils.FailIf(t, gotVal != val, "expected iterator to yield correct values")

			seen[key] = struct{}{}
			yieldedKeys = append(yieldedKeys, key)
		}

		testutils.FailIf(t, count != m.Len(), "expected iterator to yield all elements")
		testutils.FailIf(t, !slices.IsSortedFunc(yieldedKeys, strings.Compare), "expected All iterator to yield elements in sorted order")

		if m.Len() > 0 {
			iterCount := 0
			for range m.All() {
				iterCount++
				if iterCount >= 1 {
					break
				}
			}
			testutils.FailIf(t, iterCount != 1, "expected to be able to break from iterator early")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapAllRev(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newStringSortedMap[int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		var forwardKeys []string
		for key := range m.All() {
			forwardKeys = append(forwardKeys, key)
		}

		var reverseKeys []string
		for key, _ := range m.AllRev() {
			reverseKeys = append(reverseKeys, key)
		}

		testutils.FailIf(t, len(reverseKeys) != len(forwardKeys), "expected same number of keys in reverse")

		for i := range forwardKeys {
			testutils.FailIf(t, forwardKeys[i] != reverseKeys[len(reverseKeys)-1-i], "expected reverse sorted order")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapNaturalOrder(t *testing.T) {
	f := func(keys []int, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := newIntSortedMap[int]()
		uniqueKeys := make(map[int]bool)
		var sortedExpected []int

		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
			if !uniqueKeys[keys[i]] {
				uniqueKeys[keys[i]] = true
				sortedExpected = append(sortedExpected, keys[i])
			}
		}
		slices.SortFunc(sortedExpected, cmp.Compare)

		i := 0
		for key := range m.All() {
			testutils.FailIf(t, key != sortedExpected[i], "expected keys to be strictly in ascending sorted order")
			i++
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSortedMapBigInt(t *testing.T) {
	m := datatypes.NewSortedMap[big.Int, int](datatypes.ComparatorFor(reflect.TypeOf(big.Int{})))

	nums := []int64{10, -5, 100, 0, -100}
	for _, n := range nums {
		var bi big.Int
		bi.SetInt64(n)
		m.Set(bi, int(n))
	}

	expectedOrder := []int64{-100, -5, 0, 10, 100}
	i := 0
	for k, v := range m.All() {
		testutils.FailIf(t, k.Int64() != expectedOrder[i], "big.Int order mismatch")
		testutils.FailIf(t, v != int(expectedOrder[i]), "big.Int value mismatch")
		i++
	}
	testutils.FailIf(t, i != len(expectedOrder), "wrong number of elements")
}

func TestSortedMapBigFloat(t *testing.T) {
	m := datatypes.NewSortedMap[big.Float, int](datatypes.ComparatorFor(reflect.TypeOf(big.Float{})))

	nums := []float64{10.5, -5.2, 100.1, 0.0, -100.9}
	for _, n := range nums {
		var bf big.Float
		bf.SetFloat64(n)
		m.Set(bf, int(n))
	}

	expectedOrder := []float64{-100.9, -5.2, 0.0, 10.5, 100.1}
	i := 0
	for k, v := range m.All() {
		f, _ := k.Float64()
		testutils.FailIf(t, f != expectedOrder[i], "big.Float order mismatch")
		testutils.FailIf(t, v != int(expectedOrder[i]), "big.Float value mismatch")
		i++
	}
	testutils.FailIf(t, i != len(expectedOrder), "wrong number of elements")
}
