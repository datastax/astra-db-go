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
	GetSortVector(ctx context.Context) *datatypes.Vector
	// Warnings returns all warnings accumulated during cursor operations.
	//
	// Warnings are collected from each page fetch and include any non-fatal issues
	// reported by the server during query execution.
	//
	//  for cursor.Next(ctx) {
	//    // process items
	//  }
	//
	//  if warnings := cursor.warnings(); len(warnings) > 0 {
	//    // handle warnings
	//  }
	Warnings() results.Warnings
}

// findCursorImpl provides the base implementation for find-like operations
// that yield json.RawMessage.
type findCursorImpl struct {
	*findLikeCursorImpl[json.RawMessage]
}

func newFindCursorImpl(source findLikeCursorSource[json.RawMessage], fetcher findLikeCursorFetcher, target serdes.Target, initPageState *string, err error) *findCursorImpl {
	impl := &findCursorImpl{}
	impl.findLikeCursorImpl = newFindLikeCursorImpl[json.RawMessage](source, fetcher, target, initPageState, err)
	return impl
}

// mapPage implements findLikeCursorSource.mapPage
func (c *findCursorImpl) mapPage(resp *findResponse, targetCtx serdes.TargetDecodeCtx) *findLikePage[json.RawMessage] {
	return &findLikePage[json.RawMessage]{
		NextPageState: resp.Data.NextPageState,
		Results:       resp.Data.Documents,
		SortVector:    resp.Data.SortVector,
		targetCtx:     targetCtx,
	}
}

// decode implements findLikeCursorSource.decode
func (c *findCursorImpl) decode(raw json.RawMessage, result any) error {
	return serdes.Deserialize(raw, result, c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target, 0)
}

type findPayload struct {
	Filter     any            `json:"filter,omitempty"`
	Sort       sort.Sortable  `json:"sort,omitempty"`
	Projection map[string]any `json:"projection,omitempty"`
	Options    *findOptions   `json:"options,omitempty"`
}

type findOptions struct {
	Limit             *int    `json:"limit,omitempty"`
	Skip              *int    `json:"skip,omitempty"`
	IncludeSimilarity *bool   `json:"includeSimilarity,omitempty"`
	IncludeSortVector *bool   `json:"includeSortVector,omitempty"`
	PageState         *string `json:"pageState,omitempty"`
}
