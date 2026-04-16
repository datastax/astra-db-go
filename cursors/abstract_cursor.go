package cursors

import (
	"context"
	"iter"
)

type AbstractCursor[TSelf any] interface {
	State() CursorState
	Buffered() int
	Consumed() int
	All() iter.Seq2[int, func(result any)]
	Next(ctx context.Context) bool
	HasNext(ctx context.Context) bool
	Decode(result any) error
	DecodeAll(ctx context.Context, results any) error
	DecodeBuffered(ctx context.Context, results any, max ...int)
	Err() error
	Rewind(ctx context.Context) error
	Close(ctx context.Context) error
	Clone() TSelf
}

func Decode[T any, C AbstractCursor[C]](c C) (T, error) {
	var result T
	err := c.Decode(&result)
	return result, err
}

func DecodeAll[T any, C AbstractCursor[C]](c C) ([]T, error) {
	var result []T
	err := c.DecodeAll(context.Background(), &result)
	return result, err
}
