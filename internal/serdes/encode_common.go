package serdes

import (
	"fmt"
	"reflect"
	"unsafe"
)

type encoder func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error)

type encodeCtx struct {
	ptrDepth int
	ptrSeen  map[unsafe.Pointer]struct{}
}

const startDetectingCyclesAfter = 1000

func encodeError(err error) encoder {
	return func(_ encodeCtx, dst []byte, _ unsafe.Pointer) ([]byte, error) {
		return dst, err
	}
}

func encodeInlined(encode encoder) encoder {
	return func(e encodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		return encode(e, b, noescape(unsafe.Pointer(&p)))
	}
}

func encodeBoolKind(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if *(*bool)(p) {
		return append(dst, "true"...), nil
	}
	return append(dst, "false"...), nil
}

func encodeStringKind(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return appendString(dst, *(*string)(p)), nil
}

func encodeNull(_ encodeCtx, dst []byte, _ unsafe.Pointer) ([]byte, error) {
	return append(dst, "null"...), nil
}

func encodePointer(encode encoder) encoder {
	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		p = *(*unsafe.Pointer)(p)

		if p == nil {
			return encodeNull(ctx, dst, p)
		}

		if ctx.ptrDepth++; ctx.ptrDepth >= startDetectingCyclesAfter {
			if ctx.ptrSeen == nil {
				ctx.ptrSeen = make(map[unsafe.Pointer]struct{})
			}
			if _, seen := ctx.ptrSeen[p]; seen {
				return dst, fmt.Errorf("encountered a cycle via pointer %p", p)
			}
			ctx.ptrSeen[p] = struct{}{}
			defer delete(ctx.ptrSeen, p)
		}

		return encode(ctx, dst, p)
	}
}

func encodeCustom(t reflect.Type) encoder {
	if !t.Implements(astraCodecType) && !reflect.PointerTo(t).Implements(astraCodecType) {
		return encodeError(fmt.Errorf("type %v does not implement AstraCodec", t))
	}

	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		if p == nil {
			return append(dst, 0xc0), nil
		}

		codec := reflect.NewAt(t, p).Interface().(AstraCodec)
		res := codec.ToAstraValue()

		return resolveCodecCaching(reflect.TypeOf(res), seenStructs{}, false).
			encode(ctx, dst, unsafe.Pointer(&res))
	}
}

func encodeStruct(info *structInfo) encoder {
	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		start := len(dst)
		dst = append(dst, '{')
		firstField := true

		for i := range info.fields {
			f := &info.fields[i]
			v := unsafe.Pointer(uintptr(p) + f.offset)

			//if f.meta.omitempty && isZeroValue(v, f.typ) { TODO
			//	continue
			//}

			lengthBeforeKey := len(dst)

			if firstField {
				dst = append(dst, f.prefix[1:]...) // skip leading comma for the first field
				firstField = false
			} else {
				dst = append(dst, f.prefix...)
			}

			var err error
			if dst, err = f.codec.encode(ctx, dst, v); err != nil {
				//goland:noinspection GoTypeAssertionOnErrors
				if _, ok := err.(rollback); ok {
					dst = dst[:lengthBeforeKey]
					continue
				}
				return dst[:start], err
			}
		}

		return append(dst, '}'), nil
	}
}

type rollback struct{}

func (rollback) Error() string { return "rollback" }

func encodeMap(t reflect.Type, encodeKey, encodeValue encoder) encoder {
	return func(ctx encodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		m := reflect.NewAt(t, p).Elem()
		if m.IsNil() {
			return append(b, "null"...), nil
		}

		keys := m.MapKeys()

		start := len(b)
		var err error
		b = append(b, '{')

		for i, k := range keys {
			v := m.MapIndex(k)

			if i != 0 {
				b = append(b, ',')
			}

			if b, err = encodeKey(ctx, b, (*iface)(unsafe.Pointer(&k)).ptr); err != nil {
				return b[:start], err
			}

			b = append(b, ':')

			if b, err = encodeValue(ctx, b, (*iface)(unsafe.Pointer(&v)).ptr); err != nil {
				return b[:start], err
			}
		}

		b = append(b, '}')
		return b, nil
	}
}

func encodeSlice(size uintptr, encode encoder) encoder {
	return func(ctx encodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		s := (*slice)(p)

		if s.data == nil && s.len == 0 && s.cap == 0 {
			return append(b, "null"...), nil
		}

		return encodeArray(s.len, size, encode)(ctx, b, s.data)
	}
}

func encodeArray(n int, size uintptr, encode encoder) encoder {
	return func(ctx encodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		start := len(b)
		var err error
		b = append(b, '[')

		for i := range n {
			if i != 0 {
				b = append(b, ',')
			}
			if b, err = encode(ctx, b, unsafe.Pointer(uintptr(p)+(uintptr(i)*size))); err != nil {
				return b[:start], err
			}
		}

		b = append(b, ']')
		return b, nil
	}
}

func encodeInterface(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	val := *(*any)(p)

	if val == nil {
		return append(dst, "null"...), nil
	}

	t := reflect.TypeOf(val)
	c := resolveCodecCaching(t, seenStructs{}, false)

	// Extract the actual pointer to the value from the interface
	valPtr := (*iface)(unsafe.Pointer(&val)).ptr

	return c.encode(ctx, dst, valPtr)
}

func encodeEmbeddedStructPointer(encode encoder) encoder {
	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		p = *(*unsafe.Pointer)(p)
		if p == nil {
			return dst, rollback{}
		}
		return encode(ctx, dst, p)
	}
}
