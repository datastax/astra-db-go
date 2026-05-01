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
	"github.com/datastax/astra-db-go/serdes"
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

// FilterOperator represents the operation type (Eq, Gt, etc.)
type FilterOperator string

const (
	OpAnd              FilterOperator = "$and"
	OpOr               FilterOperator = "$or"
	OpNot              FilterOperator = "$not"
	OpGreaterThan      FilterOperator = "$gt"
	OpGreaterThanEqual FilterOperator = "$gte"
	OpLessThan         FilterOperator = "$lt"
	OpLessThanEqual    FilterOperator = "$lte"
	OpEqual            FilterOperator = "$eq"
	OpNotEqual         FilterOperator = "$ne"
	OpIn               FilterOperator = "$in"
	OpNotIn            FilterOperator = "$nin"
	OpExists           FilterOperator = "$exists"
	OpAll              FilterOperator = "$all"
	OpSize             FilterOperator = "$size"
	OpLexical          FilterOperator = "$lexical"
	OpMatch            FilterOperator = "$match"
)

// Filter represents a collection of filters. Compose filters with
// package-level functions like [Eq], [Gt], [And], etc. Example:
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
type Filter struct {
	// The operator. Such as "$or"
	op FilterOperator
	// The field to perform an operation on. Example: "_id".
	field string
	// The value to filter for based on `op`.
	value any
	// Child filters. Should never be populated if field/value are also populated.
	children []Filter
}

// Satisfy interface to allow Filter to be used as a filter.
func (Filter) isFilter() {}

// Construct a field filter operator. Used to reduce boilerplate.
func fieldOp(op FilterOperator, field string, value any) Filter {
	return Filter{op: op, field: field, value: value}
}

// Construct a slice filter operator. Used to reduce boilerplate.
func sliceOp(op FilterOperator, field string, vals []any) Filter {
	return Filter{op: op, field: field, value: vals}
}

func (f Filter) MarshalAstraRaw(target serdes.Target, dst []byte) ([]byte, error) {
	if len(f.children) > 0 {
		// We have child commands. Create a map and marshal them like this:
		// "$or": [...]
		filters := make(map[FilterOperator][]Filter)
		filters[f.op] = f.children
		return serdes.SerializeInto(filters, target, dst)
	}
	if len(f.field) > 0 {
		if len(f.op) == 0 || f.op == OpEqual {
			// We have a default filter which is the same as equals. Marshal it into something like:
			// "_id": 1
			filters := make(map[string]any)
			filters[f.field] = f.value
			return serdes.SerializeInto(filters, target, dst)
		}
		// We have another op. Marshal it into something like:
		// "number_of_pages": { "$lt": 300 }
		filters := make(map[string]map[FilterOperator]any)
		filters[f.field] = map[FilterOperator]any{f.op: f.value}
		return serdes.SerializeInto(filters, target, dst)
	}
	return nil, nil
}

func (f Filter) MarshalJSON() ([]byte, error) {
	return serdes.Serialize(f, serdes.TargetUnknown)
}

func Eq(key string, val any) Filter  { return fieldOp(OpEqual, key, val) }
func Ne(key string, val any) Filter  { return fieldOp(OpNotEqual, key, val) }
func Lt(key string, val any) Filter  { return fieldOp(OpLessThan, key, val) }
func Lte(key string, val any) Filter { return fieldOp(OpLessThanEqual, key, val) }
func Gt(key string, val any) Filter  { return fieldOp(OpGreaterThan, key, val) }
func Gte(key string, val any) Filter { return fieldOp(OpGreaterThanEqual, key, val) }

func Exists(key string, val bool) Filter { return fieldOp(OpExists, key, val) }
func Size(key string, val int) Filter    { return fieldOp(OpSize, key, val) }

func In(key string, vals ...any) Filter  { return sliceOp(OpIn, key, vals) }
func Nin(key string, vals ...any) Filter { return sliceOp(OpNotIn, key, vals) }
func All(key string, vals ...any) Filter { return sliceOp(OpAll, key, vals) }

func And(children ...Filter) Filter {
	return Filter{
		op:       OpAnd,
		children: children,
	}
}

func Or(children ...Filter) Filter {
	return Filter{
		op:       OpOr,
		children: children,
	}
}

func Not(child Filter) Filter {
	return Filter{op: OpNot, children: []Filter{child}}
}

// LexicalMatch creates a filter that matches documents against the collection's
// reserved $lexical field. Lexicographical matching is only available for
// [collections with lexical enabled].
//
// Example usage:
//
//	filters := filter.LexicalMatch("tree hill")
//
// [collections with lexical enabled]: https://docs.datastax.com/en/astra-db-serverless/api-reference/collection-methods/create-collection.html#example-lexical
func LexicalMatch(val string) Filter {
	return fieldOp(OpMatch, string(OpLexical), val)
}
