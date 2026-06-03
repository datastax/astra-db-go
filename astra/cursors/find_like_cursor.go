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

package cursors

import (
	"context"
	"encoding/json"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
)

// findLikePage represents a page of results from a find-like operation.
type findLikePage[Raw any] struct {
	NextPageState *string           `json:"nextPageState"`
	Results       []Raw             `json:"data"`
	SortVector    *datatypes.Vector `json:"sortVector,omitempty"`
	targetCtx     serdes.TargetDecodeCtx
}

// findLikeCursorFetcher is a function type that fetches a page of results from the server.
type findLikeCursorFetcher = func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error)

// findLikeCursorSource holds the "abstract methods" that the findLikeCursorImpl relies on.
type findLikeCursorSource[Raw any] interface {
	// mkPayload constructs the request payload for fetching the next page of results.
	mkPayload(pageState *string) any
	// includeSortVector returns whether the sort vector should be included in the response.
	includeSortVector() bool
	// apiOptions returns the API options to use for the find operation.
	apiOptions() *options.APIOptions
	// mapPage maps the raw findResponse to the generic buffer and pagination state.
	mapPage(resp *findResponse, targetCtx serdes.TargetDecodeCtx) *findLikePage[Raw]
	// decode decodes a raw item into the provided result pointer.
	decode(raw Raw, result any) error
}

// findLikeCursorImpl provides the core implementation for find-like operations.
type findLikeCursorImpl[Raw any] struct {
	*abstractCursorImpl[Raw]
	fcs         findLikeCursorSource[Raw]
	currentPage *findLikePage[Raw]
	initialPage *findLikePage[Raw]
	warnings    results.Warnings
	fetcher     findLikeCursorFetcher
	target      serdes.Target
}

// newFindLikeCursorImpl creates a new findLikeCursorImpl.
func newFindLikeCursorImpl[Raw any](source findLikeCursorSource[Raw], fetcher findLikeCursorFetcher, target serdes.Target, initPageState *string, err error) *findLikeCursorImpl[Raw] {
	impl := findLikeCursorImpl[Raw]{
		fcs:     source,
		fetcher: fetcher,
		target:  target,
	}

	impl.abstractCursorImpl = newAbstractCursorImpl[Raw](&impl, err)

	if initPageState != nil {
		impl.initialPage = &findLikePage[Raw]{
			NextPageState: initPageState,
		}
		impl.currentPage = impl.initialPage
	}

	return &impl
}

func (c *findLikeCursorImpl[Raw]) GetSortVector(ctx context.Context) *datatypes.Vector {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == CursorStateIdle && c.fcs.includeSortVector() {
		c.fetchIfEmpty(ctx)
	}
	if c.currentPage == nil {
		return nil
	}
	return c.currentPage.SortVector
}

func (c *findLikeCursorImpl[Raw]) Warnings() results.Warnings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.warnings
}

func (c *findLikeCursorImpl[Raw]) buffer() *[]Raw {
	if c.currentPage == nil {
		return &[]Raw{}
	}
	return &c.currentPage.Results
}

type findResponse struct {
	Data struct {
		Documents     []json.RawMessage `json:"documents"`
		NextPageState *string           `json:"nextPageState"`
		SortVector    *datatypes.Vector `json:"sortVector,omitempty"`
	} `json:"data"`
	Status *struct {
		DocumentResponses []struct {
			Scores map[string]float32 `json:"scores"`
		} `json:"documentResponses"`
	} `json:"status"`
}

func (c *findLikeCursorImpl[Raw]) fetchNextPage(ctx context.Context) (bool, error) {
	var pageState *string
	if c.currentPage != nil {
		pageState = c.currentPage.NextPageState
	}

	payload := c.fcs.mkPayload(pageState)
	b, warnings, schema, err := c.fetcher(ctx, payload, c.fcs.apiOptions())
	if err != nil {
		return false, err
	}

	c.warnings = append(c.warnings, warnings...)

	var resp findResponse
	if err := serdes.Deserialize(b, &resp, nil, c.target); err != nil {
		c.currentPage = nil
		return false, err
	}

	c.currentPage = c.fcs.mapPage(&resp, schema)

	return c.currentPage.NextPageState != nil, nil
}

func (c *findLikeCursorImpl[Raw]) decode(raw Raw, result any) error {
	return c.fcs.decode(raw, result)
}

func (c *findLikeCursorImpl[Raw]) rewind() {
	c.currentPage = c.initialPage
	c.warnings = nil
}

func (c *findLikeCursorImpl[Raw]) close() {
	c.currentPage = nil
}
