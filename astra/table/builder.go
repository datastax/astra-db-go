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

// columnAdder is a private interface used for F-bounded polymorphism in builders.
type columnAdder[B any] interface {
	AddColumn(name string, columnType Column) B
}

// scalarBuilder provides shared scalar column addition methods for table and UDT builders.
type scalarBuilder[B columnAdder[B]] struct {
	builder B
}

// AddTextColumn adds a text column or field.
func (s scalarBuilder[B]) AddTextColumn(name string) B {
	return s.builder.AddColumn(name, Text())
}

// AddIntColumn adds an int column or field.
func (s scalarBuilder[B]) AddIntColumn(name string) B {
	return s.builder.AddColumn(name, Int())
}

// AddBigIntColumn adds a bigint column or field.
func (s scalarBuilder[B]) AddBigIntColumn(name string) B {
	return s.builder.AddColumn(name, BigInt())
}

// AddSmallIntColumn adds a smallint column or field.
func (s scalarBuilder[B]) AddSmallIntColumn(name string) B {
	return s.builder.AddColumn(name, SmallInt())
}

// AddTinyIntColumn adds a tinyint column or field.
func (s scalarBuilder[B]) AddTinyIntColumn(name string) B {
	return s.builder.AddColumn(name, TinyInt())
}

// AddFloatColumn adds a float column or field.
func (s scalarBuilder[B]) AddFloatColumn(name string) B {
	return s.builder.AddColumn(name, Float())
}

// AddDoubleColumn adds a double column or field.
func (s scalarBuilder[B]) AddDoubleColumn(name string) B {
	return s.builder.AddColumn(name, Double())
}

// AddDecimalColumn adds a decimal column or field.
func (s scalarBuilder[B]) AddDecimalColumn(name string) B {
	return s.builder.AddColumn(name, Decimal())
}

// AddBooleanColumn adds a boolean column or field.
func (s scalarBuilder[B]) AddBooleanColumn(name string) B {
	return s.builder.AddColumn(name, Boolean())
}

// AddDateColumn adds a date column or field.
func (s scalarBuilder[B]) AddDateColumn(name string) B {
	return s.builder.AddColumn(name, Date())
}

// AddTimeColumn adds a time column or field.
func (s scalarBuilder[B]) AddTimeColumn(name string) B {
	return s.builder.AddColumn(name, Time())
}

// AddTimestampColumn adds a timestamp column or field.
func (s scalarBuilder[B]) AddTimestampColumn(name string) B {
	return s.builder.AddColumn(name, Timestamp())
}

// AddUUIDColumn adds a UUID column or field.
func (s scalarBuilder[B]) AddUUIDColumn(name string) B {
	return s.builder.AddColumn(name, UUID())
}

// AddTimeUUIDColumn adds a TimeUUID column or field.
func (s scalarBuilder[B]) AddTimeUUIDColumn(name string) B {
	return s.builder.AddColumn(name, TimeUUID())
}

// AddBlobColumn adds a blob column or field.
func (s scalarBuilder[B]) AddBlobColumn(name string) B {
	return s.builder.AddColumn(name, Blob())
}

// AddVarintColumn adds a varint column or field.
func (s scalarBuilder[B]) AddVarintColumn(name string) B {
	return s.builder.AddColumn(name, Varint())
}

// AddInetColumn adds an inet column or field.
func (s scalarBuilder[B]) AddInetColumn(name string) B {
	return s.builder.AddColumn(name, Inet())
}

// AddAsciiColumn adds an ascii column or field.
func (s scalarBuilder[B]) AddAsciiColumn(name string) B {
	return s.builder.AddColumn(name, Ascii())
}

// AddDurationColumn adds a duration column or field.
func (s scalarBuilder[B]) AddDurationColumn(name string) B {
	return s.builder.AddColumn(name, Duration())
}

// DefinitionBuilder provides a fluent API for constructing table definitions.
//
// Example using the builder pattern:
//
//	definition := table.NewDefinition().
//		AddColumn("id", table.UUID()).
//		AddTextColumn("title").
//		AddFloatColumn("rating").
//		AddListColumn("genres", table.Text()).
//		AddVectorColumn("embeddings", 1536).
//		SetPartitionBy("id").
//		Build()
type DefinitionBuilder struct {
	columns         Columns
	columnIdx       map[string]int // TODO potentially just use a LinkedMap instead
	partitionBy     []string
	partitionSort   PartitionSort
	partitionSortIx map[string]int
	scalarBuilder[*DefinitionBuilder]
}

func (b *DefinitionBuilder) build() Definition {
	return b.Build()
}

// NewDefinition creates a new DefinitionBuilder for fluent table definition construction.
func NewDefinition() *DefinitionBuilder {
	b := &DefinitionBuilder{
		columnIdx:       make(map[string]int),
		partitionBy:     []string{},
		partitionSortIx: make(map[string]int),
	}
	b.builder = b
	return b
}

// AddColumn adds a column with the specified name and type. If a column with
// the same name already exists, its type is replaced but its position is kept.
func (b *DefinitionBuilder) AddColumn(name string, columnType Column) *DefinitionBuilder {
	if i, ok := b.columnIdx[name]; ok {
		b.columns[i].Column = columnType
		return b
	}
	b.columnIdx[name] = len(b.columns)
	b.columns = append(b.columns, NamedColumn{Name: name, Column: columnType})
	return b
}

// AddVectorColumn adds a vector column with the specified dimension.
func (b *DefinitionBuilder) AddVectorColumn(name string, dimension int) *DefinitionBuilder {
	return b.AddColumn(name, Vector(dimension))
}

// AddVectorColumnWithService adds a vector column with vectorize embedding provider.
func (b *DefinitionBuilder) AddVectorColumnWithService(name string, dimension int, service *VectorService) *DefinitionBuilder {
	return b.AddColumn(name, VectorWithService(dimension, service))
}

// AddSetColumn adds a set column with the specified value type.
func (b *DefinitionBuilder) AddSetColumn(name string, valueType Column) *DefinitionBuilder {
	return b.AddColumn(name, Set(valueType))
}

// AddListColumn adds a list column with the specified value type.
func (b *DefinitionBuilder) AddListColumn(name string, valueType Column) *DefinitionBuilder {
	return b.AddColumn(name, List(valueType))
}

// AddMapColumn adds a map column with the specified key and value types.
func (b *DefinitionBuilder) AddMapColumn(name string, keyType string, valueType Column) *DefinitionBuilder {
	return b.AddColumn(name, Map(keyType, valueType))
}

// AddUDTColumn adds a user-defined type column or field.
func (s *DefinitionBuilder) AddUDTColumn(name string, udtName string) *DefinitionBuilder {
	return s.builder.AddColumn(name, UDT(udtName))
}

// SetPartitionBy sets the partition key columns.
// For a single partition key, pass one column name.
// For a composite partition key, pass multiple column names.
func (b *DefinitionBuilder) SetPartitionBy(columns ...string) *DefinitionBuilder {
	b.partitionBy = columns
	return b
}

// AddPartitionBy appends a column to the partition key.
func (b *DefinitionBuilder) AddPartitionBy(column string) *DefinitionBuilder {
	b.partitionBy = append(b.partitionBy, column)
	return b
}

// AddClusteringColumn adds a clustering column with the specified sort order.
// Use table.SortAscending (1) or table.SortDescending (-1) for the sort order.
// If the column was already added, its sort order is updated in place.
func (b *DefinitionBuilder) AddClusteringColumn(column string, sortOrder int) *DefinitionBuilder {
	if i, ok := b.partitionSortIx[column]; ok {
		b.partitionSort[i].Order = sortOrder
		return b
	}
	b.partitionSortIx[column] = len(b.partitionSort)
	b.partitionSort = append(b.partitionSort, NamedSort{Name: column, Order: sortOrder})
	return b
}

// AddClusteringColumnAsc adds a clustering column with ascending sort order.
func (b *DefinitionBuilder) AddClusteringColumnAsc(column string) *DefinitionBuilder {
	return b.AddClusteringColumn(column, SortAscending)
}

// AddClusteringColumnDesc adds a clustering column with descending sort order.
func (b *DefinitionBuilder) AddClusteringColumnDesc(column string) *DefinitionBuilder {
	return b.AddClusteringColumn(column, SortDescending)
}

// Build constructs the final Definition from the builder.
func (b *DefinitionBuilder) Build() Definition {
	pk := PrimaryKey{
		PartitionBy: b.partitionBy,
	}
	if len(b.partitionSort) > 0 {
		pk.PartitionSort = b.partitionSort
	}

	return Definition{
		Columns:    b.columns,
		PrimaryKey: pk,
	}
}

// UDTDefinitionBuilder provides a fluent API for constructing UDT definitions.
type UDTDefinitionBuilder struct {
	columns   Columns
	columnIdx map[string]int
	scalarBuilder[*UDTDefinitionBuilder]
}

// NewUDTDefinition creates a new UDTDefinitionBuilder for fluent UDT definition construction.
func NewUDTDefinition() *UDTDefinitionBuilder {
	b := &UDTDefinitionBuilder{
		columnIdx: make(map[string]int),
	}
	b.builder = b
	return b
}

// AddColumn adds a field with the specified name and type to the UDT. If a field with
// the same name already exists, its type is replaced but its position is kept.
func (b *UDTDefinitionBuilder) AddColumn(name string, columnType Column) *UDTDefinitionBuilder {
	if i, ok := b.columnIdx[name]; ok {
		b.columns[i].Column = columnType
		return b
	}
	b.columnIdx[name] = len(b.columns)
	b.columns = append(b.columns, NamedColumn{Name: name, Column: columnType})
	return b
}

// Build constructs the final UDTDefinition from the builder.
func (b *UDTDefinitionBuilder) Build() UDTDefinition {
	return UDTDefinition{
		Fields: b.columns,
	}
}
