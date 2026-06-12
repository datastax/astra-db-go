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

package utils

import (
	"fmt"
	"reflect"
)

// I know 'utils' isn't a very idiomatic or descriptive name, but sometimes a thing is just what it says on the tin

func RequireSlicePtr(v any) (reflect.Value, reflect.Value, error) {
	ptr := reflect.ValueOf(v)
	if ptr.Kind() != reflect.Ptr {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("expected pointer to slice, got %s", ptr.Kind())
	}

	slice := ptr.Elem()
	if slice.Kind() == reflect.Interface {
		slice = slice.Elem()
	}

	if slice.Kind() != reflect.Slice {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("expected pointer to slice, got pointer to %s", slice.Kind())
	}

	return ptr, slice, nil
}

func NonNilMap[M ~map[K]V, K comparable, V any](m M) M {
	if m == nil {
		return make(M)
	}
	return m
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
