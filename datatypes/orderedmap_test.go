package datatypes_test

import (
	"testing"
	"testing/quick"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/internal/testutils"
)

//goland:noinspection GoMaybeNil
func TestOrderedMapConstructors_NewOrderedMap(t *testing.T) {
	m := datatypes.NewOrderedMap[string, int]()
	testutils.FailIf(t, m.Len() != 0, "expected new map to be empty")
}

func TestOrderedMapConstructors_NewOrderedMapWithCapacity(t *testing.T) {
	f := func(capacity uint8) bool {
		m := datatypes.NewOrderedMapWithCapacity[string, int](int(capacity))
		testutils.FailIf(t, m.Len() != 0, "expected new map to be empty")
		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapSetAndGet(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()

		for i := range keys {
			val := values[i%len(values)]
			oldVal, isNew := m.Set(keys[i], val)

			if isNew {
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

func TestOrderedMapSetUpdate(t *testing.T) {
	f := func(key string, val1 int, val2 int) bool {
		m := datatypes.NewOrderedMap[string, int]()

		oldVal1, isNew1 := m.Set(key, val1)
		testutils.FailIf(t, !isNew1, "expected first set to be new")
		testutils.FailIf(t, oldVal1 != 0, "expected zero value for new key")

		oldVal2, isNew2 := m.Set(key, val2)
		testutils.FailIf(t, isNew2, "expected second set to be update")
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

func TestOrderedMapGetOrDefault(t *testing.T) {
	f := func(key string, value int, defaultValue int) bool {
		m := datatypes.NewOrderedMap[string, int]()

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

func TestOrderedMapDelete(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
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

func TestOrderedMapHas(t *testing.T) {
	f := func(key string, value int) bool {
		m := datatypes.NewOrderedMap[string, int]()

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

func TestOrderedMapLen(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
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

func TestOrderedMapClear(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
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

func TestOrderedMapFirstLast(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()

		firstNode := m.First()
		testutils.FailIf(t, firstNode != nil, "expected First to return nil for empty map")

		lastNode := m.Last()
		testutils.FailIf(t, lastNode != nil, "expected Last to return nil for empty map")

		expected := make(map[string]int)
		var firstUniqueKey string
		var lastUniqueKey string
		firstSet := false

		for i := range keys {
			val := values[i%len(values)]
			if _, exists := expected[keys[i]]; !exists {
				if !firstSet {
					firstUniqueKey = keys[i]
					firstSet = true
				}
				lastUniqueKey = keys[i]
			}
			m.Set(keys[i], val)
			expected[keys[i]] = val
		}

		firstNode = m.First()
		testutils.FailIf(t, firstNode == nil, "expected First to return non-nil for non-empty map")

		// Verify first element by iterating
		i := 0
		for key, val := range m.All() {
			if i == 0 {
				testutils.FailIf(t, key != firstUniqueKey, "expected First to return first inserted key")
				testutils.FailIf(t, val != expected[key], "expected First to return correct value")
			}
			i++
		}

		lastNode = m.Last()
		testutils.FailIf(t, lastNode == nil, "expected Last to return non-nil for non-empty map")

		// Verify last element by iterating
		i = 0
		var lastSeenKey string
		var lastSeenVal int
		for key, val := range m.All() {
			lastSeenKey = key
			lastSeenVal = val
			i++
		}
		testutils.FailIf(t, lastSeenKey != lastUniqueKey, "expected Last to return last unique inserted key")
		testutils.FailIf(t, lastSeenVal != expected[lastSeenKey], "expected Last to return correct value")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapClone(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		clone := m.Clone()
		testutils.FailIf(t, clone.Len() != m.Len(), "expected clone to have same length")

		for key, val := range m.All() {
			cloneVal, found := clone.Get(key)
			testutils.FailIf(t, !found, "expected clone to have all keys from original")
			testutils.FailIf(t, cloneVal != val, "expected clone to have same values")
		}

		origKeys := make([]string, 0, m.Len())
		for origKey := range m.Keys() {
			origKeys = append(origKeys, origKey)
		}

		cloneKeys := make([]string, 0, clone.Len())
		for cloneKey := range clone.Keys() {
			cloneKeys = append(cloneKeys, cloneKey)
		}

		for i := range origKeys {
			testutils.FailIf(t, origKeys[i] != cloneKeys[i], "expected clone to preserve order")
		}

		m.Set("new-key", 999)
		testutils.FailIf(t, clone.Has("new-key"), "expected clone to be independent")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapToMap(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		regularMap := m.ToMap()
		testutils.FailIf(t, len(regularMap) != m.Len(), "expected map to have same length")

		for key, val := range m.All() {
			mapVal, found := regularMap[key]
			testutils.FailIf(t, !found, "expected map to have all keys")
			testutils.FailIf(t, mapVal != val, "expected map to have same values")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapAll(t *testing.T) {
	f := func(keys []byte, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[byte, int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		count := 0
		seen := make(map[byte]struct{})
		for key, val := range m.All() {
			count++
			_, alreadySeen := seen[key]
			testutils.FailIf(t, alreadySeen, "expected iterator to not yield duplicate keys")

			gotVal, found := m.Get(key)
			testutils.FailIf(t, !found, "expected iterator to only yield keys in map")
			testutils.FailIf(t, gotVal != val, "expected iterator to yield correct values")

			seen[key] = struct{}{}
		}

		testutils.FailIf(t, count != m.Len(), "expected iterator to yield all elements")

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

func TestOrderedMapAllRev(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		var forwardKeys []string
		for key := range m.Keys() {
			forwardKeys = append(forwardKeys, key)
		}

		var reverseKeys []string
		for key, _ := range m.AllRev() {
			reverseKeys = append(reverseKeys, key)
		}

		testutils.FailIf(t, len(reverseKeys) != len(forwardKeys), "expected same number of keys in reverse")

		for i := range forwardKeys {
			testutils.FailIf(t, forwardKeys[i] != reverseKeys[len(reverseKeys)-1-i], "expected reverse order")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapKeys(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		count := 0
		seen := make(map[string]struct{})
		for key := range m.Keys() {
			count++
			_, alreadySeen := seen[key]
			testutils.FailIf(t, alreadySeen, "expected Keys iterator to not yield duplicate keys")
			testutils.FailIf(t, !m.Has(key), "expected Keys iterator to only yield keys in map")
			seen[key] = struct{}{}
		}

		testutils.FailIf(t, count != m.Len(), "expected Keys iterator to yield all keys")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapValues(t *testing.T) {
	f := func(keys []string, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[string, int]()
		for i := range keys {
			m.Set(keys[i], values[i%len(values)])
		}

		count := 0
		var collectedValues []int
		for val := range m.Values() {
			count++
			collectedValues = append(collectedValues, val)
		}

		testutils.FailIf(t, count != m.Len(), "expected Values iterator to yield all values")

		i := 0
		for _, val := range m.All() {
			testutils.FailIf(t, collectedValues[i] != val, "expected Values to match All order")
			i++
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestOrderedMapInsertionOrder(t *testing.T) {
	f := func(keys []int, values []int) bool {
		if len(keys) == 0 || len(values) == 0 {
			return true
		}

		m := datatypes.NewOrderedMap[int, int]()
		var insertionOrder []int
		seen := make(map[int]struct{})

		for i := range keys {
			if _, exists := seen[keys[i]]; !exists {
				insertionOrder = append(insertionOrder, keys[i])
				seen[keys[i]] = struct{}{}
			}
			m.Set(keys[i], values[i%len(values)])
		}

		i := 0
		for key := range m.Keys() {
			testutils.FailIf(t, key != insertionOrder[i], "expected keys to be in insertion order")
			i++
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
