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

package astra

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/internal/command"
	"github.com/datastax/astra-db-go/v2/astra/internal/timeout"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/internal/utils"
)

type mkCmd = func(name string, payload any, opts ...options.APIOption) command.DataAPI

// region InsertOne

type insertOneOptions struct {
	APIOptions *options.APIOptions
}

type insertOneResponse struct {
	Status struct {
		InsertedIds []json.RawMessage `json:"insertedIds"`
	} `json:"status"`
}

func insertOne(ctx context.Context, record any, mkCmd mkCmd, opts insertOneOptions, target serdes.Target, idMapper results.InternalIDMapper) (*results.InsertOneResult, error) {
	cmd := mkCmd("insertOne", map[string]any{
		"document": record,
	}, opts.APIOptions)

	b, warnings, schema, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	if err != nil {
		return nil, err
	}

	var resp insertOneResponse
	if err := serdes.Deserialize(b, &resp, nil, target, opts.APIOptions.GetDesFlags()); err != nil {
		return nil, err
	}

	if len(resp.Status.InsertedIds) == 0 {
		return nil, errors.New("no inserted ID returned from server")
	}

	return results.NewInsertOneResult(resp.Status.InsertedIds[0], warnings, schema, target, opts.APIOptions.GetDesFlags(), idMapper), nil
}

// endregion

// region InsertMany

type insertManyOptions struct {
	Ordered     *bool
	ChunkSize   *int
	Concurrency *int
	APIOptions  *options.APIOptions
}

type insertManyResponse struct {
	Status struct {
		InsertedIds []json.RawMessage `json:"insertedIds"`
	} `json:"status"`
	Errors []results.DataAPIError `json:"errors,omitempty"`
}

func insertMany(ctx context.Context, records any, baseOpts options.APIOption, mkCmd mkCmd, opts insertManyOptions, target serdes.Target, idMapper results.InternalIDMapper) (*results.InsertManyResult, error) {
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

	tm := timeout.NewMultiCall(baseOpts, opts.APIOptions)

	if *opts.Ordered {
		return insertManyOrdered(ctx, recordsVal, mkCmd, &opts, target, tm, idMapper)
	}
	return insertManyUnordered(ctx, recordsVal, mkCmd, &opts, target, tm, idMapper)
}

func insertManyOrdered(ctx context.Context, records reflect.Value, mkCmd mkCmd, opts *insertManyOptions, target serdes.Target, tm *timeout.Manager, idMapper results.InternalIDMapper) (*results.InsertManyResult, error) {
	totalDocs := records.Len()

	batches := make([]results.InternalInsertManyBatch, 0, (totalDocs+*opts.ChunkSize-1) / *opts.ChunkSize)
	var allWarnings results.Warnings
	var count int

	for i := 0; i < totalDocs; i += *opts.ChunkSize {
		end := i + *opts.ChunkSize
		if end > totalDocs {
			end = totalDocs
		}

		slice := records.Slice(i, end).Interface()

		batch, warnings, apiErrors, err := runInsertMany(ctx, slice, mkCmd, opts, tm)

		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			return nil, err
		}

		batches = append(batches, batch)
		count += len(batch.InsertedIds)

		if len(apiErrors) > 0 {
			return nil, &results.InsertManyError{
				Errors: apiErrors,
				Result: results.NewInsertManyResult(batches, count, allWarnings, target, opts.APIOptions.GetDesFlags(), idMapper),
			}
		}
	}

	return results.NewInsertManyResult(batches, count, allWarnings, target, opts.APIOptions.GetDesFlags(), idMapper), nil
}

func insertManyUnordered(ctx context.Context, records reflect.Value, mkCmd mkCmd, opts *insertManyOptions, target serdes.Target, tm *timeout.Manager, idMapper results.InternalIDMapper) (*results.InsertManyResult, error) {
	totalDocs := records.Len()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var masterIndex atomic.Int32
	var criticalErr atomic.Pointer[error]

	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	batches := make([]results.InternalInsertManyBatch, 0, (totalDocs+*opts.ChunkSize-1) / *opts.ChunkSize)
	var allApiErrors results.DataAPIErrors
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

				batch, warnings, apiErrors, err := runInsertMany(ctx, slice, mkCmd, opts, tm)

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
		return nil, &results.InsertManyError{
			Errors: allApiErrors,
			Result: results.NewInsertManyResult(batches, count, allWarnings, target, opts.APIOptions.GetDesFlags(), idMapper),
		}
	}
	return results.NewInsertManyResult(batches, count, allWarnings, target, opts.APIOptions.GetDesFlags(), idMapper), nil
}

func runInsertMany(ctx context.Context, records any, mkCmd mkCmd, opts *insertManyOptions, tm *timeout.Manager) (results.InternalInsertManyBatch, results.Warnings, results.DataAPIErrors, error) {
	cmd := mkCmd("insertMany", map[string]any{
		"documents": records,
		"options": map[string]any{
			"ordered": *opts.Ordered,
		},
	}, opts.APIOptions)

	b, warnings, schema, execErr := cmd.Execute(ctx, tm)

	batch := results.InternalInsertManyBatch{
		InsertedIds: nil,
		TargetCtx:   schema,
	}

	var apiErr *results.DataAPIError
	if execErr != nil && !errors.As(execErr, &apiErr) {
		return batch, warnings, nil, execErr
	}

	var resp insertManyResponse
	if unmarshalErr := serdes.Deserialize(b, &resp, nil, serdes.TargetNone, opts.APIOptions.GetDesFlags()); unmarshalErr != nil {
		return batch, warnings, nil, unmarshalErr
	}

	batch.InsertedIds = resp.Status.InsertedIds
	return batch, warnings, resp.Errors, nil
}

// endregion

// region FindOne

type findOneOptions struct {
	Sort              sort.Sortable
	Projection        map[string]any
	IncludeSimilarity *bool
	APIOptions        *options.APIOptions
}

func findOne(ctx context.Context, f filter.Filterable, mkCmd mkCmd, opts findOneOptions, target serdes.Target) *results.SingleResult {
	cmd := mkCmd("findOne", map[string]any{
		"filter":     f,
		"sort":       opts.Sort,
		"projection": utils.NonNilMap(opts.Projection),
		"options": map[string]any{
			"includeSimilarity": opts.IncludeSimilarity,
		},
	}, opts.APIOptions)

	b, warnings, schema, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	return results.NewSingleResult(b, warnings, schema, target, err, opts.APIOptions.GetDesFlags())
}

// endregion

// region UpdateOne

type updateOneOptions struct {
	Sort       sort.Sortable       `json:"sort,omitempty"`
	Upsert     *bool               `json:"upsert,omitempty"`
	APIOptions *options.APIOptions `json:"-"`
}

func updateOne(ctx context.Context, f filter.Filterable, u any, mkCmd mkCmd, opts updateOneOptions) ([]byte, error) {
	cmd := mkCmd("updateOne", map[string]any{
		"filter": f,
		"update": u,
		"sort":   opts.Sort,
		"options": map[string]any{
			"upsert": ptr.From(opts.Upsert),
		},
	}, opts.APIOptions)

	b, _, _, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// endregion

// region DeleteOne

type deleteOneOptions struct {
	Sort       sort.Sortable
	APIOptions *options.APIOptions
}

func deleteOne(ctx context.Context, f filter.Filterable, mkCmd mkCmd, opts deleteOneOptions) ([]byte, error) {
	payload := map[string]any{
		"filter": f,
		"sort":   opts.Sort,
	}

	cmd := mkCmd("deleteOne", payload, opts.APIOptions)
	b, _, _, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// endregion
