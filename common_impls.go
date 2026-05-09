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

package astradb

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/serdes"
)

// region Insertions

type mkInsertOneCmd = func(name string, payload any, opts *options.APIOptions) command

type insertOneOptions struct {
	APIOptions *options.APIOptions
}

// insertOneResponse is the response from insertOne command
type insertOneResponse struct {
	Status struct {
		InsertedIds []json.RawMessage `json:"insertedIds"`
	} `json:"status"`
}

func insertOne(ctx context.Context, record any, mkCmd mkInsertOneCmd, opts *insertOneOptions, target serdes.Target) (*results.InsertOneResult, error) {
	cmd := mkCmd("insertOne", map[string]any{
		"document": record,
	}, opts.APIOptions)

	b, warnings, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp insertOneResponse
	if err := serdes.Deserialize(b, &resp, nil, target); err != nil {
		return nil, err
	}

	if len(resp.Status.InsertedIds) == 0 {
		return nil, errors.New("no inserted ID returned from server")
	}

	return results.NewInsertOneResult(resp.Status.InsertedIds[0], warnings, nil, target), nil
}

type mkInsertManyCmd = func(name string, payload any, opts *options.APIOptions) command

// insertManyOptions are the common options for the collection and table insertMany operations
type insertManyOptions struct {
	Ordered     *bool
	ChunkSize   *int
	Concurrency *int
	APIOptions  *options.APIOptions
}

// insertManyResponse is the response from insertMany command
type insertManyResponse struct {
	Status struct {
		InsertedIds []json.RawMessage `json:"insertedIds"`
	} `json:"status"`
	Errors []DataAPIError `json:"errors,omitempty"`
}

func insertMany(ctx context.Context, records any, mkCmd mkInsertManyCmd, opts *insertManyOptions, target serdes.Target) (*results.InsertManyResult, error) {
	recordsVal := reflect.ValueOf(records)
	if recordsVal.Kind() != reflect.Slice {
		return nil, errors.New("records must be a slice")
	}

	if opts.ChunkSize == nil {
		opts.ChunkSize = ptr.To(50)
	}
	if opts.Concurrency == nil {
		opts.Concurrency = ptr.To(8)
	}
	if opts.Ordered == nil {
		opts.Ordered = ptr.To(false)
	}

	if *opts.Ordered {
		return insertManyOrdered(ctx, recordsVal, mkCmd, opts, target)
	}
	return insertManyUnordered(ctx, recordsVal, mkCmd, opts, target)
}

// insertManyOrdered processes documents sequentially in chunks
func insertManyOrdered(ctx context.Context, records reflect.Value, mkCmd mkInsertManyCmd, opts *insertManyOptions, target serdes.Target) (*results.InsertManyResult, error) {
	totalDocs := records.Len()

	batches := make([]results.InsertManyBatch, 0, (totalDocs+*opts.ChunkSize-1) / *opts.ChunkSize)
	var allWarnings results.Warnings
	var count int

	for i := 0; i < totalDocs; i += *opts.ChunkSize {
		end := i + *opts.ChunkSize
		if end > totalDocs {
			end = totalDocs
		}

		slice := records.Slice(i, end).Interface()

		batch, warnings, apiErrors, err := runInsertMany(ctx, slice, mkCmd, opts)

		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			return nil, err
		}

		batches = append(batches, batch)
		count += len(batch.InsertedIds)

		if len(apiErrors) > 0 {
			return nil, &InsertManyError{} // TODO
		}
	}

	return results.NewInsertManyResult(batches, count, allWarnings, target), nil
}

// insertManyUnordered processes documents concurrently using goroutines
func insertManyUnordered(ctx context.Context, records reflect.Value, mkCmd mkInsertManyCmd, opts *insertManyOptions, target serdes.Target) (*results.InsertManyResult, error) {
	totalDocs := records.Len()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var masterIndex atomic.Int32
	var criticalErr atomic.Pointer[error]

	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	batches := make([]results.InsertManyBatch, 0, (totalDocs+*opts.ChunkSize-1) / *opts.ChunkSize)
	var allApiErrors DataAPIErrors
	var allWarnings results.Warnings
	var count int

	for w := 0; w < *opts.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				claim := int(masterIndex.Add(int32(*opts.ChunkSize)))

				start := claim - *opts.ChunkSize
				end := min(claim, totalDocs)

				if start >= totalDocs {
					return
				}

				slice := records.Slice(start, end).Interface()

				if ctx.Err() != nil {
					return
				}

				batch, warnings, apiErrors, err := runInsertMany(ctx, slice, mkCmd, opts)

				if err != nil {
					if criticalErr.CompareAndSwap(nil, &err) {
						cancel()
					}
					return
				}

				resultsMu.Lock()
				batches = append(batches, batch)
				allApiErrors = append(allApiErrors, apiErrors...)
				allWarnings = append(allWarnings, warnings...)
				count += len(batch.InsertedIds)
				resultsMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if err := criticalErr.Load(); err != nil {
		return nil, *err
	}

	if len(allApiErrors) > 0 {
		return nil, &InsertManyError{} // TODO
	}
	return results.NewInsertManyResult(batches, count, allWarnings, target), nil
}

// runInsertMany executes a single insertMany command for a slice of documents
func runInsertMany(ctx context.Context, records any, mkCmd mkInsertManyCmd, opts *insertManyOptions) (results.InsertManyBatch, results.Warnings, DataAPIErrors, error) {
	cmd := mkCmd("insertMany", map[string]any{
		"documents": records,
		"options": map[string]any{
			"ordered": *opts.Ordered,
		},
	}, opts.APIOptions)

	b, warnings, schema, execErr := cmd.Execute(ctx)

	batch := results.InsertManyBatch{
		InsertedIds: nil,
		Schema:      schema,
	}

	var apiErr *DataAPIError
	if execErr != nil && !errors.As(execErr, &apiErr) {
		return batch, warnings, nil, execErr
	}

	var resp *insertManyResponse
	if unmarshalErr := json.Unmarshal(b, &resp); unmarshalErr != nil {
		return batch, warnings, nil, unmarshalErr
	}

	batch.InsertedIds = resp.Status.InsertedIds
	return batch, warnings, resp.Errors, nil
}

// endregion
