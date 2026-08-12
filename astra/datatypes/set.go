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

package datatypes

import (
	"fmt"
	"iter"
	"reflect"
	"strings"
)

// Set represents a collection of unique elements, kept in sorted order.
type Set[T any] SortedMap[T, struct{}]

// NewSet returns a new Set for the given type T.
func NewSet[T any](elements ...T) Set[T] {
	return NewSetWithComparator[T](ComparatorFor(reflect.TypeFor[T]()), elements...)
}

// NewSetWithComparator returns a new Set with a custom Comparator.
// Prefer RegisterComparator + NewSet for user-facing types; use this when you have
// a runtime-derived comparator.
func NewSetWithComparator[T any](cmp Comparator, elements ...T) Set[T] {
	s := Set[T](NewSortedMapWithComparator[T, struct{}](cmp))
	s.AddAll(elements...)
	return s
}

func (s Set[T]) m() SortedMap[T, struct{}] {
	return SortedMap[T, struct{}](s)
}

func (s Set[T]) Add(v T) {
	s.m().Set(v, struct{}{})
}

func (s Set[T]) AddAll(elements ...T) {
	for _, e := range elements {
		s.Add(e)
	}
}

func (s Set[T]) Has(v T) bool {
	return s.m().Has(v)
}

func (s Set[T]) Delete(v T) {
	s.m().Delete(v)
}

// SetAny is an internal hook for serdes to insert entries via reflection.
func (s Set[T]) SetAny(v any, _ any) bool {
	existed := s.Has(v.(T))
	s.Add(v.(T))
	return existed
}

func (s Set[T]) Pop() (T, bool) {
	if n := s.m().First(); n != nil {
		k := n.key
		s.m().Delete(k)
		return k, true
	}
	var zero T
	return zero, false
}

func (s Set[T]) Len() int {
	return s.m().Len()
}

func (s Set[T]) Clear() {
	s.m().Clear()
}

func (s Set[T]) Clone() Set[T] {
	sm := s.m()
	if sm.sortedMap == nil {
		return Set[T]{}
	}
	clone := Set[T](NewSortedMapWithComparator[T, struct{}](sm.cmp))
	for k := range s.All() {
		clone.Add(k)
	}
	return clone
}

func (s Set[T]) Equals(other Set[T]) bool {
	if s.Len() != other.Len() {
		return false
	}
	for v := range s.All() {
		if !other.Has(v) {
			return false
		}
	}
	return true
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	union := s.Clone()
	for v := range other.All() {
		union.Add(v)
	}
	return union
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	sm := s.m()
	intersection := Set[T](NewSortedMapWithComparator[T, struct{}](sm.cmp))
	for v := range s.All() {
		if other.Has(v) {
			intersection.Add(v)
		}
	}
	return intersection
}

func (s Set[T]) Difference(other Set[T]) Set[T] {
	sm := s.m()
	difference := Set[T](NewSortedMapWithComparator[T, struct{}](sm.cmp))
	for v := range s.All() {
		if !other.Has(v) {
			difference.Add(v)
		}
	}
	return difference
}

func (s Set[T]) SymmetricDifference(other Set[T]) Set[T] {
	sm := s.m()
	sd := Set[T](NewSortedMapWithComparator[T, struct{}](sm.cmp))
	for v := range s.All() {
		if !other.Has(v) {
			sd.Add(v)
		}
	}
	for v := range other.All() {
		if !s.Has(v) {
			sd.Add(v)
		}
	}
	return sd
}

func (s Set[T]) IsSubsetOf(other Set[T]) bool {
	for v := range s.All() {
		if !other.Has(v) {
			return false
		}
	}
	return true
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, s.Len())
	for v := range s.All() {
		slice = append(slice, v)
	}
	return slice
}

func (s Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s.m().All() {
			if !yield(k) {
				return
			}
		}
	}
}

func (s Set[T]) AllRev() iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s.m().AllRev() {
			if !yield(k) {
				return
			}
		}
	}
}

func (s Set[T]) First() T {
	if n := s.m().First(); n != nil {
		return n.key
	}
	var zero T
	return zero
}

func (s Set[T]) Last() T {
	if n := s.m().Last(); n != nil {
		return n.key
	}
	var zero T
	return zero
}

func (s Set[T]) String() string {
	var sb strings.Builder
	sb.WriteString("#{")
	first := true
	for v := range s.All() {
		if !first {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%v", v)
		first = false
	}
	sb.WriteString("}")
	return sb.String()
}
