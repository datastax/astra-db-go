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

	//return strconv.AppendQuote(dst, *(*string)(p)), nil
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

		c, _ := resolveCodec(reflect.TypeOf(res), seenStructs{}, false)
		return c.encode(ctx, dst, unsafe.Pointer(&res))
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

		fmt.Printf("Encoding map of type %s with %s\n", t, m)
		//fmt.Printf("Encoding map of type %s\n", t)

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
	c, _ := resolveCodec(t, seenStructs{}, false)

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

func encodeRawMessage(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	v := *(*[]byte)(p)

	if v == nil {
		return append(dst, "null"...), nil
	}

	_, err := decodeInterface(decodeCtx{}, v, unsafe.Pointer(&v))
	if err != nil {
		return dst, fmt.Errorf("invalid raw message: %w", err)
	}

	return append(dst, v...), nil
}

// Small lookup table for hex conversion
const hexTable = "0123456789abcdef"

func appendString(dst []byte, s string) []byte {
	// 1. Pre-allocate: At minimum, we need the original length + 2 quotes.
	// This reduces the number of heap allocations during append.
	dst = append(dst, '"')

	start := 0
	for i := 0; i < len(s); i++ {
		b := s[i]

		// Fast path: ASCII characters that don't need escaping.
		// We skip them to avoid frequent small appends.
		if b >= 0x20 && b != '\\' && b != '"' {
			continue
		}

		// Flush the "clean" segment we've skipped over so far
		dst = append(dst, s[start:i]...)

		// 2. Optimized switch: Use direct byte appends for common escapes.
		switch b {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			// 3. Simplified hex escape for control characters (0x00 - 0x1F)
			// JSON spec requires \u00xx for these.
			// Replacing strconv.QuoteRune with a manual hex append is much faster.
			dst = append(dst, '\\', 'u', '0', '0', hexTable[b>>4], hexTable[b&0x0f])
		}
		start = i + 1
	}

	// Append the remaining tail and the closing quote
	dst = append(dst, s[start:]...)
	return append(dst, '"')
}
