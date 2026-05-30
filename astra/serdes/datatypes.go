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

	"github.com/datastax/astra-db-go/astra/datatypes"
)

// ================================
// | UUIDs - encoded as {"$uuid":"<uuid>"} in collections and as "<uuid>" in all other contexts.
// ================================

var uuidTag = []byte("uuid")

func uuidEncoder(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target == TargetCollection {
		return encodeDollarDatatype(dst, uuidTag, func(dst []byte) ([]byte, error) {
			return encodeUUID(dst, p)
		})
	}
	return encodeUUID(dst, p)
}

func uuidDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	var uuid datatypes.UUID
	var err error

	if ctx.Target == TargetCollection {
		src, uuid, err = parseDollarDatatype(ctx, src, uuidTag, decodeUUID)
	} else {
		src, uuid, err = decodeUUID(ctx, src)
	}

	if err == nil {
		*(*datatypes.UUID)(p) = uuid
	}
	return src, err
}

func encodeUUID(dst []byte, p unsafe.Pointer) ([]byte, error) {
	dst = append(dst, '"')
	dst = (*(*datatypes.UUID)(p)).AppendString(dst)
	dst = append(dst, '"')
	return dst, nil
}

func decodeUUID(ctx DecodeCtx, src []byte) ([]byte, datatypes.UUID, error) {
	srcAfter, str, _, err := parseStringUnquote(ctx, src)
	if err != nil {
		return srcAfter, datatypes.UUID{}, err
	}

	uuid, err := datatypes.ParseUUID(unsafeString(str))
	if err != nil {
		return srcAfter, datatypes.UUID{}, ctx.syntaxErrorWrap(src, "invalid UUID string", err)
	}

	return srcAfter, uuid, nil
}

// ================================
// | ObjectIDs - encoded in collections as {"$objectId":"<objectId>"};
// | not allowed to be encoded in other targets
// |
// | TODO do we want to allow encoding these outside of collections as plain strings?
// ================================

var oidTag = []byte("objectId")

func objectIdEncoder(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target != TargetCollection {
		return dst, &UnsupportedValueError{Msg: "ObjectId is only supported for collections"}
	}

	return encodeDollarDatatype(dst, oidTag, func(dst []byte) ([]byte, error) {
		dst = append(dst, '"')
		dst = append(dst, (*(*datatypes.ObjectId)(p)).String()...)
		dst = append(dst, '"')
		return dst, nil
	})
}

func objectIdDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target != TargetCollection {
		return src, &UnsupportedValueError{Msg: "ObjectId is only supported for collections"}
	}

	src, oid, err := parseDollarDatatype(ctx, src, oidTag, func(ctx DecodeCtx, b []byte) ([]byte, datatypes.ObjectId, error) {
		src, str, _, err := parseStringUnquote(ctx, b)
		if err != nil {
			return src, datatypes.ObjectId{}, err
		}

		oid, err := datatypes.ParseObjectId(unsafeString(str))
		if err != nil {
			return src, datatypes.ObjectId{}, ctx.syntaxErrorWrap(src, "invalid ObjectId string", err)
		}

		return src, oid, nil
	})

	if err == nil {
		*(*datatypes.ObjectId)(p) = oid
	}
	return src, err
}

// ================================
// | time.Time - encoded as {"$date":<timestamp>} in collections and as ISO-8601 timestamps
// | in all other contexts.
// ================================

var dateTag = []byte("date")

func timeEncoder(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	t := (*time.Time)(p)

	if ctx.Target == TargetCollection {
		return encodeDollarDatatype(dst, dateTag, func(dst []byte) ([]byte, error) {
			return strconv.AppendInt(dst, t.UnixMilli(), 10), nil
		})
	}

	dst = append(dst, '"')
	dst = t.AppendFormat(dst, time.RFC3339Nano)
	dst = append(dst, '"')
	return dst, nil
}

func timeDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target == TargetCollection {
		src, ms, err := parseDollarDatatype(ctx, src, dateTag, func(ctx DecodeCtx, b []byte) ([]byte, int64, error) {
			return parseInt(ctx, b)
		})
		if err == nil {
			*(*time.Time)(p) = time.UnixMilli(ms)
		}
		return src, err
	}

	srcAfter, str, _, err := parseStringUnquote(ctx, src)
	if err != nil {
		return srcAfter, err
	}

	t, err := time.Parse(time.RFC3339Nano, unsafeString(str))
	if err != nil {
		return srcAfter, ctx.syntaxErrorWrap(src, "invalid timestamp string", err)
	}

	*(*time.Time)(p) = t
	return srcAfter, nil
}

// ================================
// | DateOnly - encoded as "YYYY-MM-DD" in tables; not allowed in collections
// ================================

func dateOnlyEncoder(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target == TargetCollection {
		return nil, &UnsupportedValueError{Msg: "DateOnly is not supported for collections"}
	}

	d := (*datatypes.DateOnly)(p)
	dst = append(dst, '"')
	dst = append(dst, d.String()...)
	dst = append(dst, '"')
	return dst, nil
}

func dateOnlyDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target == TargetCollection {
		return src, &UnsupportedValueError{Msg: "DateOnly is not supported for collections"}
	}

	srcAfter, str, _, err := parseStringUnquote(ctx, src)
	if err != nil {
		return srcAfter, err
	}

	d, err := datatypes.ParseDateOnly(unsafeString(str))
	if err != nil {
		return srcAfter, ctx.syntaxErrorWrap(src, "invalid date string", err)
	}

	*(*datatypes.DateOnly)(p) = d
	return srcAfter, nil
}

// ================================
// | TimeOnly - encoded as "HH:MM:SS.NNNNNNNNN" in tables; not allowed in collections
// ================================

func timeOnlyEncoder(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target == TargetCollection {
		return nil, &UnsupportedValueError{Msg: "TimeOnly is not supported for collections"}
	}

	t := (*datatypes.TimeOnly)(p)
	dst = append(dst, '"')
	dst = append(dst, t.String()...)
	dst = append(dst, '"')
	return dst, nil
}

func timeOnlyDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	if ctx.Target == TargetCollection {
		return src, &UnsupportedValueError{Msg: "TimeOnly is not supported for collections"}
	}

	srcAfter, str, _, err := parseStringUnquote(ctx, src)
	if err != nil {
		return srcAfter, err
	}

	t, err := datatypes.ParseTimeOnly(unsafeString(str))
	if err != nil {
		return srcAfter, ctx.syntaxErrorWrap(src, "invalid time string", err)
	}

	*(*datatypes.TimeOnly)(p) = t
	return srcAfter, nil
}

// ================================
// | big.Int - encoded as a raw number in all contexts
// ================================

func bigIntEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	bi := *(*big.Int)(p)
	return bi.Append(dst, 10), nil
}

func bigIntDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if b, ok := consumeNull(src); ok {
		return b, nil
	}

	srcAfter, numStr, err := parseNumber(ctx, src)
	if err != nil {
		return srcAfter, err
	}

	var bi big.Int
	if _, ok := bi.SetString(unsafeString(numStr), 10); !ok {
		return srcAfter, ctx.syntaxError(src, "invalid big.Int value")
	}

	*(*big.Int)(p) = bi
	return srcAfter, nil
}

// ================================
// | big.Float - encoded as a raw number in all contexts
// ================================

func bigFloatEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	bf := *(*big.Float)(p)
	return bf.Append(dst, 'g', -1), nil
}

func bigFloatDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if b, ok := consumeNull(src); ok {
		return b, nil
	}

	srcAfter, numStr, err := parseNumber(ctx, src)
	if err != nil {
		return srcAfter, err
	}

	var bf big.Float
	if _, ok := bf.SetString(unsafeString(numStr)); !ok {
		return srcAfter, ctx.syntaxError(src, "invalid big.Float value")
	}

	*(*big.Float)(p) = bf
	return srcAfter, nil
}

// ================================
// | Vectors - encoded as {"$binary":"<base64>"} in all contexts, but can be decoded from either
// | that format or from a float32 array
// |
// | TODO potentially add flags for encoding vectors as a plain []float32
// ================================

func vectorEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("binary"), func(dst []byte) ([]byte, error) {
		dst = append(dst, '"')
		dst = (*datatypes.DataAPIVector)(p).AppendBase64(dst)
		dst = append(dst, '"')
		return dst, nil
	})
}

func vectorDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) != 0 && src[0] == '[' {
		var arr []float32
		src, err := mkSliceDecoder(4, float32SliceType, float32Decoder)(ctx, src, unsafe.Pointer(&arr))
		if err == nil {
			vector := datatypes.NewVector(arr)
			*(*datatypes.DataAPIVector)(p) = vector
		}
		return src, err
	}

	if len(src) != 0 && src[0] == '{' {
		src, vector, err := parseDollarDatatype(ctx, src, []byte("binary"), func(ctx DecodeCtx, b []byte) ([]byte, datatypes.DataAPIVector, error) {
			src, str, isNew, err := parseStringUnquote(ctx, b)
			if err != nil {
				return src, datatypes.DataAPIVector{}, err
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

	return src, ctx.unmarshalTypeError(src, vectorType)
}

// ================================
// | Binary data ([]byte) - encoded as {"$binary":"<base64>"} in all contexts,
// | but can be decoded from either that format or from a base64 string
// ================================

func binaryEncoder(_ EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("binary"), func(dst []byte) ([]byte, error) {
		return encodeBytesAsBase64(dst, *(*[]byte)(p)), nil
	})
}

func binaryDecoder(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src = skipWS(src)

	if len(src) != 0 && src[0] == '"' {
		src, str, _, err := parseStringUnquote(ctx, src)
		if err != nil {
			return src, err
		}

		*(*[]byte)(p) = str
		return src, nil
	}

	if len(src) != 0 && src[0] == '{' {
		src, data, err := parseDollarDatatype(ctx, src, []byte("binary"), decodeBytesFromBase64)

		if err == nil {
			*(*[]byte)(p) = data
		}
		return src, err
	}

	if len(src) != 0 && src[0] == '[' {
		var arr []byte
		src, err := mkSliceDecoder(1, byteSliceType, uint8Decoder)(ctx, src, unsafe.Pointer(&arr))
		if err == nil {
			*(*[]byte)(p) = arr
		}
		return src, err
	}

	return src, ctx.unmarshalTypeError(src, byteSliceType)
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

func decodeBytesFromBase64(ctx DecodeCtx, src []byte) ([]byte, []byte, error) {
	srcAfter, str, _, err := parseStringUnquote(ctx, src)
	if err != nil {
		return srcAfter, nil, err
	}

	data := make([]byte, base64.StdEncoding.DecodedLen(len(str)))
	n, err := base64.StdEncoding.Decode(data, str)
	if err != nil {
		return srcAfter, nil, ctx.syntaxErrorWrap(src, "invalid base64 string", err)
	}

	return srcAfter, data[:n], nil
}

// ================================
// | Helpers
// ================================

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

func parseDollarDatatype[T any](ctx DecodeCtx, src []byte, datatype []byte, decode func(DecodeCtx, []byte) ([]byte, T, error)) ([]byte, T, error) {
	var zero T

	src = skipWS(src)
	if len(src) == 0 || src[0] != '{' {
		return src, zero, ctx.syntaxError(src, fmt.Sprintf("expected object for %s", datatype))
	}

	src = skipWS(src[1:])
	if !bytes.HasPrefix(src, []byte(`"$`)) {
		return src, zero, ctx.syntaxError(src, fmt.Sprintf("expected \"$%s\" key", datatype))
	}

	src = src[2:]
	if len(src) < len(datatype) || !bytes.Equal(src[:len(datatype)], datatype) {
		return src, zero, ctx.syntaxError(src, fmt.Sprintf("expected \"$%s\" key", datatype))
	}

	src = src[len(datatype):]
	if len(src) == 0 || src[0] != '"' {
		return src, zero, ctx.syntaxError(src, fmt.Sprintf("expected \"$%s\" key to be quoted", datatype))
	}

	src = skipWS(src[1:])
	if len(src) == 0 || src[0] != ':' {
		return src, zero, ctx.syntaxError(src, fmt.Sprintf("expected ':' after \"$%s\" key", datatype))
	}

	src = skipWS(src[1:])
	src, res, err := decode(ctx, src)
	if err != nil {
		return src, zero, err
	}

	src = skipWS(src)
	if len(src) == 0 || src[0] != '}' {
		return src, zero, ctx.syntaxError(src, fmt.Sprintf("expected '}' at the end of %s object", datatype))
	}

	return src[1:], res, nil
}

func decodeDollarDatatype(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error, bool) {
	initSrc := src

	src = skipWS(src)
	if len(src) == 0 || src[0] != '{' {
		return initSrc, nil, false
	}

	src = skipWS(src[1:])
	if !bytes.HasPrefix(src, []byte(`"$`)) {
		return initSrc, nil, false
	}

	src, datatype, _, err := parseStringUnquote(ctx, src)
	if err != nil {
		return src, err, true
	}

	if codec, ok := ctx.Target.dollarDatatypes()[unsafeString(datatype)]; ok {
		valPtr := reflect.New(codec.typ)

		src, err = codec.decode(ctx, initSrc, valuePtr(valPtr.Elem()))
		if err != nil {
			return src, err, true
		}

		*(*any)(p) = valPtr.Elem().Interface() // I don't like this at all...
		return src, nil, true
	}

	return initSrc, nil, false
}
