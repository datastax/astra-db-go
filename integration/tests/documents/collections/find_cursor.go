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

package collections

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/cursors"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	asort "github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("find-cursor")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	// Setup test data
	docs := []astra.NewDocument{
		{"_id": "0", "int": 0},
		{"_id": "1", "int": 1},
		{"_id": "2", "int": 2},
	}

	// Large dataset for pagination tests (95 documents)
	docs_ := make([]astra.NewDocument, 95)
	for i := 0; i < 95; i++ {
		idStr := fmt.Sprintf("%d", i)
		if i < 10 {
			idStr = "0" + idStr
		}
		docs_[i] = astra.NewDocument{"_id": idStr, "int": i}
	}

	s.Before(func(t *harness.T) {
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		_, err = t.Collection_.InsertMany(t.Ctx, docs_, options.CollectionInsertMany().SetOrdered(true))
		testlib.FailIfErr(t, err, "InsertMany for large dataset failed: %v", err)
	})

	// State and Buffered tests
	s.Run("should return correct initial state", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		testlib.FailIf(t, cursor.State() != cursors.CursorStateIdle, "expected cursor state to be idle, got %v", cursor.State())
		testlib.FailIf(t, cursor.Buffered() != 0, "expected buffered count to be 0, got %d", cursor.Buffered())
	})

	s.Run("should transition to started state after Next", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		testlib.FailIf(t, cursor.State() != cursors.CursorStateIdle, "expected initial state to be idle")

		hasNext := cursor.Next(t.Ctx)
		testlib.FailIf(t, !hasNext, "expected Next to return true")
		testlib.FailIf(t, cursor.State() != cursors.CursorStateStarted, "expected state to be started after Next")
		testlib.FailIf(t, cursor.Buffered() != 3, "expected 3 buffered documents after first Next")
	})

	s.Run("should buffer documents on first Next call", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered initially")

		cursor.Next(t.Ctx)
		testlib.FailIf(t, cursor.Buffered() != 3, "expected 3 buffered after first Next")
	})

	// Next tests
	s.Run("should iterate through all documents with Next", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		seen := make(map[string]bool)
		count := 0

		for cursor.Next(t.Ctx) {
			var doc astra.Document
			err := cursor.Decode(&doc)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)

			id := doc.MustGet("_id").(string)
			seen[id] = true
			count++
		}

		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, count != 3, "expected 3 documents, got %d", count)
		testlib.FailIf(t, !seen["0"] || !seen["1"] || !seen["2"], "not all documents were seen")
	})

	s.Run("should return false when no more documents", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		// Consume all documents
		for cursor.Next(t.Ctx) {
			var doc astra.Document
			cursor.Decode(&doc)
		}

		// Next should return false now
		hasNext := cursor.Next(t.Ctx)
		testlib.FailIf(t, hasNext, "expected Next to return false after exhausting cursor")
		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered after exhaustion")
	})

	s.Run("should close cursor when exhausted", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})

		for cursor.Next(t.Ctx) {
			var doc astra.Document
			cursor.Decode(&doc)
		}

		testlib.FailIf(t, cursor.State() != cursors.CursorStateClosed, "expected cursor to be closed after exhaustion")
	})

	s.Run("should handle pagination across multiple pages", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{})
		defer cursor.Close()

		count := 0
		for cursor.Next(t.Ctx) {
			var doc astra.Document
			err := cursor.Decode(&doc)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			count++
		}

		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, count != 95, "expected 95 documents, got %d", count)
	})

	// Decode tests
	s.Run("should decode current document without advancing", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		cursor.Next(t.Ctx)

		var doc1 astra.Document
		err := cursor.Decode(&doc1)
		testlib.FailIfErr(t, err, "first Decode failed: %v", err)

		var doc2 astra.Document
		err = cursor.Decode(&doc2)
		testlib.FailIfErr(t, err, "second Decode failed: %v", err)

		// Should be the same document
		t.NoDiff(doc1.ToMap(), doc2.ToMap())
	})

	s.Run("should fail to decode when cursor is closed", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		cursor.Close()

		var doc astra.Document
		err := cursor.Decode(&doc)
		testlib.FailIf(t, err != cursors.ErrCursorClosed, "expected ErrCursorClosed, got %v", err)
	})

	s.Run("should fail to decode when no current document", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		var doc astra.Document
		err := cursor.Decode(&doc)
		testlib.FailIf(t, err != cursors.ErrNoCurrentDocument, "expected ErrNoCurrentDocument, got %v", err)
	})

	// DecodeAll tests
	s.Run("should decode all documents with DecodeAll", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 3, "expected 3 documents, got %d", len(results))
		testlib.FailIf(t, cursor.State() != cursors.CursorStateClosed, "expected cursor to be closed after DecodeAll")
		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered after DecodeAll")
	})

	s.Run("should decode all documents across pages", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{})

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 95, "expected 95 documents, got %d", len(results))
	})

	s.Run("should return empty slice when no documents found", func(t *harness.T) {
		cursor := t.Collection.Find(filter.Eq("_id", "nonexistent"))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 0, "expected 0 documents, got %d", len(results))
	})

	// Filter tests
	s.Run("should filter documents correctly", func(t *harness.T) {
		cursor := t.Collection.Find(filter.Eq("_id", "1"))
		defer cursor.Close()

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 1, "expected 1 document, got %d", len(results))
		testlib.FailIf(t, results[0].MustGet("_id").(string) != "1", "expected _id '1', got %v", results[0].MustGet("_id"))
	})

	// Limit tests
	s.Run("should limit documents", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{}, options.CollectionFind().SetLimit(2))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 2, "expected 2 documents, got %d", len(results))
	})

	s.Run("should limit documents across pages", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetLimit(50))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 50, "expected 50 documents, got %d", len(results))
	})

	s.Run("should have no limit when limit is 0", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetLimit(0))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 95, "expected 95 documents, got %d", len(results))
	})

	// Skip tests
	s.Run("should skip documents", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{}, options.CollectionFind().SetSkip(1).SetSort(asort.Asc("_id")))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 2, "expected 2 documents, got %d", len(results))

		// Sort results by _id for comparison
		sort.Slice(results, func(i, j int) bool {
			return results[i].MustGet("_id").(string) < results[j].MustGet("_id").(string)
		})

		testlib.FailIf(t, results[0].MustGet("_id").(string) != "1", "expected first doc _id '1', got %v", results[0].MustGet("_id"))
		testlib.FailIf(t, results[1].MustGet("_id").(string) != "2", "expected second doc _id '2', got %v", results[1].MustGet("_id"))
	})

	s.Run("should skip documents across pages", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetSkip(50).SetSort(asort.Asc("_id")))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 45, "expected 45 documents, got %d", len(results))

		// First result should be document 50
		firstInt := results[0].MustGet("int").(float64)
		testlib.FailIf(t, int(firstInt) != 50, "expected first doc int 50, got %v", firstInt)
	})

	s.Run("should combine skip and limit across pages", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetSkip(50).SetLimit(20).SetSort(asort.Asc("_id")))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 20, "expected 20 documents, got %d", len(results))

		// First result should be document 50, last should be 69
		firstInt := results[0].MustGet("int").(float64)
		lastInt := results[19].MustGet("int").(float64)
		testlib.FailIf(t, int(firstInt) != 50, "expected first doc int 50, got %v", firstInt)
		testlib.FailIf(t, int(lastInt) != 69, "expected last doc int 69, got %v", lastInt)
	})

	// Sort tests
	s.Run("should sort documents", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetSort(asort.Asc("_id")))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		// Should only return first page (up to 50 docs) when sort is used
		testlib.FailIf(t, len(results) > 50, "expected at most 50 documents with sort, got %d", len(results))

		// Verify sorted order
		for i := 1; i < len(results); i++ {
			prev := results[i-1].MustGet("_id").(string)
			curr := results[i].MustGet("_id").(string)
			testlib.FailIf(t, prev > curr, "documents not sorted: %s > %s", prev, curr)
		}
	})

	s.Run("should only return one page with sort", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetSort(asort.Asc("_id")))

		// Get first document
		hasNext := cursor.Next(t.Ctx)
		testlib.FailIf(t, !hasNext, "expected at least one document")

		var firstDoc astra.Document
		err := cursor.Decode(&firstDoc)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)

		// Consume buffer
		buffered := cursor.Buffered()
		for i := 0; i < buffered; i++ {
			cursor.Next(t.Ctx)
		}

		// Should be no more documents
		hasNext = cursor.Next(t.Ctx)
		testlib.FailIf(t, hasNext, "expected no more documents after consuming first page with sort")
		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered after exhausting sorted cursor")
	})

	// Projection tests
	s.Run("should project documents", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{}, options.CollectionFind().
			SetProjection(map[string]any{"int": 0}).
			SetSort(asort.Asc("_id")))

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 3, "expected 3 documents, got %d", len(results))

		// Verify projection - should have _id but not int
		for _, doc := range results {
			_, hasID := doc.Get("_id")
			_, hasInt := doc.Get("int")
			testlib.FailIf(t, !hasID, "expected _id field in projected document")
			testlib.FailIf(t, hasInt, "expected int field to be excluded in projected document")
		}
	})

	// Rewind tests
	s.Run("should allow rewound cursor to re-fetch all data", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		// First iteration
		var results1 []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results1)
		testlib.FailIfErr(t, err, "first DecodeAll failed: %v", err)
		testlib.FailIf(t, len(results1) != 3, "expected 3 documents in first iteration, got %d", len(results1))
		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered after first DecodeAll")

		// Rewind
		cursor.Rewind()
		testlib.FailIf(t, cursor.State() != cursors.CursorStateIdle, "expected cursor state to be idle after rewind")

		// Second iteration
		var results2 []astra.Document
		err = cursor.DecodeAll(t.Ctx, &results2)
		testlib.FailIfErr(t, err, "second DecodeAll failed: %v", err)
		testlib.FailIf(t, len(results2) != 3, "expected 3 documents in second iteration, got %d", len(results2))
		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered after second DecodeAll")
	})

	// Clone tests
	s.Run("should allow cloned cursor to re-fetch all data", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		// First iteration on original cursor
		var results1 []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results1)
		testlib.FailIfErr(t, err, "DecodeAll on original failed: %v", err)
		testlib.FailIf(t, len(results1) != 3, "expected 3 documents from original, got %d", len(results1))

		// Clone and iterate
		clone := cursor.Clone()
		defer clone.Close()

		var results2 []astra.Document
		err = clone.DecodeAll(t.Ctx, &results2)
		testlib.FailIfErr(t, err, "DecodeAll on clone failed: %v", err)
		testlib.FailIf(t, len(results2) != 3, "expected 3 documents from clone, got %d", len(results2))
	})

	// Close tests
	s.Run("should return false on Next after Close", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		cursor.Close()

		hasNext := cursor.Next(t.Ctx)
		testlib.FailIf(t, hasNext, "expected Next to return false on closed cursor")
		testlib.FailIf(t, cursor.State() != cursors.CursorStateClosed, "expected cursor state to be closed")
	})

	// DecodeBuffered tests
	s.Run("should decode buffered documents without fetching next page", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		// Trigger first fetch
		cursor.Next(t.Ctx)
		testlib.FailIf(t, cursor.Buffered() != 3, "expected 3 buffered after first Next")

		// Decode buffered (should get 3 docs, total 3 including current)
		var buffered []astra.Document
		err := cursor.DecodeBuffered(&buffered, 0)
		testlib.FailIfErr(t, err, "DecodeBuffered failed: %v", err)

		testlib.FailIf(t, len(buffered) != 3, "expected 3 buffered documents, got %d", len(buffered))
		testlib.FailIf(t, cursor.Buffered() != 0, "expected 0 buffered after DecodeBuffered")
	})

	s.Run("should decode limited buffered documents", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		// Trigger first fetch
		cursor.Next(t.Ctx)
		testlib.FailIf(t, cursor.Buffered() != 3, "expected 3 buffered after first Next")

		// Decode only 1 buffered document
		var buffered []astra.Document
		err := cursor.DecodeBuffered(&buffered, 1)
		testlib.FailIfErr(t, err, "DecodeBuffered failed: %v", err)

		testlib.FailIf(t, len(buffered) != 1, "expected 1 buffered document, got %d", len(buffered))
		testlib.FailIf(t, cursor.Buffered() != 2, "expected 2 still buffered after DecodeBuffered with limit")
	})

	// Helper function tests using cursors.Decode
	s.Run("should decode using helper function", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		cursor.Next(t.Ctx)

		doc, err := cursors.Decode[astra.Document](cursor)
		testlib.FailIfErr(t, err, "cursors.Decode failed: %v", err)

		_, hasID := doc.Get("_id")
		testlib.FailIf(t, !hasID, "expected _id field in decoded document")
	})

	// Helper function tests using cursors.DecodeAll
	s.Run("should decode all using helper function", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})

		results, err := cursors.DecodeAll[astra.Document](t.Ctx, cursor)
		testlib.FailIfErr(t, err, "cursors.DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 3, "expected 3 documents, got %d", len(results))
	})

	// Typed document tests
	s.Run("should work with typed documents", func(t *harness.T) {
		type TestDoc struct {
			ID  string `json:"_id"`
			Int int    `json:"int"`
		}

		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		var results []TestDoc
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll with typed docs failed: %v", err)

		testlib.FailIf(t, len(results) != 3, "expected 3 documents, got %d", len(results))

		// Verify typed fields
		for _, doc := range results {
			testlib.FailIf(t, doc.ID == "", "expected non-empty ID")
			testlib.FailIf(t, doc.Int < 0 || doc.Int > 2, "expected int in range [0,2], got %d", doc.Int)
		}
	})

	// NextPageState tests
	s.Run("should return nil page state initially", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		pageState := cursor.NextPageState()
		testlib.FailIf(t, pageState != nil, "expected nil page state before iteration")
	})

	s.Run("should return page state after fetching first page", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{})
		defer cursor.Close()

		// Fetch first page
		cursor.Next(t.Ctx)

		pageState := cursor.NextPageState()
		// For a dataset with 95 docs, there should be a next page
		testlib.FailIf(t, pageState == nil, "expected non-nil page state after first page")
	})

	s.Run("should return nil page state when exhausted", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		// Exhaust cursor
		for cursor.Next(t.Ctx) {
			var doc astra.Document
			cursor.Decode(&doc)
		}

		pageState := cursor.NextPageState()
		testlib.FailIf(t, pageState != nil, "expected nil page state after exhaustion")
	})

	// Iterator pattern tests using cursors.All
	s.Run("should iterate using cursors.All helper", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		count := 0
		for doc, err := range cursors.All[astra.Document](t.Ctx, cursor) {
			testlib.FailIfErr(t, err, "iteration error: %v", err)
			testlib.FailIf(t, doc == nil, "expected non-nil document")
			count++
		}

		testlib.FailIf(t, count != 3, "expected 3 documents, got %d", count)
	})

	// Complex query tests
	s.Run("should handle complex query with multiple options", func(t *harness.T) {
		cursor := t.Collection_.Find(
			filter.Gte("int", 10),
			options.CollectionFind().
				SetSort(asort.Desc("int")).
				SetLimit(10).
				SetSkip(5).
				SetProjection(map[string]any{"_id": 1, "int": 1}),
		)
		defer cursor.Close()

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		testlib.FailIf(t, len(results) != 10, "expected 10 documents, got %d", len(results))

		// Verify all have int >= 10
		for _, doc := range results {
			intVal := doc.MustGet("int").(float64)
			testlib.FailIf(t, intVal < 10, "expected int >= 10, got %v", intVal)
		}
	})

	// Error handling tests
	s.Run("should handle decode errors gracefully", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{})
		defer cursor.Close()

		cursor.Next(t.Ctx)

		// Try to decode into wrong type
		var wrongType int
		err := cursor.Decode(&wrongType)
		testlib.FailIf(t, err == nil, "expected decode error when decoding to wrong type")

		// Cursor should still be usable
		var doc astra.Document
		err = cursor.Decode(&doc)
		testlib.FailIfErr(t, err, "Decode should work after previous decode error: %v", err)
	})

	// Numeric ID tests
	s.Run("should handle numeric string IDs correctly", func(t *harness.T) {
		cursor := t.Collection.Find(filter.F{}, options.CollectionFind().SetSort(asort.Asc("_id")))
		defer cursor.Close()

		var results []astra.Document
		err := cursor.DecodeAll(t.Ctx, &results)
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)

		// Verify IDs are strings and sorted
		for i, doc := range results {
			id := doc.MustGet("_id").(string)
			expectedID := strconv.Itoa(i)
			testlib.FailIf(t, id != expectedID, "expected ID %s, got %s", expectedID, id)
		}
	})

	sv := harness.ParallelSuite("find-cursor")
	sv.Truncate(harness.SelectCollections, harness.SelectBefore)

	// Sort vector tests
	sv.Run("should return sort vector when includeSortVector is true", func(t *harness.T) {
		// Insert documents with vectors
		vectorDocs := []astra.NewDocument{
			{"_id": t.Key(0), "int": 0, "$vector": []float32{1, 1, 1, 1, 1}},
			{"_id": t.Key(1), "int": 1, "$vector": []float32{1, 1, 1, 1, 1}},
			{"_id": t.Key(2), "int": 2, "$vector": []float32{1, 1, 1, 1, 1}},
		}
		_, err := t.Collection.InsertMany(t.Ctx, vectorDocs)
		testlib.FailIfErr(t, err, "InsertMany for vector docs failed: %v", err)

		cursor := t.Collection.Find(filter.F{}, options.CollectionFind().
			SetSort(asort.Vector([]float32{1, 1, 1, 1, 1})).
			SetIncludeSortVector(true))
		defer cursor.Close()

		vec := cursor.GetSortVector(t.Ctx)
		testlib.FailIf(t, vec == nil, "expected sort vector to be non-nil")

		vecArray, err := vec.AsFloatArray()
		testlib.FailIfErr(t, err, "AsFloatArray failed: %v", err)
		testlib.FailIf(t, len(vecArray) != 5, "expected vector length 5, got %d", len(vecArray))

		// Verify vector values
		for i, v := range vecArray {
			testlib.FailIf(t, v != 1.0, "expected vector[%d] to be 1.0, got %f", i, v)
		}

		// Cleanup
		for _, doc := range vectorDocs {
			t.Collection.DeleteOne(t.Ctx, filter.Eq("_id", doc["_id"]))
		}
	})

	sv.Run("should return nil sort vector when includeSortVector is false", func(t *harness.T) {
		// Insert documents with vectors
		vectorDocs := []astra.NewDocument{
			{"_id": t.Key(0), "int": 0, "$vector": []float32{1, 1, 1, 1, 1}},
		}
		_, err := t.Collection.InsertMany(t.Ctx, vectorDocs)
		testlib.FailIfErr(t, err, "InsertMany for vector docs failed: %v", err)

		cursor := t.Collection.Find(filter.F{}, options.CollectionFind().
			SetSort(asort.Vector([]float32{1, 1, 1, 1, 1})))
		defer cursor.Close()

		cursor.Next(t.Ctx)
		vec := cursor.GetSortVector(t.Ctx)
		testlib.FailIf(t, vec != nil, "expected sort vector to be nil when includeSortVector is false")

		// Cleanup
		t.Collection.DeleteOne(t.Ctx, filter.Eq("_id", t.Key(0)))
	})

	sv.Run("should return nil sort vector when no vector sort", func(t *harness.T) {
		cursor := t.Collection_.Find(filter.F{}, options.CollectionFind().SetIncludeSortVector(true))
		defer cursor.Close()

		cursor.Next(t.Ctx)
		vec := cursor.GetSortVector(t.Ctx)
		testlib.FailIf(t, vec != nil, "expected sort vector to be nil when no vector sort is used")
	})
}
