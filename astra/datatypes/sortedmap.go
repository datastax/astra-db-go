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
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

const skipListMaxLevel = 16
const skipListP = 0.25

// SortedMap is a skip list-backed map that keeps keys ordered by a Comparator.
// K doesn't need to be comparable since the Comparator handles the logic.
// All operations are O(log n) on average.
//
// Don't use the zero value; initialize it with NewSortedMap.
// It's a pointer wrapper, so it's safe to copy and pass by value.
type SortedMap[K any, V any] struct {
	*sortedMap[K, V]
}

// IMPORTANT: The field ordering of kType, vType, and cmp is extremely important:
//   - kType and vType must be at the start of the struct to ensure that no padding is allocated for them
//   - cmp is the first non-zero-size field, and must be at index 2 — both of these are relied on for reflection and pointer math
//
// `serdes/maps.go` must be updated if either of the above two invariants are changed.
//
// The idea is to treat sortedMap similar to how hmap is used in Go's built-in map implementation,
// where the public type is a thin wrapper around an internal struct that contains the actual data and
// implementation details. This helps enforce the requirement of having the actual implementation
// fields behind a pointer so people don't try to use it as a value type.
type sortedMap[K any, V any] struct {
	kType [0]K
	vType [0]V
	cmp   Comparator
	head  *SortedMapNode[K, V]
	len   int
}

// SortedMapNode is a node in the map, exposed for direct traversal via First/Last.
type SortedMapNode[K any, V any] struct {
	key   K
	value V
	next  [skipListMaxLevel]*SortedMapNode[K, V]
	level int
}

func (n *SortedMapNode[K, V]) Key() K                     { return n.key }
func (n *SortedMapNode[K, V]) Value() V                   { return n.value }
func (n *SortedMapNode[K, V]) Next() *SortedMapNode[K, V] { return n.next[0] }

// NewSortedMap returns an empty map ordered by a default Comparator for K.
// Use NewSortedMapWithComparator for custom ordering.
func NewSortedMap[K any, V any]() SortedMap[K, V] {
	return NewSortedMapWithComparator[K, V](ComparatorFor(reflect.TypeFor[K]()))
}

// NewSortedMapWithComparator returns an empty map ordered by the given Comparator.
func NewSortedMapWithComparator[K any, V any](cmp Comparator) SortedMap[K, V] {
	return SortedMap[K, V]{&sortedMap[K, V]{cmp: cmp, head: &SortedMapNode[K, V]{}}}
}

// Get returns the value for key, or false if it's missing.
func (m SortedMap[K, V]) Get(key K) (v V, found bool) {
	if m.sortedMap == nil {
		return
	}
	cur := m.head
	for i := skipListMaxLevel - 1; i >= 0; i-- {
		for cur.next[i] != nil && m.cmp(unsafe.Pointer(&cur.next[i].key), unsafe.Pointer(&key)) < 0 {
			cur = cur.next[i]
		}
	}
	cur = cur.next[0]
	if cur != nil && m.cmp(unsafe.Pointer(&cur.key), unsafe.Pointer(&key)) == 0 {
		return cur.value, true
	}
	return
}

// GetOrDefault returns the value for key, or defaultValue if it's missing.
func (m SortedMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	v, ok := m.Get(key)
	if ok {
		return v
	}
	return defaultValue
}

// Set adds or updates a key-value pair.
// Returns the old value and true if the key existed, or zero/false if it's a fresh insert.
// Panics if the map is nil (use NewSortedMap).
func (m SortedMap[K, V]) Set(key K, value V) (V, bool) {
	if m.sortedMap == nil {
		panic("assignment to entry in nil SortedMap")
	}

	var update [skipListMaxLevel]*SortedMapNode[K, V]
	cur := m.head
	for i := skipListMaxLevel - 1; i >= 0; i-- {
		for cur.next[i] != nil && m.cmp(unsafe.Pointer(&cur.next[i].key), unsafe.Pointer(&key)) < 0 {
			cur = cur.next[i]
		}
		update[i] = cur
	}

	cur = cur.next[0]
	if cur != nil && m.cmp(unsafe.Pointer(&cur.key), unsafe.Pointer(&key)) == 0 {
		old := cur.value
		cur.value = value
		return old, true
	}

	level := m.randomLevel()
	n := &SortedMapNode[K, V]{key: key, value: value, level: level}
	for i := 0; i < level; i++ {
		n.next[i] = update[i].next[i]
		update[i].next[i] = n
	}
	m.len++

	var zero V
	return zero, false
}

// SetAny is an internal hook for serdes to insert entries via reflection.
func (m SortedMap[K, V]) SetAny(key any, value any) bool {
	_, existed := m.Set(key.(K), value.(V))
	return existed
}

// Delete removes a key and returns its value.
func (m SortedMap[K, V]) Delete(key K) (v V, found bool) {
	if m.sortedMap == nil {
		return
	}
	var update [skipListMaxLevel]*SortedMapNode[K, V]
	cur := m.head
	for i := skipListMaxLevel - 1; i >= 0; i-- {
		for cur.next[i] != nil && m.cmp(unsafe.Pointer(&cur.next[i].key), unsafe.Pointer(&key)) < 0 {
			cur = cur.next[i]
		}
		update[i] = cur
	}

	cur = cur.next[0]
	if cur == nil || m.cmp(unsafe.Pointer(&cur.key), unsafe.Pointer(&key)) != 0 {
		return
	}

	for i := 0; i < cur.level; i++ {
		update[i].next[i] = cur.next[i]
	}
	m.len--

	return cur.value, true
}

// Has reports if a key is present.
func (m SortedMap[K, V]) Has(key K) bool {
	_, ok := m.Get(key)
	return ok
}

// Len is the number of entries in the map.
func (m SortedMap[K, V]) Len() int {
	if m.sortedMap == nil {
		return 0
	}
	return m.len
}

// First returns the first (smallest) node, or nil if empty.
func (m SortedMap[K, V]) First() *SortedMapNode[K, V] {
	if m.sortedMap == nil {
		return nil
	}
	return m.head.next[0]
}

// Last returns the last (largest) node, or nil if empty.
func (m SortedMap[K, V]) Last() *SortedMapNode[K, V] {
	if m.sortedMap == nil {
		return nil
	}
	cur := m.head
	for i := skipListMaxLevel - 1; i >= 0; i-- {
		for cur.next[i] != nil {
			cur = cur.next[i]
		}
	}
	if cur == m.head {
		return nil
	}
	return cur
}

// Clear removes everything from the map.
func (m SortedMap[K, V]) Clear() {
	if m.sortedMap == nil {
		return
	}
	for i := range m.head.next {
		m.head.next[i] = nil
	}
	m.len = 0
}

// All iterates entries in ascending key order.
// Supports bounds: All() is full, All(lo) is lo <= k, All(lo, hi) is lo <= k <= hi.
func (m SortedMap[K, V]) All(bounds ...K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m.sortedMap == nil {
			return
		}
		if len(bounds) > 2 {
			panic("SortedMap.All: too many bounds")
		}

		cur := m.head.next[0]
		if len(bounds) >= 1 {
			cur = m.head
			for i := skipListMaxLevel - 1; i >= 0; i-- {
				for cur.next[i] != nil && m.cmp(unsafe.Pointer(&cur.next[i].key), unsafe.Pointer(&bounds[0])) < 0 {
					cur = cur.next[i]
				}
			}
			cur = cur.next[0]
		}

		for cur != nil {
			next := cur.next[0]
			if len(bounds) == 2 && m.cmp(unsafe.Pointer(&cur.key), unsafe.Pointer(&bounds[1])) > 0 {
				return
			}
			if !yield(cur.key, cur.value) {
				return
			}
			cur = next
		}
	}
}

// AllRev iterates entries in descending key order.
// Supports bounds: AllRev() is full, AllRev(lo) is lo <= k, AllRev(lo, hi) is lo <= k <= hi.
//
// O(n) time and space: skip lists don't support backward traversal, so we
// collect matching nodes into a slice and yield them in reverse.
func (m SortedMap[K, V]) AllRev(bounds ...K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m.sortedMap == nil {
			return
		}
		if len(bounds) > 2 {
			panic("SortedMap.AllRev: too many bounds")
		}

		cur := m.head.next[0]
		if len(bounds) >= 1 {
			cur = m.head
			for i := skipListMaxLevel - 1; i >= 0; i-- {
				for cur.next[i] != nil && m.cmp(unsafe.Pointer(&cur.next[i].key), unsafe.Pointer(&bounds[0])) < 0 {
					cur = cur.next[i]
				}
			}
			cur = cur.next[0]
		}

		var nodes []*SortedMapNode[K, V]
		for ; cur != nil; cur = cur.next[0] {
			if len(bounds) == 2 && m.cmp(unsafe.Pointer(&cur.key), unsafe.Pointer(&bounds[1])) > 0 {
				break
			}
			nodes = append(nodes, cur)
		}

		for i := len(nodes) - 1; i >= 0; i-- {
			if !yield(nodes[i].key, nodes[i].value) {
				return
			}
		}
	}
}

func (m SortedMap[K, V]) Equals(other SortedMap[K, V], valCmp Comparator) bool {
	if m.Len() != other.Len() {
		return false
	}
	for k, v1 := range m.All() {
		v2, ok := other.Get(k)
		if !ok || valCmp(unsafe.Pointer(&v1), unsafe.Pointer(&v2)) != 0 {
			return false
		}
	}
	return true
}

func (m SortedMap[K, V]) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	first := true
	if m.sortedMap != nil {
		for n := m.head.next[0]; n != nil; n = n.next[0] {
			if !first {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%v: %v", n.key, n.value)
			first = false
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (m *sortedMap[K, V]) randomLevel() int {
	level := 1
	for level < skipListMaxLevel && rand.Float64() < skipListP {
		level++
	}
	return level
}

// -- Comparators --

// Comparator compares two values via pointers.
// Return <0 for a < b, 0 for a == b, and >0 for a > b.
type Comparator func(a, b unsafe.Pointer) int

// Comparable is for types that can compare themselves to another value of the same type.
// The other arg will be the same type as the receiver.
type Comparable interface {
	CompareTo(other any) int
}

// ComparatorFor returns a Comparator for common types like primitives, time.Time, or Comparable.
//
// reflect.Type is only used once at construction to pick the logic; the returned closure
// uses unsafe.Pointer casts and is allocation-free.
//
// The Comparable branch is the exception — it uses reflect.NewAt and .Interface() which allocates
// on each call, but Comparable keys are niche enough that it shouldn't matter.
func ComparatorFor(t reflect.Type) Comparator {
	if t.Implements(comparableType) {
		return func(a, b unsafe.Pointer) int {
			av := reflect.NewAt(t, a).Elem().Interface().(Comparable) // maybe replace w/ i-face trick if we need more speed
			bv := reflect.NewAt(t, b).Elem().Interface()
			return av.CompareTo(bv)
		}
	}

	switch t {
	case timeType:
		return func(a, b unsafe.Pointer) int {
			return (*time.Time)(a).Compare(*(*time.Time)(b))
		}
	case bigIntPtrType:
		return func(a, b unsafe.Pointer) int {
			return (*(**big.Int)(a)).Cmp(*(**big.Int)(b))
		}
	case bigFloatPtrType:
		return func(a, b unsafe.Pointer) int {
			return (*(**big.Float)(a)).Cmp(*(**big.Float)(b))
		}
	case jsonMsgType:
		return func(a, b unsafe.Pointer) int { return strings.Compare(*(*string)(a), *(*string)(b)) }
	}

	switch t.Kind() {
	case reflect.Int:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int)(a), *(*int)(b)) }
	case reflect.Int8:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int8)(a), *(*int8)(b)) }
	case reflect.Int16:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int16)(a), *(*int16)(b)) }
	case reflect.Int32:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int32)(a), *(*int32)(b)) }
	case reflect.Int64:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int64)(a), *(*int64)(b)) }
	case reflect.Uint, reflect.Uintptr:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint)(a), *(*uint)(b)) }
	case reflect.Uint8:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint8)(a), *(*uint8)(b)) }
	case reflect.Uint16:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint16)(a), *(*uint16)(b)) }
	case reflect.Uint32:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint32)(a), *(*uint32)(b)) }
	case reflect.Uint64:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint64)(a), *(*uint64)(b)) }
	case reflect.Float32:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*float32)(a), *(*float32)(b)) }
	case reflect.Float64:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*float64)(a), *(*float64)(b)) }
	case reflect.String:
		return func(a, b unsafe.Pointer) int { return strings.Compare(*(*string)(a), *(*string)(b)) }
	}

	panic(fmt.Sprintf("ComparatorFor: no comparator available for type %s", t))
}

var (
	timeType        = reflect.TypeOf(time.Time{})
	bigIntPtrType   = reflect.TypeFor[*big.Int]()
	bigFloatPtrType = reflect.TypeFor[*big.Float]()
	jsonMsgType     = reflect.TypeFor[json.RawMessage]()
	comparableType  = reflect.TypeOf((*interface{ CompareTo(any) int })(nil)).Elem()
)
