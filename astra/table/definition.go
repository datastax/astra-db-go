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

// Package table provides types and utilities for working with Astra DB tables.
package table

import (
	"math/big"
	"net"
	"reflect"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

// Definition represents the full Schema for a table, including column names,
// column data types, and the primary key.
//
// Example:
//
//	def := table.Definition{
//		Columns: table.Columns{
//			{"title", table.Text()},
//			{"author", table.Text()},
//			{"rating", table.Float()},
//		},
//		PrimaryKey: table.PrimaryKey{
//			PartitionBy: []string{"title"},
//		},
//	}
type Definition struct {
	// Columns defines all columns in the table with their types.
	// Order is preserved on marshal and captured on unmarshal.
	Columns Columns `json:"columns"`

	// PrimaryKey defines the primary key for the table
	PrimaryKey PrimaryKey `json:"primaryKey"`
}

func (d Definition) build() Definition {
	return d
}

// Columns is an ordered collection of named columns. It marshals as a JSON
// object, preserving insertion order on output and input order on parse.
//
// Construct with a literal:
//
//	table.Columns{
//	    {"title",  table.Text()},
//	    {"author", table.Text()},
//	}
type Columns []NamedColumn

// NamedColumn pairs a column name with its type definition.
type NamedColumn struct {
	Name   string
	Column Column
}

// Get returns the column with the given name and whether it was found.
func (c Columns) Get(name string) (Column, bool) {
	for _, nc := range c {
		if nc.Name == name {
			return nc.Column, true
		}
	}
	return Column{}, false
}

func (c Columns) MarshalAstra(_ serdes.EncodeCtx) (any, error) {
	rep := datatypes.NewLinkedMapWithCapacity[string, Column](len(c))
	for _, nc := range c {
		rep.Set(nc.Name, nc.Column)
	}
	return rep, nil
}

func (c *Columns) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	var rep datatypes.LinkedMap[string, Column]
	if err := serdes.Deserialize(value, &rep, nil, ctx.Target, ctx.Flags); err != nil {
		return err
	}

	*c = nil
	for name, col := range rep.All() {
		*c = append(*c, NamedColumn{name, col})
	}

	return nil
}

// Column represents a column's type definition.
// It can be a simple scalar type, a collection type (set, list, map),
// a vector type, or a user-defined type.
//
//goland:noinspection GoVetStructTag
type Column struct {
	// Type is the column type (text, int, float, boolean, uuid, date, vector, set, list, map, userDefined, etc.)
	Type string `json:"type"`

	// Dimension is used for vector columns to specify the vector dimension
	Dimension *int `json:"dimension,omitempty"`

	// Service is used for vector columns with vectorize embedding provider integration
	Service *VectorService `json:"service,omitempty"`

	// KeyType is used for map columns
	KeyType *string `json:"keyType,omitempty"`

	// ValueType is used for set, list, and map columns
	ValueType *Column `json:"valueType,omitempty"`

	// UDTName is used for userDefined columns to specify the UDT name
	UDTName *string `json:"udtName,omitempty"`

	// UDTDefinition is the definition of the UDT if this column is a user defined type.
	// This is returned by the server; users should not set this themselves when defining a table schema.
	UDTDefinition *UDTDefinition `json:"definition,omitempty"`
}

func (c Column) GoType() reflect.Type {
	switch c.Type {
	case TypeInt:
		return reflect.TypeFor[int32]()
	case TypeBigInt:
		return reflect.TypeFor[int64]()
	case TypeSmallInt:
		return reflect.TypeFor[int16]()
	case TypeTinyInt:
		return reflect.TypeFor[int8]()
	case TypeFloat:
		return reflect.TypeFor[float32]()
	case TypeDouble:
		return reflect.TypeFor[float64]()
	case TypeVarint:
		return reflect.TypeFor[big.Int]()
	case TypeDecimal:
		return reflect.TypeFor[big.Float]()
	case TypeText, TypeAscii:
		return reflect.TypeFor[string]()
	case TypeBoolean:
		return reflect.TypeFor[bool]()
	case TypeDate:
		return reflect.TypeFor[datatypes.DateOnly]()
	case TypeTime:
		return reflect.TypeFor[datatypes.TimeOnly]()
	case TypeTimestamp:
		return reflect.TypeFor[time.Time]()
	case TypeDuration:
		return reflect.TypeFor[time.Duration]()
	case TypeUUID, TypeTimeUUID:
		return reflect.TypeFor[datatypes.UUID]()
	case TypeBlob:
		return reflect.TypeFor[[]byte]()
	case TypeInet:
		return reflect.TypeFor[net.IP]()
	case TypeVector:
		return reflect.TypeFor[datatypes.Vector]()
	case TypeMap:
		colType := Column{Type: *c.KeyType}.GoType()

		if colType.Comparable() {
			return reflect.MapOf(colType, c.ValueType.GoType())
		}

		return reflect.TypeFor[datatypes.SortedMap[any, any]]()
	case TypeList, TypeSet:
		return reflect.SliceOf(c.ValueType.GoType())
	case TypeUDT:
		return reflect.TypeFor[map[string]any]()
	default:
		return reflect.TypeFor[any]()
	}
}

type UDTDefinition struct {
	Fields Columns `json:"fields"`
}

// VectorService defines the embedding provider configuration for vectorize
type VectorService struct {
	// Provider is the embedding provider name (e.g., "openai", "nvidia", "azureOpenAI")
	Provider string `json:"provider"`

	// ModelName is the model to use for generating embeddings
	ModelName string `json:"modelName"`

	// Authentication contains authentication configuration
	Authentication map[string]string `json:"authentication,omitempty"`

	// Parameters contains provider-specific parameters
	Parameters map[string]string `json:"parameters,omitempty"`
}

// PrimaryKey defines the primary key structure for a table.
// It can be a single column name or a compound/composite key definition.
type PrimaryKey struct {
	// PartitionBy lists the partition key columns
	PartitionBy []string `json:"partitionBy"`

	// PartitionSort defines clustering columns and their sort order (1 for ASC, -1 for DESC).
	// Order is significant — it defines the physical sort order of rows within a partition —
	// and is preserved through JSON marshaling.
	PartitionSort PartitionSort `json:"partitionSort,omitempty"`
}

// PartitionSort is an ordered collection of clustering columns with their sort
// direction. It marshals as a JSON object, preserving declared order on output
// and input order on parse.
//
// Construct with a literal:
//
//	table.PartitionSort{
//	    {"event_time", table.SortDescending},
//	    {"priority",   table.SortAscending},
//	}
type PartitionSort []NamedSort

// NamedSort pairs a clustering-column name with its sort direction
// (SortAscending or SortDescending).
type NamedSort struct {
	Name  string
	Order int
}

// Get returns the sort order for the given column and whether it was found.
func (s PartitionSort) Get(name string) (int, bool) {
	for _, ns := range s {
		if ns.Name == name {
			return ns.Order, true
		}
	}
	return 0, false
}

func (s PartitionSort) MarshalAstra(_ serdes.EncodeCtx) (any, error) {
	rep := datatypes.NewLinkedMapWithCapacity[string, int](len(s))
	for _, ns := range s {
		rep.Set(ns.Name, ns.Order)
	}
	return rep, nil
}

func (s *PartitionSort) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	var rep datatypes.LinkedMap[string, int]
	if err := serdes.Deserialize(value, &rep, nil, ctx.Target, ctx.Flags); err != nil {
		return err
	}

	*s = nil
	for name, order := range rep.All() {
		*s = append(*s, NamedSort{Name: name, Order: order})
	}
	return nil
}

func (p PrimaryKey) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	// If single partition key with no clustering columns, marshal as string
	if len(p.PartitionBy) == 1 && len(p.PartitionSort) == 0 {
		return serdes.SerializeInto(p.PartitionBy[0], ctx.Target, dst, ctx.Flags) // TODO is there really a point in special-casing this here
	}

	// Otherwise marshal as object
	type pkAlias PrimaryKey
	return serdes.SerializeInto(pkAlias(p), ctx.Target, dst, ctx.Flags)
}

func (p *PrimaryKey) UnmarshalAstraRaw(ctx serdes.DecodeCtx, data []byte) error {
	// Try to unmarshal as string first
	var singleColumn string
	if err := serdes.Deserialize(data, &singleColumn, nil, ctx.Target, ctx.Flags); err == nil {
		p.PartitionBy = []string{singleColumn}
		p.PartitionSort = nil
		return nil
	}

	// Otherwise unmarshal as object
	type pkAlias PrimaryKey
	var pk pkAlias
	if err := serdes.Deserialize(data, &pk, nil, ctx.Target, ctx.Flags); err != nil {
		return err
	}
	*p = PrimaryKey(pk)
	return nil
}

func (c *Column) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	// Try to unmarshal as a string first (e.g. "text")
	var typ string
	if err := serdes.Deserialize(value, &typ, nil, ctx.Target, ctx.Flags); err == nil {
		*c = Column{Type: typ}
		return nil
	}

	// Otherwise unmarshal as a full Column object (e.g. {"type":"text"})
	type colAlias Column
	var col colAlias
	if err := serdes.Deserialize(value, &col, nil, ctx.Target, ctx.Flags); err != nil {
		return err
	}
	*c = Column(col)
	return nil
}

// Sort order constants
const (
	SortAscending  = 1
	SortDescending = -1
)

// Column type constants
const (
	TypeText      = "text"
	TypeInt       = "int"
	TypeBigInt    = "bigint"
	TypeSmallInt  = "smallint"
	TypeTinyInt   = "tinyint"
	TypeFloat     = "float"
	TypeDouble    = "double"
	TypeDecimal   = "decimal"
	TypeBoolean   = "boolean"
	TypeDate      = "date"
	TypeTime      = "time"
	TypeTimestamp = "timestamp"
	TypeUUID      = "uuid"
	TypeTimeUUID  = "timeuuid"
	TypeBlob      = "blob"
	TypeVarint    = "varint"
	TypeInet      = "inet"
	TypeAscii     = "ascii"
	TypeVector    = "vector"
	TypeSet       = "set"
	TypeList      = "list"
	TypeMap       = "map"
	TypeUDT       = "userDefined"
	TypeDuration  = "duration"
)

// Text creates a text column
func Text() Column {
	return Column{Type: TypeText}
}

// Int creates an int column
func Int() Column {
	return Column{Type: TypeInt}
}

// BigInt creates a bigint column
func BigInt() Column {
	return Column{Type: TypeBigInt}
}

// SmallInt creates a smallint column
func SmallInt() Column {
	return Column{Type: TypeSmallInt}
}

// TinyInt creates a tinyint column
func TinyInt() Column {
	return Column{Type: TypeTinyInt}
}

// Float creates a float column
func Float() Column {
	return Column{Type: TypeFloat}
}

// Double creates a double column
func Double() Column {
	return Column{Type: TypeDouble}
}

// Decimal creates a decimal column
func Decimal() Column {
	return Column{Type: TypeDecimal}
}

// Boolean creates a boolean column
func Boolean() Column {
	return Column{Type: TypeBoolean}
}

// Date creates a date column
func Date() Column {
	return Column{Type: TypeDate}
}

// Time creates a time column
func Time() Column {
	return Column{Type: TypeTime}
}

// Timestamp creates a timestamp column
func Timestamp() Column {
	return Column{Type: TypeTimestamp}
}

// UUID creates a UUID column
func UUID() Column {
	return Column{Type: TypeUUID}
}

// TimeUUID creates a TimeUUID column
func TimeUUID() Column {
	return Column{Type: TypeTimeUUID}
}

// Blob creates a blob column
func Blob() Column {
	return Column{Type: TypeBlob}
}

// Varint creates a varint column
func Varint() Column {
	return Column{Type: TypeVarint}
}

// Inet creates an inet column
func Inet() Column {
	return Column{Type: TypeInet}
}

// Ascii creates an ascii column
func Ascii() Column {
	return Column{Type: TypeAscii}
}

// Duration creates a duration column
func Duration() Column {
	return Column{Type: TypeDuration}
}

// Vector creates a vector column with the specified dimension
func Vector(dimension int) Column {
	return Column{
		Type:      TypeVector,
		Dimension: &dimension,
	}
}

// VectorWithService creates a vector column with vectorize embedding provider
func VectorWithService(dimension int, service *VectorService) Column {
	col := Column{
		Type:    TypeVector,
		Service: service,
	}
	if dimension > 0 {
		col.Dimension = &dimension
	}
	return col
}

// Set creates a set column with the specified value type
func Set(valueType Column) Column {
	return Column{
		Type:      TypeSet,
		ValueType: &valueType,
	}
}

// List creates a list column with the specified value type
func List(valueType Column) Column {
	return Column{
		Type:      TypeList,
		ValueType: &valueType,
	}
}

// Map creates a map column with the specified key and value types
func Map(keyType string, valueType Column) Column {
	return Column{
		Type:      TypeMap,
		KeyType:   &keyType,
		ValueType: &valueType,
	}
}

// UDT creates a user-defined type column
func UDT(udtName string) Column {
	return Column{
		Type:    TypeUDT,
		UDTName: &udtName,
	}
}

// AlterOperation represents a single operation to perform on a table via table.Alter.
//
// Implementations include:
//   - AddColumns: Adds new columns to the schema.
//   - DropColumns: Removes existing columns.
//   - AddVectorize: Configures AI embedding generation for specific columns.
//   - DropVectorize: Removes AI embedding configurations.
//   - AddReranking: Enables reranking for the table.
//   - DropReranking: Disables reranking for the table.
//
// Example — Add columns:
//
//	tbl.Alter(ctx, table.AddColumns{
//	   Columns: table.Columns{
//	      "is_summer_reading": table.Boolean(),
//	      "library_branch":    table.Text(),
//	   },
//	})
//
// Example — Drop columns:
//
//	tbl.Alter(ctx, table.DropColumns{
//	   Columns: []string{"is_summer_reading", "library_branch"},
//	})
type AlterOperation interface {
	isAlterOp()
	serdes.AstraRawMarshaler
}

// AddColumns is the payload for the alterTable "add" operation.
type AddColumns struct {
	Columns Columns `json:"columns"`
}

func (a AddColumns) isAlterOp() {}

func (a AddColumns) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	type alias AddColumns
	return serdes.SerializeInto(map[string]any{"add": alias(a)}, ctx.Target, dst, ctx.Flags)
}

// DropColumns is the payload for the alterTable "drop" operation.
type DropColumns struct {
	Columns []string `json:"columns"`
}

func (d DropColumns) isAlterOp() {}

func (d DropColumns) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	type alias DropColumns
	return serdes.SerializeInto(map[string]any{"drop": alias(d)}, ctx.Target, dst, ctx.Flags)
}

// AddVectorize is the payload for the alterTable "addVectorize" operation.
type AddVectorize struct {
	Columns map[string]VectorService `json:"columns"`
}

func (v AddVectorize) isAlterOp() {}

func (v AddVectorize) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	type alias AddVectorize
	return serdes.SerializeInto(map[string]any{"addVectorize": alias(v)}, ctx.Target, dst, ctx.Flags)
}

// DropVectorize is the payload for the alterTable "dropVectorize" operation.
type DropVectorize struct {
	Columns []string `json:"columns"`
}

func (v DropVectorize) isAlterOp() {}

func (v DropVectorize) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	type alias DropVectorize
	return serdes.SerializeInto(map[string]any{"dropVectorize": alias(v)}, ctx.Target, dst, ctx.Flags)
}

// AddReranking is the payload for the alterTable "addReranking" operation.
type AddReranking struct {
	Service RerankService `json:"service"`
}

func (r AddReranking) isAlterOp() {}

func (r AddReranking) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	type alias AddReranking
	return serdes.SerializeInto(map[string]any{"addReranking": alias(r)}, ctx.Target, dst, ctx.Flags)
}

// DropReranking is the payload for the alterTable "dropReranking" operation.
type DropReranking struct{}

func (r DropReranking) isAlterOp() {}

func (r DropReranking) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	return serdes.SerializeInto(map[string]any{"dropReranking": map[string]any{}}, ctx.Target, dst, ctx.Flags)
}

// RerankService defines the configuration for reranking.
type RerankService struct {
	// Provider is the reranking provider name (e.g., "nvidia")
	Provider string `json:"provider"`

	// ModelName is the model to use for reranking
	ModelName string `json:"modelName"`

	// Authentication contains authentication configuration
	Authentication map[string]string `json:"authentication,omitempty"`

	// Parameters contains provider-specific parameters
	Parameters map[string]string `json:"parameters,omitempty"`
}

// AlterTypeOperation represents an operation to alter a user-defined type (UDT).
type AlterTypeOperation interface {
	isAlterTypeOp()
	// OpKey returns the JSON key used for this operation (e.g. "add", "rename").
	OpKey() string
}

// AddTypeFields is the payload for the alterType "add" operation.
type AddTypeFields struct {
	Fields Columns `json:"fields"`
}

func (a AddTypeFields) isAlterTypeOp() {}
func (a AddTypeFields) OpKey() string  { return "add" }

// RenameTypeFields is the payload for the alterType "rename" operation.
type RenameTypeFields struct {
	Fields map[string]string `json:"fields"`
}

func (r RenameTypeFields) isAlterTypeOp() {}
func (r RenameTypeFields) OpKey() string  { return "rename" }
