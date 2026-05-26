package astradb

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/serdes"
	"github.com/datastax/astra-db-go/table"
)

type Row interface {
	Get(path ...string) (any, bool)
	Decode(dest any, path ...string) error
	ToMap() map[string]any
}

type NewRow map[string]any

func (r NewRow) ToMap() map[string]any {
	return r
}

func (r NewRow) Get(path ...string) (any, bool) {
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

func (r NewRow) Decode(dest any, path ...string) error {
	val, ok := r.Get(path...)
	if !ok {
		return fmt.Errorf("path not found")
	}

	b, err := serdes.Serialize(val, serdes.TargetTable)
	if err != nil {
		return err
	}

	return serdes.Deserialize(b, dest, nil, serdes.TargetTable)
}

func (r NewRow) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetTable {
		return nil, fmt.Errorf("`NewRow` can only be serialized for tables, got %s", ctx.Target)
	}
	return serdes.SerializeInto(map[string]any(r), ctx.Target, dst)
}

func (r NewRow) UnmarshalAstraRaw(_ serdes.DecodeCtx, _ []byte) error {
	return fmt.Errorf("cannot deserialize into NewRow; use the astradb.Row interface for results")
}

type serverRow struct {
	data   map[string]json.RawMessage
	schema *lazySchema
}

func (s *serverRow) ToMap() map[string]any {
	schema := s.schema.Get()
	result := make(map[string]any, len(s.data))

	ctx := serdes.DecodeCtx{Target: serdes.TargetTable, TargetCtx: s.schema}

	for name, rawValue := range s.data {
		if string(rawValue) == "null" {
			result[name] = nil
			continue
		}

		col, ok := schema.Get(name)
		if ok {
			val, err := deserializeColumn(ctx, rawValue, col)
			if err != nil {
				panic(fmt.Sprintf("astradb: failed to decode field %q using schema type %q: %v", name, col.Type, err))
			}
			result[name] = val
			continue
		}

		// Generic fallback for metadata fields OR schema mismatch
		var val any
		_ = serdes.Deserialize(rawValue, &val, nil, serdes.TargetTable)
		result[name] = val
	}
	return result
}

func (s *serverRow) Get(path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	currentRaw, ok := s.data[path[0]]
	if !ok {
		return nil, false
	}

	currentCol, hasCol := s.schema.Get().Get(path[0])
	ctx := serdes.DecodeCtx{Target: serdes.TargetTable, TargetCtx: s.schema}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetTable); err != nil {
			return nil, false
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return nil, false
		}

		if hasCol {
			currentCol, hasCol = getSubColumn(currentCol, path[i])
		}
	}

	if string(currentRaw) == "null" {
		return nil, true
	}

	if hasCol {
		val, err := deserializeColumn(ctx, currentRaw, currentCol)
		if err != nil {
			panic(fmt.Sprintf("astradb: failed to decode path %v using schema type %q: %v", path, currentCol.Type, err))
		}
		return val, true
	}

	// Fallback to generic decoding if no column info
	var generic any
	if err := serdes.Deserialize(currentRaw, &generic, nil, serdes.TargetTable); err != nil {
		return nil, false
	}
	return generic, true
}

func (s *serverRow) Decode(dest any, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	currentRaw, ok := s.data[path[0]]
	if !ok {
		return fmt.Errorf("path %s not found", path[0])
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetTable); err != nil {
			return err
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return fmt.Errorf("path %s not found", path[i])
		}
	}

	return serdes.Deserialize(currentRaw, dest, nil, serdes.TargetTable)
}

func (s *serverRow) UnmarshalAstraRaw(_ serdes.DecodeCtx, value []byte) error {
	s.data = make(map[string]json.RawMessage)
	return serdes.Deserialize(value, &s.data, nil, serdes.TargetTable)
}

type lazySchema struct {
	AsRaw  json.RawMessage
	AsCols table.Columns
}

func (s *lazySchema) Get() table.Columns {
	if s.AsCols != nil {
		return s.AsCols
	}

	if s.AsRaw == nil {
		panic("no schema available")
	}

	var cols table.Columns
	if err := serdes.Deserialize(s.AsRaw, &cols, nil, serdes.TargetTable); err != nil {
		panic(fmt.Sprintf("failed to deserialize schema: %v", err))
	}

	s.AsCols = cols
	s.AsRaw = nil
	return cols
}

var rowInterfaceType = reflect.TypeFor[Row]()

func (s *lazySchema) UntypedTargetInterface() reflect.Type {
	return rowInterfaceType
}

func (s *lazySchema) NewUntypedTarget(p unsafe.Pointer) serdes.AstraRawUnmarshaler {
	row := &serverRow{schema: s}
	*(*Row)(p) = row
	return row
}

func NewRowTargetCtx(cols table.Columns) serdes.TargetDecodeCtx {
	return &lazySchema{AsCols: cols}
}

func getSubColumn(col table.Column, key string) (table.Column, bool) {
	switch col.Type {
	case table.TypeMap:
		return *col.ValueType, true
	case table.TypeUDT:
		return col.UDTDefinition.Fields.Get(key)
	default:
		return table.Column{}, false
	}
}

func deserializeColumn(ctx serdes.DecodeCtx, raw json.RawMessage, col table.Column) (any, error) {
	switch col.Type {
	case table.TypeInt:
		var v int
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeBigInt:
		var v int64
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeSmallInt:
		var v int16
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeTinyInt:
		var v int8
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeFloat:
		var v float32
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeDouble:
		var v float64
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeVarint:
		var v big.Int
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeDecimal:
		var v big.Float
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeText, table.TypeAscii:
		var v string
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeBoolean:
		var v bool
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeDate, table.TypeTime, table.TypeTimestamp:
		var v datatypes.DataAPITimestamp
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeDuration:
		var v string
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeUUID, table.TypeTimeUUID:
		var v datatypes.UUID
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeBlob:
		var v []byte
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeInet:
		var v net.IP
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeVector:
		var v datatypes.DataAPIVector
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err

	case table.TypeSet, table.TypeList:
		return deserializeListLike(ctx, raw, col)

	case table.TypeMap:
		return deserializeMap(ctx, raw, col)

	case table.TypeUDT:
		return deserializeUDT(ctx, raw, col)

	default:
		var v any
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable)
		return v, err
	}
}

func deserializeListLike(ctx serdes.DecodeCtx, raw json.RawMessage, col table.Column) (any, error) {
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

func deserializeMap(ctx serdes.DecodeCtx, raw json.RawMessage, col table.Column) (any, error) {
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

func deserializeUDT(ctx serdes.DecodeCtx, raw json.RawMessage, col table.Column) (any, error) {
	newCtx := ctx
	newCtx.TargetCtx = &lazySchema{AsCols: col.UDTDefinition.Fields}
	var r Row
	if err := serdes.Deserialize(raw, &r, newCtx.TargetCtx, serdes.TargetTable); err != nil {
		return nil, err
	}
	return r.ToMap(), nil
}
