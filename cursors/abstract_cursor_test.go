package cursors

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/datastax/astra-db-go/ptr"
)

type abstractCursorSourceImpl struct {
	FetchNextPage func(ctx context.Context) (bool, error)
	Decode        func(raw string, result any) error
	Buffer        []string
	Trace         []string
}

var _ abstractCursorSource[string] = (*abstractCursorSourceImpl)(nil)

func (a *abstractCursorSourceImpl) buffer() *[]string {
	return &a.Buffer
}

func (a *abstractCursorSourceImpl) fetchNextPage(ctx context.Context) (bool, error) {
	a.Trace = append(a.Trace, "fetchNextPage")
	return a.FetchNextPage(ctx)
}

func (a *abstractCursorSourceImpl) decode(raw string, result any) error {
	a.Trace = append(a.Trace, "decode")
	return a.Decode(raw, result)
}

func (a *abstractCursorSourceImpl) rewind() {
	a.Trace = append(a.Trace, "rewind")
}

func (a *abstractCursorSourceImpl) close() {
	a.Trace = append(a.Trace, "close")
}

func mkTestAbstractCursor() (*abstractCursorImpl[string], *abstractCursorSourceImpl) {
	source := &abstractCursorSourceImpl{
		Trace:  []string{},
		Buffer: []string{},
	}
	cursor := newAbstractCursorImpl(source)
	return cursor, source
}

func TestCursorState(t *testing.T) {
	cursor, _ := mkTestAbstractCursor()

	f := func(state CursorState) bool {
		cursor.state = state
		return cursor.State() == state
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorBuffered(t *testing.T) {
	cursor, source := mkTestAbstractCursor()

	f := func(items []string) bool {
		source.Buffer = items
		return cursor.Buffered() == len(items)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorConsumed(t *testing.T) {
	cursor, _ := mkTestAbstractCursor()

	f := func(consumed int) bool {
		cursor.consumed = consumed
		return cursor.Consumed() == consumed
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorNext_WhenBufferEmpty(t *testing.T) {
	tests := []struct {
		name            string
		initialState    CursorState
		initialNextPage bool
		initialErr      error
		fetchNextPage   func(s *abstractCursorSourceImpl) func(context.Context) (bool, error)
		wantNext        bool
		wantState       CursorState
		wantTrace       []string
		wantBuffer      []string
		wantErr         error
		wantConsumed    int
	}{
		{
			name:         "CursorAlreadyClosed",
			initialState: CursorStateClosed,
			wantNext:     false,
			wantState:    CursorStateClosed,
			wantTrace:    []string{},
			wantBuffer:   []string{},
		},
		{
			name:            "IdleButNothingToFetch",
			initialState:    CursorStateStarted,
			initialNextPage: true,
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { return false, nil }
			},
			wantNext:   false,
			wantState:  CursorStateClosed,
			wantTrace:  []string{"fetchNextPage", "close"},
			wantBuffer: []string{},
		},
		{
			name:            "StartedButNothingToFetch",
			initialState:    CursorStateStarted,
			initialNextPage: false,
			wantNext:        false,
			wantState:       CursorStateClosed,
			wantTrace:       []string{"close"},
			wantBuffer:      []string{},
		},
		{
			name:            "IdleAndFetchedSuccess",
			initialState:    CursorStateIdle,
			initialNextPage: true,
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { s.Buffer = append(s.Buffer, "a", "b", "c"); return true, nil }
			},
			wantNext:   true,
			wantState:  CursorStateStarted,
			wantTrace:  []string{"fetchNextPage"},
			wantBuffer: []string{"a", "b", "c"},
		},
		{
			name:            "StartedAndFetchedSuccess",
			initialState:    CursorStateStarted,
			initialNextPage: true,
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { s.Buffer = append(s.Buffer, "d", "e", "f"); return true, nil }
			},
			wantNext:   true,
			wantState:  CursorStateStarted,
			wantTrace:  []string{"fetchNextPage"},
			wantBuffer: []string{"d", "e", "f"},
		},
		{
			name:            "IdleAndFetchedButGotError",
			initialState:    CursorStateIdle,
			initialNextPage: true,
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { return false, fmt.Errorf("fetch failed") }
			},
			wantNext:   false,
			wantState:  CursorStateClosed,
			wantErr:    fmt.Errorf("fetch failed"),
			wantTrace:  []string{"fetchNextPage", "close"},
			wantBuffer: []string{},
		},
		{
			name:            "StartedWithMoreToFetchButHasError",
			initialState:    CursorStateStarted,
			initialNextPage: true,
			initialErr:      fmt.Errorf("fetch failed"),
			wantNext:        false,
			wantState:       CursorStateClosed,
			wantErr:         fmt.Errorf("fetch failed"),
			wantTrace:       []string{"close"},
			wantBuffer:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, source := mkTestAbstractCursor()

			cursor.state = tt.initialState
			cursor.nextPage = tt.initialNextPage
			cursor.err = tt.initialErr

			if tt.fetchNextPage != nil {
				source.FetchNextPage = tt.fetchNextPage(source)
			}

			got := cursor.Next(context.Background())

			if got != tt.wantNext {
				t.Errorf("Next() = %v, want %v", got, tt.wantNext)
			}
			if cursor.State() != tt.wantState {
				t.Errorf("State() = %v, want %v", cursor.State(), tt.wantState)
			}
			if !reflect.DeepEqual(source.Trace, tt.wantTrace) {
				t.Errorf("Trace = %v, want %v", source.Trace, tt.wantTrace)
			}
			if !reflect.DeepEqual(source.Buffer, tt.wantBuffer) {
				t.Errorf("Buffer = %v, want %v", source.Buffer, tt.wantBuffer)
			}
			if cursor.Consumed() != tt.wantConsumed {
				t.Errorf("Consumed() = %v, want %v", cursor.Consumed(), tt.wantConsumed)
			}
			if !reflect.DeepEqual(cursor.err, tt.wantErr) {
				t.Errorf("err = %v, want %v", cursor.err, tt.wantErr)
			}
		})
	}
}

func TestCursorNext_WhenBufferNonEmpty(t *testing.T) {
	f := func(buf []string, consumed int) bool {
		if len(buf) == 0 {
			return true
		}

		cursor, source := mkTestAbstractCursor()
		source.Buffer = buf
		cursor.consumed = consumed

		if !cursor.Next(context.Background()) {
			return false
		}

		return reflect.DeepEqual(source.Buffer, buf[1:]) && cursor.Consumed() == consumed+1
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorDecode(t *testing.T) {
	tests := []struct {
		name         string
		state        CursorState
		buffer       []string
		decode       func(raw string, result any) error
		result       any
		expectResult any
		expectErr    error
	}{
		{
			name:         "CursorClosed",
			state:        CursorStateClosed,
			result:       new(int),
			expectResult: new(int),
			expectErr:    ErrCursorClosed,
		},
		{
			name:         "BufferEmpty",
			state:        CursorStateStarted,
			result:       new(int),
			expectResult: new(int),
			expectErr:    ErrNoCurrentDocument,
		},
		{
			name:   "DecodeSuccess",
			state:  CursorStateStarted,
			buffer: []string{"raw1"},
			decode: func(raw string, result any) error {
				if raw != "raw1" {
					return fmt.Errorf("unexpected raw: %s", raw)
				}
				*(result.(*int)) = 123
				return nil
			},
			result:       ptr.To(56),
			expectResult: ptr.To(123),
		},
		{
			name:   "DecodeErrorDoesntOverwriteResult",
			state:  CursorStateStarted,
			buffer: []string{"raw2"},
			decode: func(raw string, result any) error {
				return fmt.Errorf("decode error")
			},
			result:       ptr.To(23),
			expectResult: ptr.To(23),
			expectErr:    fmt.Errorf("decode error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, source := mkTestAbstractCursor()
			cursor.state = tt.state
			source.Buffer = tt.buffer
			source.Decode = tt.decode

			{
				err := cursor.Decode(tt.result)

				if !reflect.DeepEqual(err, tt.expectErr) {
					t.Errorf("Decode() error = %v, want %v", err, tt.expectErr)
				}
				if !reflect.DeepEqual(tt.result, tt.expectResult) {
					t.Errorf("Decode() result = %v, want %v", tt.result, tt.expectResult)
				}
			}

			{
				result, err := Decode[int](cursor)

				if !reflect.DeepEqual(err, tt.expectErr) {
					t.Errorf("Decode() error = %v, want %v", err, tt.expectErr)
				}
				if err == nil && !reflect.DeepEqual(result, *(tt.expectResult.(*int))) {
					t.Errorf("Decode() result = %v, want %v", result, tt.expectResult)
				}
			}
		})
	}
}

func TestCursorDecodeAll(t *testing.T) {
	tests := []struct {
		name          string
		state         CursorState
		initialErr    error
		buffer        []string
		fetchNextPage func(s *abstractCursorSourceImpl) func(context.Context) (bool, error)
		decode        func(raw string, result any) error
		results       any
		expectResults any
		expectErr     error
		expectTrace   []string
		skipSecondRun bool
	}{
		{
			name:          "CursorClosed",
			state:         CursorStateClosed,
			results:       &[]string{},
			expectResults: &[]string{},
			expectErr:     ErrCursorClosed,
		},
		{
			name:          "CursorHasError",
			state:         CursorStateIdle,
			initialErr:    fmt.Errorf("previous error"),
			results:       &[]string{},
			expectResults: &[]string{},
			expectErr:     fmt.Errorf("previous error"),
		},
		{
			name:          "InvalidResultsType_NotPointer",
			state:         CursorStateIdle,
			results:       []string{},
			expectResults: []string{},
			expectErr:     fmt.Errorf("expected pointer to slice, got slice"),
			skipSecondRun: true, // DecodeAll provides its own correct results argument
		},
		{
			name:          "InvalidResultsType_NotSlicePointer",
			state:         CursorStateIdle,
			results:       new(string),
			expectErr:     fmt.Errorf("expected pointer to slice, got pointer to string"),
			skipSecondRun: true, // DecodeAll provides its own correct results argument
		},
		{
			name:   "EmptyCursor",
			state:  CursorStateIdle,
			buffer: []string{},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { return false, nil }
			},
			results:       &[]string{},
			expectResults: &[]string{},
			expectTrace:   []string{"fetchNextPage", "close"},
		},
		{
			name:   "SingleBatch",
			state:  CursorStateIdle,
			buffer: []string{"a", "b", "c"},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { return false, nil }
			},
			results:       &[]string{},
			expectResults: &[]string{"decoded_a", "decoded_b", "decoded_c"},
			expectTrace:   []string{"decode", "decode", "decode", "fetchNextPage", "close"},
		},
		{
			name:   "MultipleBatches",
			state:  CursorStateStarted,
			buffer: []string{"a", "b"},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				called := false
				return func(ctx context.Context) (bool, error) {
					if !called {
						called = true
						s.Buffer = []string{"c", "d"}
						return true, nil
					}
					return false, nil
				}
			},
			results:       &[]string{},
			expectResults: &[]string{"decoded_a", "decoded_b", "decoded_c", "decoded_d"},
			expectTrace:   []string{"decode", "decode", "fetchNextPage", "decode", "decode", "fetchNextPage", "close"},
		},
		{
			name:   "OverwriteExistingResults",
			state:  CursorStateIdle,
			buffer: []string{"a", "b"},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { return false, nil }
			},
			results:       &[]string{"existing"},
			expectResults: &[]string{"decoded_a", "decoded_b"},
			expectTrace:   []string{"decode", "decode", "fetchNextPage", "close"},
		},
		{
			name:   "DecodeErrorDuringIterationDoesntOverwriteExisting",
			state:  CursorStateIdle,
			buffer: []string{"a", "b", "c"},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) { return false, nil }
			},
			decode: func(raw string, result any) error {
				if raw == "b" {
					return fmt.Errorf("decode error on b")
				}
				*(result.(*string)) = "decoded_" + raw
				return nil
			},
			results:       &[]string{},
			expectResults: &[]string{},
			expectErr:     fmt.Errorf("decode error on b"),
			expectTrace:   []string{"decode", "decode"},
		},
		{
			name:   "FetchErrorDuringIterationDoesntOverwriteExisting",
			state:  CursorStateIdle,
			buffer: []string{"a", "b"},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) {
					return false, fmt.Errorf("fetch error")
				}
			},
			results:       &[]string{"before"},
			expectResults: &[]string{"before"},
			expectErr:     fmt.Errorf("fetch error"),
			expectTrace:   []string{"decode", "decode", "fetchNextPage", "close"},
		},
	}

	for _, tt := range tests {
		setup := func() (*abstractCursorImpl[string], *abstractCursorSourceImpl) {
			cursor, source := mkTestAbstractCursor()
			cursor.state = tt.state
			cursor.err = tt.initialErr
			source.Buffer = tt.buffer

			if tt.fetchNextPage != nil {
				source.FetchNextPage = tt.fetchNextPage(source)
			}

			if tt.decode != nil {
				source.Decode = tt.decode
			} else {
				source.Decode = func(raw string, result any) error {
					*(result.(*string)) = "decoded_" + raw
					return nil
				}
			}

			return cursor, source
		}

		t.Run(tt.name, func(t *testing.T) {
			cursor, source := setup()
			err := cursor.DecodeAll(context.Background(), tt.results)

			if !reflect.DeepEqual(err, tt.expectErr) {
				t.Errorf("DecodeAll() error = %v, want %v", err, tt.expectErr)
			}
			if tt.expectResults != nil && !reflect.DeepEqual(tt.results, tt.expectResults) {
				t.Errorf("DecodeAll() results = %v, want %v", tt.results, tt.expectResults)
			}
			if tt.expectTrace != nil && !reflect.DeepEqual(source.Trace, tt.expectTrace) {
				t.Errorf("DecodeAll() trace = %v, want %v", source.Trace, tt.expectTrace)
			}
			if tt.expectErr == nil && cursor.State() != CursorStateClosed {
				t.Errorf("State() = %v, want %v", cursor.State(), CursorStateClosed)
			}
		})

		t.Run(tt.name+" with DecodeAll helper", func(t *testing.T) {
			if tt.skipSecondRun {
				return //
			}

			cursor, source := setup()
			results, err := DecodeAll[string](context.Background(), cursor)

			if !reflect.DeepEqual(err, tt.expectErr) {
				t.Errorf("DecodeAll() error = %v, want %v", err, tt.expectErr)
			}
			if tt.expectErr == nil && !reflect.DeepEqual(results, *(tt.expectResults.(*[]string))) {
				t.Errorf("DecodeAll() results = %v, want %v", results, tt.expectResults)
			}
			if tt.expectErr != nil && results != nil {
				t.Errorf("DecodeAll() results = %v, want nil on error", results)
			}
			if tt.expectTrace != nil && !reflect.DeepEqual(source.Trace, tt.expectTrace) {
				t.Errorf("DecodeAll() trace = %v, want %v", source.Trace, tt.expectTrace)
			}
			if tt.expectErr == nil && cursor.State() != CursorStateClosed {
				t.Errorf("State() = %v, want %v", cursor.State(), CursorStateClosed)
			}
		})
	}
}

func TestCursorErr(t *testing.T) {
	cursor, _ := mkTestAbstractCursor()

	f := func(msg string) bool {
		err := errors.New(msg)
		cursor.err = err
		return errors.Is(cursor.Err(), err)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorRewind(t *testing.T) {
	f := func(state CursorState, nextPage bool, consumed int, msg string) bool {
		cursor, source := mkTestAbstractCursor()

		cursor.state = state
		cursor.nextPage = nextPage
		cursor.consumed = consumed
		cursor.err = errors.New(msg)

		cursor.Rewind()

		return cursor.State() == CursorStateIdle &&
			cursor.nextPage == true &&
			cursor.Consumed() == 0 &&
			reflect.DeepEqual(source.Trace, []string{"rewind"}) &&
			cursor.Err() == nil
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorClose(t *testing.T) {
	f := func(state CursorState, nextPage bool, consumed int, msg string) bool {
		cursor, source := mkTestAbstractCursor()
		err := errors.New(msg)

		cursor.state = state
		cursor.nextPage = nextPage
		cursor.consumed = consumed
		cursor.err = err

		cursor.Close()

		return cursor.State() == CursorStateClosed &&
			cursor.nextPage == false &&
			cursor.Consumed() == consumed &&
			reflect.DeepEqual(source.Trace, []string{"close"}) &&
			errors.Is(cursor.Err(), err)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
