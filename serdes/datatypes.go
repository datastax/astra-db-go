package serdes

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

// UUIDs

func encodeUUID(dst []byte, p unsafe.Pointer) ([]byte, error) {
	dst = append(dst, '"')
	dst = append(dst, (*(*datatypes.UUID)(p)).String()...)
	dst = append(dst, '"')
	return dst, nil
}

func decodeUUID(src []byte) ([]byte, datatypes.UUID, error) {
	src, str, _, err := parseStringUnquote(src)
	if err != nil {
		return src, datatypes.UUID{}, fmt.Errorf("invalid UUID string: %w", err)
	}

	uuid, err := datatypes.ParseUUID(unsafeString(str))
	if err != nil {
		return src, datatypes.UUID{}, fmt.Errorf("invalid UUID string: %w", err)
	}

	return src, uuid, nil
}

// ObjectIDs

func encodeObjectID(dst []byte, p unsafe.Pointer) ([]byte, error) {
	dst = append(dst, '"')
	dst = append(dst, (*(*datatypes.ObjectId)(p)).String()...)
	dst = append(dst, '"')
	return dst, nil
}

func decodeObjectID(src []byte) ([]byte, datatypes.ObjectId, error) {
	src, str, _, err := parseStringUnquote(src)
	if err != nil {
		return src, datatypes.ObjectId{}, fmt.Errorf("invalid ObjectId string: %w", err)
	}

	objectID, err := datatypes.ParseObjectId(unsafeString(str))
	if err != nil {
		return src, datatypes.ObjectId{}, fmt.Errorf("invalid ObjectId string: %w", err)
	}

	return src, objectID, nil
}

// big.Int

func encodeBigInt(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	bi := *(*big.Int)(p)
	return append(dst, bi.String()...), nil
}

func decodeBigInt(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if b, ok := consumeNull(src); ok {
		return b, nil
	}

	src, numStr, err := parseNumber(src)
	if err != nil {
		return src, fmt.Errorf("invalid big.Int: %w", err)
	}

	var bi big.Int
	if _, ok := bi.SetString(unsafeString(numStr), 10); !ok {
		return src, fmt.Errorf("invalid big.Int value: %s", numStr)
	}

	*(*big.Int)(p) = bi
	return src, nil
}

// big.Float

func encodeBigFloat(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	bf := *(*big.Float)(p)
	return append(dst, bf.Text('g', -1)...), nil
}

func decodeBigFloat(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if b, ok := consumeNull(src); ok {
		return b, nil
	}

	src, numStr, err := parseNumber(src)
	if err != nil {
		return src, fmt.Errorf("invalid big.Float: %w", err)
	}

	var bf big.Float
	if _, ok := bf.SetString(unsafeString(numStr)); !ok {
		return src, fmt.Errorf("invalid big.Float value: %s", numStr)
	}

	*(*big.Float)(p) = bf
	return src, nil
}

// Maps

func mkMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs, mkCodec func(kt, vt reflect.Type, kz, vz reflect.Value, vc codec) codec) codec {
	kt := t.Key()
	vt := t.Elem()

	kz := reflect.Zero(kt)
	vz := reflect.Zero(vt)

	vc, _ := resolveCodec(ctx, vt, seen, false)

	if inlined(vt) {
		vc.encode = encodeInlined(vc.encode)
	}

	return mkCodec(kt, vt, kz, vz, vc)
}

func encodeMap(t, kt reflect.Type, encodeKey, encodeValue encoder, open, close, sep byte) encoder {
	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		m := reflect.NewAt(t, p).Elem()
		if m.IsNil() {
			return append(dst, "null"...), nil
		}

		if m.Len() == 0 {
			return append(dst, "{}"...), nil
		}

		start := len(dst)
		toArray := open == '[' // && close == ']'

		iter := m.MapRange()
		first := true
		var err error

		dst = append(dst, open)

		for iter.Next() {
			if !first {
				dst = append(dst, ',')
			}
			first = false

			if toArray {
				dst = append(dst, open)
			}

			if dst, err = encodeKey(ctx, dst, valuePtr(iter.Key())); err != nil {
				return dst[:start], err
			}

			dst = append(dst, sep)

			if kt.Kind() == reflect.String {
				ctx.fieldHint = extractFieldHint(iter.Key().String())
			}

			if dst, err = encodeValue(ctx, dst, valuePtr(iter.Value())); err != nil {
				return dst[:start], err
			}

			if toArray {
				dst = append(dst, close)
			}
		}

		return append(dst, close), nil
	}
}

func decodeMap(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, open, close, sep byte) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if b, ok := consumeNull(src); ok {
			*(*unsafe.Pointer)(p) = nil
			return b, nil
		}

		if len(src) < 2 || src[0] != open {
			return src, fmt.Errorf("expected '%c'", open)
		}

		m := reflect.NewAt(t, p).Elem()
		if m.IsNil() {
			m = reflect.MakeMap(t)
		}

		k := reflect.New(kt).Elem()
		v := reflect.New(vt).Elem()
		kptr, vptr := valuePtr(k), valuePtr(v)

		fromArray := open == '[' // && close == ']'

		src = src[1:]
		for i := 0; ; i++ {
			src = skipWS(src)

			if len(src) != 0 && src[0] == close {
				*(*unsafe.Pointer)(p) = unsafe.Pointer(m.Pointer())
				return src[1:], nil
			}

			if i != 0 {
				if len(src) == 0 {
					return src, fmt.Errorf("unexpected end of JSON")
				}
				if src[0] != ',' {
					return src, fmt.Errorf("expected ',' but found '%c'", src[0])
				}
				src = skipWS(src[1:])
			}

			k.Set(kz)
			v.Set(vz)

			if fromArray {
				if len(src) == 0 || src[0] != '[' {
					return src, fmt.Errorf("expected '[' for table entry")
				}
				src = skipWS(src[1:])
			}

			var err error
			if src, err = decodeKey(ctx, src, kptr); err != nil {
				return src, err
			}
			src = skipWS(src)

			if len(src) == 0 || src[0] != sep {
				return src, fmt.Errorf("expected '%c' after key", sep)
			}
			src = skipWS(src[1:])

			if kt.Kind() == reflect.String {
				ctx.fieldHint = extractFieldHint(k.String())
			}

			if src, err = decodeValue(ctx, src, vptr); err != nil {
				return src, err
			}
			src = skipWS(src)

			if fromArray {
				if len(src) == 0 || src[0] != ']' {
					return src, fmt.Errorf("expected ']' after table entry")
				}
				src = skipWS(src[1:])
			}

			m.SetMapIndex(k, v)
		}
	}
}

// Vectors

func encodeVector(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("binary"), func(b []byte) ([]byte, error) {
		dst = append(dst, '"')
		dst = append(dst, (*datatypes.DataAPIVector)(p).AsBase64()...)
		dst = append(dst, '"')
		return dst, nil
	})
}

func decodeVector(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) == 0 || src[0] == '[' {
		var arr []float32
		src, err := decodeSlice(4, float32SliceType, decodeFloat32Kind)(ctx, src, unsafe.Pointer(&arr))
		if err == nil {
			vector := datatypes.NewVector(arr)
			*(*datatypes.DataAPIVector)(p) = vector
		}
		return src, err
	}

	if len(src) == 0 || src[0] == '{' {
		src, vector, err := parseDollarDatatype(src, []byte("binary"), func(b []byte) ([]byte, datatypes.DataAPIVector, error) {
			src, str, isNew, err := parseStringUnquote(b)
			if err != nil {
				return src, datatypes.DataAPIVector{}, fmt.Errorf("invalid vector string: %w", err)
			}

			if isNew {
				return src, datatypes.NewVector(unsafeString(str)), nil
			}
			return src, datatypes.NewVector(string(str)), nil
		})

		if err == nil {
			*(*datatypes.DataAPIVector)(p) = vector
		}
		return src, err
	}

	return src, fmt.Errorf("expected []float32 or {\"$binary\":\"<base64>\"} for vector value")
}

// Binary

func encodeBinary(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("binary"), func(b []byte) ([]byte, error) {
		dst = append(dst, '"')
		dst = append(dst, *(*[]byte)(p)...)
		dst = append(dst, '"')
		return dst, nil
	})
}

func decodeBinary(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) == 0 || src[0] == '"' {
		src, str, _, err := parseStringUnquote(src)
		if err != nil {
			return src, err
		}

		*(*[]byte)(p) = str
		return src, nil
	}

	if len(src) == 0 || src[0] == '{' {
		src, data, err := parseDollarDatatype(src, []byte("binary"), func(b []byte) ([]byte, []byte, error) {
			src, str, _, err := parseStringUnquote(b)
			if err != nil {
				return src, nil, fmt.Errorf("invalid binary string: %w", err)
			}
			return src, str, nil
		})

		if err == nil {
			*(*[]byte)(p) = data
		}
		return src, err
	}

	if len(src) == 0 || src[0] == '[' {
		var arr []byte
		src, err := decodeSlice(1, byteSliceType, decodeUint8Kind)(ctx, src, unsafe.Pointer(&arr))
		if err == nil {
			*(*[]byte)(p) = arr
		}
		return src, err
	}

	return src, fmt.Errorf("expected string, []byte, or {\"$binary\":\"<base64>\"} for []byte value")
}

// Helpers

func encodeDollarDatatype(dst []byte, datatype []byte, encode func([]byte) ([]byte, error)) ([]byte, error) {
	dst = append(dst, "{\"$"...)
	dst = append(dst, datatype...)
	dst = append(dst, "\":"...)
	dst, err := encode(dst)
	if err != nil {
		return dst, err
	}
	dst = append(dst, '}')
	return dst, nil
}

func parseDollarDatatype[T any](src []byte, datatype []byte, decode func([]byte) ([]byte, T, error)) ([]byte, T, error) {
	var zero T

	src = skipWS(src)
	if len(src) == 0 || src[0] != '{' {
		return src, zero, fmt.Errorf("error parsing %s: expected object", datatype)
	}

	src = skipWS(src[1:])
	if !bytes.HasPrefix(src, []byte(`"$`)) {
		return src, zero, fmt.Errorf("error parsing %s: expected object of the format {\"$%s\":...}", datatype, datatype)
	}

	src = skipWS(src[2:])
	if len(src) < len(datatype) || !bytes.Equal(src[:len(datatype)], datatype) {
		return src, zero, fmt.Errorf("error parsing %s: expected object of the format {\"$%s\":...}", datatype, datatype)
	}

	src = src[len(datatype):]
	if len(src) == 0 || src[0] != '"' {
		return src, zero, fmt.Errorf("error parsing %s: expected object of the format {\"$%s\":...}", datatype, datatype)
	}

	src = skipWS(src[1:])
	if len(src) == 0 || src[0] != ':' {
		return src, zero, fmt.Errorf("error parsing %s: expected object of the format {\"$%s\":...}", datatype, datatype)
	}

	src = skipWS(src[1:])
	src, res, err := decode(src)
	if err != nil {
		return src, zero, err
	}

	src = skipWS(src)
	if len(src) == 0 || src[0] != '}' {
		return src, zero, fmt.Errorf("error parsing %s: expected object of the format {\"$%s\":...}", datatype, datatype)
	}

	return src[1:], res, nil
}
