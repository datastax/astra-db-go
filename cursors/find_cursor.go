package cursors

import (
	"context"
	"encoding/json"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/sort"
)

// FindCursor is a lazy iterable over the results of a find operation on a collection or table.
//
// Example usage:
//
//	cursor := collection.Find(ctx, filter, opts)
//
//	for cursor.Next(ctx) {
//	  var doc MyType
//	  err := cursor.Decode(&doc)
//	}
//
// This type is goroutine safe and may be used concurrently across multiple goroutines.
type FindCursor interface {
	AbstractCursor
	// GetSortVector returns the sort vector used to perform the vector search, if applicable.
	//
	// This method will:
	// 1. Return `nil` if `IncludeSortVector` was not set to `true`
	// 2. Return the original vector if sort.Vector was used
	// 3. Return the generated vector if sort.Vectorize was used
	// 4. Return `nil` if vector search was not used
	//
	// If the sort vector is already in memory, it'll return that; otherwise it'll call the server.
	// If an error occurs while fetching a sort vector, cursor.Err() will be set.
	//
	//  vec := cursor.GetSortVector(ctx)
	//  if vec != nil {
	//    // use the sort vector
	//  }
	GetSortVector(ctx context.Context) *datatypes.DataAPIVector
	// Warnings returns all warnings accumulated during cursor operations.
	//
	// Warnings are collected from each page fetch and include any non-fatal issues
	// reported by the server during query execution.
	//
	//  for cursor.Next(ctx) {
	//    // process items
	//  }
	//
	//  if warnings := cursor.Warnings(); len(warnings) > 0 {
	//    // handle warnings
	//  }
	Warnings() results.Warnings
}

// FindPage represents a page of results from a find operation, containing the documents/rows,
// pagination state, and optional sort vector for the current page.
type FindPage struct {
	NextPageState *string                  `json:"nextPageState"`
	Results       []json.RawMessage        `json:"data"`
	SortVector    *datatypes.DataAPIVector `json:"sortVector,omitempty"`
}

// findCursorFetcher is a function type that fetches a page of results from the server,
// returning the raw response bytes, any warnings, and an error if the fetch failed.
type findCursorFetcher = func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, error)

// findCursorSource holds the "abstract methods" that the findCursorImpl
// relies on to interact with the underlying find operation (collection or table).
type findCursorSource interface {
	// mkPayload constructs the request payload for fetching the next page of results,
	// including the filter, sort, projection, and pagination state.
	mkPayload(pageState *string) *findPayload
	// includeSortVector returns whether the sort vector should be included in the response.
	includeSortVector() bool
	// apiOptions returns the API options to use for the find operation.
	apiOptions() *options.APIOptions
}

var _ FindCursor = (*findCursorImpl)(nil)
var _ abstractCursorSource[json.RawMessage] = (*findCursorImpl)(nil)

// findCursorImpl provides the core implementation of FindCursor, handling pagination,
// sort vectors, and warnings for both collection and table find operations.
type findCursorImpl struct {
	*abstractCursorImpl[json.RawMessage]
	fcs         findCursorSource
	currentPage *FindPage
	initialPage *FindPage
	warnings    results.Warnings
	fetcher     findCursorFetcher
}

// newFindCursorImpl creates a new findCursorImpl with the given source, fetcher, and optional initial page state.
func newFindCursorImpl(source findCursorSource, fetcher findCursorFetcher, initPageState *string) *findCursorImpl {
	impl := findCursorImpl{
		fcs:     source,
		fetcher: fetcher,
	}

	impl.abstractCursorImpl = newAbstractCursorImpl[json.RawMessage](&impl)

	if initPageState != nil {
		impl.initialPage = &FindPage{
			NextPageState: initPageState,
		}
		impl.currentPage = impl.initialPage
	}

	return &impl
}

// GetSortVector locks and returns the sort vector from the most recent page fetch, if available.
// If the cursor is idle and sort vectors are enabled, it triggers a fetch to retrieve the first page.
func (c *findCursorImpl) GetSortVector(ctx context.Context) *datatypes.DataAPIVector {
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

// Warnings returns all accumulated warnings with a read lock for concurrency safety.
func (c *findCursorImpl) Warnings() results.Warnings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.warnings
}

// buffer returns a pointer to the current page's result buffer for the abstractCursorImpl.
//
// This implementation returns a pointer to a random slice under the assumption that the buffer WILL NOT
// be written to by abstractCursorImpl directly if the buffer is empty.
func (c *findCursorImpl) buffer() *[]json.RawMessage {
	if c.currentPage == nil {
		return &[]json.RawMessage{}
	}
	return &c.currentPage.Results
}

// findPayload is the payload for the find command, containing the filter, sort, projection, and options.
type findPayload struct {
	Filter     any            `json:"filter,omitempty"`
	Sort       sort.Sortable  `json:"sort,omitempty"`
	Projection map[string]any `json:"projection,omitempty"`
	Options    *findOptions   `json:"options,omitempty"`
}

// findOptions contains pagination and result options for find operations.
type findOptions struct {
	Limit             *int    `json:"limit,omitempty"`
	Skip              *int    `json:"skip,omitempty"`
	IncludeSimilarity *bool   `json:"includeSimilarity,omitempty"`
	IncludeSortVector *bool   `json:"includeSortVector,omitempty"`
	PageState         *string `json:"pageState,omitempty"`
}

// findResponse is the response from the find command, containing the documents/rows and pagination state.
type findResponse struct {
	Data struct {
		Documents     []json.RawMessage        `json:"documents"`
		NextPageState *string                  `json:"nextPageState"`
		SortVector    *datatypes.DataAPIVector `json:"sortVector,omitempty"`
	} `json:"data"`
}

// fetchNextPage fetches the next page of results from the server, updating the current page and warnings.
// Returns true if more pages are available, false otherwise.
func (c *findCursorImpl) fetchNextPage(ctx context.Context) (bool, error) {
	var pageState *string
	if c.currentPage != nil {
		pageState = c.currentPage.NextPageState
	}

	payload := c.fcs.mkPayload(pageState)
	b, warnings, err := c.fetcher(ctx, payload, c.fcs.apiOptions())
	if err != nil {
		return false, err
	}

	c.warnings = append(c.warnings, warnings...)

	var resp findResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		c.currentPage = nil
		return false, err
	}

	c.currentPage = &FindPage{
		NextPageState: resp.Data.NextPageState,
		Results:       resp.Data.Documents,
		SortVector:    resp.Data.SortVector,
	}

	return resp.Data.NextPageState != nil, nil
}

// decode decodes a raw JSON message into the provided result pointer.
func (c *findCursorImpl) decode(raw json.RawMessage, result any) error {
	return json.Unmarshal(raw, result)
}

// rewind clears the current page and any warnings
func (c *findCursorImpl) rewind() {
	c.currentPage = c.initialPage
	c.warnings = nil
}

// close clears the current page
func (c *findCursorImpl) close() {
	c.currentPage = nil
}
