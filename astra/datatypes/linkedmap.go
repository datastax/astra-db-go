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
	"strings"
)

// LinkedMap is a hash map that preserves insertion order.
// Don't use the zero value; initialize it with NewLinkedMap or NewLinkedMapWithCapacity so it doesn't panic.
// It's a pointer wrapper, so it's safe to copy and pass by value.
type LinkedMap[K comparable, V any] struct {
	*linkedMap[K, V]
}

// IMPORTANT: The field ordering of kType, vType, and data is extremely important:
//   - kType and vType must be at the start of the struct to ensure that no padding is allocated for them
//   - data is the first non-zero-size field, and must be at index 2 — both of these are relied on for reflection and pointer math
//
// `serdes/maps.go` must be updated if either of the above two invariants are changed.
//
// The idea is to treat linkedMap similar to how hmap is used in Go's built-in map implementation,
// where the public type is a thin wrapper around an internal struct that contains the actual data and
// implementation details. This helps enforce the requirement of having the actual implementation
// fields behind a pointer so people don't try to use it as a value type.
type linkedMap[K comparable, V any] struct {
	kType [0]K
	vType [0]V
	data  map[K]*LinkedMapNode[K, V]
	head  *LinkedMapNode[K, V]
	tail  *LinkedMapNode[K, V]
}

// LinkedMapNode is a node in the map, exposed for direct traversal via First/Last.
type LinkedMapNode[K comparable, V any] struct {
	key   K
	value V
	prev  *LinkedMapNode[K, V]
	next  *LinkedMapNode[K, V]
}

func (n *LinkedMapNode[K, V]) Key() K                     { return n.key }
func (n *LinkedMapNode[K, V]) Value() V                   { return n.value }
func (n *LinkedMapNode[K, V]) Next() *LinkedMapNode[K, V] { return n.next }
func (n *LinkedMapNode[K, V]) Prev() *LinkedMapNode[K, V] { return n.prev }

// NewLinkedMap returns an empty LinkedMap.
func NewLinkedMap[K comparable, V any]() LinkedMap[K, V] {
	return LinkedMap[K, V]{&linkedMap[K, V]{data: make(map[K]*LinkedMapNode[K, V])}}
}

// NewLinkedMapWithCapacity returns an empty LinkedMap with the given capacity.
func NewLinkedMapWithCapacity[K comparable, V any](capacity int) LinkedMap[K, V] {
	return LinkedMap[K, V]{&linkedMap[K, V]{data: make(map[K]*LinkedMapNode[K, V], capacity)}}
}

// Get returns the value for key, or false if it's missing.
func (m LinkedMap[K, V]) Get(key K) (v V, found bool) {
	if m.linkedMap == nil {
		return
	}
	if n, ok := m.data[key]; ok {
		return n.value, true
	}
	return
}

// GetOrDefault returns the value for key, or defaultValue if it's missing.
func (m LinkedMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	v, ok := m.Get(key)
	if ok {
		return v
	}
	return defaultValue
}

// Set adds or updates a key-value pair.
// Returns the old value and true if the key existed, or zero/false if it's a fresh insert.
// Panics if the map is nil (use NewLinkedMap).
func (m LinkedMap[K, V]) Set(key K, value V) (V, bool) {
	if m.linkedMap == nil {
		panic("assignment to entry in nil LinkedMap")
	}

	if n, ok := m.data[key]; ok {
		oldValue := n.value
		n.value = value
		return oldValue, true
	}

	n := &LinkedMapNode[K, V]{key: key, value: value}
	m.data[key] = n
	m.linkAtEnd(n)

	var zero V
	return zero, false
}

// SetAny is an internal hook for serdes to insert entries via reflection.
func (m LinkedMap[K, V]) SetAny(key any, value any) bool {
	_, existed := m.Set(key.(K), value.(V))
	return existed
}

// Delete removes a key and returns its value.
func (m LinkedMap[K, V]) Delete(key K) (v V, found bool) {
	if m.linkedMap == nil {
		return
	}
	n, ok := m.data[key]
	if !ok {
		return
	}
	delete(m.data, key)
	m.unlink(n)
	return n.value, true
}

// Has reports if a key is present.
func (m LinkedMap[K, V]) Has(key K) bool {
	_, ok := m.Get(key)
	return ok
}

// Len is the number of entries in the map.
func (m LinkedMap[K, V]) Len() int {
	if m.linkedMap == nil {
		return 0
	}
	return len(m.data)
}

// First returns the first node in insertion order, or nil if empty.
func (m LinkedMap[K, V]) First() *LinkedMapNode[K, V] {
	if m.linkedMap == nil {
		return nil
	}
	return m.head
}

// Last returns the last node in insertion order, or nil if empty.
func (m LinkedMap[K, V]) Last() *LinkedMapNode[K, V] {
	if m.linkedMap == nil {
		return nil
	}
	return m.tail
}

// Clear removes everything from the map.
func (m LinkedMap[K, V]) Clear() {
	if m.linkedMap == nil {
		return
	}
	clear(m.data)
	m.head = nil
	m.tail = nil
}

// Clone returns a shallow copy that preserves insertion order.
func (m LinkedMap[K, V]) Clone() LinkedMap[K, V] {
	if m.linkedMap == nil {
		return NewLinkedMap[K, V]()
	}
	clone := NewLinkedMapWithCapacity[K, V](m.Len())
	for n := m.head; n != nil; n = n.next {
		clone.Set(n.key, n.value)
	}
	return clone
}

// ToMap returns a plain Go map with the same entries.
func (m LinkedMap[K, V]) ToMap() map[K]V {
	if m.linkedMap == nil {
		return nil
	}
	result := make(map[K]V, m.Len())
	for n := m.head; n != nil; n = n.next {
		result[n.key] = n.value
	}
	return result
}

// All iterates entries in insertion order.
func (m LinkedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		if m.linkedMap == nil {
			return
		}
		for n := m.head; n != nil; {
			next := n.next
			if !yield(n.key, n.value) {
				return
			}
			n = next
		}
	}
}

// AllRev iterates entries in reverse insertion order.
func (m LinkedMap[K, V]) AllRev() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		if m.linkedMap == nil {
			return
		}
		for n := m.tail; n != nil; {
			prev := n.prev
			if !yield(n.key, n.value) {
				return
			}
			n = prev
		}
	}
}

func (m LinkedMap[K, V]) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	if m.linkedMap != nil {
		for n := m.head; n != nil; n = n.next {
			fmt.Fprintf(&sb, "%v: %v", n.key, n.value)
			if n.next != nil {
				sb.WriteString(", ")
			}
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (m *linkedMap[K, V]) linkAtEnd(n *LinkedMapNode[K, V]) {
	if m.tail != nil {
		n.prev = m.tail
		m.tail.next = n
		m.tail = n
	} else {
		m.head = n
		m.tail = n
	}
}

func (m *linkedMap[K, V]) unlink(n *LinkedMapNode[K, V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		m.head = n.next
	}

	if n.next != nil {
		n.next.prev = n.prev
	} else {
		m.tail = n.prev
	}

	n.prev = nil
	n.next = nil
}
