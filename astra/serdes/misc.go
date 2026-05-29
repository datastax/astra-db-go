package serdes

import (
	"bytes"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/datastax/astra-db-go/astra/datatypes"
)

// Misc

func mkInlineEncoder(encode encoder) encoder {
	return func(e EncodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		return encode(e, b, noescape(unsafe.Pointer(&p)))
	}
}

// Errored

func mkErroredCodec(err error) codec {
	return codec{
		mkErrorEncoder(err),
		mkErrorDecoder(err),
	}
}

func mkErrorEncoder(err error) encoder {
	return func(_ EncodeCtx, dst []byte, _ unsafe.Pointer) ([]byte, error) {
		return dst, err
	}
}

func mkErrorDecoder(err error) decoder {
	return func(_ DecodeCtx, src []byte, _ unsafe.Pointer) ([]byte, error) {
		return src, err
	}
}

// Null

var nilCodec = codec{
	nullEncoder,
	nullDecoder,
}

func nullEncoder(_ EncodeCtx, dst []byte, _ unsafe.Pointer) ([]byte, error) {
	return append(dst, "null"...), nil
}

func nullDecoder(ctx DecodeCtx, b []byte, _ unsafe.Pointer) ([]byte, error) {
	if b, ok := consumeNull(b); ok {
		return b, nil
	}
	return b, ctx.syntaxError(b, "expected null")
}

// Booleans

func boolEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if *(*bool)(p) {
		return append(dst, "true"...), nil
	}
	return append(dst, "false"...), nil
}

func boolDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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

	return src, ctx.unmarshalTypeError(src, reflect.TypeFor[bool]())
}

// Strings

func stringEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return appendString(dst, *(*string)(p)), nil
}

func stringDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, str, isNew, err := parseStringUnquote(ctx, src)
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

// Pointers

const startDetectingCyclesAfter = 1000

func mkPointerCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	el := t.Elem()
	c := resolveCodec(ctx, el, seen, true)

	return codec{
		mkPointerEncoder(c.encode),
		mkPointerDecoder(c.decode, el),
	}
}

func mkPointerEncoder(encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		p = *(*unsafe.Pointer)(p)

		if p == nil {
			return nullEncoder(ctx, dst, p)
		}

		if ctx.ptrDepth++; ctx.ptrDepth >= startDetectingCyclesAfter {
			if ctx.ptrSeen == nil {
				ctx.ptrSeen = make(map[unsafe.Pointer]struct{})
			}
			if _, seen := ctx.ptrSeen[p]; seen {
				return dst, &UnsupportedValueError{Msg: fmt.Sprintf("encountered a cycle via pointer %p", p)}
			}
			ctx.ptrSeen[p] = struct{}{}
			defer delete(ctx.ptrSeen, p)
		}

		return encode(ctx, dst, p)
	}
}

func mkPointerDecoder(decode decoder, t reflect.Type) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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

// Custom

func mkAstraMarshalerEncoder(t reflect.Type, isPtr bool) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		v := reflect.NewAt(t, p)
		if !isPtr {
			v = v.Elem()
		}

		k := v.Kind()
		if (k == reflect.Ptr || k == reflect.Interface) && v.IsNil() {
			return append(dst, "null"...), nil
		}

		res, err := v.Interface().(AstraMarshaler).MarshalAstra(ctx)
		if err != nil {
			return dst, &MarshalerError{Type: t, Err: err}
		}

		return emptyInterfaceEncoder(ctx, dst, unsafe.Pointer(&res))
	}
}

func mkAstraRawMarshalerEncoder(t reflect.Type, isPtr bool) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		v := reflect.NewAt(t, p)
		if !isPtr {
			v = v.Elem()
		}

		k := v.Kind()
		if (k == reflect.Ptr || k == reflect.Interface) && v.IsNil() {
			return append(dst, "null"...), nil
		}

		return v.Interface().(AstraRawMarshaler).MarshalAstraRaw(ctx, dst)
	}
}

func mkAstraUnmarshalerDecoder(t reflect.Type) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		var intermediate any
		src, err := emptyInterfaceDecoder(ctx, src, unsafe.Pointer(&intermediate))
		if err != nil {
			return src, err
		}

		u := reflect.NewAt(t, p)
		if u.IsNil() {
			u.Set(reflect.New(t))
		}

		err = u.Interface().(AstraUnmarshaler).UnmarshalAstra(ctx, intermediate)
		if err != nil {
			return src, &UnmarshalerError{Type: t, Err: err}
		}
		return src, nil
	}
}

func mkAstraRawUnmarshalerDecoder(t reflect.Type) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		u := reflect.NewAt(t, p)
		if u.IsNil() {
			u.Set(reflect.New(t))
		}

		srcAfter, err := skipValue(ctx, src)
		if err != nil {
			return src, err
		}

		splitPoint := len(src) - len(srcAfter)

		err = u.Interface().(AstraRawUnmarshaler).UnmarshalAstraRaw(ctx, src[:splitPoint])
		if err != nil {
			return srcAfter, &UnmarshalerError{Type: t, Err: err}
		}
		return srcAfter, nil
	}
}

// Interfaces

func mkSomeInterfaceCodec(t reflect.Type) codec {
	return codec{
		mkSomeInterfaceEncoder(t),
		mkSomeInterfaceDecoder(t),
	}
}

func emptyInterfaceEncoder(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return SerializeInto(*(*any)(p), ctx.Target, dst)
}

func mkSomeInterfaceEncoder(t reflect.Type) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		return SerializeInto(reflect.NewAt(t, p).Elem().Interface(), ctx.Target, dst)
	}
}

func emptyInterfaceDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) == 0 {
		return src, ctx.syntaxError(src, "unexpected end of input")
	}

	var val any
	var err error

	if ctx.fieldHint == vectorField {
		var v datatypes.DataAPIVector
		src, err = vectorDecoder(ctx, src, unsafe.Pointer(&v))
		val = v
		goto decoded
	}

	if ctx.fieldHint == vectorizeField {
		var s string
		src, err = stringDecoder(ctx, src, unsafe.Pointer(&s))
		val = s
		goto decoded
	}

	switch src[0] {
	case 'n':
		if src, ok := consumeNull(src); ok {
			*(*any)(p) = nil
			return src, nil
		}
		return src, ctx.syntaxError(src, "expected null")

	case 't', 'f':
		var b bool
		src, err = boolDecoder(ctx, src, unsafe.Pointer(&b))
		val = b

	case '"':
		var s string
		src, err = stringDecoder(ctx, src, unsafe.Pointer(&s))
		val = s

	case '[':
		var arr []any
		arrType := reflect.TypeOf(arr)
		src, err = mkSliceDecoder(unsafe.Sizeof(arr[0]), arrType, emptyInterfaceDecoder)(ctx, src, unsafe.Pointer(&arr))
		val = arr

	case '{':
		var a any
		var wasDD bool
		if src, err, wasDD = decodeDollarDatatype(ctx, src, unsafe.Pointer(&a)); wasDD {
			val = a
			goto decoded
		}

		var m map[string]any
		kt := stringType
		vt := anyType
		kz := stringEmpty
		vz := anyEmpty
		mt := reflect.MapOf(kt, vt)
		src, err = mkNormalMapDecoder(mt, kt, vt, kz, vz, stringDecoder, emptyInterfaceDecoder, mkNativeMapMaker(mt))(ctx, src, unsafe.Pointer(&m)) // I hate this... hopefully it can be simplified later
		val = m

	default:
		if src[0] == '-' || (src[0] >= '0' && src[0] <= '9') {
			var f float64
			src, err = float64Decoder(ctx, src, unsafe.Pointer(&f)) // TODO figure out better way of handling numbers
			val = f
		} else {
			return src, ctx.syntaxError(src, fmt.Sprintf("unexpected character: %c", src[0]))
		}
	}

decoded:
	if err != nil {
		return src, err
	}

	*(*any)(p) = val
	return src, nil
}

func mkSomeInterfaceDecoder(t reflect.Type) decoder {
	if t.NumMethod() == 0 {
		return emptyInterfaceDecoder
	}

	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if ctx.TargetCtx != nil && ctx.TargetCtx.UntypedTargetInterface() == t {
			srcAfter, err := skipValue(ctx, src)
			if err != nil {
				return src, err
			}

			splitPoint := len(src) - len(srcAfter)
			valueBytes := src[:splitPoint]

			targetInstance := ctx.TargetCtx.NewUntypedTarget(p)
			if err := targetInstance.UnmarshalAstraRaw(ctx, valueBytes); err != nil {
				return src, err
			}

			return srcAfter, nil
		}
		return src, ctx.unmarshalTypeError(src, t)
	}
}

// Raw messages

func rawMessageEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	v := *(*[]byte)(p)

	if v == nil {
		return append(dst, "null"...), nil
	}

	var throwaway any
	_, err := emptyInterfaceDecoder(DecodeCtx{}, v, unsafe.Pointer(&throwaway))
	if err != nil {
		return dst, &MarshalerError{Type: rawMessageType, Err: err}
	}

	return append(dst, v...), nil
}

func rawMessageDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	start := src
	end, err := skipValue(ctx, src)
	if err != nil {
		return src, err
	}

	length := len(start) - len(end)
	copied := make([]byte, length)
	copy(copied, start[:length])

	*(*[]byte)(p) = copied
	return end, nil
}
