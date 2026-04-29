package serdes

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

var CollectionTarget = Target{
	typeOverrides: map[unsafe.Pointer]codec{
		typePtr(uuidType):     {encodeCollUUID, decodeCollUUID},
		typePtr(oidType):      {encodeCollObjId, decodeCollObjId},
		typePtr(dApiTimeType): {encodeCollTimestamp, decodeCollTimestamp},
	},
	kindOverrides: map[reflect.Kind]func(codecCtx, reflect.Type, seenStructs, bool) codec{
		reflect.Map: mkCollectionMapCodec,
	},
	dollarDatatypes: map[string]typedCodec{
		"$uuid":     {codec{encodeCollUUID, decodeCollUUID}, uuidType},
		"$objectId": {codec{encodeCollObjId, decodeCollObjId}, oidType},
		"$date":     {codec{encodeCollTimestamp, decodeCollTimestamp}, dApiTimeType},
		"$binary":   {codec{encodeBinary, decodeBinary}, byteSliceType},
	},
}

// UUIDs

func encodeCollUUID(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("uuid"), func(dst []byte) ([]byte, error) {
		return encodeUUID(dst, p)
	})
}

func decodeCollUUID(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, uuid, err := parseDollarDatatype(src, []byte("uuid"), decodeUUID)
	if err == nil {
		*(*datatypes.UUID)(p) = uuid
	}
	return src, err
}

// ObjectIDs

func encodeCollObjId(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("objectId"), func(dst []byte) ([]byte, error) {
		return encodeObjectID(dst, p)
	})
}

func decodeCollObjId(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, oid, err := parseDollarDatatype(src, []byte("objectId"), decodeObjectID)
	if err == nil {
		*(*datatypes.ObjectId)(p) = oid
	}
	return src, err
}

// Timestamps

func encodeCollTimestamp(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, []byte("date"), func(dst []byte) ([]byte, error) {
		ts := (*datatypes.DataAPITimestamp)(p)
		return strconv.AppendInt(dst, ts.UnixMillis(), 10), nil
	})
}

func decodeCollTimestamp(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, ms, err := parseDollarDatatype(src, []byte("date"), func(b []byte) ([]byte, int64, error) {
		return parseInt(b)
	})
	if err == nil {
		*(*datatypes.DataAPITimestamp)(p) = datatypes.DataAPITimestampFromMillis(ms)
	}
	return src, err
}

// Maps

func mkCollectionMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs, _ bool) codec {
	return mkMapCodec(ctx, t, seen, func(kt, vt reflect.Type, kz, vz reflect.Value, vc codec) codec {
		kc := kindCodecs[reflect.String]

		if kt.Kind() != reflect.String {
			return mkErroredCodec(fmt.Errorf("unsupported map key type on collections: %s", kt))
		}

		return codec{
			encodeCollectionMap(t, kt, kc.encode, vc.encode),
			decodeCollectionMap(t, kt, vt, kz, vz, kc.decode, vc.decode),
		}
	})
}

func encodeCollectionMap(t, kt reflect.Type, encodeKey, encodeValue encoder) encoder {
	return encodeMap(t, kt, encodeKey, encodeValue, '{', '}', ':')
}

func decodeCollectionMap(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder) decoder {
	return decodeMap(t, kt, vt, kz, vz, decodeKey, decodeValue, '{', '}', ':')
}
