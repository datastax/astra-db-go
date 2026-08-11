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
	"encoding/json"

	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
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
var _ findLikeCursorSource[json.RawMessage] = (*TableFindCursor)(nil)

// NewTableFindCursor creates a new table find cursor
//
// This method is not intended to be called directly by users. Instead, use Table.Find() which will create a TableFindCursor for you.
//
// Note that opts must not be nil, or it will panic.
func NewTableFindCursor(filter any, opts *options.TableFindOptions, fetcher findLikeCursorFetcher, err error) *TableFindCursor {
	cursor := &TableFindCursor{
		filter:  filter,
		options: opts,
	}
	cursor.findCursorImpl = newFindCursorImpl(cursor, fetcher, serdes.TargetTable, opts.InitialPageState, err)
	return cursor
}

// mkPayload implements findLikeCursorSource.mkPayload
func (c *TableFindCursor) mkPayload(pageState *string) any {
	return &findPayload{
		Filter:     c.filter,
		Sort:       c.options.Sort,
		Projection: c.options.Projection,
		Options: &findOptions{
			Limit:             ptr.From(c.options.Limit),
			Skip:              c.options.Skip,
			IncludeSimilarity: c.options.IncludeSimilarity,
			IncludeSortVector: c.options.IncludeSortVector,
			PageState:         pageState,
		},
	}
}

// includeSortVector implements findLikeCursorSource.includeSortVector
func (c *TableFindCursor) includeSortVector() bool {
	return ptr.From(c.options.IncludeSortVector)
}

// apiOptions implements findLikeCursorSource.apiOptions
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
