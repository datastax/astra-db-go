package serdes

import (
	"reflect"
	"sync"
)

type codec struct {
	encode encoder
	decode decoder
}

type AstraCodec interface {
	FromAstraValue(v any) error
	ToAstraValue() any
}

var (
	typeCodecs sync.Map // map[reflect.Type]codec
	kindCodecs [reflect.String + 1]codec
)

func init() {
	kindCodecs[reflect.Bool] = codec{encodeBoolKind, decodeBoolKind}

	for i := reflect.Int; i <= reflect.Int64; i++ {
		kindCodecs[i] = codec{encodeIntKind, decodeIntKind}
	}

	for i := reflect.Uint; i <= reflect.Uintptr; i++ {
		kindCodecs[i] = codec{encodeUintKind, decodeUintKind}
	}

	for i := reflect.Float32; i <= reflect.Float64; i++ {
		kindCodecs[i] = codec{encodeFloatKind, decodeFloatKind}
	}

	kindCodecs[reflect.String] = codec{encodeStringKind, decodeStringKind}
}

var nilCodec = codec{
	encodeNull,
	decodeNull,
}

type seenStructs = map[reflect.Type]*structInfo

func resolveCodecCaching(t reflect.Type, seen seenStructs, canAddr bool) codec {
	if t == nil || t == nilType {
		return nilCodec
	}

	if c, ok := typeCodecs.Load(t); ok {
		return c.(codec)
	}

	if t.Implements(astraCodecType) || (t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(astraCodecType)) {
		return mkCustomCodec(t)
	}

	k := t.Kind()
	if int(k) < len(kindCodecs) && kindCodecs[k].encode != nil {
		return kindCodecs[k]
	}

	switch k {
	case reflect.Ptr:
		return mkPointerCodec(t, seen)
	}

	switch t {
	case uuidType:
		panic("uuid codec")
	case vectorType:
		panic("vector codec")
	case timeType:
		panic("time codec")
	case ipType:
		panic("ip codec")
	}

	switch k {
	}

	panic("unsupported type: " + t.String())
}

func mkCodecCaching(t reflect.Type, c codec) codec {
	typeCodecs.Store(t, c)
	return c
}

func mkCustomCodec(t reflect.Type) codec {
	return mkCodecCaching(t, codec{
		encodeCustom(t),
		decodeCustom(t),
	})
}

func mkPointerCodec(t reflect.Type, seen seenStructs) codec {
	el := t.Elem()
	c := resolveCodecCaching(el, seen, true)

	return mkCodecCaching(t, codec{
		encodePointer(c.encode),
		decodePointer(c.decode),
	})
}

func mkStructCodec(t reflect.Type, seen seenStructs, canAddr bool) codec {

}

type structInfo struct {
	structType reflect.Type
}

type fieldInfo struct {
	name      string
	index     int
	codec     codec
	fieldType reflect.Type
}

func compileStructInfo(t reflect.Type, seen seenStructs, canAddr bool) *structInfo {

}
