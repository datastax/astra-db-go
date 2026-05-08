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
// How would we deal with users creating a row with no Schema then trying to deserialize into it when they shouldn't be doing so?
//
// What about an equivalent `Document` for collections?
//
// Should collections also be forced to go entirely through a `Document` type or can they use `any`s since they have no Schema?

type Row map[string]any

func (r Row) Get(path ...string) (any, bool) {
	current := r
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

func (r Row) ToMap() map[string]any {
	return r
}

// LazySchema TODO eventually make this internal
type LazySchema struct {
	AsRaw  json.RawMessage
	AsCols Columns
}

func (ds *LazySchema) Get() Columns {
	if ds.AsCols != nil {
		return ds.AsCols
	}

	if ds.AsRaw == nil {
		panic("no schema available") // should never happen as long a user somehow messed with it
	}

	var cols Columns
	if err := serdes.Deserialize(ds.AsRaw, &cols, nil, serdes.TargetTable); err != nil {
		panic(fmt.Sprintf("failed to deserialize schema: %v", err)) // should never happen unless the server sent something truly bizarre or the user messed with it
	}

	ds.AsCols = cols
	ds.AsRaw = nil
	return cols
}

func (r Row) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetTable {
		return nil, fmt.Errorf("`Row` can only be serialized for tables, got %s", serdes.TargetTable)
	}
	return serdes.SerializeInto(map[string]any(r), ctx.Target, dst)
}

func (rp *Row) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	if ctx.Target != serdes.TargetTable {
		return fmt.Errorf("`Row` can only be deserialized for tables, got %s", serdes.TargetTable)
	}

	schema := ctx.Schema.(*LazySchema).Get()

	if *rp == nil {
		*rp = make(Row)
	}
	r := *rp

	// parse top level struct w/ values as RawMessage so we can control the exact deserialization of each field w/ type hints
	rawMap := make(map[string]json.RawMessage)
	if err := serdes.Deserialize(value, &rawMap, nil, serdes.TargetTable); err != nil {
		return err
	}

	// fill out the map with exact types now
	// we could have written yet another map parser™ here but this helps deduplicate things a bit
	// yes it's a little less performant, but I don't have much performance sympathy for users using an untyped row
	for _, nc := range schema {
		rawValue, ok := rawMap[nc.Name]

		// technically `null` shouldn't be possible but just in case /shrug
		if !ok || *(*string)(unsafe.Pointer(&rawValue)) == "null" {
			r[nc.Name] = nil // TODO decide if nil handling needs to be fancier
			continue
		}

		val, err := deserializeColumn(ctx, rawValue, nc.Column)
		if err != nil {
			return fmt.Errorf("field %s: %w", nc.Name, err)
		}
		r[nc.Name] = val
	}

	return nil
}

func deserializeColumn(ctx serdes.DecodeCtx, raw json.RawMessage, col Column) (any, error) {
	switch col.Type {
	case TypeInt:
		var v int
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeBigInt:
		var v int64
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeSmallInt:
		var v int16
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeTinyInt:
		var v int8
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeFloat:
		var v float32
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeDouble:
		var v float64
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeVarint:
		var v big.Int
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeDecimal:
		var v big.Float
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeText, TypeAscii:
		var v string
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeBoolean:
		var v bool
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	// TODO update once temporal types are properly fleshed out
	case TypeDate, TypeTime, TypeTimestamp:
		var v datatypes.DataAPITimestamp
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	// TODO ^
	case TypeDuration:
		var v string
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeUUID, TypeTimeUUID:
		var v datatypes.UUID
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeBlob:
		var v []byte
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeInet:
		var v net.IP
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeVector:
		var v datatypes.DataAPIVector
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case TypeSet:
		// TODO should we return a `Set` instead for sets?
		return deserializeListLike(ctx, raw, col)

	case TypeList:
		return deserializeListLike(ctx, raw, col)

	case TypeMap:
		return deserializeMap(ctx, raw, col)

	case TypeUDT:
		return deserializeUDT(ctx, raw, col)

	default:
		var v any
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err
	}
}

func deserializeListLike(ctx serdes.DecodeCtx, raw json.RawMessage, col Column) (any, error) {
	if col.ValueType == nil {
		return nil, fmt.Errorf("%s column missing valueType", col.Type)
	}

	// same reasoning as explained in UnmarshalAstraRaw
	var rawArray []json.RawMessage
	if err := serdes.Deserialize(raw, &rawArray, nil, serdes.TargetTable); err != nil {
		return nil, err
	}

	result := make([]any, len(rawArray))
	for i, rawElem := range rawArray {
		val, err := deserializeColumn(ctx, rawElem, *col.ValueType)
		if err != nil {
			return nil, fmt.Errorf("%s element %d: %w", col.Type, i, err)
		}
		result[i] = val
	}

	return result, nil
}

func deserializeMap(ctx serdes.DecodeCtx, raw json.RawMessage, col Column) (any, error) {
	if col.KeyType == nil || col.ValueType == nil {
		return nil, fmt.Errorf("map column missing keyType or valueType")
	}

	// TODO figure out non-string-key maps...
	var rawMap map[string]json.RawMessage
	if err := serdes.Deserialize(raw, &rawMap, nil, serdes.TargetTable); err != nil {
		return nil, err
	}

	result := make(map[string]any, len(rawMap))

	for key, rawValue := range rawMap {
		val, err := deserializeColumn(ctx, rawValue, *col.ValueType)
		if err != nil {
			return nil, fmt.Errorf("map value for key %s: %w", key, err)
		}

		result[key] = val
	}

	return result, nil
}

func deserializeUDT(ctx serdes.DecodeCtx, raw json.RawMessage, col Column) (any, error) {
	ctx.Schema = &LazySchema{AsCols: col.UDTDefinition().Fields}
	var r Row
	if err := r.UnmarshalAstraRaw(ctx, raw); err != nil {
		return nil, err
	}
	return r.ToMap(), nil
}
