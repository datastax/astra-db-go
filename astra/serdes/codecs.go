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
	"unsafe"

	"github.com/datastax/astra-db-go/v2/astra/internal/typeutil"
	"github.com/datastax/astra-db-go/v2/internal/typecache"
)

type codec struct {
	encode encoder
	decode decoder
}

type encoder func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error)
type decoder func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error)

type EncodeCtx struct {
	codecCtx
	Target   Target
	Flags    SerFlags
	ptrDepth int
	ptrSeen  map[unsafe.Pointer]struct{}
}

type DecodeCtx struct {
	codecCtx
	Target    Target
	TargetCtx TargetDecodeCtx
	Flags     DesFlags
	payload   *[]byte
}

type AstraMarshaler interface {
	MarshalAstra(ctx EncodeCtx) (any, error)
}

type AstraRawMarshaler interface {
	MarshalAstraRaw(ctx EncodeCtx, dst []byte) ([]byte, error)
}

type AstraUnmarshaler interface {
	UnmarshalAstra(ctx DecodeCtx, value any) error
}

type AstraRawUnmarshaler interface {
	UnmarshalAstraRaw(ctx DecodeCtx, value []byte) error
}

var (
	typeCodecs typecache.Map[codec]
	kindCodecs [reflect.String + 1]codec
)

//go:generate go run -modfile=../../tools/gen-serdes/go.mod ../../tools/gen-serdes/main.go

func init() {
	kindCodecs[reflect.Bool] = codec{boolEncoder, boolDecoder}
	kindCodecs[reflect.String] = codec{stringEncoder, stringDecoder}
}

type codecCtx struct {
	fieldHint fieldHint
}

type seenStructs = map[reflect.Type]*structInfo

func resolveCodecCaching(ctx codecCtx, t reflect.Type, noCache bool) codec {
	if !noCache {
		if c, ok := typeCodecs.Load(t); ok {
			return c
		}
	}

	codec := resolveCodec(ctx, t, seenStructs{}, t.Kind() == reflect.Ptr)

	if inlined(t) {
		codec.encode = mkInlineEncoder(codec.encode)
	}

	if !noCache {
		typeCodecs.Store(t, codec)
	}

	return codec
}

func resolveCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (c codec) {
	if t == nil || t == nilType {
		return nilCodec
	}

	// we could have some of these define MarshalAstraRaw and UnmarshalAstraRaw in their own files,
	// but naive benchmarks show that having the codecs directly here is noticeably faster by a small
	// margin, and also avoids an extra allocation (see bench_test.go)
	//
	// now whether this performance really matters... probably not, but keeping the codecs
	// centralized seems clearer to me anyhow, with the tiny speed boost just being a happy bonus
	switch t {
	case anyType:
		return codec{emptyInterfaceEncoder, emptyInterfaceDecoder}
	case rawMessageType:
		return codec{rawMessageEncoder, rawMessageDecoder}
	case vectorType:
		return codec{vectorEncoder, vectorDecoder}
	case bigIntPtrType:
		return codec{bigIntEncoder, bigIntDecoder}
	case bigFloatPtrType:
		return codec{bigFloatEncoder, bigFloatDecoder}
	case byteSliceType:
		return codec{binaryEncoder, binaryDecoder}
	case uuidType:
		return codec{uuidEncoder, uuidDecoder}
	case oidType:
		return codec{objectIdEncoder, objectIdDecoder}
	case dateOnlyType:
		return codec{dateOnlyEncoder, dateOnlyDecoder}
	case timeOnlyType:
		return codec{timeOnlyEncoder, timeOnlyDecoder}
	case durationType:
		return codec{durationEncoder, durationDecoder}
	case timeType:
		return codec{timeEncoder, timeDecoder}
	case ipType:
		return codec{ipEncoder, ipDecoder}
	}

	if c.encode != nil {
		return
	}

	k := t.Kind()

	if int(k) < len(kindCodecs) && kindCodecs[k].encode != nil {
		c = kindCodecs[k]
	}

	switch k {
	case reflect.Ptr:
		c = mkPointerCodec(ctx, t, seen)
	case reflect.Struct:
		if mkCodec, yes := getCustomGenericTypeCodec(t); yes {
			return mkCodec(ctx, t, seen)
		}
		c = mkStructCodec(ctx, t, seen, canAddr)
	case reflect.Slice:
		c = mkSliceCodec(ctx, t, seen)
	case reflect.Array:
		c = mkArrayCodec(ctx, t, seen, canAddr)
	case reflect.Map:
		c = mkMapCodec(ctx, t, seen)
	case reflect.Interface:
		c = mkSomeInterfaceCodec(t)
	default:
		if c.encode == nil {
			c = mkErroredCodec(&EncodeError{Action: EncodeActionUnsupportedType, Type: t})
		}
	}

	ptr := reflect.PointerTo(t)

	switch {
	case t.Implements(astraMarshalerType):
		c.encode = mkAstraMarshalerEncoder(t, false)
	case t.Implements(astraRawMarshalerType):
		c.encode = mkAstraRawMarshalerEncoder(t, false)
	case canAddr && ptr.Implements(astraMarshalerType):
		c.encode = mkAstraMarshalerEncoder(t, true)
	case canAddr && ptr.Implements(astraRawMarshalerType):
		c.encode = mkAstraRawMarshalerEncoder(t, true)
	case t.Implements(jsonMarshalerType):
		c.encode = mkJSONMarshalerEncoder(t, false, c.encode)
	case canAddr && ptr.Implements(jsonMarshalerType):
		c.encode = mkJSONMarshalerEncoder(t, true, c.encode)
	}

	switch {
	case ptr.Implements(astraUnmarshalerType):
		c.decode = mkAstraUnmarshalerDecoder(t)
	case ptr.Implements(astraRawUnmarshalerType):
		c.decode = mkAstraRawUnmarshalerDecoder(t)
	case ptr.Implements(jsonUnmarshalerType):
		c.decode = mkJSONUnmarshalerDecoder(t, c.decode)
	}

	return
}

func getCustomGenericTypeCodec(t reflect.Type) (func(ctx codecCtx, t reflect.Type, seen seenStructs) codec, bool) {
	switch typeutil.GetCustomGenericTypeID(t) {
	case typeutil.LinkedMapType:
		return mkLinkedMapCodec, true
	case typeutil.SortedMapType:
		return mkSortedMapCodec, true
	case typeutil.SetType:
		return mkSetCodec, true
	default:
		return nil, false
	}
}
