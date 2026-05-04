package serdes

import (
	"reflect"
	"sort"
)

type mapIter struct {
	m        reflect.Value
	keys     []reflect.Value
	iter     *reflect.MapIter
	index    int
	isSorted bool
}

type comparator = func(i, j reflect.Value) bool

func newMapIterMaker(t reflect.Type, trySort bool) func(m reflect.Value) *mapIter {
	var cmp comparator
	if trySort {
		cmp = mkComparator(t.Key().Kind())
	}

	return func(m reflect.Value) *mapIter {
		wrapper := &mapIter{m: m, index: -1}

		if trySort {
			wrapper.keys = m.MapKeys()

			if len(wrapper.keys) > 1 {
				sort.Slice(wrapper.keys, func(i, j int) bool {
					return cmp(wrapper.keys[i], wrapper.keys[j])
				})
			}

			wrapper.isSorted = true
			return wrapper
		}

		wrapper.iter = m.MapRange()
		return wrapper
	}
}

// mkComparator returns the logic once so the loop doesn't have to switch
// TODO any more comparators???
func mkComparator(k reflect.Kind) comparator {
	switch {
	case k == reflect.String:
		return func(i, j reflect.Value) bool { return i.String() < j.String() }
	case k >= reflect.Int && k <= reflect.Int64:
		return func(i, j reflect.Value) bool { return i.Int() < j.Int() }
	case k >= reflect.Uint && k <= reflect.Uintptr:
		return func(i, j reflect.Value) bool { return i.Uint() < j.Uint() }
	case k == reflect.Float32 || k == reflect.Float64:
		return func(i, j reflect.Value) bool { return i.Float() < j.Float() }
	default:
		return func(i, j reflect.Value) bool { return false }
	}
}

func (u *mapIter) Next() bool {
	if u.isSorted {
		u.index++
		return u.index < len(u.keys)
	}
	return u.iter.Next()
}

func (u *mapIter) Key() reflect.Value {
	if u.isSorted {
		return u.keys[u.index]
	}
	return u.iter.Key()
}

func (u *mapIter) Value() reflect.Value {
	if u.isSorted {
		return u.m.MapIndex(u.keys[u.index])
	}
	return u.iter.Value()
}
