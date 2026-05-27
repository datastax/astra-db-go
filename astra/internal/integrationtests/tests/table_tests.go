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

package tests

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/datastax/astra-db-go/astra/cursors"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/sort"
	"github.com/datastax/astra-db-go/astra/table"
)

func init() {
	// Register table tests
	t := []harness.IntegrationTest{
		{Name: "TableCreate", Run: TableCreate},
		{Name: "TableInsertOne", Run: TableInsertOne},
		{Name: "TableInsertMany", Run: TableInsertMany},
		{Name: "TableFindOne", Run: TableFindOne},
		{Name: "TableFind", Run: TableFind},
		{Name: "TableFindWithCursor", Run: TableFindWithCursor},
		{Name: "TableFindWithSort", Run: TableFindWithSort},
		{Name: "TableFindWithProjection", Run: TableFindWithProjection},
		{Name: "TableListIndexes", Run: TableListIndexes},
		{Name: "TableListTables", Run: TableListTables},
		{Name: "TableVectorIndex", Run: TableVectorIndex},
		{Name: "TableAlter", Run: TableAlter},
		{Name: "TableDrop", Run: TableDrop},
	}
	harness.Register(t...)
}

const tableName = "go_test_books"

// TestBook represents a book for table tests
type TestBook struct {
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	NumberOfPages int      `json:"number_of_pages"`
	Rating        float32  `json:"rating"`
	IsCheckedOut  bool     `json:"is_checked_out"`
	Genres        []string `json:"genres"`
}

func TableCreate(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	// TODO: at some point we could switch this to using newer infer method.
	// But it might be better to have a separate test so we exercise both infer
	// and table.Definition.
	definition := table.Definition{
		Columns: table.Columns{
			{Name: "title", Column: table.Text()},
			{Name: "author", Column: table.Text()},
			{Name: "number_of_pages", Column: table.Int()},
			{Name: "rating", Column: table.Float()},
			{Name: "is_checked_out", Column: table.Boolean()},
			{Name: "genres", Column: table.List(table.Text())},
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy: []string{"title"},
		},
	}

	_, err := db.CreateTable(ctx, tableName, definition, options.CreateTable().SetIfNotExists(true))
	return err
}

func TableInsertOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	table := db.Table(tableName)

	book := TestBook{
		Title:         "The Great Gatsby",
		Author:        "F. Scott Fitzgerald",
		NumberOfPages: 180,
		Rating:        4.5,
		IsCheckedOut:  false,
		Genres:        []string{"Fiction", "Classic"},
	}

	resp, err := table.InsertOne(ctx, book)
	if err != nil {
		return err
	}

	insertedID, err := resp.RawID()
	if err != nil {
		return fmt.Errorf("failed to get inserted ID: %w", err)
	}

	// The API returns insertedIds as an array of primary key values
	// For a single-column primary key like "title", it returns ["The Great Gatsby"]
	pkValues, ok := insertedID.([]any)
	if !ok {
		return fmt.Errorf("expected inserted ID to be []any, got %T", insertedID)
	}
	if len(pkValues) != 1 {
		return fmt.Errorf("expected 1 primary key value, got %d", len(pkValues))
	}
	insertedTitle, ok := pkValues[0].(string)
	if !ok {
		return fmt.Errorf("expected primary key value to be string, got %T", pkValues[0])
	}
	if insertedTitle != book.Title {
		return fmt.Errorf("expected inserted ID %q, got %q", book.Title, insertedTitle)
	}

	return nil
}

func TableInsertMany(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	table := db.Table(tableName)

	books := []TestBook{
		{
			Title:         "1984",
			Author:        "George Orwell",
			NumberOfPages: 328,
			Rating:        4.7,
			IsCheckedOut:  true,
			Genres:        []string{"Dystopian", "Science Fiction"},
		},
		{
			Title:         "To Kill a Mockingbird",
			Author:        "Harper Lee",
			NumberOfPages: 281,
			Rating:        4.8,
			IsCheckedOut:  false,
			Genres:        []string{"Fiction", "Classic"},
		},
		{
			Title:         "Pride and Prejudice",
			Author:        "Jane Austen",
			NumberOfPages: 279,
			Rating:        4.6,
			IsCheckedOut:  false,
			Genres:        []string{"Romance", "Classic"},
		},
		{
			Title:         "The Catcher in the Rye",
			Author:        "J.D. Salinger",
			NumberOfPages: 234,
			Rating:        4.0,
			IsCheckedOut:  true,
			Genres:        []string{"Fiction", "Coming-of-age"},
		},
		{
			Title:         "Brave New World",
			Author:        "Aldous Huxley",
			NumberOfPages: 311,
			Rating:        4.5,
			IsCheckedOut:  false,
			Genres:        []string{"Dystopian", "Science Fiction"},
		},
	}

	resp, err := table.InsertMany(ctx, books)
	if err != nil {
		return err
	}

	// Use the new InsertedCount() method
	if resp.InsertedCount() != len(books) {
		return fmt.Errorf("expected %d inserted IDs, got %d", len(books), resp.InsertedCount())
	}

	return nil
}

func TableFindOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	table := db.Table(tableName)

	// Find the book we inserted in TableInsertOne
	var book TestBook
	err := table.FindOne(ctx, filter.Eq("title", "The Great Gatsby")).Decode(&book)
	if err != nil {
		return err
	}

	if book.Title != "The Great Gatsby" {
		return fmt.Errorf("expected title 'The Great Gatsby', got %q", book.Title)
	}
	if book.Author != "F. Scott Fitzgerald" {
		return fmt.Errorf("expected author 'F. Scott Fitzgerald', got %q", book.Author)
	}
	if book.NumberOfPages != 180 {
		return fmt.Errorf("expected 180 pages, got %d", book.NumberOfPages)
	}

	return nil
}

func TableFind(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	warningHandlerRun := false
	tbl := db.Table(tableName, options.WithWarningHandler(func(w results.Warning) {
		warningHandlerRun = true
	}))

	// Find all books that are not checked out using cursors.All()
	cursor := tbl.Find(filter.Eq("is_checked_out", false))
	defer cursor.Close()

	var books []TestBook

	for book, err := range cursors.All[TestBook](ctx, cursor) {
		if err != nil {
			return fmt.Errorf("error iterating cursor: %w", err)
		}
		if book.IsCheckedOut {
			return fmt.Errorf("expected is_checked_out to be false for book %q", book.Title)
		}
		books = append(books, *book)
	}

	if len(books) == 0 {
		return errors.New("expected to find at least one book")
	}

	if !warningHandlerRun {
		return errors.New("expected warning handler to run but it did not")
	}

	// We should have a MISSING_INDEX warning because we filtered by a non-indexed column.
	// TODO: We could be more specific and check for the exact warning code/message. For now,
	// just ensure we got some warnings. It's unclear if the code might change in the future.
	if len(cursor.Warnings()) == 0 {
		return errors.New("expected warnings for filtering on non-indexed column but got none")
	}

	// Next, create index and verify warnings go away
	if err := tbl.CreateIndex(ctx, "is_checked_out_idx", "is_checked_out"); err != nil {
		return err
	}

	// Creating an index and then immediately querying seems to cause the following error:
	// > The Data API command generated a database query that was syntactically correct and the database started processing, however too many nodes failed to complete the operation...
	// See: https://github.com/datastax/astra-db-go/issues/4
	time.Sleep(2 * time.Second)

	// Verify warnings go away after creating the index
	// Find all books that are not checked out using cursor.DecodeAll()
	idxCursor := tbl.Find(filter.Eq("is_checked_out", false))
	defer idxCursor.Close()

	if err := idxCursor.DecodeAll(ctx, &books); err != nil {
		return err
	}

	if len(idxCursor.Warnings()) > 0 {
		return fmt.Errorf("expected no warnings after index creation. Got: %v", idxCursor.Warnings())
	}

	// Let's double-create that index and make sure it doesn't error out
	if err := tbl.CreateIndex(ctx, "is_checked_out_idx", "is_checked_out", options.CreateIndex().SetIfNotExists(true)); err != nil {
		return err
	}

	// Finally - drop index
	if err := db.DropTableIndex(ctx, "is_checked_out_idx"); err != nil {
		return err
	}

	return nil
}

// TableFindWithCursor demonstrates iterating with Next/Decode pattern
func TableFindWithCursor(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	tbl := db.Table(tableName)

	// Find all books using cursor iteration
	cursor := tbl.Find(filter.F{})
	defer cursor.Close()

	var books []TestBook
	for cursor.Next(ctx) {
		var book TestBook
		if err := cursor.Decode(&book); err != nil {
			return fmt.Errorf("decode error: %w", err)
		}
		books = append(books, book)
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	if len(books) == 0 {
		return errors.New("expected to find at least one book using cursor iteration")
	}

	return nil
}

func TableFindWithSort(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	tbl := db.Table(tableName)

	// Find books sorted by rating descending using cursors.DecodeAll()
	cursor := tbl.Find(filter.F{},
		options.TableFind().
			SetSort(sort.Desc("rating")).
			SetLimit(3),
	)
	defer cursor.Close()

	books, err := cursors.DecodeAll[TestBook](ctx, cursor)
	if err != nil {
		return err
	}

	if len(books) == 0 {
		return errors.New("expected to find at least one book")
	}

	// Verify books are sorted by rating descending
	for i := 1; i < len(books); i++ {
		if books[i].Rating > books[i-1].Rating {
			return fmt.Errorf("expected books to be sorted by rating descending, but book %q (%.1f) comes after %q (%.1f)",
				books[i].Title, books[i].Rating, books[i-1].Title, books[i-1].Rating)
		}
	}

	return nil
}

func TableFindWithProjection(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	tbl := db.Table(tableName)

	// Find books with only title and author using cursor.DecodeAll()
	cursor := tbl.Find(filter.F{},
		options.TableFind().
			SetProjection(map[string]any{"title": true, "author": true}).
			SetLimit(1),
	)
	defer cursor.Close()

	var books []map[string]any
	if err := cursor.DecodeAll(ctx, &books); err != nil {
		return err
	}

	if len(books) == 0 {
		return errors.New("expected to find at least one book")
	}

	book := books[0]

	// Verify only title and author are present
	if _, ok := book["title"]; !ok {
		return errors.New("expected title to be in projection")
	}
	if _, ok := book["author"]; !ok {
		return errors.New("expected author to be in projection")
	}

	// Check that other fields are not present (or are zero/nil)
	if rating, ok := book["rating"]; ok && rating != nil && !reflect.ValueOf(rating).IsZero() {
		return fmt.Errorf("expected rating to be excluded from projection, got %v", rating)
	}

	return nil
}

func TableListIndexes(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	tbl := db.Table(tableName)

	// Create an index for testing
	indexName := "rating_idx"
	if err := tbl.CreateIndex(ctx, indexName, "rating", options.CreateIndex().SetIfNotExists(true)); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Test listing indexes without explain (names only)
	indexes, err := tbl.ListIndexes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	// Verify we got at least one index
	if len(indexes) == 0 {
		return errors.New("expected at least one index")
	}

	// Verify our index is in the list
	found := false
	for _, idx := range indexes {
		if idx.Name == indexName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("expected to find index %q in list", indexName)
	}

	// Test listing indexes with explain (full metadata)
	indexesExplain, err := tbl.ListIndexes(ctx, options.ListIndexes().SetExplain(true))
	if err != nil {
		return fmt.Errorf("failed to list indexes with explain: %w", err)
	}

	// Verify we got the same number of indexes
	if len(indexesExplain) != len(indexes) {
		return fmt.Errorf("expected %d indexes with explain, got %d", len(indexes), len(indexesExplain))
	}

	// Find our index and verify metadata
	for _, idx := range indexesExplain {
		if idx.Name == indexName {
			if idx.Definition == nil {
				return errors.New("expected definition to be present with explain=true")
			}
			if idx.Definition.Column != "rating" {
				return fmt.Errorf("expected column 'rating', got %q", idx.Definition.Column)
			}
			if idx.IndexType != "regular" {
				return fmt.Errorf("expected indexType 'regular', got %q", idx.IndexType)
			}
			break
		}
	}

	// Clean up the index
	if err := db.DropTableIndex(ctx, indexName); err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	return nil
}

// TableListTables tests listing tables with both names-only and full metadata (explain=true).
// This should be run after TableCreate and before TableDrop. One of the things I have
// thought about is improving how `IntegrationTest` registers tests to allow running
// single tests that have dependencies. Something like:
//
//	type IntegrationTest struct {
//		Name string
//		Run  func(e *TestEnv) error
//		DependsOn []string // Ensure these have run before this test
//		CleanUp   []string // Ensure these tests run after this test
//	}
//
// That would allow us to set `TEST_PREFIX=TableListTables` and just run this test with
// dependencies. Right now, to test ListTables you need to run all the table tests
// (`TEST_PREFIX=Table`).
func TableListTables(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	// Names-only listing
	names, err := db.ListTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to list table names: %w", err)
	}
	found := false
	for _, n := range names {
		if n == tableName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("expected to find table %q in ListTableNames result, got %v", tableName, names)
	}

	// Full descriptors with explain=true
	tables, err := db.ListTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	var desc *results.TableDescriptor
	for i := range tables {
		if tables[i].Name == tableName {
			desc = &tables[i]
			break
		}
	}
	if desc == nil {
		return fmt.Errorf("expected to find table %q in ListTables result", tableName)
	}
	if len(desc.Definition.Columns) == 0 {
		return errors.New("expected non-empty Definition.Columns")
	}
	// TableCreate uses PartitionBy=["title"]. This also exercises PrimaryKey.UnmarshalAstra
	// (the API returns the single-column form as a string).
	if pk := desc.Definition.PrimaryKey.PartitionBy; len(pk) != 1 || pk[0] != "title" {
		return fmt.Errorf("expected PartitionBy=[\"title\"], got %v", pk)
	}

	return nil
}

const vectorTableName = "go_test_vectors"

// TestDocument represents a document with vector embeddings for vector index tests
type TestDocument struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding"`
}

func TableVectorIndex(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	// Create a table with a vector column
	definition := table.Definition{
		Columns: table.Columns{
			{Name: "id", Column: table.Text()},
			{Name: "content", Column: table.Text()},
			{Name: "embedding", Column: table.Vector(3)}, // 3-dimensional vectors for testing
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy: []string{"id"},
		},
	}

	_, err := db.CreateTable(ctx, vectorTableName, definition, options.CreateTable().SetIfNotExists(true))
	if err != nil {
		return fmt.Errorf("failed to create vector table: %w", err)
	}

	tbl := db.Table(vectorTableName)

	// Create a vector index
	indexName := "embedding_idx"
	err = tbl.CreateVectorIndex(ctx, indexName, "embedding",
		options.CreateVectorIndex().
			SetMetric(options.MetricCosine).
			SetIfNotExists(true))
	if err != nil {
		return fmt.Errorf("failed to create vector index: %w", err)
	}

	// Wait for index to be ready
	time.Sleep(2 * time.Second)

	// Insert test documents with embeddings
	docs := []TestDocument{
		{ID: "doc1", Content: "The quick brown fox", Embedding: []float32{1.0, 0.0, 0.0}},
		{ID: "doc2", Content: "Jumped over the lazy dog", Embedding: []float32{0.0, 1.0, 0.0}},
		{ID: "doc3", Content: "A quick brown dog", Embedding: []float32{0.9, 0.1, 0.0}}, // Similar to doc1
	}

	_, err = tbl.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	// Test vector similarity search - find documents similar to [1.0, 0.0, 0.0]
	queryVector := []float32{1.0, 0.0, 0.0}
	cursor := tbl.Find(filter.F{},
		options.TableFind().
			SetSort(sort.S{"embedding": queryVector}).
			SetIncludeSimilarity(true).
			SetLimit(3),
	)
	defer cursor.Close()

	var results []map[string]any
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("failed to execute vector search: %w", err)
	}

	if len(results) == 0 {
		return errors.New("expected to find documents in vector search")
	}

	// Verify similarity scores are present and ordered correctly
	// The first result should be most similar to [1.0, 0.0, 0.0]
	firstID, ok := results[0]["id"].(string)
	if !ok {
		return fmt.Errorf("expected id to be string, got %T", results[0]["id"])
	}
	// doc1 has exact match [1.0, 0.0, 0.0], should be first
	if firstID != "doc1" {
		return fmt.Errorf("expected first result to be 'doc1' (exact match), got %q", firstID)
	}

	// Verify similarity score is present
	if _, ok := results[0]["$similarity"]; !ok {
		return errors.New("expected $similarity field in results with includeSimilarity=true")
	}

	// Verify the index appears in ListIndexes with correct type
	indexes, err := tbl.ListIndexes(ctx, options.ListIndexes().SetExplain(true))
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	foundVectorIndex := false
	for _, idx := range indexes {
		if idx.Name == indexName {
			foundVectorIndex = true
			if idx.IndexType != "vector" {
				return fmt.Errorf("expected indexType 'vector', got %q", idx.IndexType)
			}
			if idx.Definition == nil {
				return errors.New("expected definition to be present")
			}
			if idx.Definition.Column != "embedding" {
				return fmt.Errorf("expected column 'embedding', got %q", idx.Definition.Column)
			}
			if idx.Definition.Options == nil || idx.Definition.Options.Metric != "cosine" {
				return fmt.Errorf("expected metric 'cosine' in index options")
			}
			break
		}
	}
	if !foundVectorIndex {
		return fmt.Errorf("expected to find vector index %q in list", indexName)
	}

	// Clean up - drop the index
	if err := db.DropTableIndex(ctx, indexName); err != nil {
		return fmt.Errorf("failed to drop vector index: %w", err)
	}

	// Clean up - drop the table
	if err := db.DropTable(ctx, vectorTableName); err != nil {
		return fmt.Errorf("failed to drop vector table: %w", err)
	}

	return nil
}

// TableAlter exercises alterTable end-to-end against a freshly-created
// throwaway table.
func TableAlter(e *harness.TestEnv) error {
	// TODO: add vectorize add/drop.
	ctx := context.Background()
	db := e.DefaultDb()

	// Use a self-contained table so this test doesn't depend on or mess
	// up other tests.
	const alterTableName = "go_test_alter_books"

	tbl, err := db.CreateTable(ctx, alterTableName, table.Definition{
		Columns: table.Columns{
			{"title", table.Text()},
			{"author", table.Text()},
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy: []string{"title"},
		},
	}, options.CreateTable().SetIfNotExists(true))
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	defer func() {
		// Clean up after our test. If we don't already have an error, return any
		// error from cleanup. Otherwise, let the test runner show the original error
		// because it probably has more useful diagnostic info.
		cleanupErr := db.DropTable(ctx, alterTableName)
		if cleanupErr != nil && err == nil {
			err = fmt.Errorf("failed to drop table: %w", cleanupErr)
		}
	}()

	// Add some columns!
	err = tbl.Alter(ctx, table.AddColumns{
		Columns: table.Columns{
			{"is_summer_reading", table.Boolean()},
			{"library_branch", table.Text()},
		},
	})
	if err != nil {
		return fmt.Errorf("Alter Add failed: %w", err)
	}

	// AlteredBook sounds a tad sinister. But I swear it's just a book with some
	// extra columns. Not a book altered to mislead anybody.
	type AlteredBook struct {
		Title           string `json:"title"`
		Author          string `json:"author"`
		IsSummerReading bool   `json:"is_summer_reading"`
		LibraryBranch   string `json:"library_branch"`
	}

	// Insert a row populating the newly-added columns and read it back to
	// prove they made it onto the schema.
	row := AlteredBook{
		Title:           "Summer Adventures",
		Author:          "Jane Doe",
		IsSummerReading: true,
		LibraryBranch:   "Downtown",
	}
	if _, err := tbl.InsertOne(ctx, row); err != nil {
		return fmt.Errorf("insert using added columns failed: %w", err)
	}

	var got AlteredBook
	if err := tbl.FindOne(ctx, filter.Eq("title", "Summer Adventures")).Decode(&got); err != nil {
		return fmt.Errorf("findOne after add failed: %w", err)
	}
	if !got.IsSummerReading {
		return fmt.Errorf("expected is_summer_reading=true, got %v", got.IsSummerReading)
	}
	if got.LibraryBranch != "Downtown" {
		return fmt.Errorf("expected library_branch=%q, got %v", "Downtown", got.LibraryBranch)
	}

	// Drop one of the new columns and verify the remaining row no longer
	// exposes it (the other added column must survive untouched).
	err = tbl.Alter(ctx, table.DropColumns{Columns: []string{"library_branch"}})
	if err != nil {
		return fmt.Errorf("Alter Drop failed: %w", err)
	}

	// Using a map here because we want to just inspect the raw fields to verify
	// dropped columns are not present.
	var afterDrop map[string]any
	if err := tbl.FindOne(ctx, filter.Eq("title", "Summer Adventures")).Decode(&afterDrop); err != nil {
		return fmt.Errorf("findOne after drop failed: %w", err)
	}
	if _, present := afterDrop["library_branch"]; present {
		return fmt.Errorf("expected library_branch to be missing after drop, but row was %v", afterDrop)
	}
	if v, ok := afterDrop["is_summer_reading"].(bool); !ok || !v {
		return fmt.Errorf("expected is_summer_reading to survive the drop, got %v", afterDrop["is_summer_reading"])
	}
	return nil
}

func TableDrop(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	return db.DropTable(ctx, tableName)
}
