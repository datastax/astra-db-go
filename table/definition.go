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
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/datastax/astra-db-go/serdes"
)

// Definition represents the full schema for a table, including column names,
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

func (c Columns) MarshalAstraRaw(target serdes.Target, dst []byte) ([]byte, error) {
	dst = append(dst, '{')

	var err error
	for i, nc := range c {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst, err = serdes.SerializeInto(nc.Name, target, dst)
		if err != nil {
			return dst, err
		}
		dst = append(dst, ':')
		dst, err = serdes.SerializeInto(nc.Column, target, dst)
		if err != nil {
			return dst, err
		}
	}

	dst = append(dst, '}')
	return dst, nil
}

func (c *Columns) UnmarshalAstraRaw(target serdes.Target, value []byte) error {
	rep := map[string]json.RawMessage{}
	if err := serdes.Deserialize(value, &rep, target); err != nil {
		return err
	}

	var out Columns
	for k, v := range rep {
		var col Column
		if err := serdes.Deserialize(v, &col, target); err != nil {
			return err
		}
		out = append(out, NamedColumn{Name: k, Column: col})
	}
	*c = out
	return nil
}

// Column represents a column's type definition.
// It can be a simple scalar type, a collection type (set, list, map),
// a vector type, or a user-defined type.
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
}

// isSimple reports whether c is just a type name with no modifiers. The Data
// API accepts a bare string for a simple ValueType inside a set/list/map, and
// the docs examples use that shorthand.
func (c Column) isSimple() bool {
	return c.Dimension == nil && c.Service == nil &&
		c.KeyType == nil && c.ValueType == nil && c.UDTName == nil
}

// MarshalJSON emits Column as an object, collapsing a simple inner ValueType
// to a bare string to match the Data API wire format shown in the docs.
func (c Column) MarshalJSON() ([]byte, error) {
	out := struct {
		Type      string          `json:"type"`
		Dimension *int            `json:"dimension,omitempty"`
		Service   *VectorService  `json:"service,omitempty"`
		KeyType   *string         `json:"keyType,omitempty"`
		ValueType json.RawMessage `json:"valueType,omitempty"`
		UDTName   *string         `json:"udtName,omitempty"`
	}{
		Type:      c.Type,
		Dimension: c.Dimension,
		Service:   c.Service,
		KeyType:   c.KeyType,
		UDTName:   c.UDTName,
	}

	if c.ValueType != nil {
		if c.ValueType.isSimple() {
			s, err := serdes.Serialize(c.ValueType.Type, serdes.TargetUnknown)
			if err != nil {
				return nil, err
			}
			out.ValueType = s
		} else {
			b, err := serdes.Serialize(*c.ValueType, serdes.TargetUnknown)
			if err != nil {
				return nil, err
			}
			out.ValueType = b
		}
	}

	return serdes.Serialize(out, serdes.TargetUnknown)
}

func (c Column) MarshalAstraRaw(_ serdes.Target, dst []byte) ([]byte, error) {
	b, err := c.MarshalJSON()
	if err != nil {
		return dst, err
	}
	return append(dst, b...), nil
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

// MarshalJSON emits PartitionSort as a JSON object in declared order.
func (s PartitionSort) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, ns := range s {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := json.Marshal(ns.Name)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		v, err := json.Marshal(ns.Order)
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (s PartitionSort) MarshalAstraRaw(_ serdes.Target, dst []byte) ([]byte, error) {
	b, err := s.MarshalJSON()
	if err != nil {
		return dst, err
	}
	return append(dst, b...), nil
}

// UnmarshalJSON parses a JSON object into PartitionSort, preserving key order.
func (s *PartitionSort) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("partitionSort: expected JSON object, got %v", tok)
	}
	var out PartitionSort
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name, ok := tok.(string)
		if !ok {
			return fmt.Errorf("partitionSort: expected column name, got %v", tok)
		}
		var order int
		if err := dec.Decode(&order); err != nil {
			return err
		}
		out = append(out, NamedSort{Name: name, Order: order})
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	*s = out
	return nil
}

func (s *PartitionSort) UnmarshalAstraRaw(_ serdes.Target, data []byte) error {
	return s.UnmarshalJSON(data)
}

// MarshalJSON implements custom JSON marshaling for PrimaryKey.
// If only PartitionBy has a single column and PartitionSort is empty,
// it marshals as a simple string for convenience.
func (p PrimaryKey) MarshalJSON() ([]byte, error) {
	// If single partition key with no clustering columns, marshal as string
	if len(p.PartitionBy) == 1 && len(p.PartitionSort) == 0 {
		return json.Marshal(p.PartitionBy[0])
	}

	// Otherwise marshal as object
	type pkAlias PrimaryKey
	return json.Marshal(pkAlias(p))
}

func (p PrimaryKey) MarshalAstraRaw(_ serdes.Target, dst []byte) ([]byte, error) {
	b, err := p.MarshalJSON()
	if err != nil {
		return dst, err
	}
	return append(dst, b...), nil
}

// UnmarshalJSON implements custom JSON unmarshaling for PrimaryKey.
// It handles both string format (single column) and object format (compound key).
func (p *PrimaryKey) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var singleColumn string
	if err := json.Unmarshal(data, &singleColumn); err == nil {
		p.PartitionBy = []string{singleColumn}
		p.PartitionSort = nil
		return nil
	}

	// Otherwise unmarshal as object
	type pkAlias PrimaryKey
	var pk pkAlias
	if err := json.Unmarshal(data, &pk); err != nil {
		return err
	}
	*p = PrimaryKey(pk)
	return nil
}

func (p *PrimaryKey) UnmarshalAstraRaw(_ serdes.Target, data []byte) error {
	return p.UnmarshalJSON(data)
}

// UnmarshalJSON implements custom JSON unmarshaling for Column.
// It accepts either a JSON object (e.g. {"type":"text"}) or a plain
// string (e.g. "text"). The string form appears in real-world API
// responses for nested valueType fields on list/set/map columns.
// For example, this shows the simple string form:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#create-a-table-with-a-compound-primary-key
// And this shows the nested JSON object form:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/table-methods/create-table.html#example-create-table-udt
func (c *Column) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*c = Column{Type: s}
		return nil
	}
	type colAlias Column
	var col colAlias
	if err := json.Unmarshal(data, &col); err != nil {
		return err
	}
	*c = Column(col)
	return nil
}

func (c *Column) UnmarshalAstraRaw(_ serdes.Target, data []byte) error {
	return c.UnmarshalJSON(data)
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

// AlterOperation represents the operation to perform on a table via alterTable.
// Exactly one of the operation fields must be set per call — they are
// mutually exclusive.
//
// Example — add columns:
//
//	op := table.AlterOperation{
//		Add: &table.AddColumns{
//			Columns: table.Columns{
//				"is_summer_reading": table.Boolean(),
//				"library_branch":    table.Text(),
//			},
//		},
//	}
//
// Example — drop columns:
//
//	op := table.AlterOperation{
//		Drop: &table.DropColumns{Columns: []string{"borrower"}},
//	}
type AlterOperation struct {
	// Add adds new columns to the table.
	Add *AddColumns `json:"add,omitempty"`

	// Drop removes existing columns from the table.
	Drop *DropColumns `json:"drop,omitempty"`

	// AddVectorize attaches an embedding-generation integration to existing
	// vector columns.
	AddVectorize *AddVectorize `json:"addVectorize,omitempty"`

	// DropVectorize removes the embedding-generation integration from existing
	// vector columns. Stored embeddings are preserved.
	DropVectorize *DropVectorize `json:"dropVectorize,omitempty"`
}

// AddColumns is the payload for the alterTable "add" operation.
type AddColumns struct {
	// Columns maps new column names to their type definitions.
	Columns Columns `json:"columns"`
}

// DropColumns is the payload for the alterTable "drop" operation.
type DropColumns struct {
	// Columns is the list of column names to remove from the table.
	Columns []string `json:"columns"`
}

// AddVectorize is the payload for the alterTable "addVectorize" operation.
type AddVectorize struct {
	// Columns maps existing vector column names to their vectorize service
	// configuration.
	Columns map[string]VectorService `json:"columns"`
}

// DropVectorize is the payload for the alterTable "dropVectorize" operation.
type DropVectorize struct {
	// Columns is the list of vector column names to disable vectorize on.
	Columns []string `json:"columns"`
}
