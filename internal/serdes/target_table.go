package serdes

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

var TableTarget = Target{
	kind: tableKind,
	typeOverrides: map[unsafe.Pointer]codec{
		typePtr(uuidType):     {encodeTableUUID, decodeTableUUID},
		typePtr(dApiTimeType): {encodeTableTimestamp, decodeTableTimestamp},
	},
	kindOverrides: map[reflect.Kind]func(codecCtx, reflect.Type, seenStructs, bool) codec{
		reflect.Map: mkTableMapCodec,
	},
}

// UUIDs

func encodeTableUUID(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	return encodeUUID(dst, p)
}

func decodeTableUUID(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
	src, uuid, err := decodeUUID(src)
	if err == nil {
		*(*datatypes.UUID)(p) = uuid
	}
	return src, err
}

// Timestamps

func encodeTableTimestamp(_ encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
	ts := (*datatypes.DataAPITimestamp)(p)
	dst = append(dst, '"')
	dst = append(dst, ts.String()...)
	dst = append(dst, '"')
	return dst, nil
}

func decodeTableTimestamp(_ decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
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

// Maps

func mkTableMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs, _ bool) codec {
	return mkMapCodec(ctx, t, seen, func(kt, vt reflect.Type, kz, vz reflect.Value, vc codec) codec {
		kc, _ := resolveCodec(ctx, kt, seen, false)

		return codec{
			encodeTableMap(t, kt, kc.encode, vc.encode),
			decodeTableMap(t, kt, vt, kz, vz, kc.decode, vc.decode),
		}
	})
}

func encodeTableMap(t, kt reflect.Type, encodeKey, encodeValue encoder) encoder {
	return encodeMap(t, kt, encodeKey, encodeValue, '[', ']', ',')
}

func decodeTableMap(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder) decoder {
	decodeObjectMap := decodeCollectionMap(t, kt, vt, kz, vz, decodeKey, decodeValue)
	decodeArrayMap := decodeMap(t, kt, vt, kz, vz, decodeKey, decodeValue, '[', ']', ',')

	return func(ctx decodeCtx, b []byte, p unsafe.Pointer) ([]byte, error) {
		if len(b) > 0 && (b[0] == '{' || b[0] == 'n') {
			return decodeObjectMap(ctx, b, p)
		}
		return decodeArrayMap(ctx, b, p)
	}
}
