package serdes

import (
	"fmt"
	"reflect"
	"unsafe"
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

		c, err := resolveCodecCaching(anyType, seenStructs{}, false)
		if err != nil {
			return b, err
		}

		var intermediate any
		b, err = c.decode(ctx, b, reflect.ValueOf(&intermediate).Elem())
		if err != nil {
			return b, err
		}

		if err = v.Interface().(AstraCodec).FromAstraValue(intermediate); err != nil {
			return b, err
		}

		return b, nil
	}
}

func decodeStruct(info *structInfo) decoder {
	return func(ctx decodeCtx, src []byte, v reflect.Value) ([]byte, error) {
		// Ensure base addressability for offset math
		if !v.CanAddr() {
			copy := reflect.New(v.Type()).Elem()
			copy.Set(v)
			v = copy
		}
		basePtr := v.Addr().UnsafePointer()

		src = skipWS(src)
		if len(src) < 2 || src[0] != '{' {
			return src, fmt.Errorf("expected '{'")
		}
		src = src[1:] // skip '{'

		for i := 0; ; i++ {
			src = skipWS(src)
			if len(src) == 0 {
				return src, fmt.Errorf("unexpected end of JSON")
			}
			if src[0] == '}' {
				return src[1:], nil
			}

			if i > 0 {
				if src[0] != ',' {
					return src, fmt.Errorf("expected ',' after field value")
				}
				src = skipWS(src[1:])
			}

			key, n, err := parseStringRaw(src)
			if err != nil {
				return src, err
			}
			src = skipWS(src[n:])

			if len(src) == 0 || src[0] != ':' {
				return src, fmt.Errorf("expected ':' after key")
			}
			src = skipWS(src[1:])

			if fieldIdx, ok := info.offsets[key]; ok {
				f := &info.fields[fieldIdx]

				// Jump to memory location via offset instead of FieldByIndex
				ptr := unsafe.Pointer(uintptr(basePtr) + f.offset)
				fieldValue := reflect.NewAt(f.typ, ptr).Elem()

				src, err = f.codec.decode(ctx, src, fieldValue)
				if err != nil {
					return src, err
				}
			} else {
				n, err := skipValue(src, 0)
				if err != nil {
					return src, err
				}
				src = src[n:]
			}
		}
	}
}

func decodeEmbeddedStructPointer(t reflect.Type, unexported bool, offset uintptr, decode decoder) decoder {
	return func(ctx decodeCtx, src []byte, v reflect.Value) ([]byte, error) {
		// v is the reflect.Value of the pointer field (e.g., *Metadata)

		if v.IsNil() {
			if unexported {
				return nil, fmt.Errorf("json: cannot set embedded pointer to unexported struct: %s", t)
			}
			// Initialize the struct: v.Set(reflect.New(t))
			// reflect.New(t) creates a pointer to the type t
			v.Set(reflect.New(t))
		}

		// v.Elem() is the actual struct we are decoding into.
		// The 'decode' codec here is the field-specific codec for the sub-field.
		return decode(ctx, src, v.Elem())
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
