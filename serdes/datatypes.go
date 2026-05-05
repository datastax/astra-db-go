package serdes

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

// UUIDs

func uuidEncoder(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.target.kind == collectionKind {
		return encodeDollarDatatype(dst, []byte("uuid"), func(dst []byte) ([]byte, error) {
			return encodeUUID(dst, p)
		})
	}
	return encodeUUID(dst, p)
}

func uuidDecoder(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	var uuid datatypes.UUID
	var err error

	if ctx.target.kind == collectionKind {
		src, uuid, err = parseDollarDatatype(src, []byte("uuid"), decodeUUID)
	} else {
		src, uuid, err = decodeUUID(src)
	}

	if err == nil {
		*(*datatypes.UUID)(p) = uuid
	}
	return src, nil
}

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

func objectIdEncoder(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.target.kind != collectionKind {
		return dst, fmt.Errorf("cannot encode ObjectId in a non-collection")
	}

	return encodeDollarDatatype(dst, []byte("objectId"), func(dst []byte) ([]byte, error) {
		dst = append(dst, '"')
		dst = append(dst, (*(*datatypes.ObjectId)(p)).String()...)
		dst = append(dst, '"')
		return dst, nil
	})
}

func objectIdDecoder(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.target.kind != collectionKind {
		return src, fmt.Errorf("cannot decode ObjectId from a non-collection")
	}

	src, str, _, err := parseStringUnquote(src)
	if err != nil {
		return src, fmt.Errorf("invalid ObjectId string: %w", err)
	}

	oid, err := datatypes.ParseObjectId(unsafeString(str))
	if err != nil {
		return src, fmt.Errorf("invalid ObjectId string: %w", err)
	}

	*(*datatypes.ObjectId)(p) = oid
	return src, err
}

// Timestamps

func timestampEncoder(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	ts := (*datatypes.DataAPITimestamp)(p)

	if ctx.target.kind == collectionKind {
		return encodeDollarDatatype(dst, []byte("date"), func(dst []byte) ([]byte, error) {
			return strconv.AppendInt(dst, ts.UnixMillis(), 10), nil
		})
	}

	dst = append(dst, '"')
	dst = append(dst, ts.String()...)
	dst = append(dst, '"')
	return dst, nil
}

func timestampDecoder(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.target.kind == collectionKind {
		src, ms, err := parseDollarDatatype(src, []byte("date"), func(b []byte) ([]byte, int64, error) {
			return parseInt(b)
		})
		if err == nil {
			*(*datatypes.DataAPITimestamp)(p) = datatypes.DataAPITimestampFromMillis(ms)
		}
		return src, err
	}

	src, str, _, err := parseStringUnquote(src)
	if err != nil {
		return src, fmt.Errorf("invalid timestamp string: %w", err)
	}

	t, err := time.Parse(time.RFC3339Nano, unsafeString(str))
	if err != nil {
		return src, fmt.Errorf("invalid timestamp string: %w", err)
	}

	*(*datatypes.DataAPITimestamp)(p) = datatypes.NewDataAPITimestamp(t)
	return src, nil
}

// big.Int

func bigIntEncoder(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	bi := *(*big.Int)(p)
	return append(dst, bi.String()...), nil
}

func bigIntDecoder(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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

func bigFloatEncoder(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	bf := *(*big.Float)(p)
	return append(dst, bf.Text('g', -1)...), nil
}

func bigFloatDecoder(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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

func mkMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt := t.Key()
	vt := t.Elem()

	kz := reflect.Zero(kt)
	vz := reflect.Zero(vt)

	kc := resolveCodec(ctx, kt, seen, false)
	vc := resolveCodec(ctx, vt, seen, false)

	if inlined(vt) {
		vc.encode = mkInlineEncoder(vc.encode)
	}

	return codec{
		mkMapEncoder(t, kt, kc.encode, vc.encode),
		mkMapDecoder(t, kt, vt, kz, vz, kc.decode, vc.decode),
	}
}

func mkOrderedMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt, _ := t.FieldByName("kType")
	vt, _ := t.FieldByName("vType")

	kz := reflect.Zero(kt.Type)
	vz := reflect.Zero(vt.Type)

	kc := resolveCodec(ctx, kt.Type, seen, false)
	vc := resolveCodec(ctx, vt.Type, seen, false)

	if inlined(vt.Type) {
		vc.encode = mkInlineEncoder(vc.encode)
	}

	return codec{
		mkMapEncoder(t, kt.Type, kc.encode, vc.encode),
		mkMapDecoder(t, kt.Type, vt.Type, kz, vz, kc.decode, vc.decode),
	}
}

//func mkGenericMapCodec(ctx codecCtx, t, kt, vt reflect.Type, seen seenStructs) codec {
//
//}

func mkMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder) encoder {
	stringKeyEncoder := func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		dst, err := encodeKey(ctx, dst, p)
		if err != nil {
			return dst, err
		}

		dst = skipWSRev(dst)
		if len(dst) == 0 || dst[len(dst)-1] != '"' {
			return dst, rollback{}
		}

		return dst, nil
	}

	encodeObjectMap := mkNormalMapEncoder(t, kt, stringKeyEncoder, encodeValue)
	encodeArrayMap := mkGenericMapEncoder(t, kt, encodeKey, encodeValue, '[', ']', ',')

	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		dst, err := encodeObjectMap(ctx, dst, p)
		if _, ok := err.(rollback); !ok {
			return dst, err
		}

		if ctx.target.kind == tableKind {
			return encodeArrayMap(ctx, dst, p)
		}

		return dst, fmt.Errorf("cannot have a map with non-string keys in tables")
	}
}

func mkMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder) decoder {
	decodeObjectMap := mkNormalMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue)
	decodeArrayMap := mkGenericMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, '[', ']', ',')

	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if len(src) > 0 && (src[0] == '{' || src[0] == 'n') {
			return decodeObjectMap(ctx, src, p)
		}

		if ctx.target.kind == tableKind {
			return decodeArrayMap(ctx, src, p)
		}

		return src, fmt.Errorf("expected a json object or null when parsing map")
	}
}

func mkGenericMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder, open, close, sep byte) encoder {
	mkIter := newMapIterMaker(t, true) // TODO make trySort an optional flag

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

		iter := mkIter(m)
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

func mkGenericMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, open, close, sep byte) decoder {
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

func mkNormalMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder) encoder {
	return mkGenericMapEncoder(t, kt, encodeKey, encodeValue, '{', '}', ':')
}

func mkNormalMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder) decoder {
	return mkGenericMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, '{', '}', ':')
}

// Vectors

func vectorEncoder(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("binary"), func(dst []byte) ([]byte, error) {
		dst = append(dst, '"')
		dst = append(dst, (*datatypes.DataAPIVector)(p).AsBase64()...)
		dst = append(dst, '"')
		return dst, nil
	})
}

func vectorDecoder(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) == 0 || src[0] == '[' {
		var arr []float32
		src, err := mkSliceDecoder(4, float32SliceType, float32Decoder)(ctx, src, unsafe.Pointer(&arr))
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

func binaryEncoder(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("binary"), func(dst []byte) ([]byte, error) {
		return encodeBytesAsBase64(dst, *(*[]byte)(p)), nil
	})
}

func binaryDecoder(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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
		src, data, err := parseDollarDatatype(src, []byte("binary"), decodeBytesFromBase64)

		if err == nil {
			*(*[]byte)(p) = data
		}
		return src, err
	}

	if len(src) == 0 || src[0] == '[' {
		var arr []byte
		src, err := mkSliceDecoder(1, byteSliceType, uint8Decoder)(ctx, src, unsafe.Pointer(&arr))
		if err == nil {
			*(*[]byte)(p) = arr
		}
		return src, err
	}

	return src, fmt.Errorf("expected string, []byte, or {\"$binary\":\"<base64>\"} for []byte value")
}

func encodeBytesAsBase64(dst []byte, data []byte) []byte {
	if data == nil {
		return append(dst, "null"...)
	}

	b64Len := base64.StdEncoding.EncodedLen(len(data))
	reqLen := len(dst) + b64Len + 2

	if cap(dst) < reqLen {
		newDst := make([]byte, len(dst), reqLen)
		copy(newDst, dst)
		dst = newDst
	}

	dst = append(dst, '"')

	start := len(dst)
	dst = dst[:start+b64Len]
	base64.StdEncoding.Encode(dst[start:], data) // these lines could probably be simplified but eh it works

	dst = append(dst, '"')
	return dst
}

func decodeBytesFromBase64(src []byte) ([]byte, []byte, error) {
	src, str, _, err := parseStringUnquote(src)
	if err != nil {
		return src, nil, fmt.Errorf("invalid base64 string: %w", err)
	}

	data := make([]byte, base64.StdEncoding.DecodedLen(len(str)))
	n, err := base64.StdEncoding.Decode(data, str)
	if err != nil {
		return src, nil, fmt.Errorf("invalid base64 string: %w", err)
	}

	return src, data[:n], nil
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
