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
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"sync"

	"github.com/datastax/astra-db-go/v2/internal/utils"
)

// ErrCursorClosed is returned when operations are attempted on a closed cursor.
var ErrCursorClosed = errors.New("cursor is closed")

// ErrNoCurrentDocument is returned when Decode is called without a current document.
var ErrNoCurrentDocument = errors.New("no current document available")

// CursorState represents the current state of a cursor.
type CursorState int

const (
	// CursorStateIdle means the cursor has not started iteration.
	//
	// A newly created or rewound cursor will be in this state.
	CursorStateIdle CursorState = iota
	// CursorStateStarted means the cursor is actively iterating.
	//
	// A cursor which has fetched an item will be in this state,
	// even if no items were consumed.
	CursorStateStarted
	// CursorStateClosed means the cursor has been explicitly closed.
	//
	// A cursor that has been closed, exhausted, or has encountered
	// an error, will be in this state.
	CursorStateClosed
)

func (c CursorState) String() string {
	switch c {
	case CursorStateIdle:
		return "idle"
	case CursorStateStarted:
		return "started"
	case CursorStateClosed:
		return "closed"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

// AbstractCursor represents some lazy, abstract iterable over any arbitrary data, which may or may not be paginated.
//
// Example usages:
//
//	if cursor.Next(ctx) {
//	  var item MyType
//	  err := cursor.Decode(&item)
//	}
//
//	items, err := cursors.DecodeAll[MyType](ctx, cursor)
//
// This type is goroutine safe and may be used concurrently across multiple goroutines.
type AbstractCursor interface {
	// State returns the current state of the cursor.
	//
	// See CursorState for more details on the possible states.
	//
	//  cursor := ...
	//  fmt.Println(cursor.State()) // "idle"
	//
	//  cursor.Next(ctx)
	//  fmt.Println(cursor.State()) // "started"
	//
	//  cursor.Close()
	//  fmt.Println(cursor.State()) // "closed"
	State() CursorState
	// Buffered returns the number of items currently buffered in the cursor.
	//
	// This is the number of items that can be consumed without triggering
	// a fetch of the next page.
	Buffered() int
	// Next advances the cursor to the next item, returning true if there are no
	// errors and is another item available.
	//
	// If Next returns false, the loop has either reached the end of the data
	// or an error occurred during pagination; use Err() to distinguish between
	// the two. Subsequent calls will return false unless the cursor is rewound.
	//
	//  for cursor.Next(ctx) {
	//    var item MyType
	//    if err := cursor.Decode(&item); err != nil {
	//      // handle decode error
	//    }
	//  }
	//
	//  if err := cursor.Err(); err != nil {
	//    // handle cursor error
	//  }
	Next(ctx context.Context) bool
	// Decode decodes the current item into result without advancing the cursor.
	// result must be a pointer to the appropriate type.
	//
	// Multiple calls to Decode will return the same item until Next is called.
	// If the buffer is empty or the cursor has not been started, it returns an error.
	// Errors during decoding do not affect the cursor's state or the result pointer.
	//
	//  for cursor.Next(ctx) {
	//    var item MyType
	//    err := cursor.Decode(&item)
	//  }
	//
	// See cursors.Decode for a helper function that creates the result value and decodes into it in one step.
	Decode(result any) error
	// DecodeAll exhausts the cursor, decoding all remaining items into results.
	// results must be a pointer to a slice of the appropriate type.
	//
	// This is a terminal operation; except for parameter validation errors, the cursor
	// is closed upon return. If the cursor is already closed, it returns ErrCursorClosed.
	// The results slice will not be modified in the case of an error.
	//
	//  var items []MyType
	//  err := cursor.DecodeAll(ctx, &items)
	//
	// See cursors.DecodeAll for a helper function that creates the results slice and decodes into it in one step.
	DecodeAll(ctx context.Context, results any) error
	// DecodeBuffered decodes up to max items from the current buffer into results and advances the cursor.
	// The results argument must be a pointer to a slice of the appropriate type.
	//
	// If max is 0, all buffered items are decoded. If the buffer is
	// empty, results is set to an empty slice and no error is returned.
	//
	// DecodeBuffered never fetches the next page from the server. If an error occurs during decoding,
	// the cursor state remains unchanged/unadvanced, and the results slice is not modified.
	//
	//  var items []MyType
	//  err := cursor.DecodeBuffered(&items, 0)
	DecodeBuffered(results any, max int) error
	// Err returns the error, if any, that was encountered during iteration.
	//
	// If Err returns a non-nil error, it implies that the cursor is closed,
	// and no further operations can be performed on it until it is rewound.
	//
	// This method primarily reports errors encountered during pagination (e.g., network errors, server errors).
	// Errors encountered during decoding (e.g., malformed data) are returned directly by the Decode and DecodeAll
	// methods and do not affect the cursor's state.
	Err() error
	// Rewind resets the cursor to its initial state, allowing iteration to start over.
	//
	// All initial options and parameters are preserved, but any buffered data is cleared
	// and the cursor state is reset to idle, and the cursor may be used afresh.
	Rewind()
	// Close explicitly closes the cursor, clearing the buffer and preventing any further operations until it is rewound.
	//
	// The cursor is automatically closed when it is exhausted or encounters an error during pagination,
	// but Close can be used to proactively release resources.
	Close()
}

// Decode is a helper function that decodes the current item from the cursor into a new value of type T.
//
// This is a convenience function that combines the creation of the result value and the call to AbstractCursor.Decode.
//
// The zero value of T is returned in case of a decoding error.
//
// item, err := cursors.Decode[MyType](cursor)
//
//	if err != nil {
//	  // handle error
//	}
func Decode[T any](c AbstractCursor) (T, error) {
	var result T
	err := c.Decode(&result)
	return result, err
}

// DecodeAll is a helper function that exhausts the cursor, decoding all remaining items into a new slice of type T.
//
// This is a convenience function that combines the creation of the results slice and the call to AbstractCursor.DecodeAll.
//
// A nil slice is returned in case of a decoding error.
//
//	items, err := cursors.DecodeAll[MyType](ctx, cursor)
//	if err != nil {
//	  // handle error
//	}
func DecodeAll[T any](ctx context.Context, c AbstractCursor) ([]T, error) {
	var result []T
	err := c.DecodeAll(ctx, &result)
	return result, err
}

// All is a helper function that returns an iterator over all remaining items in the cursor, decoding each item into type T.
//
// This is a convenience function that combines the iteration and decoding steps into a single operation.
//
// The iterator stops when the cursor is exhausted or an error is encountered during pagination or decoding.
// Err() does not need to be called separately when using this function, as any errors will be returned through the iterator.
//
//	for item, err := range cursors.All[MyType](ctx, cursor) {
//	  if err != nil {
//	    // handle error
//	  }
//	  // use item
//	}
func All[T any](ctx context.Context, c AbstractCursor) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for c.Next(ctx) {
			var result T
			if err := c.Decode(&result); err != nil {
				yield(nil, err)
				return
			}

			if !yield(&result, nil) {
				return
			}
		}

		if err := c.Err(); err != nil {
			yield(nil, err)
		}
	}
}

// abstractCursorSource holds the "abstract methods" that the abstractCursorImpl
// relies on to interact with the underlying data source.
type abstractCursorSource[Raw any] interface {
	// buffer returns a pointer to the current buffer slice,
	// which may be modified by the abstractCursorImpl to consume items.
	//
	// buffer MUST NOT be written to if buffer is empty, as it may not
	// be pointing to a valid slice at that point; instead, fetchNextPage
	// should be called to populate the buffer first.
	buffer() *[]Raw
	// fetchNextPage attempts to fetch the next page of results from the underlying data source,
	// returning a boolean indicating whether more pages are available and any error encountered during fetching.
	//
	// fetchNextPage MUST set the buffer itself, as that's not handled by the abstractCursorImpl
	fetchNextPage(ctx context.Context) (bool, error)
	// decode decodes a raw item into the provided result pointer,
	// which is guaranteed to be a pointer to the appropriate type for the decoded item.
	decode(raw Raw, result any) error
	// rewind allows specific cursor implementations to reset any
	// internal state necessary to start iteration over from the beginning.
	rewind()
	// close allows specific cursor implementations to release any
	// internal resources necessary to explicitly close the cursor.
	close()
}

var _ AbstractCursor = (*abstractCursorImpl[any])(nil)

// abstractCursorImpl provides a goroutine-safe implementation of the AbstractCursor interface,
// relying on an abstractCursorSource to interact with the underlying data source.
type abstractCursorImpl[Raw any] struct {
	acs        abstractCursorSource[Raw]
	mu         sync.RWMutex
	state      CursorState
	nextPage   bool
	err        error
	persistErr bool // used for validation errors where the cursor is fully unusable, even if cloned or rewound
}

// newAbstractCursorImpl creates a new abstractCursorImpl with the given source.
func newAbstractCursorImpl[Raw any](source abstractCursorSource[Raw], err error) *abstractCursorImpl[Raw] {
	impl := abstractCursorImpl[Raw]{
		acs:      source,
		state:    CursorStateIdle,
		nextPage: true,
	}

	if err != nil {
		impl.err = err
		impl.persistErr = true
	}

	return &impl
}

// State returns the current state of the cursor with a read lock for concurrency safety.
func (c *abstractCursorImpl[Raw]) State() CursorState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Buffered returns the length of the current buffer with a read lock for concurrency safety.
// It shouldn't be used in internal code that already holds a lock; in those cases, buffered() should be used instead.
func (c *abstractCursorImpl[Raw]) Buffered() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buffered()
}

// buffered is an internal helper that returns the length of the current buffer without acquiring a lock.
func (c *abstractCursorImpl[Raw]) buffered() int {
	return len(*c.acs.buffer())
}

// Next locks and advances the cursor to the next item, fetching the next page if necessary,
// and returns true if there are no errors and another item is available.
func (c *abstractCursorImpl[Raw]) Next(ctx context.Context) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.buffered() > 0 {
		*c.acs.buffer() = (*c.acs.buffer())[1:]
	}

	if c.buffered() == 0 {
		return c.fetchIfEmpty(ctx)
	}

	return true
}

// fetchIfEmpty is an internal helper that attempts to fetch the next page if the buffer is empty
// and there are more pages to fetch, updating the cursor state and error accordingly.
//
// Locking must be provided by the caller; this method does not acquire its own lock.
func (c *abstractCursorImpl[Raw]) fetchIfEmpty(ctx context.Context) bool {
	if c.state == CursorStateClosed {
		return false
	}

	c.state = CursorStateStarted

	for {
		if c.err != nil || (!c.nextPage && c.buffered() == 0) {
			c.close()
			return false
		}
		if c.buffered() > 0 {
			break
		}
		c.nextPage, c.err = c.acs.fetchNextPage(ctx)
	}

	return true
}

// Decode decodes the current item into result without advancing the cursor with a read lock for safety.
func (c *abstractCursorImpl[Raw]) Decode(result any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	if c.buffered() == 0 {
		return ErrNoCurrentDocument
	}

	raw := (*c.acs.buffer())[0]
	if err := c.acs.decode(raw, result); err != nil {
		return err
	}

	return nil
}

// DecodeAll locks and exhausts the cursor, decoding all remaining items into results, and then closes the cursor.
func (c *abstractCursorImpl[Raw]) DecodeAll(ctx context.Context, results any) error {
	resultsPtr, sliceVal, err := utils.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.close()

	defer func() {
		resultsPtr.Elem().Set(sliceVal)
	}()

	if c.err != nil {
		return c.err
	}

	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	for i := 0; ; {
		i, err = c.decodeBuffered(&sliceVal, i, 0)
		if err != nil {
			return err
		}

		if c.buffered() == 0 && !c.fetchIfEmpty(ctx) {
			break
		}
	}

	return c.err
}

// DecodeBuffered locks and decodes up to max items from the current buffer into results and advances the cursor, without fetching the next page.
func (c *abstractCursorImpl[Raw]) DecodeBuffered(results any, max int) error {
	resultsPtr, sliceVal, err := utils.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err = c.decodeBuffered(&sliceVal, 0, max)
	resultsPtr.Elem().Set(sliceVal)
	return err
}

// decodeBuffered is an internal helper that decodes up to max items from the current buffer into a new slice of the given element type and advances the cursor accordingly.
//
// Locking must be provided by the caller; this method does not acquire its own lock.
func (c *abstractCursorImpl[Raw]) decodeBuffered(sliceValue *reflect.Value, start int, max int) (int, error) {
	buffered := c.buffered()
	if 0 < max && max < buffered {
		buffered = max
	}

	bufPtr := c.acs.buffer()
	toTake := (*bufPtr)[:buffered]

	end := start + buffered
	if sliceValue.Cap() < end {
		sliceValue.Grow(end - sliceValue.Len()) // requires Go 1.20+
	}
	sliceValue.SetLen(end)

	for i, raw := range toTake {
		targetAddr := sliceValue.Index(i + start).Addr().Interface() // TODO optimize
		if err := c.acs.decode(raw, targetAddr); err != nil {
			end = start + i
			sliceValue.SetLen(end)
			return end, err
		}
	}

	// We don't set the buffer in case of err
	//
	// This doesn't matter for DecodeAll() since we close on return anyway,
	// but for DecodeBuffered() it lets the caller decode the same items again
	// if they want to handle the error and try again
	*bufPtr = (*bufPtr)[buffered:]
	return end, nil
}

// Err returns the error, if any, that was encountered during iteration with a read lock for concurrency safety.
func (c *abstractCursorImpl[Raw]) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

// Rewind locks resets the cursor to its initial state, allowing iteration to start over.
func (c *abstractCursorImpl[Raw]) Rewind() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = CursorStateIdle
	c.nextPage = true

	if !c.persistErr {
		c.err = nil
	}

	c.acs.rewind()
}

// Close locks and explicitly closes the cursor, clearing the buffer and preventing any further operations until it is rewound.
// It shouldn't be used in internal code that already holds a lock; in those cases, close() should be used instead.
func (c *abstractCursorImpl[Raw]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.close()
}

// close is an internal helper that explicitly closes the cursor, clearing the buffer and preventing any further operations until it is rewound.
//
// Locking must be provided by the caller; this method does not acquire its own lock.
func (c *abstractCursorImpl[Raw]) close() {
	if c.state != CursorStateClosed {
		c.state = CursorStateClosed
		c.nextPage = false
		c.acs.close()
	}
}
