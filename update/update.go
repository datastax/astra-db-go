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

//go:generate go run -modfile=../tools/gen-update/go.mod ../tools/gen-update/main.go -pkg .

import "encoding/json"

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
//	update.Set("name", "Bob").Set("age", 30)
type U map[string]any

// Updater accumulates update operators and serializes directly to JSON.
// Construct with New() or any operator helper (Set, Unset, Inc, etc.).
type Updater struct {
	ops map[string]map[string]any
}

func (u *Updater) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.ops)
}

// New returns an empty Updater.
func New() *Updater {
	return &Updater{ops: make(map[string]map[string]any)}
}

// setField ensures ops is non-nil before setting the given operator and field.
func (u *Updater) setField(op, field string, value any) *Updater {
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

// #region Field Operators

// The $set operator sets the value of the specified field to the specified value.
//
// Example usage:
//
//	update.Set("number_of_pages", 423).Set("rating", 4.5)
func (u *Updater) Set(field string, value any) *Updater {
	return u.setField("$set", field, value)
}

// The $setOnInsert operator sets the value of the specified field only if an upsert is performed.
//
// Example usage:
//
//	update.SetOnInsert("rating", 5.0).SetOnInsert("is_checked_out", false)
func (u *Updater) SetOnInsert(field string, value any) *Updater {
	return u.setField("$setOnInsert", field, value)
}

// The $unset operator removes the specified field(s).
//
// Example usage:
//
//	update.Unset("borrower", "due_date")
func (b *Updater) Unset(fields ...string) *Updater {
	for _, f := range fields {
		b.setField("$unset", f, "")
	}
	return b
}

// The $currentDate operator sets the value of the specified field to the current date and time.
//
// Example usage:
//
//	update.CurrentDate("due_date")
func (b *Updater) CurrentDate(field string) *Updater {
	return b.setField("$currentDate", field, true)
}

// The $inc operator increments the value of the specified field by the specified amount.
//
// Example usage:
//
//	update.Inc("number_of_pages", 25)
func (b *Updater) Inc(field string, amount any) *Updater {
	return b.setField("$inc", field, amount)
}

// The $min operator updates the specified field only if the specified value is less than the existing field value.
//
// Example usage:
//
//	update.Min("rating", 3.9)
func (b *Updater) Min(field string, value any) *Updater {
	return b.setField("$min", field, value)
}

// The $max operator updates the specified field only if the specified value is greater than the existing field value.
//
// Example usage:
//
//	update.Max("rating", 3.9)
func (b *Updater) Max(field string, value any) *Updater {
	return b.setField("$max", field, value)
}

// The $mul operator multiplies the value of the specified field.
//
// Example usage:
//
//	update.Mul("rating", 1.2)
func (b *Updater) Mul(field string, value any) *Updater {
	return b.setField("$mul", field, value)
}

// The $rename operator renames the specified field.
//
// Example usage:
//
//	update.Rename("old_field", "new_field").Rename("other_old_field", "other_new_field")
func (u *Updater) Rename(field, newName string) *Updater {
	return u.setField("$rename", field, newName)
}

// #endregion

// #region Array Operators

// The $addToSet operator adds an item to an array only if the item does not already exist in the array.
//
// Example usage:
//
//	update.AddToSet("genres", "SciFi")
func (u *Updater) AddToSet(field string, value any) *Updater {
	return u.setField("$addToSet", field, value)
}

// AddToSetEach combines [AddToSet] and $each to add items to an array if the items do not
// already exist in the array.
//
// Example usage:
//
//	update.AddToSetEach("genres", "SciFi", "Fantasy")
func (u *Updater) AddToSetEach(field string, values ...any) *Updater {
	return u.setField("$addToSet", field, map[string]any{"$each": values})
}

// The $pop operator removes the first or last item of the array, depending on the value of the operator.
// Use -1 to remove the first item. Use 1 to remove the last item.
//
// Example usage:
//
//	update.Pop("genres", -1) // Removes the first item in the genres array.
func (u *Updater) Pop(field string, value int) *Updater {
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
//	update.Push("genres", "SciFi")
func (u *Updater) Push(field string, value any) *Updater {
	return u.setField("$push", field, value)
}

// The $push operator with $each modifier appends multiple values to an array field.
//
// Example usage:
//
//	update.PushEach("genres", "SciFi", "Fantasy")
func (u *Updater) PushEach(field string, values ...any) *Updater {
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
//	update.PushEachPosition("genres", 1, "SciFi", "Fantasy")
func (u *Updater) PushEachPosition(field string, position int, values ...any) *Updater {
	return u.setField("$push", field, map[string]any{"$each": values, "$position": position})
}

// #endregion
