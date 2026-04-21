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
			if !reflect.DeepEqual(cursor.err, tt.wantErr) {
				t.Errorf("err = %v, want %v", cursor.err, tt.wantErr)
			}
		})
	}
}

func TestCursorNext_WhenBufferNonEmpty(t *testing.T) {
	f := func(buf []string, nextPage bool, nextBuf []string) bool {
		if len(buf) == 0 {
			return true
		}

		cursor, source := mkTestAbstractCursor()
		source.Buffer = buf
		cursor.nextPage = nextPage
		cursor.state = CursorStateStarted // state should be started since buffer is non-empty
		source.FetchNextPage = func(ctx context.Context) (bool, error) {
			source.Buffer = nextBuf
			return false, nil
		}

		isNext := cursor.Next(context.Background())

		switch true {
		case len(buf) == 1 && nextPage && len(nextBuf) > 0:
			if !isNext || cursor.State() != CursorStateStarted || !reflect.DeepEqual(source.Buffer, nextBuf) {
				t.Fatalf("isNext = %v, State() = %v, Buffer = %v, want isNext=true, State=Started, Buffer=%v", isNext, cursor.State(), source.Buffer, nextBuf)
			}
		case len(buf) == 1 && (!nextPage || len(nextBuf) == 0):
			if isNext || cursor.State() != CursorStateClosed || !reflect.DeepEqual(source.Buffer, []string{}) {
				t.Fatalf("isNext = %v, State() = %v, Buffer = %v, want isNext=false, State=Closed, Buffer=[]", isNext, cursor.State(), source.Buffer)
			}
		default:
			if !isNext || cursor.State() != CursorStateStarted || !reflect.DeepEqual(source.Buffer, buf[1:]) {
				t.Fatalf("isNext = %v, State() = %v, Buffer = %v, want isNext=true, State=Started, Buffer=%v", isNext, cursor.State(), source.Buffer, buf[1:])
			}
		}

		return true
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
			skipSecondRun: true,
		},
		{
			name:          "InvalidResultsType_NotSlicePointer",
			state:         CursorStateIdle,
			results:       new(string),
			expectResults: new(string),
			expectErr:     fmt.Errorf("expected pointer to slice, got pointer to string"),
			skipSecondRun: true,
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
			name:   "DecodeErrorDuringIteration",
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
			expectResults: &[]string{"decoded_a"},
			expectErr:     fmt.Errorf("decode error on b"),
			expectTrace:   []string{"decode", "decode", "close"},
		},
		{
			name:   "FetchErrorDuringIteration",
			state:  CursorStateIdle,
			buffer: []string{"a", "b"},
			fetchNextPage: func(s *abstractCursorSourceImpl) func(context.Context) (bool, error) {
				return func(ctx context.Context) (bool, error) {
					return false, fmt.Errorf("fetch error")
				}
			},
			results:       &[]string{"before"},
			expectResults: &[]string{"decoded_a", "decoded_b"},
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
			if !reflect.DeepEqual(tt.results, tt.expectResults) {
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
				return // helper function provides its own pointer
			}

			cursor, source := setup()
			results, err := DecodeAll[string](context.Background(), cursor)

			if !reflect.DeepEqual(err, tt.expectErr) {
				t.Errorf("DecodeAll() error = %v, want %v", err, tt.expectErr)
			}

			expectedSlice := *(tt.expectResults.(*[]string))
			if len(results) != len(expectedSlice) || (len(results) > 0 && !reflect.DeepEqual(results, expectedSlice)) {
				t.Errorf("DecodeAll() results = %v, want %v", results, expectedSlice)
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
	f := func(state CursorState, nextPage bool, msg string) bool {
		cursor, source := mkTestAbstractCursor()

		cursor.state = state
		cursor.nextPage = nextPage
		cursor.err = errors.New(msg)

		cursor.Rewind()

		return cursor.State() == CursorStateIdle &&
			cursor.nextPage == true &&
			reflect.DeepEqual(source.Trace, []string{"rewind"}) &&
			cursor.Err() == nil
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorClose(t *testing.T) {
	f := func(state CursorState, nextPage bool, msg string) bool {
		cursor, source := mkTestAbstractCursor()
		err := errors.New(msg)

		cursor.state = state
		cursor.nextPage = nextPage
		cursor.err = err

		cursor.Close()

		return cursor.State() == CursorStateClosed &&
			cursor.nextPage == false &&
			reflect.DeepEqual(source.Trace, []string{"close"}) &&
			errors.Is(cursor.Err(), err)
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCursorLifecycle(t *testing.T) {
	cursor, source := mkTestAbstractCursor()

	testManualStepByStepIteration(t, cursor, source)

	testDecodingRemainderOfCursor(t, cursor)

	testErrorDuringStepByStepIteration(t, cursor, source)

	cursor.Rewind()

	testIteration(t, cursor, source, func(body func(item string, err error)) {
		for cursor.Next(context.Background()) {
			var item string
			body(item, cursor.Decode(&item))
		}
	})

	cursor.Rewind()

	testIteration(t, cursor, source, func(body func(item string, err error)) {
		for cursor.Next(context.Background()) {
			body(Decode[string](cursor))
		}
	})

	cursor.Rewind()

	testIteration(t, cursor, source, func(body func(item string, err error)) {
		for item, err := range All[string](context.Background(), cursor) {
			body(ptr.From(item), err)
		}
	})
}

func testManualStepByStepIteration(t *testing.T, cursor *abstractCursorImpl[string], source *abstractCursorSourceImpl) {
	source.FetchNextPage = func(ctx context.Context) (bool, error) {
		source.Buffer = []string{"a", "b", "c"}
		return true, nil
	}

	source.Decode = func(raw string, result any) error {
		*(result.(*string)) = "decoded_" + raw
		return nil
	}

	if cursor.Buffered() != 0 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}

	if !cursor.Next(context.Background()) {
		t.Fatal("expected Next() to return true")
	}

	if cursor.Buffered() != 3 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}

	var decoded string
	if err := cursor.Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	} else if decoded != "decoded_a" {
		t.Errorf("Decode() result = %v", decoded)
	}

	if cursor.Buffered() != 3 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}

	if !cursor.Next(context.Background()) {
		t.Fatal("expected Next() to return true")
	}

	if cursor.Buffered() != 2 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}

	if err := cursor.Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	} else if decoded != "decoded_b" {
		t.Errorf("Decode() result = %v", decoded)
	}
}

func testDecodingRemainderOfCursor(t *testing.T, cursor *abstractCursorImpl[string]) {
	var buffered []string
	if err := cursor.DecodeBuffered(&buffered, 0); err != nil {
		t.Fatalf("DecodeBuffered() error = %v", err)
	}

	if !reflect.DeepEqual(buffered, []string{"decoded_b", "decoded_c"}) {
		t.Errorf("DecodeBuffered() result = %v", buffered)
	}

	if cursor.Buffered() != 0 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}
}

func testErrorDuringStepByStepIteration(t *testing.T, cursor *abstractCursorImpl[string], source *abstractCursorSourceImpl) {
	source.FetchNextPage = func(ctx context.Context) (bool, error) {
		return false, fmt.Errorf("fetch error")
	}

	if cursor.State() != CursorStateStarted {
		t.Errorf("State() = %v", cursor.State())
	}

	if cursor.Next(context.Background()) {
		t.Fatal("expected Next() to return false on fetch error")
	}

	if !reflect.DeepEqual(cursor.Err(), fmt.Errorf("fetch error")) {
		t.Errorf("Err() = %v", cursor.Err())
	}

	if cursor.State() != CursorStateClosed {
		t.Errorf("State() = %v", cursor.State())
	}

	if cursor.Buffered() != 0 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}
}

func testIteration(t *testing.T, cursor *abstractCursorImpl[string], source *abstractCursorSourceImpl, iter func(body func(item string, err error))) {
	if cursor.State() != CursorStateIdle {
		t.Errorf("State() = %v", cursor.State())
	}

	if cursor.Err() != nil {
		t.Errorf("Err() = %v", cursor.Err())
	}

	docs, i := [6]string{}, 0

	source.FetchNextPage = func(ctx context.Context) (bool, error) {
		source.Buffer = []string{"1", "2"}
		return i < 4, nil
	}

	iter(func(item string, err error) {
		docs[i] = item
		if err != nil {
			t.Fatalf("iteration error = %v", err)
		}
		if cursor.Buffered() != 2-(i%2) {
			t.Errorf("Buffered() = %v, want %v", cursor.Buffered(), 2-(i%2))
		}
		i++
	})

	if i != 6 {
		t.Errorf("iterated %v documents, want 3", i)
	}

	if !reflect.DeepEqual(docs, [6]string{"decoded_1", "decoded_2", "decoded_1", "decoded_2", "decoded_1", "decoded_2"}) {
		t.Errorf("docs = %v", docs)
	}

	if cursor.Err() != nil {
		t.Errorf("Err() = %v", cursor.Err())
	}

	if cursor.State() != CursorStateClosed {
		t.Errorf("State() = %v", cursor.State())
	}

	if cursor.Buffered() != 0 {
		t.Errorf("Buffered() = %v", cursor.Buffered())
	}
}
