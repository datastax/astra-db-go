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

package results

import (
	"encoding/json"
	"errors"
)

// InsertOneResult represents the result of an insertOne operation.
type InsertOneResult struct {
	insertedId any
	warnings   Warnings
}

// InsertManyResult represents the result of an insertMany operation.
type InsertManyResult struct {
	insertedIds []any
	warnings    Warnings
}

// NewInsertOneResult creates a new InsertOneResult.
func NewInsertOneResult(insertedId any, warnings Warnings) *InsertOneResult {
	return &InsertOneResult{
		insertedId: insertedId,
		warnings:   warnings,
	}
}

// NewInsertManyResult creates a new InsertManyResult.
func NewInsertManyResult(insertedIds []any, warnings Warnings) *InsertManyResult {
	return &InsertManyResult{
		insertedIds: insertedIds,
		warnings:    warnings,
	}
}

// Warnings returns any warnings from the API response.
func (r *InsertOneResult) Warnings() Warnings {
	return r.warnings
}

// Warnings returns any warnings from the API response.
func (r *InsertManyResult) Warnings() Warnings {
	return r.warnings
}

// InsertedCount returns the number of documents successfully inserted.
func (r *InsertManyResult) InsertedCount() int {
	return len(r.insertedIds)
}

// RawID returns the raw inserted ID as any.
func (r *InsertOneResult) RawID() (any, error) {
	if r.insertedId == nil {
		return nil, errors.New("no inserted ID available")
	}
	return r.insertedId, nil
}

// DecodeID unmarshalls the inserted ID into v.
// v should be a pointer to the appropriate ID type (string, int, ObjectId, UUID, etc.)
func (r *InsertOneResult) DecodeID(v any) error {
	rawId, err := r.RawID()
	if err != nil {
		return err
	}

	b, err := json.Marshal(rawId)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// RawIDs returns the raw inserted IDs as a slice of any.
func (r *InsertManyResult) RawIDs() ([]any, error) {
	if r.insertedIds == nil {
		return nil, errors.New("no inserted IDs available")
	}
	return r.insertedIds, nil
}

// DecodeIDs unmarshalls the inserted IDs into v.
// v should be a pointer to a slice of the appropriate ID type.
func (r *InsertManyResult) DecodeIDs(v any) error {
	rawIds, err := r.RawIDs()
	if err != nil {
		return err
	}

	b, err := json.Marshal(rawIds)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
