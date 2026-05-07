package datatypes

import (
	"fmt"
	"iter"
)

type OrderedMap[K comparable, V any] struct {
	*orderedMap[K, V]
}

// TODO should I just replace this with an associative list type? I need a map which allows non comparable keys anyways
// so I might as well just use that for everything and not have to maintain two separate implementations.
//
// this implementation isn't super performant anyways.

// IMPORTANT: The field ordering of kType, vType, and data is extremely important:
// - kType and vType must are at the start of the struct to ensure that no padding is allocated for them
// - data is the first non-zero-size field, and must be at index 2 – both of these are relied on for reflection and pointer math
//
// `serdes/maps.go` must be updated of either of the above two invariants are changed.
//
// The idea is to treat orderedMap similar to how hmap is used in Go's built-in map implementation,
// where the public type is a thin wrapper around an internal struct that contains the actual data and
// implementation details.
//
// This should make the public API a little cleaner and also, importantly, helps enforce the requirement of having the
// actual implementation fields behind a pointer so people don't try to use it as a value type
type orderedMap[K comparable, V any] struct {
	// Preserves type information for serdes purposes
	kType [0]K
	vType [0]V

	// The actual backing map for all the basic constant time operations
	data map[K]*OrderedMapNode[K, V]

	// Pointers to the ends of the linked list used when iterating in insertion order
	head *OrderedMapNode[K, V]
	tail *OrderedMapNode[K, V]
}

type OrderedMapNode[K comparable, V any] struct {
	key   K
	value V
	prev  *OrderedMapNode[K, V]
	next  *OrderedMapNode[K, V]
}

func NewOrderedMap[K comparable, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{&orderedMap[K, V]{data: make(map[K]*OrderedMapNode[K, V])}}
}

func NewOrderedMapWithCapacity[K comparable, V any](capacity int) OrderedMap[K, V] {
	return OrderedMap[K, V]{&orderedMap[K, V]{data: make(map[K]*OrderedMapNode[K, V], capacity)}}
}

func (m OrderedMap[K, V]) Get(key K) (v V, found bool) {
	if n, ok := m.data[key]; ok {
		return n.value, true
	}
	return
}

func (m OrderedMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if n, ok := m.data[key]; ok {
		return n.value
	}
	return defaultValue
}

func (m OrderedMap[K, V]) Set(key K, value V) (V, bool) {
	m.ensureInit()

	if n, ok := m.data[key]; ok {
		oldValue := n.value
		n.value = value
		return oldValue, false
	}

	n := &OrderedMapNode[K, V]{key: key, value: value}
	m.data[key] = n
	m.linkAtEnd(n)

	var zero V
	return zero, true
}

func (m OrderedMap[K, V]) SetAny(key any, value any) bool {
	_, set := m.Set(key.(K), value.(V))
	return set
}

func (m OrderedMap[K, V]) Delete(key K) (v V, found bool) {
	n, ok := m.data[key]
	if !ok {
		return
	}

	delete(m.data, key)
	m.unlink(n)

	return n.value, true
}

func (m OrderedMap[K, V]) Has(key K) bool {
	_, ok := m.data[key]
	return ok
}

func (m OrderedMap[K, V]) First() *OrderedMapNode[K, V] {
	return m.head
}

func (m OrderedMap[K, V]) Last() *OrderedMapNode[K, V] {
	return m.tail
}

func (m OrderedMap[K, V]) Len() int {
	return len(m.data)
}

func (m OrderedMap[K, V]) Clear() {
	clear(m.data)
	m.head = nil
	m.tail = nil
}

func (m OrderedMap[K, V]) Clone() OrderedMap[K, V] {
	clone := NewOrderedMapWithCapacity[K, V](m.Len())

	for oldNode := m.head; oldNode != nil; oldNode = oldNode.next {
		newNode := &OrderedMapNode[K, V]{
			key:   oldNode.key,
			value: oldNode.value,
		}

		clone.data[newNode.key] = newNode
		clone.linkAtEnd(newNode)
	}

	return clone
}

func (m OrderedMap[K, V]) ToMap() map[K]V {
	result := make(map[K]V, m.Len())
	for n := m.head; n != nil; n = n.next {
		result[n.key] = n.value
	}
	return result
}

func (m OrderedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.key, n.value) {
				return
			}
		}
	}
}

func (m OrderedMap[K, V]) AllRev() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := m.tail; n != nil; n = n.prev {
			if !yield(n.key, n.value) {
				return
			}
		}
	}
}

func (m OrderedMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(key K) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.key) {
				return
			}
		}
	}
}

func (m OrderedMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(value V) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.value) {
				return
			}
		}
	}
}

func (m OrderedMap[K, V]) String() string {
	var result string
	result += "{"
	for n := m.head; n != nil; n = n.next {
		result += fmt.Sprintf("%v: %v", n.key, n.value)
		if n.next != nil {
			result += ", "
		}
	}
	result += "}"
	return result
}

func (m OrderedMap[K, V]) ensureInit() {
	if m.orderedMap == nil {
		panic("OrderedMap is not initialized. Use NewOrderedMap() to create an instance before reading.")
	}
}

func (m *orderedMap[K, V]) linkAtEnd(n *OrderedMapNode[K, V]) {
	if m.tail != nil {
		n.prev = m.tail
		m.tail.next = n
		m.tail = n
	} else {
		m.head = n
		m.tail = n
	}
}

func (m *orderedMap[K, V]) unlink(n *OrderedMapNode[K, V]) {
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
