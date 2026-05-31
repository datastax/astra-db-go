package cursors

import (
	"context"
	"encoding/json"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/sort"
)

// FindCursor is a lazy iterable over the results of a find operation on a collection or table.
type FindCursor interface {
	AbstractCursor
	// GetSortVector returns the sort vector used to perform the vector search, if applicable.
	GetSortVector(ctx context.Context) *datatypes.Vector
	// Warnings returns all warnings accumulated during cursor operations.
	Warnings() results.Warnings
}

// FindPage represents a page of results from a find operation, containing the documents/rows,
// pagination state, and optional sort vector for the current page.
type FindPage = findPage[json.RawMessage]

// findCursorImpl provides the base implementation for find-like operations
// that yield json.RawMessage.
type findCursorImpl struct {
	*findLikeCursorImpl[json.RawMessage]
}

func newFindCursorImpl(source findCursorSource[json.RawMessage], fetcher findCursorFetcher, target serdes.Target, initPageState *string, err error) *findCursorImpl {
	impl := &findCursorImpl{}
	impl.findLikeCursorImpl = newFindLikeCursorImpl[json.RawMessage](source, fetcher, target, initPageState, err)
	return impl
}

func (c *findCursorImpl) mapPage(resp *findResponse, targetCtx serdes.TargetDecodeCtx) *findPage[json.RawMessage] {
	return &findPage[json.RawMessage]{
		NextPageState: resp.Data.NextPageState,
		Results:       resp.Data.Documents,
		SortVector:    resp.Data.SortVector,
		targetCtx:     targetCtx,
	}
}

func (c *findCursorImpl) decode(raw json.RawMessage, result any) error {
	return serdes.Deserialize(raw, result, c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target)
}

// findPayload is the payload for the find command.
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
