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

func encodeCustom(t reflect.Type) (encoder, error) {
	if !t.Implements(astraCodecType) && !reflect.PointerTo(t).Implements(astraCodecType) {
		return nil, fmt.Errorf("type %v does not implement AstraCodec", t)
	}

	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		if p == nil {
			return append(dst, 0xc0), nil
		}

		// NewAt(t, p) creates a Value representing a pointer to the data at p.
		// Interface() boxes that pointer. Go handles the method dispatch
		// whether AstraCodec is on the value or the pointer.
		codec := reflect.NewAt(t, p).Interface().(AstraCodec)

		res := codec.ToAstraValue()
		c, err := resolveCodecCaching(reflect.TypeOf(res), seenStructs{}, false)
		if err != nil {
			return dst, err
		}

		// We use the address of 'res' because it's an interface{}
		return c.encode(ctx, dst, unsafe.Pointer(&res))
	}, nil
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

func encodeEmbeddedStructPointer(encode encoder) encoder {
	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		p = *(*unsafe.Pointer)(p)
		if p == nil {
			return dst, rollback{}
		}
		return encode(ctx, dst, p)
	}
}
