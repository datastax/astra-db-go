package cursors

import (
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
)

// TableFindCursor is a cursor for iterating over rows returned by a table find operation.
//
// Example usage:
//
//	cursor := table.Find(ctx, filter, opts)
//	defer cursor.Close()
//
//	for cursor.Next(ctx) {
//	  var row MyRow
//	  if err := cursor.Decode(&row); err != nil {
//	    // handle decode error
//	  }
//	  // use row
//	}
//
//	if err := cursor.Err(); err != nil {
//	  // handle cursor error
//	}
//
// This type is goroutine safe and may be used concurrently across multiple goroutines.
type TableFindCursor struct {
	*findCursorImpl
	filter  any
	options *options.TableFindOptions
}

var _ FindCursor = (*TableFindCursor)(nil)
var _ findCursorSource = (*TableFindCursor)(nil)

// NewTableFindCursor creates a new table find cursor
//
// This method is not intended to be called directly by users. Instead, use Table.Find() which will create a TableFindCursor for you.
//
// Note that opts must not be nil, or it will panic.
func NewTableFindCursor(filter any, opts *options.TableFindOptions, fetcher findCursorFetcher, err error) *TableFindCursor {
	cursor := &TableFindCursor{
		filter:  filter,
		options: opts,
	}
	cursor.findCursorImpl = newFindCursorImpl(cursor, fetcher, opts.InitialPageState, err)
	return cursor
}

// mkPayload constructs the request payload for fetching the next page of table results.
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

// includeSortVector returns whether the sort vector should be included in the response.
func (c *TableFindCursor) includeSortVector() bool {
	return ptr.From(c.options.IncludeSortVector)
}

// apiOptions returns the API options to use for the find operation.
func (c *TableFindCursor) apiOptions() *options.APIOptions {
	return c.options.APIOptions
}

// Clone creates a new TableFindCursor with the same filter, options, and fetcher.
//
// This allows you to create multiple independent cursors that iterate over the same query results.
//
//	cursor1 := table.Find(ctx, filter, opts)
//	cursor2 := cursor1.Clone()
//
//	// cursor1 and cursor2 can be used independently
func (c *TableFindCursor) Clone() *TableFindCursor {
	var err error
	if c.persistErr {
		err = c.err
	}
	return NewTableFindCursor(c.filter, c.options, c.fetcher, err)
}
