// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package astra

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/table"
)

// region Interfaces

// Document represents an untyped document used for a collection operation (as opposed to using a specific struct).
//
// To use a Document as an input for collection operations (Insert, Update, etc.),
// use [NewDocument] to create a document from a map:
//
//	doc := astra.NewDocument{"name": "token", "value": 123}
//	res, err := collection.InsertOne(ctx, doc)
//
// To use a Document as a target for collection results (FindOne, Find, etc.),
// pass a pointer to a [Document] interface variable to Decode methods:
//
//	var doc astra.Document
//	if err := result.Decode(&doc); err == nil {
//	    val := doc.MustGet("name").(string)
//	    fmt.Println(val)
//	}
//
// The returned Document allows for dynamic access to the data.
type Document interface {
	isDocument()

	// Get looks up a value in the document using a sequence of keys to navigate
	// through nested maps.
	//
	// It returns (nil, false) if the path is empty, if any of the intermediate
	// keys don't point to a map, or if the final key isn't found.
	//
	// Example:
	//
	//	if val, ok := doc.Get("metadata", "priority"); ok {
	//	    fmt.Printf("Priority: %v\n", val)
	//	}
	Get(path ...string) (any, bool)

	// MustGet is like [Document.Get] but panics if the path doesn't exist.
	// Use this when you're certain the document has the structure you expect.
	//
	// Example:
	//
	//	name := doc.MustGet("user", "name").(string)
	MustGet(path ...string) any

	// Decode tries to extract the value at the path and store it in dest.
	// You must provide a non-nil pointer for dest.
	//
	// It automatically handles type conversions, such as filling out nested
	// structs from maps or parsing dates into time.Time.
	//
	// Example:
	//
	//	var tags []string
	//	if err := doc.Decode(&tags, "metadata", "tags"); err == nil {
	//	    fmt.Printf("Tags: %v\n", tags)
	//	}
	Decode(dest any, path ...string) error

	// ToMap converts the entire document into a standard map[string]any.
	//
	// This is useful when you want to work with the data using standard map
	// operations or need to pass it to functions that don't know about the
	// Document interface.
	ToMap() map[string]any
}

// Row represents an untyped row used for a table operation (as opposed to using a specific struct).
//
// To use a Row as an input for table operations (Insert, Update, etc.),
// use [NewRow] to create a row from a map:
//
//	row := astra.NewRow{"id": 1, "name": "item"}
//	res, err := table.InsertOne(ctx, row)
//
// To use a Row as a target for table results (FindOne, Find, etc.),
// pass a pointer to a [Row] interface variable to Decode methods:
//
//	var row astra.Row
//	if err := result.Decode(&row); err == nil {
//	    val := row.MustGet("name").(string)
//	    fmt.Println(val)
//	}
//
// The returned Row allows for dynamic access to the data.
type Row interface {
	isRow()

	// Get looks up a value in the row using a sequence of keys to navigate
	// through nested structures like maps or UDTs.
	//
	// It returns (nil, false) if the path is empty, if an intermediate key
	// can't be traversed, or if the final key isn't found.
	//
	// Example:
	//
	//	if val, ok := row.Get("address", "city"); ok {
	//	    fmt.Printf("City: %v\n", val)
	//	}
	Get(path ...string) (any, bool)

	// MustGet is like [Row.Get] but panics if the path doesn't exist.
	// Use this when you're certain the row has the structure you expect.
	//
	// Example:
	//
	//	city := row.MustGet("address", "city").(string)
	MustGet(path ...string) any

	// Decode tries to extract the value at the path and store it in dest.
	// You must provide a non-nil pointer for dest.
	//
	// It handles all necessary type conversions, such as parsing strings into
	// UUIDs or filling out structs from nested maps or UDTs.
	//
	// Example:
	//
	//	var id datatypes.UUID
	//	if err := row.Decode(&id, "id"); err == nil {
	//	    fmt.Printf("ID: %s\n", id)
	//	}
	Decode(dest any, path ...string) error

	// ToMap converts the entire row into a standard map[string]any.
	//
	// This is useful when you want to work with the data using standard map
	// operations or need to pass it to functions that don't know about the
	// Row interface.
	ToMap() map[string]any
}

// endregion

// region NewDocument

// NewDocument is a map-based implementation of [Document], primarily used for insertion.
// While standard maps can still be used for insertion, NewDocument provides a
// convenient way to implement the [Document] interface if needed.
//
// Note that you must use the [Document] interface for retrieval when decoding
// results from the server.
//
// Example:
//
//	doc := astra.NewDocument{
//	    "name": "token",
//	    "metadata": map[string]any{
//	        "created_at": time.Now(),
//	        "tags": []string{"active", "priority"},
//	    },
//	}
//
// To retrieve values from a NewDocument, use [NewDocument.Get], [NewDocument.MustGet], or [NewDocument.Decode].
type NewDocument map[string]any

func (NewDocument) isDocument() {}

// ToMap returns the document as a standard map[string]any.
//
// Since NewDocument is just a map under the hood, this returns a reference
// to the actual data. Any changes you make to the map will affect the
// document.
func (d NewDocument) ToMap() map[string]any {
	return d
}

// Get looks up a value in the document using a sequence of keys to navigate
// through nested maps.
//
// It returns (nil, false) if the path is empty, if any of the intermediate
// keys don't point to a map, or if the final key isn't found.
//
// It can not traverse into nested structures that aren't maps, such as lists or structs.
//
// Example:
//
//	val, ok := doc.Get("metadata", "tags")
func (d NewDocument) Get(path ...string) (any, bool) {
	return getDeepFromMap(d, path...)
}

// MustGet is like [NewDocument.Get] but panics if the path doesn't exist.
// Use this when you're certain the document has the structure you expect.
//
// Example:
//
//	tags := doc.MustGet("metadata", "tags").([]string)
func (d NewDocument) MustGet(path ...string) any {
	return mustGet(d.Get, path, "Document")
}

// Decode tries to extract the value at the path and store it in dest.
// You must provide a non-nil pointer for dest.
//
// It will automatically handle type conversions if the source value doesn't
// exactly match what you're asking for—like parsing dates into time.Time
// or filling out structs from nested maps. If the value's type already matches,
// it performs a direct assignment to keep things fast.
//
// Example:
//
//	var metadata MyMetadataStruct
//	if err := doc.Decode(&metadata, "metadata"); err == nil {
//	    fmt.Printf("Tags: %v\n", metadata.Tags)
//	}
//
// It returns an error if the path is missing or if the value can't be
// converted to the requested type.
func (d NewDocument) Decode(dest any, path ...string) error {
	return decodeFromMap(d, path, dest, serdes.TargetCollection)
}

func (d NewDocument) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetCollection {
		return nil, fmt.Errorf("`NewDocument` can only be serialized for collections, got %s", ctx.Target)
	}
	return serdes.SerializeInto(map[string]any(d), ctx.Target, dst, ctx.Flags)
}

func (d NewDocument) UnmarshalAstraRaw(_ serdes.DecodeCtx, _ []byte) error {
	return fmt.Errorf("cannot deserialize into NewDocument; use the astra.Document interface for results")
}

// endregion

// region NewRow

// NewRow is a map-based implementation of [Row], primarily used for insertion.
// While standard maps can still be used for insertion, NewRow provides a
// convenient way to implement the [Row] interface if needed.
//
// Note that you must use the [Row] interface for retrieval when decoding
// results from the server.
//
// Example:
//
//	row := astra.NewRow{
//	    "id": 123,
//	    "name": "token",
//	    "address": map[string]any{
//	        "city": "New York",
//	        "zip": 10001,
//	    },
//	}
//
// To retrieve values from a NewRow, use [NewRow.Get], [NewRow.MustGet], or [NewRow.Decode].
type NewRow map[string]any

func (NewRow) isRow() {}

// ToMap returns the underlying map representation of the row.
func (r NewRow) ToMap() map[string]any {
	return r
}

// Get retrieves a value from the row at the specified path.
// It returns (nil, false) if the path does not exist or if an intermediate
// element is not a map.
//
// Example:
//
//	val, ok := row.Get("address", "city")
func (r NewRow) Get(path ...string) (any, bool) {
	return getDeepFromMap(r, path...)
}

// MustGet retrieves a value from the row at the specified path.
// It panics if the path does not exist.
func (r NewRow) MustGet(path ...string) any {
	return mustGet(r.Get, path, "Row")
}

// Decode tries to extract the value at the path and store it in dest.
// You must provide a non-nil pointer for dest.
//
// It will automatically handle type conversions if the source value doesn't
// exactly match what you're asking for—like parsing strings into UUIDs
// or filling out structs from nested maps. If the value's type already matches,
// it performs a direct assignment to keep things fast.
//
// Example:
//
//	var address MyAddressStruct
//	if err := row.Decode(&address, "address"); err == nil {
//	    fmt.Printf("City: %s\n", address.City)
//	}
//
// It returns an error if the path is missing or if the value can't be
// converted to the requested type.
func (r NewRow) Decode(dest any, path ...string) error {
	return decodeFromMap(r, path, dest, serdes.TargetTable)
}

func (r NewRow) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetTable {
		return nil, fmt.Errorf("`NewRow` can only be serialized for tables, got %s", ctx.Target)
	}
	return serdes.SerializeInto(map[string]any(r), ctx.Target, dst, ctx.Flags)
}

func (r NewRow) UnmarshalAstraRaw(_ serdes.DecodeCtx, _ []byte) error {
	return fmt.Errorf("cannot deserialize into NewRow; use the astra.Row interface for results")
}

// endregion

// region serverDocument

type serverDocument struct {
	data  map[string]json.RawMessage
	flags serdes.DesFlags
}

func (d *serverDocument) isDocument() {}

func (d *serverDocument) ToMap() map[string]any {
	result := make(map[string]any, len(d.data))

	for name, rawValue := range d.data {
		if string(rawValue) == "null" {
			result[name] = nil
			continue
		}

		var val any
		_ = serdes.Deserialize(rawValue, &val, nil, serdes.TargetCollection, d.flags)
		result[name] = val
	}
	return result
}

func (d *serverDocument) Get(path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	currentRaw, ok := d.data[path[0]]
	if !ok {
		return nil, false
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetCollection, d.flags); err != nil {
			return nil, false
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return nil, false
		}
	}

	var generic any
	if err := serdes.Deserialize(currentRaw, &generic, nil, serdes.TargetCollection, d.flags); err != nil {
		return nil, false
	}
	return generic, true
}

func (d *serverDocument) MustGet(path ...string) any {
	return mustGet(d.Get, path, "Document")
}

func (d *serverDocument) Decode(dest any, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("astra: empty path for Decode")
	}

	currentRaw, ok := d.data[path[0]]
	if !ok {
		return fmt.Errorf("astra: path %q not found", path[0])
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetCollection, d.flags); err != nil {
			return fmt.Errorf("astra: failed to decode intermediate path %q: %w", path[i-1], err)
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return fmt.Errorf("astra: path %q not found", path[i])
		}
	}

	return serdes.Deserialize(currentRaw, dest, nil, serdes.TargetCollection, d.flags)
}

func (d *serverDocument) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	d.data = make(map[string]json.RawMessage)
	d.flags = ctx.Flags
	return serdes.Deserialize(value, &d.data, nil, serdes.TargetCollection, d.flags)
}

func (d *serverDocument) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetCollection {
		return nil, fmt.Errorf("`Document` can only be serialized for collections, got %s", ctx.Target)
	}
	return serdes.SerializeInto(d.data, ctx.Target, dst, ctx.Flags)
}

// endregion

// region Document TargetCtx

var documentInterfaceType = reflect.TypeFor[Document]()

type documentTargetCtx struct{}

func (documentTargetCtx) UntypedTargetInterface() reflect.Type {
	return documentInterfaceType
}

func (documentTargetCtx) NewUntypedTarget(ctx serdes.DecodeCtx, p unsafe.Pointer) serdes.AstraRawUnmarshaler {
	doc := &serverDocument{flags: ctx.Flags}
	*(*Document)(p) = doc
	return doc
}

var documentCtx = documentTargetCtx{}

func NewDocumentTargetCtx() serdes.TargetDecodeCtx {
	return documentCtx
}

// endregion

// region serverRow

type serverRow struct {
	data   map[string]json.RawMessage
	schema *lazySchema
	flags  serdes.DesFlags
}

func (r *serverRow) isRow() {}

func (r *serverRow) ToMap() map[string]any {
	schema := r.schema.Get()
	result := make(map[string]any, len(r.data))

	ctx := serdes.DecodeCtx{Target: serdes.TargetTable, TargetCtx: r.schema, Flags: r.flags}

	for name, rawValue := range r.data {
		if string(rawValue) == "null" {
			result[name] = nil
			continue
		}

		col, ok := schema.Get(name)
		if ok {
			val, err := deserializeColumn(ctx, rawValue, col)
			if err != nil {
				panic(fmt.Sprintf("astra: failed to decode field %q using schema type %q: %v", name, col.Type, err))
			}
			result[name] = val
			continue
		}

		// Generic fallback for metadata fields OR schema mismatch
		var val any
		_ = serdes.Deserialize(rawValue, &val, nil, serdes.TargetTable, r.flags)
		result[name] = val
	}

	if r.flags&serdes.SparseRows == 0 {
		for _, col := range schema {
			if _, ok := result[col.Name]; !ok {
				result[col.Name] = nil
			}
		}
	}

	return result
}

func (r *serverRow) Get(path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	currentRaw, ok := r.data[path[0]]
	if !ok {
		if r.flags&serdes.SparseRows == 0 {
			if _, ok := r.schema.Get().Get(path[0]); ok {
				return nil, true
			}
		}
		return nil, false
	}

	currentCol, hasCol := r.schema.Get().Get(path[0])
	ctx := serdes.DecodeCtx{Target: serdes.TargetTable, TargetCtx: r.schema, Flags: r.flags}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetTable, r.flags); err != nil {
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

	if hasCol {
		val, err := deserializeColumn(ctx, currentRaw, currentCol)
		if err != nil {
			panic(fmt.Sprintf("astra: failed to decode path %v using schema type %q: %v", path, currentCol.Type, err))
		}
		return val, true
	}

	// Fallback to generic decoding if no column info
	var generic any
	if err := serdes.Deserialize(currentRaw, &generic, nil, serdes.TargetTable, r.flags); err != nil {
		return nil, false
	}
	return generic, true
}

func (r *serverRow) MustGet(path ...string) any {
	return mustGet(r.Get, path, "Row")
}

func (r *serverRow) Decode(dest any, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("astra: empty path for Decode")
	}

	currentRaw, ok := r.data[path[0]]
	if !ok {
		return fmt.Errorf("astra: path %q not found", path[0])
	}

	currentCol, hasCol := r.schema.Get().Get(path[0])

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetTable, r.flags); err != nil {
			return fmt.Errorf("astra: failed to decode intermediate path %q: %w", path[i-1], err)
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return fmt.Errorf("astra: path %q not found", path[i])
		}

		if hasCol {
			currentCol, hasCol = getSubColumn(currentCol, path[i])
		}
	}

	if _, ok := dest.(*any); ok && hasCol {
		ctx := serdes.DecodeCtx{Target: serdes.TargetTable, TargetCtx: r.schema, Flags: r.flags}
		val, err := deserializeColumn(ctx, currentRaw, currentCol)
		if err != nil {
			return err
		}
		*dest.(*any) = val
		return nil
	}

	var targetCtx serdes.TargetDecodeCtx
	if hasCol && currentCol.Type == table.TypeUDT {
		targetCtx = &lazySchema{AsCols: currentCol.UDTDefinition.Fields}
	}

	return serdes.Deserialize(currentRaw, dest, targetCtx, serdes.TargetTable, r.flags)
}

func (r *serverRow) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	r.data = make(map[string]json.RawMessage)
	r.flags = ctx.Flags
	return serdes.Deserialize(value, &r.data, nil, serdes.TargetTable, r.flags)
}

func (r *serverRow) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetTable {
		return nil, fmt.Errorf("`Row` can only be serialized for tables, got %s", ctx.Target)
	}
	return serdes.SerializeInto(r.data, ctx.Target, dst, ctx.Flags)
}

// endregion

// region Row TargetCtx

type lazySchema struct {
	AsRaw  json.RawMessage
	AsCols table.Columns
}

func (s *lazySchema) Get() table.Columns {
	if s.AsCols != nil {
		return s.AsCols
	}

	if len(s.AsRaw) == 0 || string(s.AsRaw) == "null" {
		return nil
	}

	var cols table.Columns
	if err := serdes.Deserialize(s.AsRaw, &cols, nil, serdes.TargetTable); err != nil {
		panic(fmt.Sprintf("astra: failed to deserialize schema: %v", err))
	}

	s.AsCols = cols
	s.AsRaw = nil
	return cols
}

var rowInterfaceType = reflect.TypeFor[Row]()

func (s *lazySchema) UntypedTargetInterface() reflect.Type {
	return rowInterfaceType
}

func (s *lazySchema) NewUntypedTarget(ctx serdes.DecodeCtx, p unsafe.Pointer) serdes.AstraRawUnmarshaler {
	row := &serverRow{schema: s, flags: ctx.Flags}
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

// endregion

// region Column Deserialization

func deserializeColumn(ctx serdes.DecodeCtx, raw json.RawMessage, col table.Column) (any, error) {
	if string(raw) == "null" {
		return nil, nil
	}

	switch col.Type {
	case table.TypeInt:
		var v int
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeBigInt:
		var v int64
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeSmallInt:
		var v int16
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeTinyInt:
		var v int8
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeFloat:
		var v float32
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeDouble:
		var v float64
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeVarint:
		var v big.Int
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeDecimal:
		var v big.Float
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeText, table.TypeAscii:
		var v string
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeBoolean:
		var v bool
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeDate:
		var v datatypes.DateOnly
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeTime:
		var v datatypes.TimeOnly
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeTimestamp:
		var v time.Time
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeDuration:
		var v datatypes.Duration
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeUUID, table.TypeTimeUUID:
		var v datatypes.UUID
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeBlob:
		var v []byte
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeInet:
		var v net.IP
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeVector:
		var v datatypes.Vector
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err

	case table.TypeSet, table.TypeList:
		return deserializeListLike(ctx, raw, col)

	case table.TypeMap:
		return deserializeMap(ctx, raw, col)

	case table.TypeUDT:
		return deserializeUDT(ctx, raw, col)

	default:
		var v any
		err := serdes.Deserialize(raw, &v, nil, serdes.TargetTable, ctx.Flags)
		return v, err
	}
}

func deserializeListLike(ctx serdes.DecodeCtx, raw json.RawMessage, col table.Column) (any, error) {
	var rawArray []json.RawMessage
	if err := serdes.Deserialize(raw, &rawArray, nil, serdes.TargetTable, ctx.Flags); err != nil {
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
	if err := serdes.Deserialize(raw, &rawMap, nil, serdes.TargetTable, ctx.Flags); err != nil {
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
	if err := serdes.Deserialize(raw, &r, newCtx.TargetCtx, serdes.TargetTable, ctx.Flags); err != nil {
		return nil, err
	}
	return r.ToMap(), nil
}

// endregion

// region Helpers

func getDeepFromMap(m map[string]any, path ...string) (any, bool) {
	current := m
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

func mustGet(get func(path ...string) (any, bool), path []string, target string) any {
	val, ok := get(path...)
	if !ok {
		panic(fmt.Sprintf("astra: path %v not found in %s", path, target))
	}
	return val
}

func decodeFromMap(m map[string]any, path []string, dest any, target serdes.Target) error {
	val, ok := getDeepFromMap(m, path...)
	if !ok {
		return fmt.Errorf("astra: path %v not found", path)
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("astra: destination must be a non-nil pointer")
	}

	if val != nil {
		srcVal := reflect.ValueOf(val)
		if srcVal.Type().AssignableTo(rv.Elem().Type()) {
			rv.Elem().Set(srcVal)
			return nil
		}
	} else {
		elem := rv.Elem()
		switch elem.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map:
			elem.Set(reflect.Zero(elem.Type()))
			return nil
		}
	}

	b, err := serdes.Serialize(val, target)
	if err != nil {
		return err
	}

	return serdes.Deserialize(b, dest, nil, target)
}

// endregion
