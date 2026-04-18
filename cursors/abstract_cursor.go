package cursors

import (
	"context"
	"errors"
	"iter"
	"reflect"

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
	DecodeBuffered(results any) error
	DecodeBufferedN(results any, max int) error
	Err() error
	Rewind()
	Close()
}

func Decode[T any](c AbstractCursor) (T, error) {
	var result T
	err := c.Decode(&result)
	return result, err
}

func DecodeAll[T any](c AbstractCursor) ([]T, error) {
	var result []T
	err := c.DecodeAll(context.Background(), &result)
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
	abstractCursorSource[Raw]
	state    CursorState
	nextPage bool
	consumed int
	err      error
}

func newAbstractCursorImpl[Raw any](source abstractCursorSource[Raw]) abstractCursorImpl[Raw] {
	return abstractCursorImpl[Raw]{
		abstractCursorSource: source,
		state:                CursorStateIdle,
		nextPage:             true,
	}
}

func (c *abstractCursorImpl[Raw]) State() CursorState {
	return c.state
}

func (c *abstractCursorImpl[Raw]) Buffered() int {
	return len(*c.buffer())
}

func (c *abstractCursorImpl[Raw]) Consumed() int {
	return c.consumed
}

func (c *abstractCursorImpl[Raw]) Next(ctx context.Context) bool {
	if c.state == CursorStateClosed {
		return false
	}

	c.state = CursorStateStarted

	if c.Buffered() == 0 {
		for c.Buffered() == 0 {
			if c.err != nil || !c.nextPage {
				c.Close()
				return false
			}
			c.nextPage, c.err = c.fetchNextPage(ctx)
		}
	} else {
		c.consumed++
		*c.buffer() = (*c.buffer())[1:]
	}

	return true
}

func (c *abstractCursorImpl[Raw]) Decode(result any) error {
	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	bufPtr := c.buffer()
	if len(*bufPtr) == 0 {
		return ErrNoCurrentDocument
	}

	raw := (*bufPtr)[0]
	if err := c.decode(raw, result); err != nil {
		return err
	}

	return nil
}

func (c *abstractCursorImpl[Raw]) DecodeAll(ctx context.Context, results any) error {
	resultPtr, sliceVal, err := guards.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	if c.err != nil {
		return c.err
	}

	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	for {
		nextBatch, err := c.decodeBufferedN(sliceVal, 0)
		if err != nil {
			return err
		}

		sliceVal = reflect.AppendSlice(sliceVal, nextBatch)

		if !c.Next(ctx) {
			break
		}
	}

	if c.err != nil {
		return c.err
	}

	resultPtr.Elem().Set(sliceVal)
	return nil
}

func (c *abstractCursorImpl[Raw]) DecodeBuffered(results any) error {
	return c.DecodeBufferedN(results, c.Buffered())
}

func (c *abstractCursorImpl[Raw]) DecodeBufferedN(results any, max int) error {
	resultPtr, sliceVal, err := guards.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	sliceVal, err = c.decodeBufferedN(sliceVal, max)
	if err != nil {
		return err
	}

	resultPtr.Elem().Set(sliceVal)
	return nil
}

func (c *abstractCursorImpl[Raw]) decodeBufferedN(sliceVal reflect.Value, max int) (reflect.Value, error) {
	bufPtr := c.buffer()
	numToTake := len(*bufPtr)
	if max > 0 && max < numToTake {
		numToTake = max
	}

	if sliceVal.Cap() < numToTake {
		sliceVal = reflect.MakeSlice(sliceVal.Type(), numToTake, numToTake)
	} else {
		sliceVal = sliceVal.Slice(0, numToTake)
	}

	toTake := (*bufPtr)[:numToTake]

	for i, raw := range toTake {
		targetAddr := sliceVal.Index(i).Addr().Interface()
		if err := c.decode(raw, targetAddr); err != nil {
			return reflect.Value{}, err
		}
	}

	*bufPtr = (*bufPtr)[numToTake:]
	c.consumed += numToTake

	return sliceVal, nil
}

func (c *abstractCursorImpl[Raw]) Err() error {
	return c.err
}

func (c *abstractCursorImpl[Raw]) Rewind() {
	c.consumed = 0
	c.state = CursorStateIdle
	c.nextPage = true
	c.rewind()
}

func (c *abstractCursorImpl[Raw]) Close() {
	c.state = CursorStateClosed
	c.nextPage = false
	c.close()
}
