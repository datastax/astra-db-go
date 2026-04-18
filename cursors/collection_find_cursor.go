package cursors

import (
	"github.com/datastax/astra-db-go/options"
)

// CollectionFindCursor is a cursor for iterating over collection find results
type CollectionFindCursor struct {
	findCursorImpl
	filter  any
	options *options.CollectionFindOptions
}

var _ FindCursor = (*CollectionFindCursor)(nil)
var _ findCursorSource = (*CollectionFindCursor)(nil)

// NewCollectionFindCursor creates a new collection find cursor
func NewCollectionFindCursor(filter any, opts *options.CollectionFindOptions, fetcher findCursorFetcher) *CollectionFindCursor {
	cursor := &CollectionFindCursor{
		filter:  filter,
		options: opts,
	}
	cursor.findCursorImpl = newFindCursorImpl(cursor, fetcher)
	return cursor
}

func (c *CollectionFindCursor) mkPayload(pageState *string) *findPayload {
	return &findPayload{
		Filter:     c.filter,
		Sort:       c.options.Sort,
		Projection: c.options.Projection,
		Options: &findOptions{
			Limit:             c.options.Limit,
			Skip:              c.options.Skip,
			IncludeSimilarity: c.options.IncludeSimilarity,
			IncludeSortVector: c.options.IncludeSortVector,
			PageState:         pageState,
		},
	}
}

func (c *CollectionFindCursor) apiOptions() *options.APIOptions {
	return c.options.APIOptions
}

func (c *CollectionFindCursor) Clone() *CollectionFindCursor {
	return NewCollectionFindCursor(c.filter, c.options, c.fetcher)
}
