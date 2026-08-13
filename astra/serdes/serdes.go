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

	"github.com/datastax/astra-db-go/v2/internal/refl"
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

// TODO the method signatures of Serialize and Deserialize are just getting longer and longer
// maybe we can require the user to create the ctx objects themselves and just have Serialize/Deserialize
// take the ctx and data?
//
// that could help with being able to pass private ctx fields across interface boundaries as well

func Serialize(data any, target Target, flags ...SerFlags) ([]byte, error) {
	buf := encodingBufferPool.Get()
	defer encodingBufferPool.Put(buf)

	dst, err := SerializeInto(data, target, *buf, flags...)

	if err != nil {
		return nil, err
	}

	ret := make([]byte, len(dst))
	copy(ret, dst)
	return ret, nil
}

func SerializeInto(data any, target Target, dst []byte, flags ...SerFlags) ([]byte, error) {
	if data == nil {
		return append(dst, "null"...), nil
	}

	var f SerFlags
	for _, flag := range flags {
		f |= flag
	}

	t := reflect.TypeOf(data)
	p := (*refl.IFace)(unsafe.Pointer(&data)).Ptr

	ctx := EncodeCtx{Target: target, Flags: f}
	c := resolveCodecCaching(ctx.codecCtx, t, f&SerNoCache != 0)

	var err error
	dst, err = c.encode(ctx, dst, p)
	runtime.KeepAlive(data)

	return dst, wrapStruct(err, t.Name())
}

func Deserialize(data []byte, res any, targetDecodeCtx TargetDecodeCtx, target Target, flags ...DesFlags) error {
	var f DesFlags
	for _, flag := range flags {
		f |= flag
	}

	ctx := DecodeCtx{Target: target, TargetCtx: targetDecodeCtx, Flags: f}
	return deserializeWithContext(data, res, ctx)
}

func deserializeWithContext(data []byte, res any, ctx DecodeCtx) error {
	if res == nil {
		return &DecodeError{Action: DecodeActionInvalid}
	}

	t := reflect.TypeOf(res)
	p := (*refl.IFace)(unsafe.Pointer(&res)).Ptr

	if t.Kind() != reflect.Ptr {
		return &DecodeError{Action: DecodeActionInvalid, Type: t}
	}

	ctx.payload = &data
	c := resolveCodecCaching(ctx.codecCtx, t.Elem(), ctx.Flags&DesNoCache != 0)

	srcAfter, err := c.decode(ctx, data, p)
	runtime.KeepAlive(res)
	if err != nil {
		return wrapStruct(err, t.Elem().Name())
	}

	srcAfter = skipWS(srcAfter)
	if len(srcAfter) > 0 {
		return ctx.syntaxError(srcAfter, fmt.Sprintf("invalid character '%c' after top-level value", srcAfter[0]))
	}

	return nil
}
