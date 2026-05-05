package datatypes

import (
	"iter"
)

type OrderedMap[K comparable, V any] struct {
	data  map[K]*node[K, V]
	head  *node[K, V]
	tail  *node[K, V]
	kType [0]K
	vType [0]V
}

type node[K comparable, V any] struct {
	key   K
	value V
	prev  *node[K, V]
	next  *node[K, V]
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{data: make(map[K]*node[K, V])}
}

func NewOrderedMapWithCapacity[K comparable, V any](capacity int) *OrderedMap[K, V] {
	return &OrderedMap[K, V]{data: make(map[K]*node[K, V], capacity)}
}

func (m *OrderedMap[K, V]) Get(key K) (v V, found bool) {
	if n, ok := m.data[key]; ok {
		return n.value, true
	}
	return
}

func (m *OrderedMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if n, ok := m.data[key]; ok {
		return n.value
	}
	return defaultValue
}

func (m *OrderedMap[K, V]) Set(key K, value V) (V, bool) {
	if n, ok := m.data[key]; ok {
		oldValue := n.value
		n.value = value
		return oldValue, false
	}

	n := &node[K, V]{key: key, value: value}
	m.data[key] = n
	m.linkAtEnd(n)

	var zero V
	return zero, true
}

func (m *OrderedMap[K, V]) Delete(key K) (v V, found bool) {
	n, ok := m.data[key]
	if !ok {
		return
	}

	delete(m.data, key)
	m.unlink(n)

	return n.value, true
}

func (m *OrderedMap[K, V]) Has(key K) bool {
	_, ok := m.data[key]
	return ok
}

func (m *OrderedMap[K, V]) First() (K, V, bool) {
	return m.nthFromStart(0)
}

func (m *OrderedMap[K, V]) Last() (K, V, bool) {
	return m.nthFromEnd(0)
}

func (m *OrderedMap[K, V]) Nth(n int) (K, V, bool) {
	if n < 0 {
		return m.nthFromEnd(-n - 1)
	}
	return m.nthFromStart(n)
}

func (m *OrderedMap[K, V]) nthFromStart(n int) (k K, v V, found bool) {
	for i, node := 0, m.head; node != nil; i, node = i+1, node.next {
		if i == n {
			return node.key, node.value, true
		}
	}
	return
}

func (m *OrderedMap[K, V]) nthFromEnd(n int) (k K, v V, found bool) {
	for i, node := 0, m.tail; node != nil; i, node = i+1, node.prev {
		if i == n {
			return node.key, node.value, true
		}
	}
	return
}

func (m *OrderedMap[K, V]) Len() int {
	return len(m.data)
}

func (m *OrderedMap[K, V]) Clear() {
	m.data = make(map[K]*node[K, V])
	m.head = nil
	m.tail = nil
}

func (m *OrderedMap[K, V]) Clone() *OrderedMap[K, V] {
	clone := NewOrderedMapWithCapacity[K, V](m.Len())

	for oldNode := m.head; oldNode != nil; oldNode = oldNode.next {
		newNode := &node[K, V]{
			key:   oldNode.key,
			value: oldNode.value,
		}

		clone.data[newNode.key] = newNode
		clone.linkAtEnd(newNode)
	}

	return clone
}

func (m *OrderedMap[K, V]) ToMap() map[K]V {
	result := make(map[K]V, m.Len())
	for n := m.head; n != nil; n = n.next {
		result[n.key] = n.value
	}
	return result
}

func (m *OrderedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.key, n.value) {
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) AllRev() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := m.tail; n != nil; n = n.prev {
			if !yield(n.key, n.value) {
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(key K) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.key) {
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(value V) bool) {
		for n := m.head; n != nil; n = n.next {
			if !yield(n.value) {
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) linkAtEnd(n *node[K, V]) {
	if m.tail != nil {
		n.prev = m.tail
		m.tail.next = n
		m.tail = n
	} else {
		m.head = n
		m.tail = n
	}
}

func (m *OrderedMap[K, V]) unlink(n *node[K, V]) {
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
