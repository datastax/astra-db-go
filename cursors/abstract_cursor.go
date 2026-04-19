package cursors

import (
	"context"
	"errors"
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
	CursorStateIdle CursorState = iota
	// CursorStateStarted means the cursor is actively iterating.
	CursorStateStarted
	// CursorStateClosed means the cursor has been explicitly closed.
	CursorStateClosed
)

type AbstractCursor interface {
	State() CursorState
	Buffered() int
	Consumed() int
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
	consumed int
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
	return len(*c.acs.buffer())
}

func (c *abstractCursorImpl[Raw]) Consumed() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consumed
}

func (c *abstractCursorImpl[Raw]) Next(ctx context.Context) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(*c.acs.buffer()) == 0 {
		return c.fetchIfEmpty(ctx)
	}

	*c.acs.buffer() = (*c.acs.buffer())[1:]
	c.consumed++
	return true
}

func (c *abstractCursorImpl[Raw]) fetchIfEmpty(ctx context.Context) bool {
	if c.state == CursorStateClosed {
		return false
	}

	c.state = CursorStateStarted

	for {
		if c.err != nil || (!c.nextPage && len(*c.acs.buffer()) == 0) {
			c.closeLocked()
			return false
		}
		if len(*c.acs.buffer()) > 0 {
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

	bufPtr := c.acs.buffer()
	if len(*bufPtr) == 0 {
		return ErrNoCurrentDocument
	}

	raw := (*bufPtr)[0]
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
	defer c.closeLocked()

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

		if len(*c.acs.buffer()) == 0 {
			if !c.fetchIfEmpty(ctx) {
				break
			}
			continue
		}

		*c.acs.buffer() = (*c.acs.buffer())[1:]
		c.consumed++
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
	bufPtr := c.acs.buffer()
	numToTake := len(*bufPtr)
	if max > 0 && max < numToTake {
		numToTake = max
	}

	tempSlice := reflect.MakeSlice(elemType, numToTake, numToTake)
	toTake := (*bufPtr)[:numToTake]

	for i, raw := range toTake {
		targetAddr := tempSlice.Index(i).Addr().Interface()
		if err := c.acs.decode(raw, targetAddr); err != nil {
			return reflect.Value{}, err
		}
	}

	*bufPtr = (*bufPtr)[numToTake:]
	c.consumed += numToTake

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

	c.consumed = 0
	c.state = CursorStateIdle
	c.nextPage = true
	c.err = nil
	c.acs.rewind()
}

func (c *abstractCursorImpl[Raw]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

func (c *abstractCursorImpl[Raw]) closeLocked() {
	c.state = CursorStateClosed
	c.nextPage = false
	c.acs.close()
}
