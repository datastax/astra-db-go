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
	"maps"
)

type Set[T comparable] struct {
	// IMPORTANT: The field ordering of tType and data is extremely important:
	// - tType is at the start of the struct to ensure that no extra padding is allocated for it
	// - data is the first non-zero-size field, and must be at index 1 – both of these are relied on for reflection and pointer math
	//
	// `serdes/encode.go` must be updated of either of the above two invariants are changed.
	//
	// The idea is to treat data similar to how hmap is used in Go's built-in map implementation,
	// where the public type (Set) is a thin wrapper around an internal struct that contains the actual data and
	// implementation details, to help make the public API a little bit cleaner.
	//
	// tType simply preserves type information for serdes purposes, and does nothing else
	tType [0]T

	// data is used similar to how HashSets are implemented in Rust, where it's really just a HashMap<T, ()>
	data map[T]struct{}
}

func NewSet[T comparable](elements ...T) Set[T] {
	s := Set[T]{data: make(map[T]struct{}, len(elements))}
	for _, e := range elements {
		s.Add(e)
	}
	return s
}

func NewSetWithCapacity[T comparable](capacity int) Set[T] {
	if capacity < 0 {
		panic("NewSetWithCapacity called with negative capacity")
	}
	return Set[T]{data: make(map[T]struct{}, capacity)}
}

func (s Set[T]) Add(v T) {
	s.data[v] = struct{}{}
}

func (s Set[T]) Has(v T) bool {
	_, ok := s.data[v]
	return ok
}

func (s Set[T]) Delete(v T) {
	delete(s.data, v)
}

func (s Set[T]) Pop() (T, bool) {
	for v := range s.data {
		s.Delete(v)
		return v, true
	}
	var zero T
	return zero, false
}

func (s Set[T]) Len() int {
	return len(s.data)
}

func (s Set[T]) Clear() {
	clear(s.data)
}

func (s Set[T]) Clone() Set[T] {
	return Set[T]{data: maps.Clone(s.data)}
}

func (s Set[T]) Equals(other Set[T]) bool {
	if s.Len() != other.Len() {
		return false
	}
	for v := range s.data {
		if !other.Has(v) {
			return false
		}
	}
	return true
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	union := s.Clone()
	for v := range other.data {
		union.Add(v)
	}
	return union
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	intersection := NewSet[T]()
	for v := range s.data {
		if other.Has(v) {
			intersection.Add(v)
		}
	}
	return intersection
}

func (s Set[T]) Difference(other Set[T]) Set[T] {
	difference := NewSet[T]()
	for v := range s.data {
		if !other.Has(v) {
			difference.Add(v)
		}
	}
	return difference
}

func (s Set[T]) SymmetricDifference(other Set[T]) Set[T] {
	symmetricDifference := NewSet[T]()
	for v := range s.data {
		if !other.Has(v) {
			symmetricDifference.Add(v)
		}
	}
	for v := range other.data {
		if !s.Has(v) {
			symmetricDifference.Add(v)
		}
	}
	return symmetricDifference
}

func (s Set[T]) IsSubsetOf(other Set[T]) bool {
	for v := range s.data {
		if !other.Has(v) {
			return false
		}
	}
	return true
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, s.Len())
	for v := range s.data {
		slice = append(slice, v)
	}
	return slice
}

func (s Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s.data {
			if !yield(v) {
				return
			}
		}
	}
}

func (s Set[T]) String() string {
	str := "#{"
	first := true
	for v := range s.data {
		if !first {
			str += ", "
		}
		str += fmt.Sprintf("%v", v)
		first = false
	}
	str += "}"
	return str
}
