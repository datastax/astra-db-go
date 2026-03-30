// Copyright DataStax, Inc.
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

// Package update provides helpers for constructing update documents for collection and table updates.
//
// The update operators supported for collections and tables differ, so be sure to check the docs for
// which operators are supported for each resource type:
//
//   - [Collection Update Operators]
//   - [Table Update Operators]
//
// [Collection Update Operators]: https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operator-collections.html
// [Table Update Operators]: https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operators-tables.html
package update

import "encoding/json"

// CollectionUpdate is implemented by types that can be used as an update document
// for collection operations. It is satisfied by [CollectionUpdater] and [U].
type CollectionUpdate interface {
	json.Marshaler
	isCollectionUpdate()
}

// TableUpdate is implemented by types that can be used as an update document
// for table operations. It is satisfied by [TableUpdater] and [U].
type TableUpdate interface {
	json.Marshaler
	isTableUpdate()
}

// U represents an update document as a map.
//
// Use operator keys directly:
//
//	// update.U{"name": "new"} is equivalent to map[string]any{"name": "new"}.
//	update.U{"$set": update.U{"name": "Bob", "age": 30}},
//
// You can also chain fluent functions:
//
//	// Equivalent to the above example.
//	update.Coll().Set("name", "Bob").Set("age", 30)
type U map[string]any

func (U) isCollectionUpdate() {}
func (U) isTableUpdate()      {}

func (u U) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any(u))
}

// CollectionUpdater accumulates [Collection Update Operators] and serializes directly to JSON.
//
// Construct with Coll():
//
//	update.Coll().Set("number_of_pages", 423).Unset("rating").Inc("copies_sold", 1)
//
// [Collection Update Operators]: https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operator-collections.html
type CollectionUpdater struct {
	ops map[string]map[string]any
}

// dummy implementation to satisfy interface.
func (*CollectionUpdater) isCollectionUpdate() {}

func (u *CollectionUpdater) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.ops)
}

// Coll returns an empty [CollectionUpdater]. Chain methods like [Set], [Unset], etc. to add operators and fields.
//
// Example usage:
//
//	update.Coll().Set("number_of_pages", 423).Unset("rating").Inc("copies_sold", 1)
func Coll() *CollectionUpdater {
	return &CollectionUpdater{ops: make(map[string]map[string]any)}
}

// setField ensures ops is non-nil before setting the given operator and field.
func (u *CollectionUpdater) setField(op, field string, value any) *CollectionUpdater {
	// Ensure ops is non-nil to avoid panic.
	if u.ops == nil {
		u.ops = make(map[string]map[string]any)
	}
	// Op will be something like "$set".
	if _, ok := u.ops[op]; !ok {
		u.ops[op] = make(map[string]any)
	}
	u.ops[op][field] = value
	return u
}

// #region Coll Field Operators

// The $set operator sets the value of the specified field to the specified value.
//
// Example usage:
//
//	update.Coll().Set("number_of_pages", 423).Set("rating", 4.5)
func (u *CollectionUpdater) Set(field string, value any) *CollectionUpdater {
	return u.setField("$set", field, value)
}

// The $setOnInsert operator sets the value of the specified field only if an upsert is performed.
//
// Example usage:
//
//	update.Coll().SetOnInsert("rating", 5.0).SetOnInsert("is_checked_out", false)
func (u *CollectionUpdater) SetOnInsert(field string, value any) *CollectionUpdater {
	return u.setField("$setOnInsert", field, value)
}

// The $unset operator removes the specified field(s).
//
// Example usage:
//
//	update.Coll().Unset("borrower", "due_date")
func (b *CollectionUpdater) Unset(fields ...string) *CollectionUpdater {
	for _, f := range fields {
		b.setField("$unset", f, "")
	}
	return b
}

// The $currentDate operator sets the value of the specified field to the current date and time.
//
// Example usage:
//
//	update.Coll().CurrentDate("due_date")
func (b *CollectionUpdater) CurrentDate(field string) *CollectionUpdater {
	return b.setField("$currentDate", field, true)
}

// The $inc operator increments the value of the specified field by the specified amount.
//
// Example usage:
//
//	update.Coll().Inc("number_of_pages", 25)
func (b *CollectionUpdater) Inc(field string, amount any) *CollectionUpdater {
	return b.setField("$inc", field, amount)
}

// The $min operator updates the specified field only if the specified value is less than the existing field value.
//
// Example usage:
//
//	update.Coll().Min("rating", 3.9)
func (b *CollectionUpdater) Min(field string, value any) *CollectionUpdater {
	return b.setField("$min", field, value)
}

// The $max operator updates the specified field only if the specified value is greater than the existing field value.
//
// Example usage:
//
//	update.Coll().Max("rating", 3.9)
func (b *CollectionUpdater) Max(field string, value any) *CollectionUpdater {
	return b.setField("$max", field, value)
}

// The $mul operator multiplies the value of the specified field.
//
// Example usage:
//
//	update.Coll().Mul("rating", 1.2)
func (b *CollectionUpdater) Mul(field string, value any) *CollectionUpdater {
	return b.setField("$mul", field, value)
}

// The $rename operator renames the specified field.
//
// Example usage:
//
//	update.Coll().Rename("old_field", "new_field").Rename("other_old_field", "other_new_field")
func (u *CollectionUpdater) Rename(field, newName string) *CollectionUpdater {
	return u.setField("$rename", field, newName)
}

// #endregion

// #region Coll Array Operators

// The $addToSet operator adds an item to an array only if the item does not already exist in the array.
//
// Example usage:
//
//	update.Coll().AddToSet("genres", "SciFi")
func (u *CollectionUpdater) AddToSet(field string, value any) *CollectionUpdater {
	return u.setField("$addToSet", field, value)
}

// AddToSetEach combines [AddToSet] and $each to add items to an array if the items do not
// already exist in the array.
//
// Example usage:
//
//	update.Coll().AddToSetEach("genres", "SciFi", "Fantasy")
func (u *CollectionUpdater) AddToSetEach(field string, values ...any) *CollectionUpdater {
	return u.setField("$addToSet", field, map[string]any{"$each": values})
}

// The $pop operator removes the first or last item of the array, depending on the value of the operator.
// Use -1 to remove the first item. Use 1 to remove the last item.
//
// Example usage:
//
//	update.Coll().Pop("genres", -1) // Removes the first item in the genres array.
func (u *CollectionUpdater) Pop(field string, value int) *CollectionUpdater {
	return u.setField("$pop", field, value)
}

// The $push operator appends data to the field value. The specified field must be an array or must not yet exist.
//
// If the specified field does not exist, the field is created, and the value is an array containing the pushed items.
//
// Use [PushEach] to append multiple properties. Use [PushEachPosition] to modify the position of the new items in the array.
//
// Example usage:
//
//	update.Coll().Push("genres", "SciFi")
func (u *CollectionUpdater) Push(field string, value any) *CollectionUpdater {
	return u.setField("$push", field, value)
}

// The $push operator with $each modifier appends multiple values to an array field.
//
// Example usage:
//
//	update.Coll().PushEach("genres", "SciFi", "Fantasy")
func (u *CollectionUpdater) PushEach(field string, values ...any) *CollectionUpdater {
	return u.setField("$push", field, map[string]any{"$each": values})
}

// The $position operator modifies the $push operator to specify the position in the array to add items.
//
//   - Use a positive position value to count from the start of the array. For example, 2 inserts the items after the first two items in the array.
//   - Use a negative position value to count from the end of the array. For example, -2 inserts the items before the last two items in the array.
//   - Use a position value of 0 to insert items at the start of the array
//
// When you use position, the $each operator is required, even if you want to insert a single item at the specified position.
//
// Example usage:
//
//	update.Coll().PushEachPosition("genres", 1, "SciFi", "Fantasy")
func (u *CollectionUpdater) PushEachPosition(field string, position int, values ...any) *CollectionUpdater {
	return u.setField("$push", field, map[string]any{"$each": values, "$position": position})
}

// #endregion

// TableUpdater accumulates [Table Update Operators] and serializes directly to JSON.
//
// Construct with Table():
//
//	update.Table().Set("name", "Bob").Unset("phone")
//
// [Table Update Operators]: https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operators-tables.html
type TableUpdater struct {
	ops map[string]map[string]any
}

func (*TableUpdater) isTableUpdate() {}

func (u *TableUpdater) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.ops)
}

// Table returns an empty TableUpdater.
func Table() *TableUpdater {
	return &TableUpdater{ops: make(map[string]map[string]any)}
}

// setField ensures ops is non-nil before setting the given operator and field.
func (u *TableUpdater) setField(op, field string, value any) *TableUpdater {
	if u.ops == nil {
		u.ops = make(map[string]map[string]any)
	}
	if _, ok := u.ops[op]; !ok {
		u.ops[op] = make(map[string]any)
	}
	u.ops[op][field] = value
	return u
}

// #region Table operators

// The $set operator sets the value of the specified field to the specified value.
// To update a value to a map that includes non-string keys, you must use an array
// of key-value pairs to update the map column. Otherwise, you can use an array of
// key-value pairs or a normal map.
//
// Example setting a non-map column:
//
//	update.Table().Set("name", "Bob").Set("age", 30)
//
// Example setting a map column with string keys (normal map):
//
//	update.Table().Set("metadata", update.U{"language": "English", "edition": "First"})
//
// Example setting a map column with non-string keys (array of key-value pairs):
//
//	update.Table().Set("map_column_int_str", [][]any{{1, "value1"}, {2, "value2}})
func (u *TableUpdater) Set(field string, value any) *TableUpdater {
	return u.setField("$set", field, value)
}

// The $unset operator sets the specified column’s value to null or the equivalent empty form,
// such as [] or {} for map, list, and set types.
//
// Unsetting a column produces a [tombstone]. Excessive tombstones can impact query performance.
//
// Example usage:
//
//	update.Table().Unset("borrower", "due_date")
//
// [tombstone]: https://docs.datastax.com/en/hyper-converged-database/1.2/architecture/database-internals/architecture-tombstones.html
func (u *TableUpdater) Unset(fields ...string) *TableUpdater {
	for _, f := range fields {
		u.setField("$unset", f, "")
	}
	return u
}

// The $push operator appends a single element to a map, list, or set.
// To append multiple items, use [PushEach].
//
// Example usage:
//
//	update.Table().Push("genres", "SciFi")
func (u *TableUpdater) Push(field string, value any) *TableUpdater {
	return u.setField("$push", field, value)
}

// The $push operator with $each modifier appends multiple values to an array field.
//
// Example usage:
//
//	update.Table().PushEach("topics", "robots", "AI")
func (u *TableUpdater) PushEach(field string, values ...any) *TableUpdater {
	return u.setField("$push", field, map[string]any{"$each": values})
}

// The $pullAll operator removes the specified elements from a list or set, or removes
// entries that match the specified keys from a map.
//
// Example usage:
//
//	update.Table().PullAll("genres", "SciFi", "Romance")
func (u *TableUpdater) PullAll(field string, values ...any) *TableUpdater {
	return u.setField("$pullAll", field, values)
}

// #endregion
