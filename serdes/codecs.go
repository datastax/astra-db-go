package serdes

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync/atomic"
	"unsafe"
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
	ptrDepth int
	ptrSeen  map[unsafe.Pointer]struct{}
}

type DecodeCtx struct {
	codecCtx
	Target Target
	Schema any
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
	typeCodecs atomic.Pointer[map[unsafe.Pointer]codec] // TODO may be able to just reuse the same cache for all targets and let the resolution be at execution time?
	kindCodecs [reflect.String + 1]codec
)

//go:generate go run -modfile=../tools/gen-serdes/go.mod ../tools/gen-serdes/main.go

func init() {
	kindCodecs[reflect.Bool] = codec{boolEncoder, boolDecoder}
	kindCodecs[reflect.String] = codec{stringEncoder, stringDecoder}
}

type codecCtx struct {
	fieldHint fieldHint
}

type seenStructs = map[reflect.Type]*structInfo

func resolveCodecCaching(ctx codecCtx, t reflect.Type) codec {
	tid := typePtr(t)
	cache := cacheLoad()

	if c, ok := cache[tid]; ok {
		return c
	}

	codec := resolveCodec(ctx, t, seenStructs{}, t.Kind() == reflect.Ptr)

	return cacheSet(cache, t, codec)
}

func cacheLoad() map[unsafe.Pointer]codec {
	p := typeCodecs.Load()
	if p == nil {
		return map[unsafe.Pointer]codec{}
	}
	return *p
}

// I don't fully understand why using the "old" cache from the initial cacheLoad() as the base for the updated cache
// instead of reloading the cache before updating is the right way to do this, but two major libraries do this:
// - goccy/go-json: https://github.com/goccy/go-json/blob/e4877d51d546f8c67b1cd9b49ab002ba3af37785/internal/encoder/compiler.go#L46
// - segmentio/encoding/json: https://github.com/segmentio/encoding/blob/fd406855de30c54110d23eace25478ab9c6fa2cc/json/codec.go#L75
//
// I do have my suspicions as to why they're doing this over reloading the cache, but I haven't taken the time to properly verify
// that this is indeed the best way to do this (I do agree with CoW > mutex or using CaS here at least, not that it's likely
// to make a huge difference either way)
//
// Anyway, at the risk of cargo-culting, I'm going to follow their lead here and do it this way regardless as
// the authors of those libraries are highly talented and much more knowledgeable about this kind of thing than I am
func cacheSet(oldCache map[unsafe.Pointer]codec, t reflect.Type, c codec) codec {
	if inlined(t) {
		c.encode = mkInlineEncoder(c.encode)
	}

	newCache := make(map[unsafe.Pointer]codec, len(oldCache)+1)
	newCache[typePtr(t)] = c
	maps.Copy(newCache, oldCache)
	typeCodecs.Store(&newCache)

	return c
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
	case bigIntType:
		return codec{bigIntEncoder, bigIntDecoder}
	case bigFloatType:
		return codec{bigFloatEncoder, bigFloatDecoder}
	case byteSliceType:
		return codec{binaryEncoder, binaryDecoder}
	case uuidType:
		return codec{uuidEncoder, uuidDecoder}
	case oidType:
		return codec{objectIdEncoder, objectIdDecoder}
	case dApiTimeType:
		return codec{timestampEncoder, timestampDecoder}
	case timeType:
		return codec{timeEncoder, timeDecoder}
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
		if mkCodec, yes := isSpecialGenericDatatype(t); yes {
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
			c = mkErroredCodec(fmt.Errorf("unsupported type %s", t.String()))
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
	}

	switch {
	case ptr.Implements(astraUnmarshalerType):
		c.decode = mkAstraUnmarshalerDecoder(t)
	case ptr.Implements(astraRawUnmarshalerType):
		c.decode = mkAstraRawUnmarshalerDecoder(t)
	}

	return
}

func isSpecialGenericDatatype(t reflect.Type) (func(ctx codecCtx, t reflect.Type, seen seenStructs) codec, bool) {
	if t.PkgPath() != datatypesPkgPath {
		return nil, false
	}

	tName := t.Name()

	switch {
	case strings.HasPrefix(tName, someOrderedMapTypeName):
		return mkOrderedMapCodec, true
	case strings.HasPrefix(tName, someSetTypeName):
		return mkSetCodec, true
	default:
		return nil, false
	}
}
