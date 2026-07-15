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

// Package filter defines filtering options for Astra DB queries.
package filter

import (
	"fmt"

	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

// Filterable is implemented by types that can be used as query filters.
type Filterable interface {
	isFilter()
}

// F represents a map of filters to be applied to an Astra DB query.
// Use this in conjunction with [A] if you want to pass filters as
// they appear in JSON data.
//
// Example:
//
//	filters := filter.F{
//		"$and": filter.A{
//			filter.F{"$or": filter.A{
//				filter.F{"is_checked_out": false},
//				filter.F{"number_of_pages": filter.F{"$lt": 300}},
//			}},
//			filter.F{"$or": filter.A{
//				filter.F{"genres": filter.F{"$in": filter.A{"Fantasy", "Romance"}}},
//				filter.F{"publication_year": filter.F{"$gte": 2002}},
//			}},
//		},
//	}
//
// See [FilterOperator] for available operators.
type F map[string]any

// Satisfy interface to allow F to be used as a filter.
func (F) isFilter() {}

// A represents a slice/array of filters to be applied to an Astra DB query.
// Use this in conjunction with [F] if you want to pass filters as
// they appear in JSON data.
//
// Example:
//
//	filters := filter.F{
//		"$and": filter.A{
//			filter.F{"$or": filter.A{
//				filter.F{"is_checked_out": false},
//				filter.F{"number_of_pages": filter.F{"$lt": 300}},
//			}},
//			filter.F{"$or": filter.A{
//				filter.F{"genres": filter.F{"$in": filter.A{"Fantasy", "Romance"}}},
//				filter.F{"publication_year": filter.F{"$gte": 2002}},
//			}},
//		},
//	}
type A []any

// Operator represents the operation type (Eq, Gt, etc.)
type Operator string

const (
	OpAnd              Operator = "$and"
	OpOr               Operator = "$or"
	OpNot              Operator = "$not"
	OpGreaterThan      Operator = "$gt"
	OpGreaterThanEqual Operator = "$gte"
	OpLessThan         Operator = "$lt"
	OpLessThanEqual    Operator = "$lte"
	OpEqual            Operator = "$eq"
	OpNotEqual         Operator = "$ne"
	OpIn               Operator = "$in"
	OpNotIn            Operator = "$nin"
	OpExists           Operator = "$exists"
	OpAll              Operator = "$all"
	OpKeys             Operator = "$keys"
	OpValues           Operator = "$values"
	OpSize             Operator = "$size"
	OpLexical          Operator = "$lexical"
	OpMatch            Operator = "$match"
)

// Filter represents a filter clause. Compose filters with package-level
// functions like [Eq], [Gt], [And], etc. Example:
//
//	filters := filter.And(
//		filter.Or(
//			filter.Eq("is_checked_out", false),
//			filter.Lt("number_of_pages", 300),
//		),
//		filter.Or(
//			filter.In("genres", "Fantasy", "Romance"),
//			filter.Gte("publication_year", 2002),
//		),
//	)
//
// For collection-specific operators (e.g. [CollFilter.Size], [CollFilter.All],
// [CollFilter.LexicalMatch]), use [Coll] to access them.
// For table-specific operators, use [Table].
type Filter struct {
	// The operator. Such as "$or"
	op Operator
	// The field to perform an operation on. Example: "_id".
	field string
	// The value to filter for based on `op`.
	value any
	// Child filters. Should never be populated if field/value are also populated.
	children any
}

// Satisfy interface to allow Filter to be used as a filter.
func (Filter) isFilter() {}

// CollFilter is a namespace for collection-specific filter operators.
// Obtain one via [Coll]. The returned [Filter] values compose with [And],
// [Or], and [Not] like any other filter.
//
// Example:
//
//	filter.And(
//		filter.Eq("genre", "fantasy"),
//		filter.Coll.Size("tags", 3),
//		filter.Coll.LexicalMatch("dragon"),
//	)
type CollFilter struct{}

// Coll is a [CollFilter] that provides access to collection-specific
// filter operators
var Coll CollFilter

// Exists matches documents that have the specified field, even if the field value is null.
// Collection-only.
func (CollFilter) Exists(key string, val bool) Filter { return fieldOp(OpExists, key, val) }

// Size filters documents where the array field has the given number of elements.
// Collection-only.
func (CollFilter) Size(key string, val int) Filter { return fieldOp(OpSize, key, val) }

// Not negates the given filter.
// All Filterable types ([Filter], [F], [A]) are accepted.
// Collection-only.
func (CollFilter) Not(child Filterable) Filter { return Filter{op: OpNot, children: child} }

// LexicalMatch creates a filter that matches documents against the collection's
// reserved $lexical field. Only available for collections with lexical enabled.
func (CollFilter) LexicalMatch(val string) Filter { return fieldOp(OpMatch, string(OpLexical), val) }

// TableFilter is a namespace for table-specific filter operators.
// Obtain one via [Table]. The returned [Filter] values compose with [And],
// [Or], and [Not] like any other filter.
//
// Example:
//
//	filter.And(
//		filter.Eq("genre", "fantasy"),
//		filter.Table.Keys(filter.In("metadata", "Language", "Edition")),
//		filter.Table.LexicalMatch("dragon"),
//	)
type TableFilter struct{}

// Table is a [TableFilter] that provides access to table-specific
// filter operators.
var Table TableFilter

// Keys works with In, All, and Nin to filter on map columns
// Table-only.
func (TableFilter) Keys(f Filter) Filter { return mapFieldOp(OpKeys, f) }

// Values works with In, All, and Nin to filter on map columns
// Table-only.
func (TableFilter) Values(f Filter) Filter { return mapFieldOp(OpValues, f) }

// LexicalMatch creates a filter that matches documents against a table column
// with a text index associated.
func (TableFilter) LexicalMatch(key string, val string) Filter { return fieldOp(OpMatch, key, val) }

// Construct a field filter operator. Used to reduce boilerplate.
func fieldOp(op Operator, field string, value any) Filter {
	return Filter{op: op, field: field, value: value}
}

// Construct a slice filter operator. Used to reduce boilerplate.
func sliceOp(op Operator, field string, vals []any) Filter {
	return fieldOp(op, field, vals)
}

func mapFieldOp(op Operator, f Filter) Filter {
	if f.op != OpIn && f.op != OpAll && f.op != OpNotIn {
		panic(fmt.Sprintf("filter passed to %s must be In, All, or Nin; got %s", op, f.op))
	}
	return fieldOp(op, f.field, Filter{field: string(f.op), value: f.value}) // cheating by using the op as the field so it marshals properly
}

func (f Filter) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if f.children != nil {
		// We have child commands. Create a map and marshal them like this:
		// "$or": [...]
		filters := make(map[Operator]any)
		filters[f.op] = f.children
		return serdes.SerializeInto(filters, ctx.Target, dst, ctx.Flags)
	}
	if len(f.field) > 0 {
		if len(f.op) == 0 || f.op == OpEqual {
			// We have a default filter which is the same as equals. Marshal it into something like:
			// "_id": 1
			filters := make(map[string]any)
			filters[f.field] = f.value
			return serdes.SerializeInto(filters, ctx.Target, dst, ctx.Flags)
		}
		// We have another op. Marshal it into something like:
		// "number_of_pages": { "$lt": 300 }
		filters := make(map[string]map[Operator]any)
		filters[f.field] = map[Operator]any{f.op: f.value}
		return serdes.SerializeInto(filters, ctx.Target, dst, ctx.Flags)
	}
	return append(dst, "null"...), nil
}

func Eq(key string, val any) Filter  { return fieldOp(OpEqual, key, val) }
func Ne(key string, val any) Filter  { return fieldOp(OpNotEqual, key, val) }
func Lt(key string, val any) Filter  { return fieldOp(OpLessThan, key, val) }
func Lte(key string, val any) Filter { return fieldOp(OpLessThanEqual, key, val) }
func Gt(key string, val any) Filter  { return fieldOp(OpGreaterThan, key, val) }
func Gte(key string, val any) Filter { return fieldOp(OpGreaterThanEqual, key, val) }

func In(key string, vals ...any) Filter  { return sliceOp(OpIn, key, vals) }
func Nin(key string, vals ...any) Filter { return sliceOp(OpNotIn, key, vals) }
func All(key string, vals ...any) Filter { return sliceOp(OpAll, key, vals) }

// And combines the given filters with a logical AND.
// All Filterable types ([Filter], [F], [A]) are accepted.
func And(children ...Filterable) Filter {
	return Filter{op: OpAnd, children: children}
}

// Or combines the given filters with a logical OR.
// All Filterable types ([Filter], [F], [A]) are accepted.
func Or(children ...Filterable) Filter {
	return Filter{op: OpOr, children: children}
}
