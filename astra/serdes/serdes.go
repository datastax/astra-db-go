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

	dst, err := SerializeInto(data, target, *buf)

	if err != nil {
		return nil, err
	}

	ret := make([]byte, len(dst))
	copy(ret, dst)
	return ret, nil
}

func SerializeInto(data any, target Target, dst []byte) ([]byte, error) {
	if data == nil {
		return append(dst, "null"...), nil
	}

	t := reflect.TypeOf(data)
	p := (*iface)(unsafe.Pointer(&data)).ptr

	ctx := EncodeCtx{Target: target}
	c := resolveCodecCaching(ctx.codecCtx, t)

	var err error
	dst, err = c.encode(ctx, dst, p)
	runtime.KeepAlive(data)

	return dst, wrapStruct(err, t.Name())
}

func Deserialize(data []byte, res any, targetDecodeCtx TargetDecodeCtx, target Target) error {
	if res == nil {
		return &InvalidUnmarshalError{Type: nil}
	}

	t := reflect.TypeOf(res)
	p := (*iface)(unsafe.Pointer(&res)).ptr

	if t.Kind() != reflect.Ptr {
		return &InvalidUnmarshalError{Type: t}
	}

	ctx := DecodeCtx{Target: target, TargetCtx: targetDecodeCtx, originalSrc: data}
	c := resolveCodecCaching(ctx.codecCtx, t.Elem())

	_, err := c.decode(ctx, data, p)
	return wrapStruct(err, t.Elem().Name())
}
