package serdes

import (
	"fmt"
	"reflect"
)

type decoder func(ctx decodeCtx, src []byte, ret reflect.Value) ([]byte, error)

type decodeCtx struct {
}

func decodeBoolKind(_ decodeCtx, src []byte, ret reflect.Value) ([]byte, error) {
	off, err := decodeBool(src, ret)
	return src[off:], err
}

func decodeIntKind(_ decodeCtx, src []byte, ret reflect.Value) ([]byte, error) {
	off, err := decodeInt(src, ret)
	return src[off:], err
}

func decodeUintKind(_ decodeCtx, src []byte, ret reflect.Value) ([]byte, error) {
	off, err := decodeUint(src, ret)
	return src[off:], err
}

func decodeFloatKind(_ decodeCtx, src []byte, ret reflect.Value) ([]byte, error) {
	off, err := decodeFloat(src, ret)
	return src[off:], err
}

func decodeStringKind(_ decodeCtx, src []byte, ret reflect.Value) ([]byte, error) {
	off, err := decodeString(src, ret)
	return src[off:], err
}

func decodeNull(_ decodeCtx, b []byte, v reflect.Value) ([]byte, error) {
	if b, ok := consumeNull(b); ok {
		v.Set(reflect.Zero(v.Type()))
		return b, nil
	}
	return nil, fmt.Errorf("expected null")
}

func decodePointer(decode decoder) decoder {
	return func(ctx decodeCtx, b []byte, v reflect.Value) ([]byte, error) {
		elemType := v.Type().Elem()

		if b, ok := consumeNull(b); ok {
			if !v.IsNil() && elemType.Kind() == reflect.Ptr {
				return decode(ctx, b, v.Elem())
			}
			v.Set(reflect.Zero(elemType))
			return b, nil
		}

		if v.IsNil() {
			v.Set(reflect.New(elemType))
		}

		return decode(ctx, b, v.Elem())
	}
}

func decodeCustom(t reflect.Type) decoder {
	baseImplements := t.Implements(astraCodecType)

	return func(ctx decodeCtx, b []byte, v reflect.Value) ([]byte, error) {
		if !baseImplements && v.CanAddr() {
			v = v.Addr()
		}

		var intermediate any
		b, err := resolveCodecCaching(anyType).decode(ctx, b, reflect.ValueOf(&intermediate).Elem())
		if err != nil {
			return b, err
		}

		if err = v.Interface().(AstraCodec).FromAstraValue(intermediate); err != nil {
			return b, err
		}

		return b, nil
	}
}

func consumeNull(b []byte) ([]byte, bool) {
	b = skipWS(b)
	if len(b) >= 4 && b[0] == 'n' && b[1] == 'u' && b[2] == 'l' && b[3] == 'l' {
		return b[4:], true
	}
	return b, false
}

func skipWS(b []byte) []byte {
	for i := 0; i < len(b); i++ {
		if b[i] > ' ' {
			return b[i:]
		}
	}
	return nil
}
