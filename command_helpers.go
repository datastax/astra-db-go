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
)

// #region InsertMany helpers

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
		InsertedIds []any `json:"insertedIds"`
	} `json:"status"`
	Errors []DataAPIError `json:"errors,omitempty"`
}

func insertMany(ctx context.Context, records any, mkCmd mkInsertManyCmd, opts insertManyOptions) (*results.InsertManyResult, error) {
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
		return insertManyOrdered(ctx, recordsVal, mkCmd, &opts)
	}
	return insertManyUnordered(ctx, recordsVal, mkCmd, &opts)
}

// insertManyOrdered processes documents sequentially in chunks
func insertManyOrdered(ctx context.Context, records reflect.Value, mkCmd mkInsertManyCmd, opts *insertManyOptions) (*results.InsertManyResult, error) {
	totalDocs := records.Len()

	allInsertedIds := make([]any, 0, totalDocs)
	var allWarnings results.Warnings

	for i := 0; i < totalDocs; i += *opts.ChunkSize {
		end := i + *opts.ChunkSize
		if end > totalDocs {
			end = totalDocs
		}

		slice := records.Slice(i, end).Interface()

		result, warnings, err := runInsertMany(ctx, slice, mkCmd, opts)
		allWarnings = append(allWarnings, warnings...)

		if err != nil {
			return nil, err
		}

		allInsertedIds = append(allInsertedIds, result.Status.InsertedIds...)

		if result.Errors != nil {
			return nil, &InsertManyError{
				Errors:      result.Errors,
				InsertedIds: allInsertedIds,
			}
		}
	}

	return results.NewInsertManyResult(allInsertedIds, allWarnings), nil
}

// insertManyUnordered processes documents concurrently using goroutines
func insertManyUnordered(ctx context.Context, records reflect.Value, mkCmd mkInsertManyCmd, opts *insertManyOptions) (*results.InsertManyResult, error) {
	totalDocs := records.Len()

	var masterIndex atomic.Int32

	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	insertedIds := make([]any, 0, totalDocs)
	var apiErrors DataAPIErrors
	var allWarnings results.Warnings
	errChan := make(chan error, 1)

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
				res, warn, err := runInsertMany(ctx, slice, mkCmd, opts)

				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}

				resultsMu.Lock()
				insertedIds = append(insertedIds, res.Status.InsertedIds...)
				allWarnings = append(allWarnings, warn...)
				if len(res.Errors) > 0 {
					apiErrors = append(apiErrors, res.Errors...)
				}
				resultsMu.Unlock()
			}
		}()
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	if len(apiErrors) > 0 {
		return nil, &InsertManyError{
			Errors:      apiErrors,
			InsertedIds: insertedIds,
		}
	}
	return results.NewInsertManyResult(insertedIds, allWarnings), nil
}

// runInsertMany executes a single insertMany command for a slice of documents
func runInsertMany(ctx context.Context, records any, mkCmd mkInsertManyCmd, opts *insertManyOptions) (*insertManyResponse, results.Warnings, error) {
	cmd := mkCmd("insertMany", map[string]any{
		"documents": records,
		"options": map[string]any{
			"ordered": *opts.Ordered,
		},
	}, opts.APIOptions)

	b, warnings, execErr := cmd.Execute(ctx)
	if execErr != nil && !errors.Is(execErr, &DataAPIError{}) {
		return nil, warnings, execErr
	}

	var resp *insertManyResponse
	if unmarshalErr := json.Unmarshal(b, &resp); unmarshalErr != nil {
		return nil, warnings, unmarshalErr
	}

	return resp, warnings, nil
}

// #endregion
