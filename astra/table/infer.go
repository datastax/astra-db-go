package table

import (
	"fmt"
	"reflect"
	"sort"
)

type DefinitionLike interface {
	build() Definition
}

type UDTDefinitionLike interface {
	build() Definition
}

// Infer generates a table [Definition] from a Go struct's type information.
//
// Column names are taken from json tags (falling back to field names).
// Primary key configuration and type overrides come from astra tags.
//
// See the astra tag grammar:
//   - astra:"pk" or astra:"pk[N]" — partition key (N = ordinal for composite keys)
//   - astra:"ck", "ck[N], or astra:"ck[N,asc|desc]" — clustering key
//   - astra:"-" — skip field
//   - astra:"type=<T>" — override inferred column type
//   - astra:"dim=<N>" — vector dimension
//
// The <T> in type= supports:
//   - scalars: text, ascii, int, bigint, smallint, tinyint, float, double,
//     decimal, boolean, date, time, timestamp, uuid, timeuuid, blob, varint,
//     inet, duration, vector (vector additionally requires dim=<N>)
//   - containers: set[T], list[T], map[K]V — Type parameters are required, except for type=set, as a handy shortcut
//   - user-defined types: udt[<name>]
//
// Overrides for fields with complex options may be provided within the Infer call itself, e.g. to specify
// vectorize configuration.
//
// Composite example: map[uuid]udt[address]
//
// Example:
//
//	type Book struct {
//	    Title   string              `json:"title"   astra:"pk"`
//	    Authors []string            `json:"author"  astra:"type=set"
//	    Rating  float32             `json:"rating"`
//	    Vector  datatypes.Vectorize `json:"desc_vector"`
//	}
//
//	def, err := table.Infer[Book](
//	  table.VectorWithService("desc_vector", ...)
//	)
//	tbl, err := db.CreateTable(ctx, "books", def)
func Infer[T any](overrides ...DefinitionLike) (Definition, error) {
	t, cols, pks, cks, err := infer[T](overrides, func(def DefinitionLike) Columns { return def.build().Columns })
	if err != nil {
		return Definition{}, fmt.Errorf("table.Infer[%s]: %w", t.Name(), err)
	}

	primaryKey, err := buildPrimaryKey(pks, cks)
	if err != nil {
		return Definition{}, fmt.Errorf("table.Infer[%s]: %w", t.Name(), err)
	}

	return Definition{
		Columns:    cols,
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
//	    City   string `json:"city"` astra:"type=ascii"
//	}
//
//	def, err := table.InferUDT[Address]()
//	err := db.CreateType(ctx, "address", def)
func InferUDT[T any](overrides ...UDTDefinitionLike) (UDTDefinition, error) {
	t, cols, pks, cks, err := infer[T](overrides, func(def UDTDefinitionLike) Columns { return def.build().Columns })
	if err != nil {
		return UDTDefinition{}, fmt.Errorf("table.InferUDT[%s]: %w", t.Name(), err)
	}

	if len(pks) > 0 || len(cks) > 0 {
		return UDTDefinition{}, fmt.Errorf("table.InferUDT[%s]: UDTs cannot have primary or clustering keys", t.Name())
	}

	return UDTDefinition{
		Fields: cols,
	}, nil
}

func infer[T, B any](overrides []B, extractCols func(B) Columns) (reflect.Type, Columns, []tableKeyInfo, []tableKeyInfo, error) {
	t := reflect.TypeFor[T]()

	fields, err := compileFields(t)
	if err != nil {
		return t, nil, nil, nil, err
	}

	cols := make(Columns, 0, fields.Len())
	var pks, cks []tableKeyInfo

	for name, info := range fields.All() {
		col, err := resolveTypeExpr(info.typeExpr, info.modifier)
		if err != nil {
			return t, nil, nil, nil, fmt.Errorf("field %q: %w", name, err)
		}
		cols = append(cols, NamedColumn{Name: name, Column: col})

		if mod, ok := info.modifier.(pkFieldMod); ok {
			pks = append(pks, tableKeyInfo{name: name, ord: mod.ord})
		}

		if mod, ok := info.modifier.(ckFieldMod); ok {
			cks = append(cks, tableKeyInfo{name: name, ord: mod.ord, desc: mod.desc})
		}
	}

	for _, override := range overrides {
		for _, nc := range extractCols(override) {
			found := false
			for i, existing := range cols {
				if existing.Name == nc.Name {
					cols[i].Column = nc.Column
					found = true
					break
				}
			}
			if !found {
				return t, nil, nil, nil, fmt.Errorf("override column %q not found in struct", nc.Name)
			}
		}
	}

	return t, cols, pks, cks, nil
}

type tableKeyInfo struct {
	name string
	ord  int
	desc bool
}

// buildPrimaryKey assembles the PrimaryKey from collected partition and clustering key fields.
func buildPrimaryKey(pks, cks []tableKeyInfo) (PrimaryKey, error) {
	if len(pks) == 0 {
		return PrimaryKey{}, fmt.Errorf("no partition key defined (tag at least one field with astra:\"pk\")")
	}

	if err := sortAndValidateOrdinals("pk", pks); err != nil {
		return PrimaryKey{}, err
	}

	if err := sortAndValidateOrdinals("ck", cks); err != nil {
		return PrimaryKey{}, err
	}

	partitionBy := make([]string, len(pks))
	for i, pk := range pks {
		partitionBy[i] = pk.name
	}

	partitionSort := make(PartitionSort, 0, len(cks))
	for _, ck := range cks {
		order := SortAscending
		if ck.desc {
			order = SortDescending
		}
		partitionSort = append(partitionSort, NamedSort{Name: ck.name, Order: order})
	}

	return PrimaryKey{partitionBy, partitionSort}, nil
}

// validateOrdinals checks that sorted key fields have contiguous 1-based ordinals
// with no duplicates.
func sortAndValidateOrdinals(label string, keys []tableKeyInfo) error {
	if len(keys) == 0 {
		return nil
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].ord < keys[j].ord })

	if keys[0].ord != 0 {
		return fmt.Errorf("%s ordinals must start counting from 0 (got %d for %q)", label, keys[0].ord, keys[0].name)
	}

	if len(keys) > 1 && keys[1].ord == 0 {
		return fmt.Errorf("if multiple %ss are present, explicit ordinals in the form of %s[N] must be used", label, label)
	}

	for i, k := range keys {
		if i > 0 && k.ord == keys[i-1].ord {
			return fmt.Errorf("duplicate %s ordinal %d: %q and %q", label, k.ord, keys[i-1].name, k.name)
		}
		if k.ord != i {
			return fmt.Errorf("%s ordinals must be contiguous starting from 0 (expected %d, got %d for %q)", label, i, k.ord, k.name)
		}
	}

	return nil
}
