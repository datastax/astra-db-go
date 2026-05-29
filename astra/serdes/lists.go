package serdes

import (
	"fmt"
	"reflect"
	"unsafe"
)

// this file is full of ugly gross code that I plan to revisit later

func mkSliceCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	elem := t.Elem()
	c := resolveCodec(ctx, elem, seen, true)
	size := alignedSize(elem)

	return codec{
		mkSliceEncoder(size, c.encode),
		mkSliceDecoder(size, t, c.decode),
	}
}

func mkArrayCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) codec {
	elem := t.Elem()
	size := alignedSize(elem)
	c := resolveCodec(ctx, elem, seen, canAddr)
	n := t.Len()

	return codec{
		mkArrayEncoder(n, size, c.encode),
		mkArrayDecoder(n, size, t, c.decode),
	}
}

func encodeArray(ctx EncodeCtx, dst []byte, p unsafe.Pointer, n int, size uintptr, encode encoder) ([]byte, error) {
	start := len(dst)
	var err error
	dst = append(dst, '[')

	for i := range n {
		if i != 0 {
			dst = append(dst, ',')
		}
		if dst, err = encode(ctx, dst, unsafe.Pointer(uintptr(p)+(uintptr(i)*size))); err != nil {
			return dst[:start], err
		}
	}

	dst = append(dst, ']')
	return dst, nil
}

func mkSliceEncoder(size uintptr, encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		s := (*slice)(p)

		if s.data == nil && s.len == 0 && s.cap == 0 {
			return append(dst, "null"...), nil
		}

		return encodeArray(ctx, dst, s.data, s.len, size, encode)
	}
}

func mkArrayEncoder(n int, size uintptr, encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		return encodeArray(ctx, dst, p, n, size, encode)
	}
}

func mkSliceDecoder(size uintptr, t reflect.Type, decode decoder) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			*(*slice)(p) = slice{}
			return src, nil
		}

		if len(src) == 0 || src[0] != '[' {
			return src, ctx.unmarshalTypeError(src, t)
		}
		src = src[1:]

		s := (*slice)(p)
		s.len = 0

		for {
			src = skipWS(src)

			if len(src) != 0 && src[0] == ']' {
				if s.data == nil {
					s.data = unsafe.Pointer(&struct{}{})
				}
				return src[1:], nil
			}

			if s.len != 0 {
				if len(src) == 0 || src[0] != ',' {
					return src, ctx.syntaxError(src, "expected ','")
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
				return src, wrapPath(err, fmt.Sprintf("[%d]", s.len))
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

func mkArrayDecoder(n int, size uintptr, t reflect.Type, decode decoder) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			return src, nil
		}

		if len(src) == 0 || src[0] != '[' {
			return src, ctx.unmarshalTypeError(src, t)
		}
		src = src[1:]

		for i := 0; i < n; i++ {
			src = skipWS(src)

			if i > 0 {
				if len(src) == 0 || src[0] != ',' {
					return src, ctx.syntaxError(src, "expected ','")
				}
				src = skipWS(src[1:])
			}

			elemPtr := unsafe.Pointer(uintptr(p) + uintptr(i)*size)
			var err error
			src, err = decode(ctx, src, elemPtr)
			if err != nil {
				return src, wrapPath(err, fmt.Sprintf("[%d]", i))
			}
		}

		src = skipWS(src)

		for {
			if len(src) == 0 {
				return src, ctx.syntaxError(src, "expected ']'")
			}

			if src[0] == ']' {
				return src[1:], nil
			}

			if src[0] != ',' {
				return src, ctx.syntaxError(src, "expected ',' or ']'")
			}
			src = skipWS(src[1:])

			var err error
			src, err = skipValue(ctx, src)
			if err != nil {
				return src, err
			}
		}
	}
}
