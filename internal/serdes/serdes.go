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

package serdes

import (
	"fmt"
	"reflect"
	"unsafe"
)

func Serialize(data any) ([]byte, error) {
	t := reflect.TypeOf(data)
	p := (*iface)(unsafe.Pointer(&data)).ptr

	c, err := resolveCodecCaching(t, seenStructs{}, t.Kind() == reflect.Ptr)
	if err != nil {
		return nil, err
	}

	return c.encode(encodeCtx{}, []byte{}, p)
}

func Deserialize(data []byte, res any) error {
	t := reflect.TypeOf(res)
	p := (*iface)(unsafe.Pointer(&res)).ptr

	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("deserialize requires a pointer, got %v", t)
	}

	c, err := resolveCodecCaching(t.Elem(), seenStructs{}, true)
	if err != nil {
		return err
	}

	_, err = c.decode(decodeCtx{}, data, p)
	return err
}
