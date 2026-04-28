package serdes

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

var CollectionTarget = Target{
	kind: collectionKind,
	typeOverrides: map[unsafe.Pointer]codec{
		typePtr(uuidType): {encodeCollUUID, decodeCollUUID},
		typePtr(oidType):  {encodeCollObjId, decodeCollObjId},
	},
	kindOverrides: map[reflect.Kind]func(codecCtx, reflect.Type, seenStructs, bool) codec{
		reflect.Map: mkCollectionMapCodec,
	},
}

// UUIDs

func encodeCollUUID(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, uuidTag, func(dst []byte) ([]byte, error) {
		return encodeUUID(dst, p)
	})
}

func decodeCollUUID(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, uuid, err := parseDollarDatatype(src, uuidTag, decodeUUID)
	if err == nil {
		*(*datatypes.UUID)(p) = uuid
	}
	return src, err
}

// ObjectIDs

func encodeCollObjId(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeDollarDatatype(dst, oidTag, func(dst []byte) ([]byte, error) {
		return encodeObjectID(dst, p)
	})
}

func decodeCollObjId(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, oid, err := parseDollarDatatype(src, oidTag, decodeObjectID)
	if err == nil {
		*(*datatypes.ObjectId)(p) = oid
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
