package serdes

import (
	"bytes"
	"fmt"
	"reflect"
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
	src, str, isNew, err := parseStringUnquote(src)
	if err != nil {
		return src, err
	}

	if isNew {
		*(*string)(p) = unsafeString(str)
	} else {
		*(*string)(p) = string(str)
	}

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

		c, _ := resolveCodec(reflect.TypeOf((*any)(nil)).Elem(), seenStructs{}, false)

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

		var key []byte
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

			src, key, _, err = parseStringUnquote(src)
			if err != nil {
				return src, err
			}
			src = skipWS(src)

			if len(src) == 0 || src[0] != ':' {
				return src, fmt.Errorf("expected ':' after key")
			}
			src = skipWS(src[1:])

			if fieldIdx, ok := info.offsets[unsafeString(key)]; ok {
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
		return src, fmt.Errorf("unexpected end of input")
	}

	switch src[0] {
	case '"':
		src, _, _, err := parseString(src)
		if err != nil {
			return src, err
		}
		return src, nil

	case '{', '[':
		depth, open, cls := 0, src[0], byte('}')
		if open == '[' {
			cls = ']'
		}

		for i := 0; i < len(src); i++ {
			if src[i] == open {
				depth++
			} else if src[i] == cls {
				depth--
				if depth == 0 {
					return src[i+1:], nil
				}
			}
		}
		return src, fmt.Errorf("unexpected end of input while skipping value")

	default:
		i := 0
		for i < len(src) && src[i] != ',' && src[i] != '}' && src[i] != ']' {
			i++
		}
		return src[i:], nil
	}
}

var empty struct{}

func decodeMap(kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder) decoder {
	return func(ctx decodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		if b, ok := consumeNull(b); ok {
			*(*unsafe.Pointer)(p) = nil
			return b, nil
		}

		if len(b) < 2 || b[0] != '{' {
			return b, fmt.Errorf("expected '{'")
		}
		i := 0
		m := reflect.NewAt(reflect.MapOf(kt, vt), p).Elem()

		k := reflect.New(kt).Elem()
		v := reflect.New(vt).Elem()

		kptr := (*iface)(unsafe.Pointer(&k)).ptr
		vptr := (*iface)(unsafe.Pointer(&v)).ptr

		if m.IsNil() {
			m = reflect.MakeMap(reflect.MapOf(kt, vt))
		}

		var err error
		b = b[1:]
		for {
			k.Set(kz)
			v.Set(vz)
			b = skipWS(b)

			if len(b) != 0 && b[0] == '}' {
				*(*unsafe.Pointer)(p) = unsafe.Pointer(m.Pointer())
				return b[1:], nil
			}

			if i != 0 {
				if len(b) == 0 {
					return b, fmt.Errorf("unexpected end of JSON input after object field value")
				}
				if b[0] != ',' {
					return b, fmt.Errorf("expected ',' after object field value but found '%c'", b[0])
				}
				b = skipWS(b[1:])
			}

			if b, ok := consumeNull(b); ok {
				return b, fmt.Errorf("cannot decode object key string from 'null' value")
			}

			if b, err = decodeKey(ctx, b, kptr); err != nil {
				return b, err
			}
			b = skipWS(b)

			if len(b) == 0 {
				return b, fmt.Errorf("unexpected end of JSON input after object field key")
			}
			if b[0] != ':' {
				return b, fmt.Errorf("expected ':' after object field key but found '%c'", b[0])
			}
			b = skipWS(b[1:])

			if b, err = decodeValue(ctx, b, vptr); err != nil {
				return b, err
			}

			m.SetMapIndex(k, v)
			i++
		}
	}
}

func decodeSlice(size uintptr, t reflect.Type, decode decoder) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			*(*slice)(p) = slice{}
			return src, nil
		}

		if len(src) == 0 || src[0] != '[' {
			return src, fmt.Errorf("expected '['")
		}
		src = src[1:]

		s := (*slice)(p)
		s.len = 0

		for {
			src = skipWS(src)

			if len(src) != 0 && src[0] == ']' {
				if s.data == nil {
					s.data = unsafe.Pointer(&empty)
				}
				return src[1:], nil
			}

			if s.len != 0 {
				if len(src) == 0 || src[0] != ',' {
					return src, fmt.Errorf("expected ','")
				}
				src = skipWS(src[1:])
			}

			if s.len == s.cap {
				c := s.cap
				if c == 0 {
					c = 10
				} else {
					c *= 2
				}
				*s = extendSlice(t, s, c)
			}

			elemPtr := unsafe.Pointer(uintptr(s.data) + uintptr(s.len)*size)
			var err error
			src, err = decode(ctx, src, elemPtr)
			if err != nil {
				return src, err
			}

			s.len++
		}
	}
}

func extendSlice(t reflect.Type, s *slice, newCap int) slice {
	newSlice := reflect.MakeSlice(t, s.len, newCap)
	if s.len > 0 {
		reflect.Copy(newSlice, reflect.NewAt(t, unsafe.Pointer(s)).Elem())
	}
	return slice{
		data: unsafe.Pointer(newSlice.Pointer()),
		len:  s.len,
		cap:  newCap,
	}
}

func decodeArray(n int, size uintptr, t reflect.Type, decode decoder) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			return src, nil
		}

		if len(src) == 0 || src[0] != '[' {
			return src, fmt.Errorf("expected '['")
		}
		src = src[1:]

		for i := 0; i < n; i++ {
			src = skipWS(src)

			if i > 0 {
				if len(src) == 0 || src[0] != ',' {
					return src, fmt.Errorf("expected ','")
				}
				src = skipWS(src[1:])
			}

			elemPtr := unsafe.Pointer(uintptr(p) + uintptr(i)*size)
			var err error
			src, err = decode(ctx, src, elemPtr)
			if err != nil {
				return src, err
			}
		}

		src = skipWS(src)

		for {
			if len(src) == 0 {
				return src, fmt.Errorf("expected ']'")
			}

			if src[0] == ']' {
				return src[1:], nil
			}

			if src[0] != ',' {
				return src, fmt.Errorf("expected ',' or ']'")
			}
			src = skipWS(src[1:])

			src, err := skipValue(src)
			if err != nil {
				return src, err
			}
		}
	}
}

func decodeInterface(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) == 0 {
		return src, fmt.Errorf("unexpected end of input")
	}

	var val any
	var err error

	switch src[0] {
	case 'n':
		if src, ok := consumeNull(src); ok {
			*(*any)(p) = nil
			return src, nil
		}
		return src, fmt.Errorf("expected null")

	case 't', 'f':
		var b bool
		src, err = decodeBoolKind(decodeCtx{}, src, unsafe.Pointer(&b))
		val = b

	case '"':
		var s string
		src, err = decodeStringKind(decodeCtx{}, src, unsafe.Pointer(&s))
		val = s

	case '[':
		var arr []any
		arrType := reflect.TypeOf(arr)
		src, err = decodeSlice(unsafe.Sizeof(arr[0]), arrType, decodeInterface)(decodeCtx{}, src, unsafe.Pointer(&arr))
		val = arr

	case '{':
		var m map[string]any
		kt := reflect.TypeOf("")
		vt := reflect.TypeOf((*any)(nil)).Elem()
		kz := reflect.Zero(kt)
		vz := reflect.Zero(vt)
		src, err = decodeMap(kt, vt, kz, vz, decodeStringKind, decodeInterface)(decodeCtx{}, src, unsafe.Pointer(&m))
		val = m

	default:
		if src[0] == '-' || (src[0] >= '0' && src[0] <= '9') {
			var f float64
			src, err = decodeFloat64Kind(decodeCtx{}, src, unsafe.Pointer(&f))
			val = f
		} else {
			return src, fmt.Errorf("unexpected character: %c", src[0])
		}
	}

	if err != nil {
		return src, err
	}

	*(*any)(p) = val
	return src, nil
}

func decodeRawMessage(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	start := src
	end, err := skipValue(src)
	if err != nil {
		return src, err
	}

	length := len(start) - len(end)
	copied := make([]byte, length)
	copy(copied, start[:length])

	*(*[]byte)(p) = copied
	return end, nil
}
