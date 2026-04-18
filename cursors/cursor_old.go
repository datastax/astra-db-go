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

// Package cursor provides iterator-style access to query results.
package cursors

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/datastax/astra-db-go/results"
)

// ErrCursorClosed is returned when operations are attempted on a closed cursor.
var ErrCursorClosed = errors.New("cursor is closed")

// ErrNoCurrentDocument is returned when Decode is called without calling Next first.
var ErrNoCurrentDocument = errors.New("no current document; call Next() first")

// PageFetcher is a function that fetches the next page of results.
// It receives the page state (nil for first page) and returns:
// - documents: raw JSON array of documents
// - nextPageState: state for fetching the next page (nil if no more pages)
// - warnings: any warnings from the API response
// - error: any error that occurred
type PageFetcher func(ctx context.Context, pageState *string) (documents []json.RawMessage, nextPageState *string, warnings results.Warnings, err error)

// Cursor provides an iterator interface for query results.
//
// Cursors automatically handle pagination, fetching new pages as needed
// when iterating through results.
//
// Example usage:
//
//	cursor := collection.Find(ctx, filter.F{"active": true})
//	defer cursor.Close(ctx)
//
//	for cursor.Next(ctx) {
//	    var doc MyDocument
//	    if err := cursor.Decode(&doc); err != nil {
//	        return err
//	    }
//	    // Process doc
//	}
//	if err := cursor.Err(); err != nil {
//	    return err
//	}
//
// Or to get all results at once:
//
//	cursor := collection.Find(ctx, filter.F{})
//	var docs []MyDocument
//	if err := cursor.All(ctx, &docs); err != nil {
//	    return err
//	}
type Cursor struct {
	mu sync.Mutex

	// Fetcher function to get next page
	fetcher PageFetcher

	// Current state
	state CursorState

	// Buffer of documents from current page
	buffer []json.RawMessage

	// Current position in buffer (-1 means not started)
	position int

	// Page state for fetching next page
	nextPageState *string

	// Any error that occurred
	err error

	// Whether we've fetched the first page
	initialized bool

	// Accumulated warnings from all fetched pages
	warnings results.Warnings
}

// New creates a new Cursor with the given page fetcher function.
func New(fetcher PageFetcher) *Cursor {
	return &Cursor{
		fetcher:  fetcher,
		state:    CursorStateIdle,
		buffer:   nil,
		position: -1,
	}
}

// NewWithError creates a Cursor that immediately returns the given error.
// This is useful when validation fails before any fetching can occur.
func NewWithError(err error) *Cursor {
	return &Cursor{
		state: CursorStateExhausted,
		err:   err,
	}
}

// NewWithInitialData creates a new Cursor pre-populated with data from an initial response.
// This is useful when the first page has already been fetched.
func NewWithInitialData(documents []json.RawMessage, nextPageState *string, warnings results.Warnings, fetcher PageFetcher) *Cursor {
	state := CursorStateIdle
	if len(documents) == 0 && (nextPageState == nil || *nextPageState == "") {
		state = CursorStateExhausted
	}

	return &Cursor{
		fetcher:       fetcher,
		state:         state,
		buffer:        documents,
		position:      -1,
		nextPageState: nextPageState,
		initialized:   true,
		warnings:      warnings,
	}
}

// State returns the current state of the cursor.
func (c *Cursor) State() CursorState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Next advances the cursor to the next document.
// It returns true if there is a document available, false otherwise.
// When Next returns false, check Err() for any errors that may have occurred.
//
// Example:
//
//	for cursor.Next(ctx) {
//	    var doc MyDoc
//	    cursor.Decode(&doc)
//	}
//	if err := cursor.Err(); err != nil {
//	    // handle error
//	}
func (c *Cursor) Next(ctx context.Context) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == CursorStateClosed {
		c.err = ErrCursorClosed
		return false
	}

	if c.state == CursorStateExhausted {
		return false
	}

	// If we haven't initialized yet, fetch first page
	if !c.initialized {
		if err := c.fetchPageLocked(ctx, nil); err != nil {
			c.err = err
			return false
		}
		c.initialized = true
	}

	c.state = CursorStateActive

	// Try to advance within current buffer
	if c.position+1 < len(c.buffer) {
		c.position++
		return true
	}

	// Need to fetch next page
	if c.nextPageState == nil || *c.nextPageState == "" {
		// No more pages
		c.state = CursorStateExhausted
		return false
	}

	// Fetch next page
	if err := c.fetchPageLocked(ctx, c.nextPageState); err != nil {
		c.err = err
		return false
	}

	// Check if we got any documents
	if len(c.buffer) == 0 {
		c.state = CursorStateExhausted
		return false
	}

	c.position = 0
	return true
}

// fetchPageLocked fetches a page of results. Must be called with mutex held.
func (c *Cursor) fetchPageLocked(ctx context.Context, pageState *string) error {
	documents, nextState, warnings, err := c.fetcher(ctx, pageState)
	if err != nil {
		return err
	}

	c.buffer = documents
	c.position = -1
	c.nextPageState = nextState

	// Accumulate warnings from each page
	if len(warnings) > 0 {
		c.warnings = append(c.warnings, warnings...)
	}

	return nil
}

// Decode unmarshals the current document into the provided value.
// Call Next() before calling Decode().
//
// Example:
//
//	var doc MyDocument
//	if err := cursor.Decode(&doc); err != nil {
//	    return err
//	}
func (c *Cursor) Decode(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	if c.position < 0 || c.position >= len(c.buffer) {
		return ErrNoCurrentDocument
	}

	return json.Unmarshal(c.buffer[c.position], v)
}

// TryDecode is like Decode but returns (value, error) instead of taking a pointer.
// Useful with generics in Go 1.18+.
func TryDecode[T any](c *Cursor) (T, error) {
	var v T
	err := c.Decode(&v)
	return v, err
}

// All decodes all remaining documents into the provided slice.
// The slice must be a pointer to a slice type.
// After All returns, the cursor will be exhausted.
//
// Example:
//
//	var docs []MyDocument
//	if err := cursor.All(ctx, &docs); err != nil {
//	    return err
//	}
func (c *Cursor) All(ctx context.Context, results any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}

	if c.state == CursorStateClosed {
		return ErrCursorClosed
	}

	// Collect all raw documents
	var allDocs []json.RawMessage

	// If we haven't initialized, fetch first page
	if !c.initialized {
		if err := c.fetchPageLocked(ctx, nil); err != nil {
			c.err = err
			return err
		}
		c.initialized = true
	}

	// Add any remaining documents from current buffer
	if c.position < 0 {
		allDocs = append(allDocs, c.buffer...)
	} else if c.position+1 < len(c.buffer) {
		allDocs = append(allDocs, c.buffer[c.position+1:]...)
	}

	// Fetch remaining pages
	for c.nextPageState != nil && *c.nextPageState != "" {
		if err := c.fetchPageLocked(ctx, c.nextPageState); err != nil {
			c.err = err
			return err
		}
		allDocs = append(allDocs, c.buffer...)
	}

	c.state = CursorStateExhausted

	// Marshal collected documents to JSON array and unmarshal into results
	if len(allDocs) == 0 {
		// Return empty but don't error
		return json.Unmarshal([]byte("[]"), results)
	}

	// Build JSON array from raw messages
	arrayJSON, err := json.Marshal(allDocs)
	if err != nil {
		return err
	}

	return json.Unmarshal(arrayJSON, results)
}

// Err returns any error that occurred during iteration.
// Check this after Next() returns false.
func (c *Cursor) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Warnings returns all warnings accumulated during iteration.
// This includes warnings from all pages that have been fetched.
func (c *Cursor) Warnings() results.Warnings {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.warnings
}

// Close closes the cursor, releasing any resources.
// After Close is called, the cursor cannot be used.
func (c *Cursor) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == CursorStateClosed {
		return nil
	}

	c.state = CursorStateClosed
	c.buffer = nil
	c.nextPageState = nil

	return nil
}

// RemainingBatchLength returns the number of documents remaining in the current batch.
func (c *Cursor) RemainingBatchLength() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.position < 0 {
		return len(c.buffer)
	}
	remaining := len(c.buffer) - c.position - 1
	if remaining < 0 {
		return 0
	}
	return remaining
}

// HasNextPage returns true if there are more pages to fetch.
// Note: This doesn't guarantee more documents exist, just that pagination continues.
func (c *Cursor) HasNextPage() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nextPageState != nil && *c.nextPageState != ""
}

// ID returns a unique identifier for this cursor (for compatibility).
// Since Astra DB doesn't use server-side cursors, this returns 0.
func (c *Cursor) ID() int64 {
	return 0
}

// Current returns the current document as raw JSON.
// Returns nil if Next() hasn't been called or if there's no current document.
func (c *Cursor) Current() json.RawMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.position < 0 || c.position >= len(c.buffer) {
		return nil
	}
	return c.buffer[c.position]
}

// Iterate is a convenience method that calls a function for each document.
// Iteration stops if the function returns an error or io.EOF.
// Returning io.EOF stops iteration without error.
//
// Example:
//
//	err := cursor.Iterate(ctx, func(raw json.RawMessage) error {
//	    var doc MyDoc
//	    if err := json.Unmarshal(raw, &doc); err != nil {
//	        return err
//	    }
//	    fmt.Println(doc.Name)
//	    return nil
//	})
func (c *Cursor) Iterate(ctx context.Context, fn func(json.RawMessage) error) error {
	for c.Next(ctx) {
		doc := c.Current()
		if err := fn(doc); err != nil {
			if errors.Is(err, io.EOF) {
				return nil // Stop iteration without error
			}
			return err
		}
	}
	return c.Err()
}
