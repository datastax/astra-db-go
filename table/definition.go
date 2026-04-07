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

import "encoding/json"

// Definition represents the full schema for a table, including column names,
// column data types, and the primary key.
//
// Example:
//
//	def := table.Definition{
//		Columns: map[string]table.Column{
//			"title":  table.Text(),
//			"author": table.Text(),
//			"rating": table.Float(),
//		},
//		PrimaryKey: table.PrimaryKey{
//			PartitionBy: []string{"title"},
//		},
//	}
type Definition struct {
	// Columns defines all columns in the table with their types
	Columns map[string]Column `json:"columns"`

	// PrimaryKey defines the primary key for the table
	PrimaryKey PrimaryKey `json:"primaryKey"`
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

	// ValueType is used for set and list columns
	ValueType *Column `json:"valueType,omitempty"`

	// KeyType is used for map columns
	KeyType *string `json:"keyType,omitempty"`

	// UDTName is used for userDefined columns to specify the UDT name
	UDTName *string `json:"udtName,omitempty"`
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

	// PartitionSort defines clustering columns and their sort order (1 for ASC, -1 for DESC)
	// This is optional and used for compound primary keys
	PartitionSort map[string]int `json:"partitionSort,omitempty"`
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
