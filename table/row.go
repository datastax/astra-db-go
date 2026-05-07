package table

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/serdes"
)

// TODO figure out how we want to portray Rows to users
//
// Should they be readonly or should users use them to write too?
//
// Should users be allowed to write `any`s and `map[string]any`s too or should they be forced to go through Row for symmetry?
//
// Should `Data` be public or remain private and only accessible through `Get` and `ToMap`?
//
// How would we deal with users creating a row with no schema then trying to deserialize into it when they shouldn't be doing so?
//
// What about an equivalent `Document` for collections?
//
// Should collections also be forced to go entirely through a `Document` type or can they use `any`s since they have no schema?

type Row struct {
	data   map[string]any
	schema Columns
}

func (r *Row) Get(path ...string) (any, bool) {
	current := r.data
	for i, p := range path {
		val, ok := current[p]
		if !ok {
			return nil, false
		}

		if i == len(path)-1 {
			return val, true
		}

		nextMap, ok := val.(map[string]any)
		if !ok {
			return nil, false
		}
		current = nextMap
	}
	return nil, false
}

func (r *Row) ToMap() map[string]any {
	return r.data
}

func (r *Row) Schema() Columns {
	return r.schema
}

func (r *Row) UnmarshalAstraRaw(target serdes.Target, value []byte) error {
	if !target.Is(serdes.TargetTable) {
		return fmt.Errorf("`Row` can only be deserialized for tables, got %s", serdes.TargetTable)
	}

	// parse top level struct w/ values as RawMessage so we can control the exact deserialization of each field w/ type hints
	rawMap := make(map[string]json.RawMessage)
	if err := serdes.Deserialize(value, &rawMap, serdes.TargetTable); err != nil {
		return err
	}

	// fill out the map with exact types now
	// we could have written yet another map parser™ here but this helps deduplicate things a bit
	// yes it's a little less performant, but I don't have much performance sympathy for users using an untyped row
	r.data = make(map[string]any, len(r.schema))
	for _, nc := range r.schema {
		rawValue, ok := rawMap[nc.Name]

		// technically `null` shouldn't be possible but just in case /shrug
		if !ok || *(*string)(unsafe.Pointer(&rawValue)) == "null" {
			r.data[nc.Name] = nil // TODO decide if nil handling needs to be fancier
			continue
		}

		val, err := deserializeColumn(rawValue, nc.Column)
		if err != nil {
			return fmt.Errorf("field %s: %w", nc.Name, err)
		}
		r.data[nc.Name] = val
	}

	return nil
}

func deserializeColumn(raw json.RawMessage, col Column) (any, error) {
	switch col.Type {
	case TypeInt:
		var v int
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeBigInt:
		var v int64
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeSmallInt:
		var v int16
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeTinyInt:
		var v int8
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeFloat:
		var v float32
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeDouble:
		var v float64
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeVarint:
		var v big.Int
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeDecimal:
		var v big.Float
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeText, TypeAscii:
		var v string
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeBoolean:
		var v bool
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	// TODO update once temporal types are properly fleshed out
	case TypeDate, TypeTime, TypeTimestamp:
		var v datatypes.DataAPITimestamp
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	// TODO ^
	case TypeDuration:
		var v string
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeUUID, TypeTimeUUID:
		var v datatypes.UUID
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeBlob:
		var v []byte
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeInet:
		var v net.IP
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeVector:
		var v datatypes.DataAPIVector
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err

	case TypeSet:
		// TODO should we return a `Set` instead for sets?
		return deserializeListLike(raw, col)

	case TypeList:
		return deserializeListLike(raw, col)

	case TypeMap:
		return deserializeMap(raw, col)

	case TypeUDT:
		return deserializeUDT(raw, col)

	default:
		var v any
		err := serdes.Deserialize(raw, &v, serdes.TargetTable)
		return v, err
	}
}

func deserializeListLike(raw json.RawMessage, col Column) (any, error) {
	if col.ValueType == nil {
		return nil, fmt.Errorf("%s column missing valueType", col.Type)
	}

	// same reasoning as explained in UnmarshalAstraRaw
	var rawArray []json.RawMessage
	if err := serdes.Deserialize(raw, &rawArray, serdes.TargetTable); err != nil {
		return nil, err
	}

	result := make([]any, len(rawArray))
	for i, rawElem := range rawArray {
		val, err := deserializeColumn(rawElem, *col.ValueType)
		if err != nil {
			return nil, fmt.Errorf("%s element %d: %w", col.Type, i, err)
		}
		result[i] = val
	}

	return result, nil
}

func deserializeMap(raw json.RawMessage, col Column) (any, error) {
	if col.KeyType == nil || col.ValueType == nil {
		return nil, fmt.Errorf("map column missing keyType or valueType")
	}

	// TODO figure out non-string-key maps...
	var rawMap map[string]json.RawMessage
	if err := serdes.Deserialize(raw, &rawMap, serdes.TargetTable); err != nil {
		return nil, err
	}

	result := make(map[string]any, len(rawMap))

	for key, rawValue := range rawMap {
		val, err := deserializeColumn(rawValue, *col.ValueType)
		if err != nil {
			return nil, fmt.Errorf("map value for key %s: %w", key, err)
		}

		result[key] = val
	}

	return result, nil
}

func deserializeUDT(raw json.RawMessage, col Column) (any, error) {
	asRow := &Row{schema: col.UDTDefinition().Fields}

	if err := asRow.UnmarshalAstraRaw(serdes.TargetTable, raw); err != nil {
		return nil, err
	}

	return asRow.ToMap(), nil
}
