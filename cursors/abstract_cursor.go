package cursors

import (
	"context"
	"iter"
	"reflect"

	"github.com/datastax/astra-db-go/internal/guards"
)

// CursorState represents the current state of a cursor.
type CursorState int

const (
	// CursorStateIdle means the cursor has not started iteration.
	CursorStateIdle CursorState = iota
	// CursorStateActive means the cursor is actively iterating.
	CursorStateActive
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

type abstractCursorInternal[Raw any] interface {
	buffer() *[]Raw
	fetchNextPage(ctx context.Context) error
	decode(raw Raw, result any) error
	rewind()
	close()
}

var _ AbstractCursor = (*abstractCursorImpl)(nil)

type abstractCursorImpl[Raw any] struct {
	impl     abstractCursorInternal[Raw]
	state    CursorState
	nextPage bool
	consumed int
	err      error
}

func (c *abstractCursorImpl[Raw]) State() CursorState {
	return c.state
}

func (c *abstractCursorImpl[Raw]) Buffered() int {
	return len(*c.impl.buffer())
}

func (c *abstractCursorImpl[Raw]) Consumed() int {
	return c.consumed
}

func (c *abstractCursorImpl[Raw]) Next(ctx context.Context) bool {
	//TODO implement me
	panic("implement me")
}

func (c *abstractCursorImpl[Raw]) Decode(result any) error {
	//TODO implement me
	panic("implement me")
}

func (c *abstractCursorImpl[Raw]) DecodeAll(ctx context.Context, results any) error {
	//TODO implement me
	panic("implement me")
}

func (c *abstractCursorImpl[Raw]) DecodeBuffered(results any) error {
	return c.DecodeBufferedN(results, c.Buffered())
}

func (c *abstractCursorImpl[Raw]) DecodeBufferedN(results any, max int) error {
	resultPtr, sliceVal, err := guards.RequireSlicePtr(results)
	if err != nil {
		return err
	}

	bufPtr := c.impl.buffer()
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
		if err := c.impl.decode(raw, targetAddr); err != nil {
			return err
		}
	}

	resultPtr.Elem().Set(sliceVal)
	*bufPtr = (*bufPtr)[numToTake:]
	c.consumed += numToTake

	return nil
}

func (c *abstractCursorImpl[Raw]) Err() error {
	return c.err
}

func (c *abstractCursorImpl[Raw]) Rewind() {
	c.consumed = 0
	c.state = CursorStateIdle
	c.nextPage = true
	c.impl.rewind()
}

func (c *abstractCursorImpl[Raw]) Close() {
	c.state = CursorStateClosed
	c.nextPage = false
	c.impl.close()
}
