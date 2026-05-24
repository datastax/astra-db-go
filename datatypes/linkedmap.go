package datatypes

import (
	"fmt"
	"iter"
	"strings"
)

type LinkedMap[K comparable, V any] struct {
	*linkedMap[K, V]
}

// IMPORTANT: The field ordering of kType, vType, and data is extremely important:
// - kType and vType must are at the start of the struct to ensure that no padding is allocated for them
// - data is the first non-zero-size field, and must be at index 2 – both of these are relied on for reflection and pointer math
//
// `serdes/maps.go` must be updated of either of the above two invariants are changed.
//
// The idea is to treat linkedMap similar to how hmap is used in Go's built-in map implementation,
// where the public type is a thin wrapper around an internal struct that contains the actual data and
// implementation details.
//
// This should make the public API a little cleaner and also, importantly, helps enforce the requirement of having the
// actual implementation fields behind a pointer so people don't try to use it as a value type
type linkedMap[K comparable, V any] struct {
	// Preserves type information for serdes purposes
	kType [0]K
	vType [0]V

	// The actual backing map for all the basic constant time operations
	data map[K]*LinkedMapNode[K, V]

	// Pointers to the ends of the linked list used when iterating in insertion order
	head *LinkedMapNode[K, V]
	tail *LinkedMapNode[K, V]
}

type LinkedMapNode[K comparable, V any] struct {
	key   K
	value V
	prev  *LinkedMapNode[K, V]
	next  *LinkedMapNode[K, V]
}

func NewLinkedMap[K comparable, V any]() LinkedMap[K, V] {
	return LinkedMap[K, V]{&linkedMap[K, V]{data: make(map[K]*LinkedMapNode[K, V])}}
}

func NewLinkedMapWithCapacity[K comparable, V any](capacity int) LinkedMap[K, V] {
	return LinkedMap[K, V]{&linkedMap[K, V]{data: make(map[K]*LinkedMapNode[K, V], capacity)}}
}

// Get returns the value for key, or the zero value and false if not present.
func (m LinkedMap[K, V]) Get(key K) (v V, found bool) {
	if n, ok := m.data[key]; ok {
		return n.value, true
	}
	return
}

// GetOrDefault returns the value for key, or defaultValue if not present.
func (m LinkedMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if n, ok := m.data[key]; ok {
		return n.value
	}
	return defaultValue
}

// Set inserts or updates key with value.
// Returns the previous value and true if the key already existed, or the zero value and false if it was newly inserted.
// Panics if the LinkedMap is nil (use NewLinkedMap to initialize).
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

func (m LinkedMap[K, V]) SetAny(key any, value any) bool {
	_, existed := m.Set(key.(K), value.(V))
	return existed
}

// Delete removes key and returns its value and true, or the zero value and false if not present.
func (m LinkedMap[K, V]) Delete(key K) (v V, found bool) {
	n, ok := m.data[key]
	if !ok {
		return
	}

	delete(m.data, key)
	m.unlink(n)

	return n.value, true
}

// Has reports whether key is present.
func (m LinkedMap[K, V]) Has(key K) bool {
	_, ok := m.data[key]
	return ok
}

// First returns the first node in insertion order, or nil if empty.
func (m LinkedMap[K, V]) First() *LinkedMapNode[K, V] {
	return m.head
}

// Last returns the last node in insertion order, or nil if empty.
func (m LinkedMap[K, V]) Last() *LinkedMapNode[K, V] {
	return m.tail
}

// Len returns the number of entries.
func (m LinkedMap[K, V]) Len() int {
	return len(m.data)
}

func (m LinkedMap[K, V]) Clear() {
	clear(m.data)
	m.head = nil
	m.tail = nil
}

func (m LinkedMap[K, V]) Clone() LinkedMap[K, V] {
	clone := NewLinkedMapWithCapacity[K, V](m.Len())

	for oldNode := m.head; oldNode != nil; oldNode = oldNode.next {
		newNode := &LinkedMapNode[K, V]{
			key:   oldNode.key,
			value: oldNode.value,
		}

		clone.data[newNode.key] = newNode
		clone.linkAtEnd(newNode)
	}

	return clone
}

func (m LinkedMap[K, V]) ToMap() map[K]V {
	result := make(map[K]V, m.Len())
	for n := m.head; n != nil; n = n.next {
		result[n.key] = n.value
	}
	return result
}

// All iterates entries in insertion order.
func (m LinkedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.key, n.value) {
				return
			}
		}
	}
}

// AllRev iterates entries in reverse insertion order.
func (m LinkedMap[K, V]) AllRev() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := m.tail; n != nil; n = n.prev {
			if !yield(n.key, n.value) {
				return
			}
		}
	}
}

// Keys iterates keys in insertion order.
func (m LinkedMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(key K) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.key) {
				return
			}
		}
	}
}

// Values iterates values in insertion order.
func (m LinkedMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(value V) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.value) {
				return
			}
		}
	}
}

func (m LinkedMap[K, V]) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	for n := m.head; n != nil; n = n.next {
		fmt.Fprintf(&sb, "%v: %v", n.key, n.value)
		if n.next != nil {
			sb.WriteString(", ")
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
