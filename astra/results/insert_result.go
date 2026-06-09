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

	"github.com/datastax/astra-db-go/v2/astra/internal/utils"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

// InsertOneResult represents the result of an insertOne operation.
type InsertOneResult struct {
	insertedId json.RawMessage
	targetCtx  serdes.TargetDecodeCtx
	warnings   Warnings
	target     serdes.Target
	desFlags   serdes.DesFlags
}

// InsertManyResult represents the result of an insertMany operation.
type InsertManyResult struct {
	batches  []InsertManyBatch
	count    int
	warnings Warnings
	target   serdes.Target
	desFlags serdes.DesFlags
}

type InsertManyBatch struct {
	InsertedIds []json.RawMessage
	TargetCtx   serdes.TargetDecodeCtx
}

// NewInsertOneResult creates a new InsertOneResult.
func NewInsertOneResult(insertedId json.RawMessage, warnings Warnings, targetCtx serdes.TargetDecodeCtx, target serdes.Target, desFlags serdes.DesFlags) *InsertOneResult {
	return &InsertOneResult{insertedId, targetCtx, warnings, target, desFlags}
}

// NewInsertManyResult creates a new InsertManyResult.
func NewInsertManyResult(batches []InsertManyBatch, count int, warnings Warnings, target serdes.Target, desFlags serdes.DesFlags) *InsertManyResult {
	return &InsertManyResult{batches, count, warnings, target, desFlags}
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
	return r.count
}

// RawID returns the raw inserted ID as any.
func (r *InsertOneResult) RawID() (json.RawMessage, error) {
	if r.insertedId == nil {
		return nil, errors.New("no inserted ID available")
	}
	return r.insertedId, nil
}

// DecodeID unmarshalls the inserted ID into v.
// v should be a pointer to the appropriate ID type (string, int, ObjectId, UUID, etc.)
func (r *InsertOneResult) DecodeID(v any) error {
	if r.insertedId == nil {
		return errors.New("no inserted ID available")
	}
	return serdes.Deserialize(r.insertedId, v, r.targetCtx, r.target, r.desFlags)
}

// RawIDs returns the raw inserted IDs as a slice of any.
func (r *InsertManyResult) RawIDs() ([]json.RawMessage, error) {
	ids := make([]json.RawMessage, 0, r.count)
	for _, batch := range r.batches {
		for _, rawId := range batch.InsertedIds {
			ids = append(ids, rawId)
		}
	}
	return ids, nil
}

// DecodeIDs unmarshalls the inserted IDs into v.
// v should be a pointer to a slice of the appropriate ID type.
func (r *InsertManyResult) DecodeIDs(v any) error {
	resultsPtr, sliceVal, err := utils.RequireSlicePtr(v) // TODO optimize
	if err != nil {
		return err
	}

	defer func() {
		resultsPtr.Elem().Set(sliceVal)
	}()

	if sliceVal.Cap() < r.count {
		sliceVal.Grow(r.count - sliceVal.Len())
	}
	sliceVal.SetLen(r.count)

	idx := 0
	for _, batch := range r.batches {
		for _, rawId := range batch.InsertedIds {
			targetAddr := sliceVal.Index(idx).Addr().Interface()
			if err := serdes.Deserialize(rawId, targetAddr, batch.TargetCtx, r.target, r.desFlags); err != nil {
				sliceVal.SetLen(idx)
				return err
			}
			idx++
		}
	}

	return nil
}
