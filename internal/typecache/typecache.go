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

package typecache

import (
	"maps"
	"reflect"
	"sync/atomic"

	"github.com/datastax/astra-db-go/v2/internal/refl"
)

// Map is a performant read-optimized COW cache for reflect.Type -> V mappings,
// based heavily on implementations by segmentio/encoding/json & goccy/go-json
type Map[V any] atomic.Pointer[map[refl.TypeID]V]

func (m *Map[V]) ptr() *atomic.Pointer[map[refl.TypeID]V] {
	return (*atomic.Pointer[map[refl.TypeID]V])(m)
}

// Load returns the value stored for t, and whether it was found.
func (m *Map[V]) Load(t reflect.Type) (V, bool) {
	snapshot := m.snapshot()
	v, ok := snapshot[refl.GetTypeID(t)]
	return v, ok
}

// Store inserts or replaces the value for t.
// It copies the current snapshot into a new map before swapping, so concurrent
// readers always see a consistent (if possibly stale) view.
func (m *Map[V]) Store(t reflect.Type, v V) {
	old := m.snapshot()
	next := make(map[refl.TypeID]V, len(old)+1)
	maps.Copy(next, old)
	next[refl.GetTypeID(t)] = v
	m.ptr().Store(&next)
}

// Reset clears all entries from the cache.
func (m *Map[V]) Reset() {
	m.ptr().Store(nil)
}

// Len returns the number of entries currently in the cache.
func (m *Map[V]) Len() int {
	return len(m.snapshot())
}

func (m *Map[V]) snapshot() map[refl.TypeID]V {
	if p := m.ptr().Load(); p != nil {
		return *p
	}
	return map[refl.TypeID]V{}
}
