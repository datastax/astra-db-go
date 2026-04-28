package serdes

import (
	"fmt"
	"reflect"
	"unsafe"
)

type encoder func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error)

type encodeCtx struct {
	codecCtx
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

		c := resolveCodecCaching(ctx.codecCtx, reflect.TypeOf(res), seenStructs{})
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
				dst = append(dst, f.prefix[1:]...) // skip leading comma for the first fieldHint
				firstField = false
			} else {
				dst = append(dst, f.prefix...)
			}

			ctx.fieldHint = extractFieldHint(f.meta.name)

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
	return serializeAppend(*(*any)(p), ctx.target, dst)
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

// small lut for hex conversion
const hexTable = "0123456789abcdef"

// appendString efficiently encodes a string to JSON, but does not handle 100% of edge cases
// It's designed to be fast and simple for all common and even most uncommon cases, but it'll
// rely on the server to cover the last 0.01% of edge cases that are extremely rare in practice
func appendString(dst []byte, s string) []byte {
	dst = append(dst, '"')

	start := 0
	for i := 0; i < len(s); i++ {
		b := s[i]

		if b >= 0x20 && b != '\\' && b != '"' {
			continue
		}

		dst = append(dst, s[start:i]...)

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

	dst = append(dst, s[start:]...)
	return append(dst, '"')
}
