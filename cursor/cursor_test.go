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

package cursor_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/datastax/astra-db-go/cursor"
	"github.com/datastax/astra-db-go/results"
)

type testDoc struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func makeRawMessages(docs []testDoc) []json.RawMessage {
	result := make([]json.RawMessage, len(docs))
	for i, doc := range docs {
		b, _ := json.Marshal(doc)
		result[i] = b
	}
	return result
}

func TestCursor_SinglePage(t *testing.T) {
	docs := []testDoc{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	fetchCount := 0
	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		fetchCount++
		if pageState != nil {
			return nil, nil, nil, nil // No more pages
		}
		return makeRawMessages(docs), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	var results []testDoc
	for c.Next(context.Background()) {
		var doc testDoc
		if err := c.Decode(&doc); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		results = append(results, doc)
	}

	if err := c.Err(); err != nil {
		t.Fatalf("Cursor error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if results[0].Name != "Alice" {
		t.Errorf("expected first doc to be Alice, got %s", results[0].Name)
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}
}

func TestCursor_MultiplePages(t *testing.T) {
	page1 := []testDoc{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	page2 := []testDoc{{ID: 3, Name: "Charlie"}, {ID: 4, Name: "Diana"}}
	page3 := []testDoc{{ID: 5, Name: "Eve"}}

	pageState1 := "page2"
	pageState2 := "page3"

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		if pageState == nil {
			return makeRawMessages(page1), &pageState1, nil, nil
		}
		if *pageState == "page2" {
			return makeRawMessages(page2), &pageState2, nil, nil
		}
		if *pageState == "page3" {
			return makeRawMessages(page3), nil, nil, nil
		}
		return nil, nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	var testResults []testDoc
	for c.Next(context.Background()) {
		var doc testDoc
		if err := c.Decode(&doc); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		testResults = append(testResults, doc)
	}

	if err := c.Err(); err != nil {
		t.Fatalf("Cursor error: %v", err)
	}

	if len(testResults) != 5 {
		t.Errorf("expected 5 results, got %d", len(testResults))
	}

	expectedNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve"}
	for i, expected := range expectedNames {
		if testResults[i].Name != expected {
			t.Errorf("result[%d]: expected %s, got %s", i, expected, testResults[i].Name)
		}
	}
}

func TestCursor_All(t *testing.T) {
	page1 := []testDoc{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	page2 := []testDoc{{ID: 3, Name: "Charlie"}}

	pageState1 := "page2"

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		if pageState == nil {
			return makeRawMessages(page1), &pageState1, nil, nil
		}
		return makeRawMessages(page2), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	var testResults []testDoc
	if err := c.All(context.Background(), &testResults); err != nil {
		t.Fatalf("All failed: %v", err)
	}

	if len(testResults) != 3 {
		t.Errorf("expected 3 results, got %d", len(testResults))
	}

	if c.State() != cursor.CursorStateExhausted {
		t.Errorf("expected cursor to be exhausted after All()")
	}
}

func TestCursor_EmptyResults(t *testing.T) {
	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return []json.RawMessage{}, nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	if c.Next(context.Background()) {
		t.Error("expected Next() to return false for empty results")
	}

	if err := c.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var testResults []testDoc
	if err := c.All(context.Background(), &testResults); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(testResults) != 0 {
		t.Errorf("expected empty results, got %d", len(testResults))
	}
}

func TestCursor_FetchError(t *testing.T) {
	expectedErr := errors.New("network error")

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return nil, nil, nil, expectedErr
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	if c.Next(context.Background()) {
		t.Error("expected Next() to return false on error")
	}

	if c.Err() != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, c.Err())
	}
}

func TestCursor_Close(t *testing.T) {
	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages([]testDoc{{ID: 1, Name: "Alice"}}), nil, nil, nil
	}

	c := cursor.New(fetcher)

	// Close the cursor
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations should fail after close
	if c.Next(context.Background()) {
		t.Error("expected Next() to return false after Close()")
	}

	if c.Err() != cursor.ErrCursorClosed {
		t.Errorf("expected ErrCursorClosed, got %v", c.Err())
	}

	if err := c.Decode(&testDoc{}); err != cursor.ErrCursorClosed {
		t.Errorf("expected ErrCursorClosed from Decode, got %v", err)
	}

	// Double close should be safe
	if err := c.Close(context.Background()); err != nil {
		t.Errorf("double Close failed: %v", err)
	}
}

func TestCursor_DecodeWithoutNext(t *testing.T) {
	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages([]testDoc{{ID: 1, Name: "Alice"}}), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	// Decode without calling Next should fail
	var doc testDoc
	if err := c.Decode(&doc); err != cursor.ErrNoCurrentDocument {
		t.Errorf("expected ErrNoCurrentDocument, got %v", err)
	}
}

func TestCursor_WithInitialData(t *testing.T) {
	initialDocs := []testDoc{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	page2 := []testDoc{{ID: 3, Name: "Charlie"}}

	pageState := "page2"
	fetchCalled := false

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		fetchCalled = true
		return makeRawMessages(page2), nil, nil, nil
	}

	c := cursor.NewWithInitialData(makeRawMessages(initialDocs), &pageState, nil, fetcher)
	defer c.Close(context.Background())

	var testResults []testDoc
	for c.Next(context.Background()) {
		var doc testDoc
		if err := c.Decode(&doc); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		testResults = append(testResults, doc)
	}

	if !fetchCalled {
		t.Error("expected fetcher to be called for page 2")
	}

	if len(testResults) != 3 {
		t.Errorf("expected 3 results, got %d", len(testResults))
	}
}

func TestCursor_Current(t *testing.T) {
	docs := []testDoc{{ID: 1, Name: "Alice"}}

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages(docs), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	// Current before Next should be nil
	if c.Current() != nil {
		t.Error("expected Current() to be nil before Next()")
	}

	c.Next(context.Background())

	current := c.Current()
	if current == nil {
		t.Fatal("expected Current() to return document")
	}

	var doc testDoc
	if err := json.Unmarshal(current, &doc); err != nil {
		t.Fatalf("failed to unmarshal current: %v", err)
	}
	if doc.Name != "Alice" {
		t.Errorf("expected Alice, got %s", doc.Name)
	}
}

func TestCursor_Iterate(t *testing.T) {
	docs := []testDoc{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages(docs), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	var names []string
	err := c.Iterate(context.Background(), func(raw json.RawMessage) error {
		var doc testDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			return err
		}
		names = append(names, doc.Name)
		return nil
	})

	if err != nil {
		t.Fatalf("Iterate failed: %v", err)
	}

	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
}

func TestCursor_IterateEarlyStop(t *testing.T) {
	docs := []testDoc{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages(docs), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	count := 0
	err := c.Iterate(context.Background(), func(raw json.RawMessage) error {
		count++
		if count >= 2 {
			return io.EOF // Stop early
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Iterate should not return error for io.EOF: %v", err)
	}

	if count != 2 {
		t.Errorf("expected to process 2 items, got %d", count)
	}
}

func TestCursor_TryDecode(t *testing.T) {
	docs := []testDoc{{ID: 1, Name: "Alice"}}

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages(docs), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	c.Next(context.Background())

	doc, err := cursor.TryDecode[testDoc](c)
	if err != nil {
		t.Fatalf("TryDecode failed: %v", err)
	}
	if doc.Name != "Alice" {
		t.Errorf("expected Alice, got %s", doc.Name)
	}
}

func TestCursor_RemainingBatchLength(t *testing.T) {
	docs := []testDoc{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	fetcher := func(ctx context.Context, pageState *string) ([]json.RawMessage, *string, results.Warnings, error) {
		return makeRawMessages(docs), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	// Before any iteration
	if c.RemainingBatchLength() != 0 {
		t.Errorf("expected 0 remaining before iteration, got %d", c.RemainingBatchLength())
	}

	c.Next(context.Background()) // Move to first document
	if c.RemainingBatchLength() != 2 {
		t.Errorf("expected 2 remaining after first Next(), got %d", c.RemainingBatchLength())
	}

	c.Next(context.Background()) // Move to second
	if c.RemainingBatchLength() != 1 {
		t.Errorf("expected 1 remaining, got %d", c.RemainingBatchLength())
	}

	c.Next(context.Background()) // Move to third
	if c.RemainingBatchLength() != 0 {
		t.Errorf("expected 0 remaining, got %d", c.RemainingBatchLength())
	}
}

func TestCursor_HasNextPage(t *testing.T) {
	pageState := "page2"

	fetcher := func(ctx context.Context, ps *string) ([]json.RawMessage, *string, results.Warnings, error) {
		if ps == nil {
			return makeRawMessages([]testDoc{{ID: 1}}), &pageState, nil, nil
		}
		return makeRawMessages([]testDoc{{ID: 2}}), nil, nil, nil
	}

	c := cursor.New(fetcher)
	defer c.Close(context.Background())

	// Before first fetch
	if c.HasNextPage() {
		t.Error("expected HasNextPage() false before iteration")
	}

	c.Next(context.Background()) // Fetch first page

	if !c.HasNextPage() {
		t.Error("expected HasNextPage() true after first page")
	}

	c.Next(context.Background()) // Exhaust first page, fetch second

	if c.HasNextPage() {
		t.Error("expected HasNextPage() false after last page")
	}
}
