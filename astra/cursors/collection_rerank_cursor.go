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
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

// CollectionFindAndRerankCursor is a cursor for iterating over documents returned by a collection findAndRerank operation.
//
// Example usage:
//
//	cursor := collection.FindAndRerank(ctx, filter, opts)
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
type CollectionFindAndRerankCursor struct {
	*findAndRerankCursorImpl
	filter  any
	options *options.CollectionFindAndRerankOptions
}

var _ FindAndRerankCursor = (*CollectionFindAndRerankCursor)(nil)
var _ findLikeCursorSource[rawRerankedResult] = (*CollectionFindAndRerankCursor)(nil)

// NewCollectionFindAndRerankCursor creates a new collection findAndRerank cursor.
//
// This method is not intended to be called directly by users. Instead, use Collection.FindAndRerank()
// which will create a CollectionFindAndRerankCursor for you.
//
// Note that opts must not be nil, or it will panic.
func NewCollectionFindAndRerankCursor(filter any, opts *options.CollectionFindAndRerankOptions, fetcher findLikeCursorFetcher, err error) *CollectionFindAndRerankCursor {
	cursor := &CollectionFindAndRerankCursor{
		filter:  filter,
		options: opts,
	}
	cursor.findAndRerankCursorImpl = newFindAndRerankCursorImpl(cursor, fetcher, serdes.TargetCollection, opts.InitialPageState, err)
	return cursor
}

// mkPayload constructs the request payload for fetching the next page of results.
func (c *CollectionFindAndRerankCursor) mkPayload(pageState *string) any {
	return &findAndRerankPayload{
		Filter:     c.filter,
		Sort:       c.options.Sort,
		Projection: c.options.Projection,
		Options: &findAndRerankOptions{
			Limit:             ptr.From(c.options.Limit),
			HybridLimits:      c.options.HybridLimits,
			IncludeScores:     c.options.IncludeScores,
			IncludeSortVector: c.options.IncludeSortVector,
			RerankOn:          c.options.RerankOn,
			RerankQuery:       c.options.RerankQuery,
			PageState:         pageState,
			Rerank:            c.options.Rerank,
		},
	}
}

// includeSortVector returns whether the sort vector should be included in the response.
func (c *CollectionFindAndRerankCursor) includeSortVector() bool {
	return ptr.From(c.options.IncludeSortVector)
}

// apiOptions returns the API options to use for the operation.
func (c *CollectionFindAndRerankCursor) apiOptions() *options.APIOptions {
	return c.options.APIOptions
}

// Clone creates a new CollectionFindAndRerankCursor with the same configuration.
func (c *CollectionFindAndRerankCursor) Clone() *CollectionFindAndRerankCursor {
	var err error
	if c.persistErr {
		err = c.err
	}
	return NewCollectionFindAndRerankCursor(c.filter, c.options, c.fetcher, err)
}
