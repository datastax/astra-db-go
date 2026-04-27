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
	"sync"
	"unsafe"
)

type bufferPool struct {
	pool sync.Pool
}

var encodingBufferPool = bufferPool{
	pool: sync.Pool{
		New: func() any {
			b := make([]byte, 0, 1024)
			return &b
		},
	},
}

func (bp *bufferPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

func (bp *bufferPool) Put(b *[]byte) {
	if cap(*b) <= 65536 {
		*b = (*b)[:0]
		bp.pool.Put(b)
	}
}

func Serialize(data any) ([]byte, error) {
	if data == nil {
		return []byte("null"), nil
	}

	t := reflect.TypeOf(data)
	p := (*iface)(unsafe.Pointer(&data)).ptr

	c := resolveCodecCaching(t, seenStructs{}, t.Kind() == reflect.Ptr)

	buf := encodingBufferPool.Get()

	var err error
	*buf, err = c.encode(encodeCtx{}, *buf, p)
	ret := make([]byte, len(*buf))
	copy(ret, *buf)

	encodingBufferPool.Put(buf)
	return ret, err
}

func Deserialize(data []byte, res any) error {
	t := reflect.TypeOf(res)
	p := (*iface)(unsafe.Pointer(&res)).ptr

	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("deserialize requires a pointer, got %v", t)
	}

	c := resolveCodecCaching(t.Elem(), seenStructs{}, true)

	_, err := c.decode(decodeCtx{}, data, p)
	return err
}
