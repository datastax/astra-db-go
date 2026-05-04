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
	"runtime"
	"sync"
	"unsafe"
)

type bufferPool struct {
	pool sync.Pool
}

var encodingBufferPool = bufferPool{
	pool: sync.Pool{
		New: func() any {
			b := make([]byte, 0, 4096)
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

func Serialize(data any, target Target) ([]byte, error) {
	buf := encodingBufferPool.Get()
	defer encodingBufferPool.Put(buf)

	b, err := SerializeInto(data, target, *buf)

	if err != nil {
		return nil, err
	}

	ret := make([]byte, len(b))
	copy(ret, b)
	return ret, nil
}

func SerializeInto(data any, target Target, dst []byte) ([]byte, error) {
	if data == nil {
		return append(dst, "null"...), nil
	}

	t := reflect.TypeOf(data)
	p := (*iface)(unsafe.Pointer(&data)).ptr

	ctx := encodeCtx{target: target}
	c := resolveCodecCaching(ctx.codecCtx, t, seenStructs{})

	var err error
	dst, err = c.encode(ctx, dst, p)
	runtime.KeepAlive(data)

	return dst, err
}

func Deserialize(data []byte, res any, target Target) error {
	t := reflect.TypeOf(res)
	p := (*iface)(unsafe.Pointer(&res)).ptr

	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("deserialize requires a pointer, got %v", t)
	}

	ctx := decodeCtx{target: target}
	c := resolveCodecCaching(ctx.codecCtx, t.Elem(), seenStructs{})

	_, err := c.decode(ctx, data, p)
	return err
}
