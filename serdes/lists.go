package serdes

import (
	"fmt"
	"reflect"
	"unsafe"
)

// this file is full of ugly gross code that I plan to revisit later

func mkSetCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt, _ := t.FieldByName("kType")
	et := kt.Type.Elem()

	c := resolveCodec(ctx, et, seen, false)

	return codec{
		mkSetEncoder(t, c.encode),
		mkSetDecoder(et, t, c.decode),
	}
}

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
		mkArrayDecoder(n, size, c.decode),
	}
}

func mkSetEncoder(setT reflect.Type, encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		m := reflect.NewAt(setT, p).Elem()

		if mapIsNil(m) {
			return append(dst, "null"...), nil
		}

		var err error
		first := true

		dst = append(dst, '[')

		it := newMapIterFromSortedMap(m)
		for it.Next() {
			if !first {
				dst = append(dst, ',')
			}
			first = false

			if dst, err = encode(ctx, dst, valuePtr(it.Key())); err != nil {
				return dst, err
			}
		}

		dst = append(dst, ']')
		return dst, nil
	}
}

func mkSliceEncoder(size uintptr, encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		s := (*slice)(p)

		if s.data == nil && s.len == 0 && s.cap == 0 {
			return append(dst, "null"...), nil
		}

		return mkArrayEncoder(s.len, size, encode)(ctx, dst, s.data)
	}
}

func mkArrayEncoder(n int, size uintptr, encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
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
}

func mkSetDecoder(kt, setT reflect.Type, decode decoder) decoder {
	maker := mkSortedMapMaker(setT)

	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			reflect.NewAt(setT, p).Elem().Set(reflect.Zero(setT))
			return src, nil
		}

		if len(src) == 0 || src[0] != '[' {
			return src, fmt.Errorf("expected '['")
		}
		src = src[1:]

		m := reflect.NewAt(setT, p).Elem()

		if mapIsNil(m) {
			*(*unsafe.Pointer)(p) = *(*unsafe.Pointer)(valuePtr(maker.makeMap()))
		}

		var err error

		k := reflect.New(kt).Elem()
		kptr := valuePtr(k)

		for {
			src = skipWS(src)

			if len(src) != 0 && src[0] == ']' {
				return src[1:], nil
			}

			k.Set(reflect.Zero(kt))

			src, err = decode(ctx, src, kptr)
			if err != nil {
				return src, err
			}

			maker.setMap(m, k, emptyEmpty)
			src = skipWS(src)

			if len(src) != 0 && src[0] == ',' {
				src = skipWS(src[1:])
			}
		}
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
			return src, fmt.Errorf("expected '['")
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

func mkArrayDecoder(n int, size uintptr, decode decoder) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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
