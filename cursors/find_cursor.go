package cursors

import (
	"context"
	"encoding/json"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/sort"
)

type FindCursor interface {
	AbstractCursor
	GetSortVector(ctx context.Context) *datatypes.DataAPIVector
	Warnings() results.Warnings
}

// FindPage represents a page of results from a find operation
type FindPage struct {
	NextPageState *string                  `json:"nextPageState"`
	Result        []json.RawMessage        `json:"data"`
	SortVector    *datatypes.DataAPIVector `json:"sortVector,omitempty"`
}

type findCursorFetcher = func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, error)

type findCursorSource interface {
	mkPayload(pageState *string) *findPayload
	includeSortVector() bool
	apiOptions() *options.APIOptions
}

var _ FindCursor = (*findCursorImpl)(nil)
var _ abstractCursorSource[json.RawMessage] = (*findCursorImpl)(nil)

type findCursorImpl struct {
	*abstractCursorImpl[json.RawMessage]
	fcs         findCursorSource
	currentPage *FindPage
	initialPage *FindPage
	warnings    results.Warnings
	fetcher     findCursorFetcher
}

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

func (c *findCursorImpl) Warnings() results.Warnings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.warnings
}

func (c *findCursorImpl) buffer() *[]json.RawMessage {
	if c.currentPage == nil {
		return &[]json.RawMessage{}
	}
	return &c.currentPage.Result
}

// findPayload is the payload for the find command on collections
type findPayload struct {
	Filter     any             `json:"filter,omitempty"`
	Sort       sort.Sortable   `json:"sort,omitempty"`
	Projection map[string]bool `json:"projection,omitempty"`
	Options    *findOptions    `json:"options,omitempty"`
}

// findOptions contains options for collection find operations
type findOptions struct {
	Limit             *int    `json:"limit,omitempty"`
	Skip              *int    `json:"skip,omitempty"`
	IncludeSimilarity *bool   `json:"includeSimilarity,omitempty"`
	IncludeSortVector *bool   `json:"includeSortVector,omitempty"`
	PageState         *string `json:"pageState,omitempty"`
}

// findResponse is the response from the find command
type findResponse struct {
	Data struct {
		Documents     []json.RawMessage        `json:"documents"`
		NextPageState *string                  `json:"nextPageState"`
		SortVector    *datatypes.DataAPIVector `json:"sortVector,omitempty"`
	} `json:"data"`
}

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
		Result:        resp.Data.Documents,
		SortVector:    resp.Data.SortVector,
	}

	return resp.Data.NextPageState != nil, nil
}

func (c *findCursorImpl) decode(raw json.RawMessage, result any) error {
	return json.Unmarshal(raw, result)
}

func (c *findCursorImpl) rewind() {
	c.currentPage = c.initialPage
	c.warnings = nil
}

func (c *findCursorImpl) close() {
	c.currentPage = nil
}
