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

package untyped

import (
	"fmt"
	"reflect"

	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

func getDeepFromMap(m map[string]any, path ...string) (any, bool) {
	current := m
	for i, p := range path {
		val, ok := current[p]
		if !ok {
			return nil, false
		}

		if i == len(path)-1 {
			return val, true
		}

		nextMap, ok := val.(map[string]any)
		if !ok {
			return nil, false
		}
		current = nextMap
	}
	return nil, false
}

func mustGet(get func(path ...string) (any, bool), path []string, target string) any {
	val, ok := get(path...)
	if !ok {
		panic(fmt.Sprintf("astra: path %v not found in %s", path, target))
	}
	return val
}

func decodeFromMap(m map[string]any, path []string, dest any, target serdes.Target) error {
	val, ok := getDeepFromMap(m, path...)
	if !ok {
		return fmt.Errorf("astra: path %v not found", path)
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("astra: destination must be a non-nil pointer")
	}

	if val != nil {
		srcVal := reflect.ValueOf(val)
		if srcVal.Type().AssignableTo(rv.Elem().Type()) {
			rv.Elem().Set(srcVal)
			return nil
		}
	} else {
		elem := rv.Elem()
		switch elem.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map:
			elem.Set(reflect.Zero(elem.Type()))
			return nil
		}
	}

	b, err := serdes.Serialize(val, target)
	if err != nil {
		return err
	}

	return serdes.Deserialize(b, dest, nil, target)
}
