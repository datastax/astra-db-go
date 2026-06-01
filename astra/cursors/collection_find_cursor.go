package cursors

import (
	"encoding/json"

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/ptr"
	"github.com/datastax/astra-db-go/astra/serdes"
)

// CollectionFindCursor is a cursor for iterating over documents returned by a collection find operation.
//
// Example usage:
//
//	cursor := collection.Find(ctx, filter, opts)
//	defer cursor.Close()
//
//	for cursor.Next(ctx) {
//	  var doc MyDocument
//	  if err := cursor.Decode(&doc); err != nil {
//	    // handle decode error
//	  }
//	  // use doc
//	}
//
//	if err := cursor.Err(); err != nil {
//	  // handle cursor error
//	}
//
// This type is goroutine safe and may be used concurrently across multiple goroutines.
type CollectionFindCursor struct {
	*findCursorImpl
	filter  any
	options *options.CollectionFindOptions
}

var _ FindCursor = (*CollectionFindCursor)(nil)
var _ findLikeCursorSource[json.RawMessage] = (*CollectionFindCursor)(nil)

// NewCollectionFindCursor creates a new collection find cursor
//
// This method is not intended to be called directly by users. Instead, use Collection.Find() which will create a CollectionFindCursor for you.
//
// Note that opts must not be nil, or it will panic.
func NewCollectionFindCursor(filter any, opts *options.CollectionFindOptions, fetcher findLikeCursorFetcher, err error) *CollectionFindCursor {
	cursor := &CollectionFindCursor{
		filter:  filter,
		options: opts,
	}
	cursor.findCursorImpl = newFindCursorImpl(cursor, fetcher, serdes.TargetCollection, opts.InitialPageState, err)
	return cursor
}

// mkPayload implements findLikeCursorSource.mkPayload
func (c *CollectionFindCursor) mkPayload(pageState *string) any {
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

// includeSortVector implements findLikeCursorSource.includeSortVector
func (c *CollectionFindCursor) includeSortVector() bool {
	return ptr.From(c.options.IncludeSortVector)
}

// apiOptions implements findLikeCursorSource.apiOptions
func (c *CollectionFindCursor) apiOptions() *options.APIOptions {
	return c.options.APIOptions
}

// Clone creates a new CollectionFindCursor with the same filter, options, and fetcher.
//
// This allows you to create multiple independent cursors that iterate over the same query results.
//
//	cursor1 := collection.Find(ctx, filter, opts)
//	cursor2 := cursor1.Clone()
//
//	// cursor1 and cursor2 can be used independently
func (c *CollectionFindCursor) Clone() *CollectionFindCursor {
	var err error
	if c.persistErr {
		err = c.err
	}
	return NewCollectionFindCursor(c.filter, c.options, c.fetcher, err)
}
