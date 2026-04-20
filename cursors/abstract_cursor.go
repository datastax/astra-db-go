package cursors

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"sync"

	"github.com/datastax/astra-db-go/internal/guards"
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
	// A cursor which has fetched an item will be in this state, even if no items were consumed.
	CursorStateStarted
	// CursorStateClosed means the cursor has been explicitly closed.
	//
	// A cursor that has been closed, exhausted, or has encountered an error, will be in this state.
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
	// This is the number of items that can be consumed without triggering a fetch of the next page.
	Buffered() int
	Next(ctx context.Context) bool
	Decode(result any) error
	DecodeAll(ctx context.Context, results any) error
	DecodeBuffered(results any, max int) error
	Err() error
	Rewind()
	Close()
}

func Decode[T any](c AbstractCursor) (T, error) {
	var result T
	err := c.Decode(&result)
	return result, err
}

func DecodeAll[T any](ctx context.Context, c AbstractCursor) ([]T, error) {
	var result []T
	err := c.DecodeAll(ctx, &result)
	return result, err
}

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

type abstractCursorSource[Raw any] interface {
	buffer() *[]Raw
	fetchNextPage(ctx context.Context) (bool, error)
	decode(raw Raw, result any) error
	rewind()
	close()
}

var _ AbstractCursor = (*abstractCursorImpl[any])(nil)

type abstractCursorImpl[Raw any] struct {
	acs      abstractCursorSource[Raw]
	mu       sync.RWMutex
	state    CursorState
	nextPage bool
	err      error
}

func newAbstractCursorImpl[Raw any](source abstractCursorSource[Raw]) *abstractCursorImpl[Raw] {
	return &abstractCursorImpl[Raw]{
		acs:      source,
		state:    CursorStateIdle,
		nextPage: true,
	}
}

func (c *abstractCursorImpl[Raw]) State() CursorState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *abstractCursorImpl[Raw]) Buffered() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buffered()
}

func (c *abstractCursorImpl[Raw]) buffered() int {
	return len(*c.acs.buffer())
}

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

func (c *abstractCursorImpl[Raw]) DecodeAll(ctx context.Context, results any) error {
	resultPtr, elemType, err := guards.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.close()

	if c.err != nil {
		return c.err
	}

	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	result := reflect.MakeSlice(elemType, 0, 0)

	for {
		next, err := c.decodeBuffered(elemType, 0)
		if err != nil {
			return err
		}

		result = reflect.AppendSlice(result, next)

		if c.buffered() == 0 && !c.fetchIfEmpty(ctx) {
			break
		}
	}

	if c.err != nil {
		return c.err
	}

	resultPtr.Elem().Set(result)
	return nil
}

func (c *abstractCursorImpl[Raw]) DecodeBuffered(results any, max int) error {
	resultPtr, elemType, err := guards.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := c.decodeBuffered(elemType, max)
	if err != nil {
		return err
	}

	resultPtr.Elem().Set(result)
	return nil
}

func (c *abstractCursorImpl[Raw]) decodeBuffered(elemType reflect.Type, max int) (reflect.Value, error) {
	numToTake := c.buffered()
	if 0 < max && max < numToTake {
		numToTake = max
	}

	bufPtr := c.acs.buffer()
	toTake := (*bufPtr)[:numToTake]

	tempSlice := reflect.MakeSlice(elemType, numToTake, numToTake)

	for i, raw := range toTake {
		targetAddr := tempSlice.Index(i).Addr().Interface()
		if err := c.acs.decode(raw, targetAddr); err != nil {
			return reflect.Value{}, err
		}
	}

	*bufPtr = (*bufPtr)[numToTake:]

	return tempSlice, nil
}

func (c *abstractCursorImpl[Raw]) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func (c *abstractCursorImpl[Raw]) Rewind() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = CursorStateIdle
	c.nextPage = true
	c.err = nil
	c.acs.rewind()
}

func (c *abstractCursorImpl[Raw]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.close()
}

func (c *abstractCursorImpl[Raw]) close() {
	if c.state != CursorStateClosed {
		c.state = CursorStateClosed
		c.nextPage = false
		c.acs.close()
	}
}
