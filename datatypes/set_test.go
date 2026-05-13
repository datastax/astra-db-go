package datatypes_test

import (
	"slices"
	"testing"
	"testing/quick"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/internal/testutils"
)

func TestSetConstructors_NewSet(t *testing.T) {
	f := func(elements []string) bool {
		s := datatypes.NewSet(elements...)
		unique := make(map[string]struct{})
		for _, e := range elements {
			unique[e] = struct{}{}
		}
		return s.Len() == len(unique)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetConstructors_NewSetWithCapacity(t *testing.T) {
	f := func(capacity int) bool {
		if capacity < 0 {
			var panicked bool
			func() {
				defer func() {
					panicked = recover() != nil
				}()
				datatypes.NewSetWithCapacity[int](capacity)
			}()

			testutils.FailIf(t, !panicked, "expected NewSetWithCapacity to panic when given negative capacity")
		} else {
			datatypes.NewSetWithCapacity[int](capacity)
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetBasics(t *testing.T) {
	f := func(v int, others []int) bool {
		s := datatypes.NewSet[int]()

		s.Add(v)
		for _, val := range others {
			s.Add(val)
		}

		testutils.FailIf(t, !s.Has(v), "expected set to have value after adding it")
		s.Delete(v)
		testutils.FailIf(t, s.Has(v), "expected set to not have value after deleting it")
		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Errorf("Property failed: %v", err)
	}
}

func TestSetPop(t *testing.T) {
	type comparableStruct struct {
		id   byte
		name string
	}

	f := func(ids []byte, names []string) bool {
		s := datatypes.NewSet[comparableStruct]()

		if len(ids) == 0 || len(names) == 0 {
			return true
		}

		for i := range ids {
			s.Add(comparableStruct{id: ids[i], name: names[i%len(names)]})
		}

		seen := make(map[comparableStruct]struct{})
		size := s.Len()

		for i := 0; i < size; i++ {
			found, ok := s.Pop()
			_, seenHas := seen[found]

			testutils.FailIf(t, !ok, "expected s.Pop() to return ok == true")
			testutils.FailIf(t, seenHas, "expected popped value to not have been seen before")
			testutils.FailIf(t, s.Has(found), "expected popped value to no longer be in the set")

			seen[found] = struct{}{}
		}

		testutils.FailIf(t, s.Len() != 0, "expected set to be empty after popping all elements")

		for range 5 {
			found, ok := s.Pop()
			testutils.FailIf(t, ok, "expected s.Pop() to return ok == false when popping from empty set")
			testutils.FailIf(t, found != (comparableStruct{}), "expected popped value to be zero value when popping from empty set")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Errorf("Property failed: %v", err)
	}
}

func TestSetLen(t *testing.T) {
	f := func(elements []int) bool {
		if slices.Contains(elements, -999999) {
			return true
		}

		s := datatypes.NewSet(elements...)
		unique := make(map[int]struct{})
		for _, e := range elements {
			unique[e] = struct{}{}
		}

		testutils.FailIf(t, s.Len() != len(unique), "expected set length to match number of unique elements")

		if len(elements) > 0 {
			initialLen := s.Len()
			s.Add(elements[0])
			testutils.FailIf(t, s.Len() != initialLen, "expected length to remain same after adding duplicate")

			newVal := -999999
			s.Add(newVal)
			testutils.FailIf(t, s.Len() != initialLen+1, "expected length to increase after adding new element")

			s.Delete(newVal)
			testutils.FailIf(t, s.Len() != initialLen, "expected length to decrease after deleting element")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetClear(t *testing.T) {
	f := func(elements []rune) bool {
		if len(elements) == 0 {
			return true
		}
		s := datatypes.NewSet(elements...)
		testutils.FailIf(t, s.Len() == 0, "expected set to have elements after construction")
		s.Clear()
		testutils.FailIf(t, s.Len() != 0, "expected set to be empty after clear")
		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetClone(t *testing.T) {
	f := func(elements []string) bool {
		if slices.Contains(elements, "new element") {
			return true
		}

		s := datatypes.NewSet(elements...)
		clone := s.Clone()

		testutils.FailIf(t, s.Len() != clone.Len(), "expected clone to have same length as original")
		for _, e := range elements {
			testutils.FailIf(t, !s.Has(e), "expected original set to have element")
			testutils.FailIf(t, !clone.Has(e), "expected cloned set to have element")
		}

		s.Add("new element")
		testutils.FailIf(t, clone.Has("new element"), "expected clone to not have new element added to original")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetEquals(t *testing.T) {
	f := func(elements []string) bool {
		if slices.Contains(elements, "new element") {
			return true
		}

		s1 := datatypes.NewSet(elements...)
		s2 := datatypes.NewSet(elements...)

		testutils.FailIf(t, !s1.Equals(s2), "expected sets with same elements to be equal")

		if len(elements) > 0 {
			s2.Add("new element")
			testutils.FailIf(t, s1.Equals(s2), "expected sets with different elements to not be equal")
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetUnion(t *testing.T) {
	f := func(elements1 []int, elements2 []int) bool {
		s1 := datatypes.NewSet(elements1...)
		s2 := datatypes.NewSet(elements2...)

		union := s1.Union(s2)

		for _, e := range elements1 {
			testutils.FailIf(t, !union.Has(e), "expected union to contain element from first set")
		}
		for _, e := range elements2 {
			testutils.FailIf(t, !union.Has(e), "expected union to contain element from second set")
		}

		testutils.FailIf(t, s1.Len() != datatypes.NewSet(elements1...).Len(), "expected original set to be unchanged")
		testutils.FailIf(t, s2.Len() != datatypes.NewSet(elements2...).Len(), "expected original set to be unchanged")

		testutils.FailIf(t, union.Len() > s1.Len()+s2.Len(), "expected union size to be at most sum of both sets")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetIntersection(t *testing.T) {
	f := func(elements1 []int, elements2 []int) bool {
		s1 := datatypes.NewSet(elements1...)
		s2 := datatypes.NewSet(elements2...)

		intersection := s1.Intersection(s2)

		for v := range intersection.All() {
			testutils.FailIf(t, !s1.Has(v), "expected intersection element to be in first set")
			testutils.FailIf(t, !s2.Has(v), "expected intersection element to be in second set")
		}

		for _, e := range elements1 {
			if s2.Has(e) {
				testutils.FailIf(t, !intersection.Has(e), "expected common element to be in intersection")
			}
		}

		testutils.FailIf(t, s1.Len() != datatypes.NewSet(elements1...).Len(), "expected original set to be unchanged")
		testutils.FailIf(t, s2.Len() != datatypes.NewSet(elements2...).Len(), "expected original set to be unchanged")

		testutils.FailIf(t, intersection.Len() > min(s1.Len(), s2.Len()), "expected intersection size to be at most size of smaller set")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetDifference(t *testing.T) {
	f := func(elements1 []int, elements2 []int) bool {
		s1 := datatypes.NewSet(elements1...)
		s2 := datatypes.NewSet(elements2...)

		difference := s1.Difference(s2)

		for v := range difference.All() {
			testutils.FailIf(t, !s1.Has(v), "expected difference element to be in first set")
			testutils.FailIf(t, s2.Has(v), "expected difference element to not be in second set")
		}

		for _, e := range elements1 {
			if !s2.Has(e) {
				testutils.FailIf(t, !difference.Has(e), "expected element in s1 but not s2 to be in difference")
			}
		}

		for _, e := range elements1 {
			if s2.Has(e) {
				testutils.FailIf(t, difference.Has(e), "expected element in both sets to not be in difference")
			}
		}

		testutils.FailIf(t, s1.Len() != datatypes.NewSet(elements1...).Len(), "expected original set to be unchanged")
		testutils.FailIf(t, s2.Len() != datatypes.NewSet(elements2...).Len(), "expected original set to be unchanged")

		testutils.FailIf(t, difference.Len() > s1.Len(), "expected difference size to be at most size of first set")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetSymmetricDifference(t *testing.T) {
	f := func(elements1 []int, elements2 []int) bool {
		s1 := datatypes.NewSet(elements1...)
		s2 := datatypes.NewSet(elements2...)

		symDiff := s1.SymmetricDifference(s2)

		for v := range symDiff.All() {
			inS1 := s1.Has(v)
			inS2 := s2.Has(v)
			testutils.FailIf(t, inS1 && inS2, "expected symmetric difference element to not be in both sets")
			testutils.FailIf(t, !inS1 && !inS2, "expected symmetric difference element to be in at least one set")
		}

		for _, e := range elements1 {
			if !s2.Has(e) {
				testutils.FailIf(t, !symDiff.Has(e), "expected element only in s1 to be in symmetric difference")
			}
		}
		for _, e := range elements2 {
			if !s1.Has(e) {
				testutils.FailIf(t, !symDiff.Has(e), "expected element only in s2 to be in symmetric difference")
			}
		}

		for _, e := range elements1 {
			if s2.Has(e) {
				testutils.FailIf(t, symDiff.Has(e), "expected element in both sets to not be in symmetric difference")
			}
		}

		testutils.FailIf(t, s1.Len() != datatypes.NewSet(elements1...).Len(), "expected original set to be unchanged")
		testutils.FailIf(t, s2.Len() != datatypes.NewSet(elements2...).Len(), "expected original set to be unchanged")

		symDiff2 := s2.SymmetricDifference(s1)
		testutils.FailIf(t, !symDiff.Equals(symDiff2), "expected symmetric difference to be commutative")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetIsSubsetOf(t *testing.T) {
	f := func(elements1 []int, elements2 []int) bool {
		s1 := datatypes.NewSet(elements1...)
		s2 := datatypes.NewSet(elements2...)

		testutils.FailIf(t, !s1.IsSubsetOf(s1), "expected set to be subset of itself")
		testutils.FailIf(t, !s2.IsSubsetOf(s2), "expected set to be subset of itself")

		empty := datatypes.NewSet[int]()
		testutils.FailIf(t, !empty.IsSubsetOf(s1), "expected empty set to be subset of any set")
		testutils.FailIf(t, !empty.IsSubsetOf(s2), "expected empty set to be subset of any set")

		if s1.IsSubsetOf(s2) {
			for _, e := range elements1 {
				testutils.FailIf(t, !s2.Has(e), "expected superset to contain all subset elements")
			}
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}

	subset := datatypes.NewSet(1, 2, 3)
	superset := datatypes.NewSet(1, 2, 3, 4, 5)
	testutils.FailIf(t, !subset.IsSubsetOf(superset), "expected subset to be subset of superset")
	testutils.FailIf(t, superset.IsSubsetOf(subset), "expected superset to not be subset of subset")
}

func TestSetToSlice(t *testing.T) {
	f := func(elements []int) bool {
		s := datatypes.NewSet(elements...)
		slice := s.ToSlice()

		testutils.FailIf(t, len(slice) != s.Len(), "expected slice length to match set length")

		for _, e := range slice {
			testutils.FailIf(t, !s.Has(e), "expected slice element to be in set")
		}

		sliceMap := make(map[int]struct{})
		for _, e := range slice {
			sliceMap[e] = struct{}{}
		}
		for v := range s.All() {
			_, ok := sliceMap[v]
			testutils.FailIf(t, !ok, "expected set element to be in slice")
		}

		empty := datatypes.NewSet[int]()
		emptySlice := empty.ToSlice()
		testutils.FailIf(t, len(emptySlice) != 0, "expected empty set to produce empty slice")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestSetAll(t *testing.T) {
	f := func(elements []string) bool {
		s := datatypes.NewSet(elements...)

		count := 0
		seen := make(map[string]struct{})
		for v := range s.All() {
			count++
			_, alreadySeen := seen[v]
			testutils.FailIf(t, alreadySeen, "expected iterator to not yield duplicate elements")
			testutils.FailIf(t, !s.Has(v), "expected iterator to only yield elements in set")
			seen[v] = struct{}{}
		}

		testutils.FailIf(t, count != s.Len(), "expected iterator to yield all elements exactly once")

		if s.Len() > 0 {
			terminatedEarly := false
			iterCount := 0
			for range s.All() {
				iterCount++
				if iterCount >= 1 {
					terminatedEarly = true
					break
				}
			}
			testutils.FailIf(t, !terminatedEarly && s.Len() > 1, "expected to be able to break from iterator early")
		}

		empty := datatypes.NewSet[string]()
		emptyCount := 0
		for range empty.All() {
			emptyCount++
		}
		testutils.FailIf(t, emptyCount != 0, "expected empty set iterator to yield no elements")

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
