package cursors

import (
	"context"
	"iter"
)

type AbstractCursor interface {
	State() CursorState
	Buffered() int
	Consumed() int
	Iter() iter.Seq2[int, func(result any) error]
	Next(ctx context.Context) bool
	HasNext(ctx context.Context) bool
	Decode(result any) error
	DecodeAll(ctx context.Context, results any) error
	DecodeBuffered(ctx context.Context, results any)
	DecodeBufferedN(ctx context.Context, results any)
	Err() error
	Rewind() error
	Close() error
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
