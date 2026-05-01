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

type set[T comparable] struct {
	vals map[T]struct{}
}

func Set[T comparable]() *set[T] {
	return &set[T]{vals: make(map[T]struct{})}
}

func (s *set[T]) Add(v T)      { s.vals[v] = struct{}{} }
func (s *set[T]) Has(v T) bool { _, ok := s.vals[v]; return ok }
func (s *set[T]) Delete(v T)   { delete(s.vals, v) }
func (s *set[T]) Len() int     { return len(s.vals) }
