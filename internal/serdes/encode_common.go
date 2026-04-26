package serdes

import (
	"fmt"
	"reflect"
	"strconv"
)

type encoder func(ctx encodeCtx, dst []byte, v reflect.Value) ([]byte, error)

type encodeCtx struct {
	ptrDepth int
	ptrSeen  map[uintptr]struct{}
}

const startDetectingCyclesAfter = 1000

func encodeBoolKind(_ encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
	return strconv.AppendBool(dst, v.Bool()), nil
}

func encodeIntKind(_ encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
	return strconv.AppendInt(dst, v.Int(), 10), nil
}

func encodeUintKind(_ encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
	return strconv.AppendUint(dst, v.Uint(), 10), nil
}

func encodeFloatKind(_ encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
	return strconv.AppendFloat(dst, v.Float(), 'g', -1, 64), nil
}

func encodeStringKind(_ encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
	return appendString(dst, v.String()), nil
}

func encodeNull(_ encodeCtx, dst []byte, _ reflect.Value) ([]byte, error) {
	return append(dst, "null"...), nil
}

func encodePointer(encode encoder) encoder {
	return func(ctx encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
		if v.IsNil() {
			return encodeNull(ctx, dst, v)
		}

		// Cycle detection
		ptr := v.Pointer()
		if ctx.ptrDepth++; ctx.ptrDepth >= startDetectingCyclesAfter {
			if ctx.ptrSeen == nil {
				ctx.ptrSeen = make(map[uintptr]struct{})
			}
			if _, seen := ctx.ptrSeen[ptr]; seen {
				return dst, fmt.Errorf("encountered a cycle via %s", v.Type())
			}
			ctx.ptrSeen[ptr] = struct{}{}
			defer delete(ctx.ptrSeen, ptr)
		}

		return encode(ctx, dst, v.Elem())
	}
}

func encodeCustom(t reflect.Type) encoder {
	baseImplements := t.Implements(astraCodecType)

	return func(ctx encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
		if baseImplements && v.CanAddr() {
			v = v.Addr()
		}

		if v.Kind() == reflect.Pointer && v.IsNil() {
			return encodeNull(ctx, dst, v)
		}

		res := v.Interface().(AstraCodec).ToAstraValue()
		resValue := reflect.ValueOf(res)

		c, err := resolveCodecCaching(resValue.Type(), seenStructs{}, resValue.CanAddr())
		if err != nil {
			return dst, err
		}

		return c.encode(ctx, dst, resValue)
	}
}

func encodeStruct(info *structInfo) encoder {
	return func(ctx encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
		//basePtr := (*[3]unsafe.Pointer)(unsafe.Pointer(&v))[1]

		start := len(dst)
		dst = append(dst, '{')

		n := 0
		for i := range info.fields {
			f := &info.fields[i]

			// Calculate field address using the flattened offset
			//ptr := unsafe.Pointer(uintptr(basePtr) + f.offset)

			// Reconstruct the reflect.Value from the pointer
			//fieldValue := reflect.NewAt(f.typ, ptr).Elem()
			fieldValue := v.FieldByIndex(f.path)

			if f.meta.omitempty && fieldValue.IsZero() {
				continue
			}

			lengthBeforeKey := len(dst)

			if n != 0 {
				dst = append(dst, f.prefix...)
			} else {
				dst = append(dst, f.prefix[1:]...) // skip leading comma for the first field
			}

			var err error
			dst, err = f.codec.encode(ctx, dst, fieldValue)

			if err != nil {
				//goland:noinspection GoTypeAssertionOnErrors
				if _, ok := err.(rollback); ok {
					dst = dst[:lengthBeforeKey]
					continue
				}
				return dst[:start], err
			}
			n++
		}

		return append(dst, '}'), nil
	}
}

type rollback struct{}

func (rollback) Error() string { return "rollback" }

func encodeEmbeddedStructPointer(encode encoder) encoder {
	return func(ctx encodeCtx, dst []byte, v reflect.Value) ([]byte, error) {
		if v.IsNil() {
			return dst, rollback{}
		}
		// Dereference to the actual struct and pass it to the next encoder
		return encode(ctx, dst, v.Elem())
	}
}
