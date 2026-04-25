package serdes

import (
	"reflect"
	"strconv"
)

type encoder func(ctx encodeCtx, dst []byte, v reflect.Value) ([]byte, error)

type encodeCtx struct {
}

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

		return resolveCodecCaching(resValue.Type(), seenStructs{}, resValue.CanAddr()).
			encode(ctx, dst, resValue)
	}
}
