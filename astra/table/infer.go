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

package table

import (
	"fmt"
	"math/big"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/datastax/astra-db-go/astra/datatypes"
)

// Well-known reflect types for comparison in type mapping.
var (
	reflectUUID      = reflect.TypeFor[datatypes.UUID]()
	reflectVector    = reflect.TypeFor[datatypes.Vector]()
	reflectTime      = reflect.TypeFor[time.Time]()
	reflectDuration  = reflect.TypeFor[datatypes.Duration]()
	reflectIP        = reflect.TypeFor[net.IP]()
	reflectByteSlice = reflect.TypeFor[[]byte]()
	reflectDateOnly  = reflect.TypeFor[datatypes.DateOnly]()
	reflectTimeOnly  = reflect.TypeFor[datatypes.TimeOnly]()
	reflectBigInt    = reflect.TypeFor[big.Int]()
	reflectBigFloat  = reflect.TypeFor[big.Float]()
)

// fieldData holds processed metadata about a single struct field.
type fieldData struct {
	columnName string
	goType     reflect.Type
	tag        tagInfo
	fieldIdx   int // position for stable ordering
}

// collectFields iterates over the fields of a struct type (including promoted
// fields from embedded structs at any depth) and returns their processed
// metadata. Outer fields shadow embedded fields with the same column name,
// matching encoding/json promotion semantics. Two direct fields on the outer
// struct with the same column name are rejected as a user error.
func collectFields(t reflect.Type) ([]fieldData, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", t.Kind())
	}

	seen := make(map[string]bool)
	var result []fieldData
	var queue []reflect.Type

	// Direct fields on the outer struct. Duplicate column names here are a
	// user error (two fields with the same json tag on the same struct).
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			if ft := derefStruct(f.Type); ft != nil {
				queue = append(queue, ft)
			}
			continue
		}
		fd, skip, err := processField(f, i)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		if seen[fd.columnName] {
			return nil, fmt.Errorf("duplicate column name %q", fd.columnName)
		}
		seen[fd.columnName] = true
		result = append(result, fd)
	}

	// BFS over embedded types: outer-most level wins, nested embeds are
	// descended into so promotion works at arbitrary depth.
	baseOffset := t.NumField()
	for len(queue) > 0 {
		et := queue[0]
		queue = queue[1:]
		for i := 0; i < et.NumField(); i++ {
			f := et.Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Anonymous {
				if ft := derefStruct(f.Type); ft != nil {
					queue = append(queue, ft)
				}
				continue
			}
			fd, skip, err := processField(f, baseOffset+i)
			if err != nil {
				return nil, err
			}
			if skip || seen[fd.columnName] {
				continue
			}
			seen[fd.columnName] = true
			result = append(result, fd)
		}
		baseOffset += et.NumField()
	}

	return result, nil
}

// derefStruct unwraps pointer types and returns the underlying struct type,
// or nil if t does not resolve to a struct.
func derefStruct(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return nil
}

// processField extracts fieldData from a single struct field.
func processField(f reflect.StructField, idx int) (fieldData, bool, error) {
	name, include := columnName(f)
	if !include {
		return fieldData{}, true, nil
	}
	// Skip $-prefixed fields (API metadata: $similarity, $vector, $vectorize)
	if strings.HasPrefix(name, "$") {
		return fieldData{}, true, nil
	}

	tag, err := parseAstraTag(f.Tag.Get("astra"))
	if err != nil {
		return fieldData{}, false, fmt.Errorf("field %q: %w", f.Name, err)
	}
	if tag.skip {
		return fieldData{}, true, nil
	}

	return fieldData{
		columnName: name,
		goType:     f.Type,
		tag:        tag,
		fieldIdx:   idx,
	}, false, nil
}

// goTypeToColumn converts a Go reflect.Type to a table Column using tag metadata.
func goTypeToColumn(t reflect.Type, info tagInfo) (Column, error) {
	// Unwrap pointers
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// jsonString — field is stored as a text column
	if info.isJSONString {
		return Text(), nil
	}

	// Vectorize produces a vector column regardless of Go type
	if info.hasVectorize {
		svc := &VectorService{
			Provider:  info.provider,
			ModelName: info.model,
		}
		return VectorWithService(info.dimension, svc), nil
	}

	// Explicit vector modifier
	if info.isVector {
		if info.dimension <= 0 {
			return Column{}, fmt.Errorf("vector modifier requires dim=N")
		}
		return Vector(info.dimension), nil
	}

	// Type override
	if info.typeOverride != "" {
		expr, err := parseTypeExpr(info.typeOverride)
		if err != nil {
			return Column{}, fmt.Errorf("type=%s: %w", info.typeOverride, err)
		}
		return resolveTypeExpr(expr, t, info)
	}

	// Standalone dim= implies vector
	if info.dimension > 0 {
		return Vector(info.dimension), nil
	}

	// Known named types (by reflect.Type identity)
	switch t {
	case reflectUUID:
		return UUID(), nil
	case reflectVector:
		return Column{}, fmt.Errorf("Vector requires dim=N in astra tag")
	case reflectTime:
		return Timestamp(), nil
	case reflectDuration:
		return Column{Type: "duration"}, nil
	case reflectIP:
		return Inet(), nil
	case reflectByteSlice:
		return Blob(), nil
	case reflectDateOnly:
		return Date(), nil
	case reflectTimeOnly:
		return Time(), nil
	case reflectBigInt:
		return Varint(), nil
	case reflectBigFloat:
		return Decimal(), nil
	}

	// Kind-based mapping
	switch t.Kind() {
	case reflect.String:
		return Text(), nil
	case reflect.Int, reflect.Int32:
		return Int(), nil
	case reflect.Int64:
		return BigInt(), nil
	case reflect.Int16:
		return SmallInt(), nil
	case reflect.Int8, reflect.Uint8:
		return TinyInt(), nil
	case reflect.Float32:
		return Float(), nil
	case reflect.Float64:
		return Double(), nil
	case reflect.Bool:
		return Boolean(), nil
	case reflect.Slice:
		elem, err := goTypeToColumn(t.Elem(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("list element: %w", err)
		}
		return List(elem), nil
	case reflect.Map:
		keyCol, err := goTypeToColumn(t.Key(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("map key: %w", err)
		}
		valCol, err := goTypeToColumn(t.Elem(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("map value: %w", err)
		}
		return Map(keyCol.Type, valCol), nil
	case reflect.Interface:
		return Column{}, fmt.Errorf("interface type requires vectorize or type= modifier")
	case reflect.Struct:
		return Column{}, fmt.Errorf("struct type %s requires jsonString or type= modifier", t.Name())
	default:
		return Column{}, fmt.Errorf("unsupported Go type %s", t)
	}
}

// resolveTypeExpr walks a parsed typeExpr and produces a Column. It consults
// goType only at "infer" leaves and at container boundaries where the Go
// type's shape (slice / map) governs how values will be marshaled. For
// container leaves declared explicitly (e.g. set[ascii]), the declared leaf
// wins and the Go element type is not consulted — callers are trusted to
// serialize values compatibly.
func resolveTypeExpr(expr typeExpr, t reflect.Type, info tagInfo) (Column, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch expr.name {
	case "infer":
		return goTypeToColumn(t, tagInfo{})

	case "udt":
		if expr.udtName == "" {
			return Column{}, fmt.Errorf("udt requires a name")
		}
		return UDT(expr.udtName), nil

	case TypeSet:
		if t.Kind() != reflect.Slice {
			return Column{}, fmt.Errorf("type=set requires slice Go type, got %s", t)
		}
		elem, err := resolveTypeExpr(*expr.elem, t.Elem(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("set element: %w", err)
		}
		return Set(elem), nil

	case TypeList:
		if t.Kind() != reflect.Slice {
			return Column{}, fmt.Errorf("type=list requires slice Go type, got %s", t)
		}
		elem, err := resolveTypeExpr(*expr.elem, t.Elem(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("list element: %w", err)
		}
		return List(elem), nil

	case TypeMap:
		if t.Kind() != reflect.Map {
			return Column{}, fmt.Errorf("type=map requires map Go type, got %s", t)
		}
		keyCol, err := resolveTypeExpr(*expr.key, t.Key(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("map key: %w", err)
		}
		valCol, err := resolveTypeExpr(*expr.elem, t.Elem(), tagInfo{})
		if err != nil {
			return Column{}, fmt.Errorf("map value: %w", err)
		}
		return Map(keyCol.Type, valCol), nil

	case TypeVector:
		if info.dimension <= 0 {
			return Column{}, fmt.Errorf("type=vector requires dim=N")
		}
		return Vector(info.dimension), nil

	default:
		if factory, ok := scalarFactories[expr.name]; ok {
			return factory(), nil
		}
		return Column{}, fmt.Errorf("unknown type override %q", expr.name)
	}
}

// scalarFactories maps a scalar type name to the Column factory that produces
// the corresponding column. Vector and the container types are handled
// directly in resolveTypeExpr because they need extra inputs.
var scalarFactories = map[string]func() Column{
	TypeText:      Text,
	TypeAscii:     Ascii,
	TypeInt:       Int,
	TypeBigInt:    BigInt,
	TypeSmallInt:  SmallInt,
	TypeTinyInt:   TinyInt,
	TypeFloat:     Float,
	TypeDouble:    Double,
	TypeDecimal:   Decimal,
	TypeBoolean:   Boolean,
	TypeDate:      Date,
	TypeTime:      Time,
	TypeTimestamp: Timestamp,
	TypeUUID:      UUID,
	TypeTimeUUID:  TimeUUID,
	TypeBlob:      Blob,
	TypeVarint:    Varint,
	TypeInet:      Inet,
	TypeDuration:  Duration,
}

// keyField is used during primary key assembly.
type keyField struct {
	name    string
	ordinal int
	idx     int  // struct field index for stable ordering
	desc    bool // clustering key: true = descending
}

// buildPrimaryKey assembles the PrimaryKey from collected partition and clustering key fields.
func buildPrimaryKey(pks, cks []keyField) (PrimaryKey, error) {
	if len(pks) == 0 {
		return PrimaryKey{}, fmt.Errorf("no partition key defined (tag at least one field with astra:\"pk\")")
	}

	// Determine ordering strategy: all ordinals or all struct-order
	hasOrd, noOrd := false, false
	for _, pk := range pks {
		if pk.ordinal > 0 {
			hasOrd = true
		} else {
			noOrd = true
		}
	}
	if hasOrd && noOrd {
		return PrimaryKey{}, fmt.Errorf("partition key fields must all have ordinals or all omit them")
	}

	if hasOrd {
		sort.Slice(pks, func(i, j int) bool { return pks[i].ordinal < pks[j].ordinal })
		if err := validateOrdinals("partition key", pks); err != nil {
			return PrimaryKey{}, err
		}
	} else {
		sort.Slice(pks, func(i, j int) bool { return pks[i].idx < pks[j].idx })
	}

	partitionBy := make([]string, len(pks))
	for i, pk := range pks {
		partitionBy[i] = pk.name
	}

	var partitionSort PartitionSort
	if len(cks) > 0 {
		sort.Slice(cks, func(i, j int) bool { return cks[i].ordinal < cks[j].ordinal })
		if err := validateOrdinals("clustering key", cks); err != nil {
			return PrimaryKey{}, err
		}
		partitionSort = make(PartitionSort, 0, len(cks))
		for _, ck := range cks {
			order := SortAscending
			if ck.desc {
				order = SortDescending
			}
			partitionSort = append(partitionSort, NamedSort{Name: ck.name, Order: order})
		}
	}

	pk := PrimaryKey{PartitionBy: partitionBy}
	if len(partitionSort) > 0 {
		pk.PartitionSort = partitionSort
	}
	return pk, nil
}

// validateOrdinals checks that sorted key fields have contiguous 1-based ordinals
// with no duplicates.
func validateOrdinals(label string, keys []keyField) error {
	for i, k := range keys {
		if i > 0 && k.ordinal == keys[i-1].ordinal {
			return fmt.Errorf("duplicate %s ordinal %d: %q and %q", label, k.ordinal, keys[i-1].name, k.name)
		}
		if k.ordinal != i+1 {
			return fmt.Errorf("%s ordinals must be contiguous starting from 1 (expected %d, got %d for %q)", label, i+1, k.ordinal, k.name)
		}
	}
	return nil
}

// Infer generates a table [Definition] from a Go struct's type information.
//
// Column names are taken from json tags (falling back to field names).
// Primary key configuration and type overrides come from astra tags.
//
// See the astra tag grammar:
//   - astra:"pk" or astra:"pk,N" — partition key (N = ordinal for composite keys)
//   - astra:"ck,N,asc" or astra:"ck,N,desc" — clustering key
//   - astra:"-" — skip field
//   - astra:"type=<T>" — override inferred column type
//   - astra:"dim=<N>" — vector dimension
//   - astra:"vectorize,provider=<P>,model=<M>" — vectorize service
//
// The <T> in type= supports:
//   - scalars: text, ascii, int, bigint, smallint, tinyint, float, double,
//     decimal, boolean, date, time, timestamp, uuid, timeuuid, blob, varint,
//     inet, duration, vector (vector additionally requires dim=<N>)
//   - containers: set[T], list[T], map[K]V. Bare set / list / map default to
//     inferring their inner types from the Go field type.
//   - user-defined types: udt[<name>]
//   - infer — only valid inside brackets; reuses the Go field's inferred type
//     at that position. E.g. map[uuid]infer overrides only the key type.
//
// Composed example: astra:"type=map[uuid]set[ascii]"
//
// Example:
//
//	type Book struct {
//	    Title  string  `json:"title"  astra:"pk"`
//	    Author string  `json:"author"`
//	    Rating float32 `json:"rating"`
//	}
//
//	def, err := table.Infer[Book]()
//	tbl, err := db.CreateTable(ctx, "books", def)
func Infer[T any]() (Definition, error) {
	t := reflect.TypeFor[T]()

	fields, err := collectFields(t)
	if err != nil {
		return Definition{}, fmt.Errorf("table.Infer[%s]: %w", t.Name(), err)
	}
	if len(fields) == 0 {
		return Definition{}, fmt.Errorf("table.Infer[%s]: no columns found", t.Name())
	}

	columns := make(Columns, 0, len(fields))
	var pks, cks []keyField

	for _, fd := range fields {
		col, err := goTypeToColumn(fd.goType, fd.tag)
		if err != nil {
			return Definition{}, fmt.Errorf("table.Infer[%s]: field %q: %w", t.Name(), fd.columnName, err)
		}
		columns = append(columns, NamedColumn{Name: fd.columnName, Column: col})

		if fd.tag.isPK {
			pks = append(pks, keyField{name: fd.columnName, ordinal: fd.tag.pkOrdinal, idx: fd.fieldIdx})
		}
		if fd.tag.isCK {
			cks = append(cks, keyField{name: fd.columnName, ordinal: fd.tag.ckOrdinal, idx: fd.fieldIdx, desc: fd.tag.ckDescending})
		}
	}

	primaryKey, err := buildPrimaryKey(pks, cks)
	if err != nil {
		return Definition{}, fmt.Errorf("table.Infer[%s]: %w", t.Name(), err)
	}

	return Definition{
		Columns:    columns,
		PrimaryKey: primaryKey,
	}, nil
}

// InferUDT generates a [UDTDefinition] from a Go struct's type information.
//
// Field names are taken from json tags (falling back to field names).
// Type overrides come from astra tags.
//
// See [Infer] for details on the astra tag grammar.
//
// Example:
//
//	type Address struct {
//	    Street string `json:"street"`
//	    City   string `json:"city"`
//	}
//
//	def, err := table.InferUDT[Address]()
//	err := db.CreateType(ctx, "address", def)
func InferUDT[T any]() (UDTDefinition, error) { // TODO somehow try to check if people are using the right Infer... maybe use pk tags to detect misuse?
	t := reflect.TypeFor[T]()

	fields, err := collectFields(t)
	if err != nil {
		return UDTDefinition{}, fmt.Errorf("table.InferUDT[%s]: %w", t.Name(), err)
	}
	if len(fields) == 0 {
		return UDTDefinition{}, fmt.Errorf("table.InferUDT[%s]: no fields found", t.Name())
	}

	columns := make(Columns, 0, len(fields))
	for _, fd := range fields {
		col, err := goTypeToColumn(fd.goType, fd.tag)
		if err != nil {
			return UDTDefinition{}, fmt.Errorf("table.InferUDT[%s]: field %q: %w", t.Name(), fd.columnName, err)
		}
		columns = append(columns, NamedColumn{Name: fd.columnName, Column: col})
	}

	return UDTDefinition{
		Fields: columns,
	}, nil
}
