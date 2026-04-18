package cursors

import (
	"github.com/datastax/astra-db-go/options"
)

// TableFindCursor is a cursor for iterating over table find results
type TableFindCursor struct {
	findCursorImpl
	filter  any
	options *options.TableFindOptions
}

var _ FindCursor = (*TableFindCursor)(nil)
var _ findCursorSource = (*TableFindCursor)(nil)

// NewTableFindCursor creates a new table find cursor
func NewTableFindCursor(filter any, opts *options.TableFindOptions, fetcher findCursorFetcher) *TableFindCursor {
	cursor := &TableFindCursor{
		filter:  filter,
		options: opts,
	}
	cursor.findCursorImpl = newFindCursorImpl(cursor, fetcher)
	return cursor
}

func (c *TableFindCursor) mkPayload(pageState *string) *findPayload {
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

func (c *TableFindCursor) apiOptions() *options.APIOptions {
	return c.options.APIOptions
}

func (c *TableFindCursor) Clone() *TableFindCursor {
	return NewTableFindCursor(c.filter, c.options, c.fetcher)
}
