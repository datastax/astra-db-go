package serdes

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

type decoder func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error)

type decodeCtx struct {
}

func decodeError(err error) decoder {
	return func(_ decodeCtx, src []byte, _ unsafe.Pointer) ([]byte, error) {
		return src, err
	}
}

func decodeBoolKind(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if b, ok := consumeNull(src); ok {
		return b, nil
	}

	if bytes.HasPrefix(src, []byte("true")) {
		*(*bool)(p) = true
		return src[4:], nil
	} else if bytes.HasPrefix(src, []byte("false")) {
		*(*bool)(p) = false
		return src[5:], nil
	}

	return src, fmt.Errorf("expected boolean")
}

func decodeStringKind(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, str, err := decodeString(src)
	if err != nil {
		return src, err
	}

	*(*string)(p) = str
	return src, nil
}

func decodeNull(_ decodeCtx, b []byte, _ unsafe.Pointer) ([]byte, error) {
	if b, ok := consumeNull(b); ok {
		return b, nil
	}
	return nil, fmt.Errorf("expected null")
}

func decodePointer(decode decoder, t reflect.Type) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if b, ok := consumeNull(src); ok {
			pp := *(*unsafe.Pointer)(p)
			if pp != nil {
				return decode(ctx, b, pp)
			}
			*(*unsafe.Pointer)(p) = nil
			return b, nil
		}

		v := *(*unsafe.Pointer)(p)
		if v == nil {
			v = unsafe.Pointer(reflect.New(t).Pointer())
			*(*unsafe.Pointer)(p) = v
		}

		return decode(ctx, src, v)
	}
}

func decodeCustom(t reflect.Type) decoder {
	if !t.Implements(astraCodecType) && !reflect.PointerTo(t).Implements(astraCodecType) {
		return decodeError(fmt.Errorf("type %v does not implement AstraCodec", t))
	}

	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if p == nil {
			return src, fmt.Errorf("cannot decode into nil pointer for %v", t)
		}

		var codec AstraCodec
		codec = reflect.NewAt(t, p).Interface().(AstraCodec)

		c := resolveCodecCaching(reflect.TypeOf((*any)(nil)).Elem(), seenStructs{}, false)

		var intermediate any
		src, err := c.decode(ctx, src, unsafe.Pointer(&intermediate))
		if err != nil {
			return src, err
		}

		if err = codec.FromAstraValue(intermediate); err != nil {
			return src, err
		}

		return src, nil
	}
}

func decodeStruct(info *structInfo) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			return src, nil
		}

		if len(src) == 0 || src[0] != '{' {
			return src, fmt.Errorf("expected '{'")
		}
		src = src[1:] // skip '{'

		var key string
		var err error

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

			src, key, err = decodeString(src)
			if err != nil {
				return src, err
			}
			src = skipWS(src)

			if len(src) == 0 || src[0] != ':' {
				return src, fmt.Errorf("expected ':' after key")
			}
			src = skipWS(src[1:])

			if fieldIdx, ok := info.offsets[key]; ok {
				f := &info.fields[fieldIdx]

				// Jump to memory location via offset instead of FieldByIndex
				ptr := unsafe.Pointer(uintptr(p) + f.offset)

				src, err = f.codec.decode(ctx, src, ptr)
				if err != nil {
					return src, err
				}
			} else {
				src, err = skipValue(src)
			}
		}
	}
}

func decodeEmbeddedStructPointer(t reflect.Type, unexported bool, offset uintptr, decode decoder) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		v := *(*unsafe.Pointer)(p)

		if v == nil {
			if unexported {
				return nil, fmt.Errorf("json: cannot set embedded pointer to unexported struct: %s", t)
			}
			v = unsafe.Pointer(reflect.New(t).Pointer())
			*(*unsafe.Pointer)(p) = v
		}

		return decode(ctx, src, unsafe.Pointer(uintptr(v)+offset))
	}
}

func decodeInt(src []byte) ([]byte, int64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] == '-' || (src[end] >= '0' && src[end] <= '9')) {
		end++
	}

	if end == 0 {
		return src, 0, fmt.Errorf("expected integer")
	}

	num, err := strconv.ParseInt(string(src[:end]), 10, 64)
	return src[end:], num, err
}

func decodeUint(src []byte) ([]byte, uint64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] >= '0' && src[end] <= '9') {
		end++
	}

	if end == 0 {
		return src, 0, fmt.Errorf("expected unsigned integer")
	}

	num, err := strconv.ParseUint(string(src[:end]), 10, 64)
	return src[end:], num, err
}

func decodeFloat(src []byte) ([]byte, float64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] == '-' || src[end] == '.' || (src[end] >= '0' && src[end] <= '9') || src[end] == 'e' || src[end] == 'E' || src[end] == '+') {
		end++
	}

	if end == 0 {
		return src, 0, fmt.Errorf("expected float")
	}

	f, err := strconv.ParseFloat(string(src[:end]), 64)
	return src[end:], f, err
}

func decodeString(src []byte) ([]byte, string, error) {
	src = skipWS(src)
	if len(src) == 0 || src[0] != '"' {
		return src, "", fmt.Errorf("expected string")
	}

	end := 1
	for end < len(src) {
		if src[end] == '\\' {
			end += 2
		} else if src[end] == '"' {
			break
		} else {
			end++
		}
	}

	if end >= len(src) {
		return src, "", fmt.Errorf("unterminated string")
	}

	return src[end+1:], string(src[1:end]), nil
}

func consumeNull(src []byte) ([]byte, bool) {
	src = skipWS(src)
	if len(src) >= 4 && src[0] == 'n' && src[1] == 'u' && src[2] == 'l' && src[3] == 'l' {
		return src[4:], true
	}
	return src, false
}

func skipWS(src []byte) []byte {
	for i := range src {
		if src[i] > ' ' {
			return src[i:]
		}
	}

	return nil
}

func skipValue(src []byte) ([]byte, error) {
	src = skipWS(src)
	if len(src) == 0 {
		return src, errInvalid
	}

	switch src[0] {
	case '"':
		src, _, err := decodeString(src)
		if err != nil {
			return src, err
		}
		return src, nil

	case '{', '[':
		depth, open, cls := 0, src[0], src[0]+2
		if open == '{' {
			cls = '}'
		}
		for i := 0; i < len(src); i++ {
			if src[i] == open {
				depth++
			} else if src[i] == cls {
				depth--
			}
			if depth == 0 {
				return src[i+1:], nil
			}
		}

	default:
		i := 0
		for i < len(src) && src[i] != ',' && src[i] != '}' && src[i] != ']' {
			i++
		}
		return src[i:], nil
	}

	return src, errInvalid
}
