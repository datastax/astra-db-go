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
	"log/slog"
	"reflect"
	"slices"
	"time"

	astradb "github.com/datastax/astra-db-go"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/sort"
	"github.com/datastax/astra-db-go/update"
)

func init() {
	// Register our tests
	t := []harness.IntegrationTest{
		{Name: "CollectionCreate", Run: CollectionCreate},
		{Name: "CollectionListCollections", Run: CollectionListCollections},
		{Name: "CollectionOptions", Run: CollectionOptions},
		{Name: "CollectionListCollectionNames", Run: CollectionListCollectionNames},
		{Name: "CollectionInsertMany", Run: CollectionInsertMany},
		{Name: "CollectionItemAlreadyExists", Run: CollectionItemAlreadyExists},
		{Name: "CollectionCount", Run: CollectionCount},
		{Name: "CollectionCountUpperBound", Run: CollectionCountUpperBound},
		{Name: "CollectionEstimatedCount", Run: CollectionEstimatedCount},
		{Name: "CollectionFind", Run: CollectionFind},
		{Name: "CollectionFindOne", Run: CollectionFindOne},
		{Name: "CollectionCursorPagination", Run: CollectionCursorPagination},
		{Name: "CollectionUpdateOne", Run: CollectionUpdateOne},
		{Name: "CollectionUpdateOneUpsert", Run: CollectionUpdateOneUpsert},
		{Name: "CollectionUpdateOneNoMatch", Run: CollectionUpdateOneNoMatch},
		{Name: "CollectionUpdateMany", Run: CollectionUpdateMany},
		{Name: "CollectionUpdateManyUpsert", Run: CollectionUpdateManyUpsert},
		{Name: "CollectionUpdateManyNoMatch", Run: CollectionUpdateManyNoMatch},
		{Name: "CollectionFindOneAndUpdate", Run: CollectionFindOneAndUpdate},
		{Name: "CollectionFindOneAndUpdateAfter", Run: CollectionFindOneAndUpdateAfter},
		{Name: "CollectionFindOneAndUpdateUpsert", Run: CollectionFindOneAndUpdateUpsert},
		{Name: "CollectionFindOneAndUpdateProjection", Run: CollectionFindOneAndUpdateProjection},
		{Name: "CollectionReplaceOne", Run: CollectionReplaceOne},
		{Name: "CollectionReplaceOneUpsert", Run: CollectionReplaceOneUpsert},
		{Name: "CollectionReplaceOneNoMatch", Run: CollectionReplaceOneNoMatch},
		{Name: "CollectionFindOneAndReplace", Run: CollectionFindOneAndReplace},
		{Name: "CollectionFindOneAndReplaceAfter", Run: CollectionFindOneAndReplaceAfter},
		{Name: "CollectionFindOneAndReplaceUpsert", Run: CollectionFindOneAndReplaceUpsert},
		{Name: "CollectionFindOneAndReplaceProjection", Run: CollectionFindOneAndReplaceProjection},
		{Name: "CollectionDeleteOne", Run: CollectionDeleteOne},
		{Name: "CollectionFindOneAndDelete", Run: CollectionFindOneAndDelete},
		{Name: "CollectionFindOneAndDeleteProjection", Run: CollectionFindOneAndDeleteProjection},
		{Name: "CollectionDrop", Run: CollectionDrop},
		// Vector search tests
		{Name: "CollectionVectorCreate", Run: CollectionVectorCreate},
		{Name: "CollectionVectorInsert", Run: CollectionVectorInsert},
		{Name: "CollectionVectorSearch", Run: CollectionVectorSearch},
		{Name: "CollectionVectorSearchWithSimilarity", Run: CollectionVectorSearchWithSimilarity},
		{Name: "CollectionFindWithSort", Run: CollectionFindWithSort},
		{Name: "CollectionFindWithProjection", Run: CollectionFindWithProjection},
		{Name: "CollectionFindWithLimit", Run: CollectionFindWithLimit},
		{Name: "CollectionFindWithSkip", Run: CollectionFindWithSkip},
		{Name: "CollectionFindCombined", Run: CollectionFindCombined},
		{Name: "CollectionVectorCollectionDrop", Run: CollectionVectorCollectionDrop},
	}
	harness.Register(t...)
}

const collectionName = "GoTest"

func CollectionCreate(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	_, err := db.CreateCollection(ctx, collectionName)
	return err
}

func CollectionListCollections(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	collections, err := db.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("ListCollections failed: %w", err)
	}

	var predicate = func(c results.CollectionDescriptor) bool {
		return c.Name == collectionName && c.Definition.DefaultId != nil && c.Definition.Vector == nil && c.Definition.Indexing == nil
	}

	if !slices.ContainsFunc(collections, predicate) {
		return fmt.Errorf("expected to find collection '%s' with simple definition in list", collectionName)
	}
	return nil
}

func CollectionListCollectionNames(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	names, err := db.ListCollectionNames(ctx)
	if err != nil {
		return fmt.Errorf("ListCollectionNames failed: %w", err)
	}

	if !slices.Contains(names, collectionName) {
		return fmt.Errorf("expected to find collection '%s' in names list", collectionName)
	}

	return nil
}

func CollectionOptions(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	collection := db.Collection(collectionName)

	c, err := collection.Options(ctx)
	if err != nil {
		return fmt.Errorf("collection.Options() failed: %w", err)
	}

	if c.Name != collectionName || c.Definition.DefaultId == nil || c.Definition.Vector != nil || c.Definition.Indexing != nil {
		return fmt.Errorf("collection options did not match expected values. Got: %+v", c)
	}
	return nil
}

func getSimpleObjects(rows int) []SimpleObject {
	data := make([]SimpleObject, rows)
	for i := 0; i < rows; i++ {
		name := fmt.Sprintf("Object #%v", i)
		data[i] = SimpleObject{
			Name: name,
			Properties: Properties{
				PropertyOne: fmt.Sprintf("I'm number %v! What about extended characters? ☠️き", i),
				PropertyTwo: `Bet you didn't see this newline coming....
	did you?`,
				IntProperty:         i,
				StringArrayProperty: []string{"Test1", "test2"},
				BoolProperty:        true,
				TimeProperty:        time.Now().AddDate(i, i, i),
				UTCTime:             time.Now().UTC(),
			},
		}
	}
	return data
}

func CollectionInsertMany(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	collection := db.Collection(collectionName)
	_, err := collection.InsertMany(ctx, getSimpleObjects(30))
	return err
}

func CollectionItemAlreadyExists(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)
	// Generate test data and insert a document
	item := getSimpleObjects(1)[0]
	resp, err := c.InsertOne(ctx, item)
	if err != nil {
		return err
	}
	// Now set inserted ID to existing one and insert again
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}
	item.ID = insertedID
	resp, err = c.InsertOne(ctx, item)
	if err == nil {
		return errors.New("expecting duplicate insert error. Got nil")
	}
	var errs *astradb.DataAPIErrors
	if errors.As(err, &errs) {
		expecting := "DOCUMENT_ALREADY_EXISTS"
		if len(*errs) != 1 {
			return fmt.Errorf("expecting len(errs) to = 1. Got %d", len(*errs))
		}
		apiError := (*errs)[0]
		if apiError.ErrorCode != expecting {
			return fmt.Errorf("expecting Code %v. got %v", expecting, apiError.ErrorCode)
		}
	} else {
		return fmt.Errorf("expecting error of type astradb.DataAPIError. Got %s", err)
	}
	return nil
}

func CollectionCount(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	collection := db.Collection(collectionName)
	count, err := collection.CountDocuments(ctx, filter.Gte("properties.intProperty", 13), 1000)
	if count == 0 {
		return errors.New("was expecting non-zero count")
	}
	return err
}

func CollectionCountUpperBound(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	collection := db.Collection(collectionName)
	_, err := collection.CountDocuments(ctx, nil, 1)
	if err != results.ErrTooManyDocumentsToCount {
		return fmt.Errorf("expecting err:%v. Got: %v", results.ErrTooManyDocumentsToCount, err)
	}
	return nil
}

func CollectionEstimatedCount(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	collection := db.Collection(collectionName)
	count, err := collection.EstimatedDocumentCount(ctx)
	if count < 0 {
		return errors.New("was expecting non-negative count")
	}
	return err
}

func CollectionFindOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)
	// Generate test data and insert a document
	original := getSimpleObjects(1)[0]
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return err
	}
	// Get inserted ID and use it to then find our newly-inserted record
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}
	var document SimpleObject
	err = c.FindOne(ctx, filter.F{"_id": insertedID}).Decode(&document)
	if err != nil {
		return err
	}
	// Before deep equal, set ID on original doc so deep equal succeeds
	original.ID = insertedID
	if !reflect.DeepEqual(original, document) {
		slog.Debug("Original", "struct", original)
		slog.Debug("Returned document", "struct", document)
		return errors.New("original != what was selected from DB")
	}
	return nil
}

func CollectionFind(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Use cursor to find documents
	cursor, _ := c.Find(filter.Gte("properties.intProperty", 20))
	defer cursor.Close()

	var documents []SimpleObject
	if err := cursor.DecodeAll(ctx, &documents); err != nil {
		return err
	}

	if len(documents) == 0 {
		return errors.New("expected to find documents with intProperty >= 20")
	}

	return nil
}

// CollectionCursorPagination tests server-side cursor pagination by inserting
// enough documents to span multiple pages and iterating through them all.
//
// The Data API typically returns ~20 documents per page, so we insert 50+
// documents to ensure we get multiple pages.
func CollectionCursorPagination(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	// Create a dedicated collection for pagination test
	paginationCollectionName := "GoTestPagination"
	_, err := db.CreateCollection(ctx, paginationCollectionName)
	if err != nil {
		return fmt.Errorf("failed to create pagination test collection: %w", err)
	}

	// Clean up at the end
	defer func() {
		db.DropCollection(ctx, paginationCollectionName)
	}()

	c := db.Collection(paginationCollectionName)

	// Insert 50 documents
	const totalDocs = 50
	docs := make([]map[string]any, totalDocs)
	for i := 0; i < totalDocs; i++ {
		docs[i] = map[string]any{
			"index":     i,
			"batchId":   "pagination-test",
			"name":      fmt.Sprintf("Document %d", i),
			"timestamp": time.Now().UnixNano(),
		}
	}

	// Insert in batches
	batchSize := 20
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		batch := docs[i:end]
		_, err := c.InsertMany(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to insert batch starting at %d: %w", i, err)
		}
	}

	// Now use the cursor to iterate through ALL documents
	cursor, _ := c.Find(filter.Eq("batchId", "pagination-test"))
	defer cursor.Close()

	// Track pagination stats
	var fetchedDocs []map[string]any
	pagesFetched := 0
	docsInCurrentPage := 0

	for cursor.Next(ctx) {
		var doc map[string]any
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("failed to decode document: %w", err)
		}
		fetchedDocs = append(fetchedDocs, doc)
		docsInCurrentPage++

		// Check if we just finished a page (remaining batch length is 0 and there's a next page)
		if cursor.Buffered() == 0 {
			slog.Info("Fetched page",
				"pageNumber", pagesFetched+1,
				"docsInPage", docsInCurrentPage,
				"totalDocsSoFar", len(fetchedDocs),
			)
			pagesFetched++
			docsInCurrentPage = 0
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	// Verify we got all documents
	if len(fetchedDocs) != totalDocs {
		return fmt.Errorf("expected %d documents, got %d", totalDocs, len(fetchedDocs))
	}

	// Verify we actually had to paginate (more than 1 page)
	if pagesFetched < 2 {
		slog.Warn("Pagination test may not have tested multiple pages",
			"pagesFetched", pagesFetched,
			"totalDocs", totalDocs,
		)
	}

	// Verify document indices are all present (no duplicates or missing)
	seen := make(map[int]bool)
	for _, doc := range fetchedDocs {
		idx, ok := doc["index"].(float64) // JSON numbers are float64
		if !ok {
			return fmt.Errorf("document missing or invalid index field: %v", doc)
		}
		intIdx := int(idx)
		if seen[intIdx] {
			return fmt.Errorf("duplicate document with index %d", intIdx)
		}
		seen[intIdx] = true
	}

	if len(seen) != totalDocs {
		return fmt.Errorf("expected %d unique indices, got %d", totalDocs, len(seen))
	}

	slog.Info("Cursor pagination test completed successfully",
		"totalDocuments", len(fetchedDocs),
		"pagesFetched", pagesFetched,
	)

	return nil
}

func CollectionUpdateOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document to update
	original := SimpleObject{Name: "UpdateOneOriginal"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// Update the document's name
	result, err := c.UpdateOne(ctx, filter.F{"_id": insertedID}, update.Coll().Set("name", "UpdateOneModified"))
	if err != nil {
		return fmt.Errorf("UpdateOne failed: %w", err)
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("expected MatchedCount 1, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 1 {
		return fmt.Errorf("expected ModifiedCount 1, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 0 {
		return fmt.Errorf("expected UpsertedCount 0, got %d", result.UpsertedCount)
	}

	// Verify the update by reading back
	var doc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": insertedID}).Decode(&doc); err != nil {
		return fmt.Errorf("FindOne after update failed: %w", err)
	}
	if doc.Name != "UpdateOneModified" {
		return fmt.Errorf("expected name 'UpdateOneModified', got '%s'", doc.Name)
	}

	return nil
}

func CollectionUpdateOneUpsert(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Adding timestamp so this test survives retries. At some point, it might be good
	// to either make ALL tests survive retries, or add a "cleanup" flag to tests that
	// indicate they should run even if other tests fail. Or maybe it is more like we
	// make these grouped tests "suites" and we have setup and teardown functions for
	// suites.
	upsertID := fmt.Sprintf("upsert-test-id-%d", time.Now().Unix())

	// Upsert a document that doesn't exist
	result, err := c.UpdateOne(ctx,
		filter.F{"_id": upsertID},
		update.Coll().Set("name", "UpsertedDoc"),
		options.CollectionUpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("UpdateOne upsert failed: %w", err)
	}

	if result.MatchedCount != 0 {
		return fmt.Errorf("expected MatchedCount 0, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 0 {
		return fmt.Errorf("expected ModifiedCount 0, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 1 {
		return fmt.Errorf("expected UpsertedCount 1, got %d", result.UpsertedCount)
	}
	if result.UpsertedId == nil {
		return errors.New("expected UpsertedId to be non-nil")
	}

	// Verify the upserted document
	var doc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": upsertID}).Decode(&doc); err != nil {
		return fmt.Errorf("FindOne after upsert failed: %w", err)
	}
	if doc.Name != "UpsertedDoc" {
		return fmt.Errorf("expected name 'UpsertedDoc', got '%s'", doc.Name)
	}

	return nil
}

func CollectionUpdateOneNoMatch(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	result, err := c.UpdateOne(ctx,
		filter.F{"_id": "nonexistent-id-xyz"},
		update.Coll().Set("name", "ShouldNotExist"),
	)
	if err != nil {
		return fmt.Errorf("UpdateOne no-match failed: %w", err)
	}

	if result.MatchedCount != 0 {
		return fmt.Errorf("expected MatchedCount 0, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 0 {
		return fmt.Errorf("expected ModifiedCount 0, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 0 {
		return fmt.Errorf("expected UpsertedCount 0, got %d", result.UpsertedCount)
	}

	return nil
}

func CollectionUpdateMany(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert several documents with a shared tag for targeting
	tag := fmt.Sprintf("update-many-%d", time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		_, err := c.InsertOne(ctx, map[string]any{
			"tag":   tag,
			"index": i,
			"name":  fmt.Sprintf("doc-%d", i),
		})
		if err != nil {
			return fmt.Errorf("failed to insert doc %d: %w", i, err)
		}
	}

	// UpdateMany: set a new field on all docs with our tag
	result, err := c.UpdateMany(ctx,
		filter.F{"tag": tag},
		update.Coll().Set("updated", true),
	)
	if err != nil {
		return fmt.Errorf("UpdateMany failed: %w", err)
	}

	if result.MatchedCount != 5 {
		return fmt.Errorf("expected MatchedCount 5, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 5 {
		return fmt.Errorf("expected ModifiedCount 5, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 0 {
		return fmt.Errorf("expected UpsertedCount 0, got %d", result.UpsertedCount)
	}

	// Verify the update by reading back
	cur, _ := c.Find(filter.F{"tag": tag})
	defer cur.Close()
	var docs []map[string]any
	if err := cur.DecodeAll(ctx, &docs); err != nil {
		return fmt.Errorf("Find after UpdateMany failed: %w", err)
	}
	for i, doc := range docs {
		if doc["updated"] != true {
			return fmt.Errorf("doc %d not updated", i)
		}
	}

	return nil
}

func CollectionUpdateManyUpsert(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	upsertTag := fmt.Sprintf("upsert-many-%d", time.Now().UnixNano())

	result, err := c.UpdateMany(ctx,
		filter.F{"tag": upsertTag},
		update.Coll().Set("name", "UpsertedMany"),
		options.CollectionUpdateMany().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("UpdateMany upsert failed: %w", err)
	}

	if result.MatchedCount != 0 {
		return fmt.Errorf("expected MatchedCount 0, got %d", result.MatchedCount)
	}
	if result.UpsertedCount != 1 {
		return fmt.Errorf("expected UpsertedCount 1, got %d", result.UpsertedCount)
	}
	if result.UpsertedId == nil {
		return errors.New("expected UpsertedId to be non-nil")
	}

	// Verify the upserted document exists
	var doc map[string]any
	if err := c.FindOne(ctx, filter.F{"tag": upsertTag}).Decode(&doc); err != nil {
		return fmt.Errorf("FindOne after upsert failed: %w", err)
	}
	if doc["name"] != "UpsertedMany" {
		return fmt.Errorf("expected name 'UpsertedMany', got '%s'", doc["name"])
	}

	return nil
}

func CollectionUpdateManyNoMatch(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	result, err := c.UpdateMany(ctx,
		filter.F{"_id": "nonexistent-update-many-xyz"},
		update.Coll().Set("name", "ShouldNotExist"),
	)
	if err != nil {
		return fmt.Errorf("UpdateMany no-match failed: %w", err)
	}

	if result.MatchedCount != 0 {
		return fmt.Errorf("expected MatchedCount 0, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 0 {
		return fmt.Errorf("expected ModifiedCount 0, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 0 {
		return fmt.Errorf("expected UpsertedCount 0, got %d", result.UpsertedCount)
	}

	return nil
}

func CollectionFindOneAndUpdate(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document to update
	original := SimpleObject{Name: "FindOneAndUpdateOriginal"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndUpdate — default returnDocument is "before"
	var doc SimpleObject
	err = c.FindOneAndUpdate(ctx,
		filter.F{"_id": insertedID},
		update.Coll().Set("name", "FindOneAndUpdateModified"),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndUpdate failed: %w", err)
	}

	// Should return the document BEFORE the update
	if doc.Name != "FindOneAndUpdateOriginal" {
		return fmt.Errorf("expected name 'FindOneAndUpdateOriginal' (before update), got '%s'", doc.Name)
	}

	// Verify the DB has the updated value
	var dbDoc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": insertedID}).Decode(&dbDoc); err != nil {
		return fmt.Errorf("FindOne after FindOneAndUpdate failed: %w", err)
	}
	if dbDoc.Name != "FindOneAndUpdateModified" {
		return fmt.Errorf("expected DB name 'FindOneAndUpdateModified', got '%s'", dbDoc.Name)
	}

	return nil
}

func CollectionFindOneAndUpdateAfter(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document
	original := SimpleObject{Name: "BeforeAfterTest"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndUpdate with ReturnDocumentAfter
	var doc SimpleObject
	err = c.FindOneAndUpdate(ctx,
		filter.F{"_id": insertedID},
		update.Coll().Set("name", "AfterValue"),
		options.CollectionFindOneAndUpdate().SetReturnDocument(options.ReturnDocumentAfter),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndUpdate with returnDocument=after failed: %w", err)
	}

	// Should return the document AFTER the update
	if doc.Name != "AfterValue" {
		return fmt.Errorf("expected name 'AfterValue' (after update), got '%s'", doc.Name)
	}

	return nil
}

func CollectionFindOneAndUpdateUpsert(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	upsertID := fmt.Sprintf("find-and-update-upsert-%d", time.Now().Unix())

	// FindOneAndUpdate with upsert on a non-existent document, return after
	var doc map[string]any
	err := c.FindOneAndUpdate(ctx,
		filter.F{"_id": upsertID},
		update.Coll().Set("name", "UpsertedViaFindOneAndUpdate"),
		options.CollectionFindOneAndUpdate().
			SetUpsert(true).
			SetReturnDocument(options.ReturnDocumentAfter),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndUpdate upsert failed: %w", err)
	}

	if doc["name"] != "UpsertedViaFindOneAndUpdate" {
		return fmt.Errorf("expected name 'UpsertedViaFindOneAndUpdate', got '%v'", doc["name"])
	}

	// Verify document exists in DB
	var dbDoc map[string]any
	if err := c.FindOne(ctx, filter.F{"_id": upsertID}).Decode(&dbDoc); err != nil {
		return fmt.Errorf("FindOne after upsert failed: %w", err)
	}
	if dbDoc["name"] != "UpsertedViaFindOneAndUpdate" {
		return fmt.Errorf("expected DB name 'UpsertedViaFindOneAndUpdate', got '%v'", dbDoc["name"])
	}

	return nil
}

func CollectionFindOneAndUpdateProjection(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document with multiple fields
	original := SimpleObject{Name: "ProjectionTest", Properties: Properties{
		PropertyOne: "val1",
		IntProperty: 42,
	}}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndUpdate with projection — only include "name"
	var doc map[string]any
	err = c.FindOneAndUpdate(ctx,
		filter.F{"_id": insertedID},
		update.Coll().Set("name", "ProjectionTestUpdated"),
		options.CollectionFindOneAndUpdate().
			SetReturnDocument(options.ReturnDocumentAfter).
			SetProjection(map[string]any{"name": true}),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndUpdate with projection failed: %w", err)
	}

	// Should have _id and name, but NOT properties
	if _, ok := doc["_id"]; !ok {
		return errors.New("expected _id in projected result")
	}
	if doc["name"] != "ProjectionTestUpdated" {
		return fmt.Errorf("expected name 'ProjectionTestUpdated', got '%v'", doc["name"])
	}
	if _, ok := doc["properties"]; ok {
		return errors.New("expected properties to be excluded by projection")
	}

	return nil
}

func CollectionReplaceOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document to update
	original := SimpleObject{Name: "ReplaceOneOriginal"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// Replace the document
	result, err := c.ReplaceOne(ctx, filter.F{"_id": insertedID}, SimpleObject{Name: "ReplaceOneNew"})
	if err != nil {
		return fmt.Errorf("ReplaceOne failed: %w", err)
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("expected MatchedCount 1, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 1 {
		return fmt.Errorf("expected ModifiedCount 1, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 0 {
		return fmt.Errorf("expected UpsertedCount 0, got %d", result.UpsertedCount)
	}

	// Verify the replacement by reading back
	var doc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": insertedID}).Decode(&doc); err != nil {
		return fmt.Errorf("FindOne after replace failed: %w", err)
	}
	if doc.Name != "ReplaceOneNew" {
		return fmt.Errorf("expected name 'ReplaceOneNew', got '%s'", doc.Name)
	}
	if doc.ID != insertedID {
		return fmt.Errorf("expected ID '%v' to remain unchanged, got '%v'", insertedID, doc.ID)
	}

	return nil
}

func CollectionReplaceOneUpsert(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Adding timestamp so this test survives retries. At some point, it might be good
	// to either make ALL tests survive retries, or add a "cleanup" flag to tests that
	// indicate they should run even if other tests fail. Or maybe it is more like we
	// make these grouped tests "suites" and we have setup and teardown functions for
	// suites.
	upsertID := fmt.Sprintf("replace-upsert-id-%d", time.Now().Unix())

	// Replace a document that doesn't exist
	result, err := c.ReplaceOne(ctx,
		filter.F{"_id": upsertID},
		SimpleObject{ID: upsertID, Name: "UpsertedDoc"},
		options.CollectionReplaceOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("ReplaceOne upsert failed: %w", err)
	}

	if result.MatchedCount != 0 {
		return fmt.Errorf("expected MatchedCount 0, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 0 {
		return fmt.Errorf("expected ModifiedCount 0, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 1 {
		return fmt.Errorf("expected UpsertedCount 1, got %d", result.UpsertedCount)
	}
	if result.UpsertedId == nil {
		return errors.New("expected UpsertedId to be non-nil")
	}

	// Verify the document was created correctly
	var doc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": upsertID}).Decode(&doc); err != nil {
		return fmt.Errorf("FindOne after ReplaceOne upsert failed: %w", err)
	}
	if doc.Name != "UpsertedDoc" {
		return fmt.Errorf("expected name 'UpsertedDoc', got '%s'", doc.Name)
	}

	return nil
}

func CollectionReplaceOneNoMatch(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	result, err := c.ReplaceOne(ctx,
		filter.F{"_id": "nonexistent-replace-id"},
		SimpleObject{Name: "ShouldNotBeCreated"},
	)
	if err != nil {
		return fmt.Errorf("ReplaceOne no-match failed: %w", err)
	}

	if result.MatchedCount != 0 {
		return fmt.Errorf("expected MatchedCount 0, got %d", result.MatchedCount)
	}
	if result.ModifiedCount != 0 {
		return fmt.Errorf("expected ModifiedCount 0, got %d", result.ModifiedCount)
	}
	if result.UpsertedCount != 0 {
		return fmt.Errorf("expected UpsertedCount 0, got %d", result.UpsertedCount)
	}

	return nil
}

func CollectionFindOneAndReplace(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document to update
	original := SimpleObject{Name: "FindOneAndReplaceOriginal"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndReplace — default returnDocument is "before"
	var doc SimpleObject
	err = c.FindOneAndReplace(ctx,
		filter.F{"_id": insertedID},
		SimpleObject{Name: "FindOneAndReplaceModified"},
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndReplace failed: %w", err)
	}

	// Should return the document BEFORE the update
	if doc.Name != "FindOneAndReplaceOriginal" {
		return fmt.Errorf("expected name 'FindOneAndReplaceOriginal' (before update), got '%s'", doc.Name)
	}

	// Verify the DB has the updated value
	var dbDoc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": insertedID}).Decode(&dbDoc); err != nil {
		return fmt.Errorf("FindOne after FindOneAndReplace failed: %w", err)
	}
	if dbDoc.Name != "FindOneAndReplaceModified" {
		return fmt.Errorf("expected DB name 'FindOneAndReplaceModified', got '%s'", dbDoc.Name)
	}

	return nil
}

func CollectionFindOneAndReplaceAfter(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document
	original := SimpleObject{Name: "BeforeAfterReplaceTest"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndReplace with ReturnDocumentAfter
	var doc SimpleObject
	err = c.FindOneAndReplace(ctx,
		filter.F{"_id": insertedID},
		SimpleObject{Name: "AfterValueReplace"},
		options.CollectionFindOneAndReplace().SetReturnDocument(options.ReturnDocumentAfter),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndReplace with returnDocument=after failed: %w", err)
	}

	// Should return the document AFTER the update
	if doc.Name != "AfterValueReplace" {
		return fmt.Errorf("expected name 'AfterValueReplace' (after update), got '%s'", doc.Name)
	}

	return nil
}

func CollectionFindOneAndReplaceUpsert(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	upsertID := fmt.Sprintf("find-and-replace-upsert-%d", time.Now().Unix())

	// FindOneAndReplace with upsert on a non-existent document, return after
	var doc map[string]any
	err := c.FindOneAndReplace(ctx,
		filter.F{"_id": upsertID},
		SimpleObject{Name: "UpsertedViaFindOneAndReplace"},
		options.CollectionFindOneAndReplace().
			SetUpsert(true).
			SetReturnDocument(options.ReturnDocumentAfter),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndReplace upsert failed: %w", err)
	}

	if doc["name"] != "UpsertedViaFindOneAndReplace" {
		return fmt.Errorf("expected name 'UpsertedViaFindOneAndReplace', got '%v'", doc["name"])
	}

	// Verify document exists in DB
	var dbDoc map[string]any
	if err := c.FindOne(ctx, filter.F{"_id": upsertID}).Decode(&dbDoc); err != nil {
		return fmt.Errorf("FindOne after upsert failed: %w", err)
	}
	if dbDoc["name"] != "UpsertedViaFindOneAndReplace" {
		return fmt.Errorf("expected DB name 'UpsertedViaFindOneAndReplace', got '%v'", dbDoc["name"])
	}

	return nil
}

func CollectionFindOneAndReplaceProjection(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document with multiple fields
	original := SimpleObject{Name: "ReplaceProjectionTest", Properties: Properties{
		PropertyOne: "val1",
		IntProperty: 42,
	}}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndReplace with projection — only include "name"
	var doc map[string]any
	err = c.FindOneAndReplace(ctx,
		filter.F{"_id": insertedID},
		SimpleObject{Name: "ReplaceProjectionUpdated"},
		options.CollectionFindOneAndReplace().
			SetReturnDocument(options.ReturnDocumentAfter).
			SetProjection(map[string]any{"name": true}),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndReplace with projection failed: %w", err)
	}

	// Should have _id and name, but NOT properties
	if _, ok := doc["_id"]; !ok {
		return errors.New("expected _id in projected result")
	}
	if doc["name"] != "ReplaceProjectionUpdated" {
		return fmt.Errorf("expected name 'ReplaceProjectionUpdated', got '%v'", doc["name"])
	}
	if _, ok := doc["properties"]; ok {
		return errors.New("expected properties to be excluded by projection")
	}

	return nil
}

func CollectionDeleteOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document to delete
	original := SimpleObject{Name: "DeleteOneTest"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// Delete the document
	result, err := c.DeleteOne(ctx, filter.F{"_id": insertedID})
	if err != nil {
		return fmt.Errorf("DeleteOne failed: %w", err)
	}
	if result.DeletedCount != 1 {
		return fmt.Errorf("expected DeletedCount 1, got %d", result.DeletedCount)
	}
	return nil
}

func CollectionFindOneAndDelete(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document to update
	original := SimpleObject{Name: "FindOneAndDeleteOriginal"}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndDelete — returnDocument is always "before"
	var doc SimpleObject
	err = c.FindOneAndDelete(ctx,
		filter.F{"_id": insertedID},
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndDelete failed: %w", err)
	}

	// Should return the document BEFORE the update
	if doc.Name != "FindOneAndDeleteOriginal" {
		return fmt.Errorf("expected name 'FindOneAndDeleteOriginal' (before update), got '%s'", doc.Name)
	}

	// Verify the DB does not have the updated value
	var dbDoc SimpleObject
	if err := c.FindOne(ctx, filter.F{"_id": insertedID}).Decode(&dbDoc); !errors.Is(err, results.ErrNoDocuments) {
		return fmt.Errorf("FindOne after FindOneAndDelete did not return ErrNoDocuments: %w", err)
	}
	return nil
}

func CollectionFindOneAndDeleteProjection(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(collectionName)

	// Insert a document with multiple fields
	original := SimpleObject{Name: "ProjectionTest", Properties: Properties{
		PropertyOne: "val1",
		IntProperty: 42,
	}}
	resp, err := c.InsertOne(ctx, original)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}

	// FindOneAndDelete with projection — only include "name"
	var doc map[string]any
	err = c.FindOneAndDelete(ctx,
		filter.F{"_id": insertedID},
		options.CollectionFindOneAndDelete().SetProjection(map[string]any{"name": true}),
	).Decode(&doc)
	if err != nil {
		return fmt.Errorf("FindOneAndDelete with projection failed: %w", err)
	}

	// Should have _id and name, but NOT properties
	if _, ok := doc["_id"]; !ok {
		return errors.New("expected _id in projected result")
	}
	if doc["name"] != "ProjectionTest" {
		return fmt.Errorf("expected name 'ProjectionTest', got '%v'", doc["name"])
	}
	if _, ok := doc["properties"]; ok {
		return errors.New("expected properties to be excluded by projection")
	}

	return nil
}

func CollectionDrop(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	err := db.Collection(collectionName).Drop(ctx)
	return err
}

// #region Vector Search Integration Tests
// Based on AstraPy examples from:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html

const vectorCollectionName = "GoTestVector"
const vectorDimension = 3

// VectorDocument represents a document with vector embeddings
type VectorDocument struct {
	ID            string    `json:"_id,omitempty"`
	Title         string    `json:"title"`
	Rating        int       `json:"rating"`
	IsCheckedOut  bool      `json:"is_checked_out"`
	NumberOfPages int       `json:"number_of_pages"`
	Vector        []float32 `json:"$vector,omitempty"`
	Metadata      Metadata  `json:"metadata"`
}

type Metadata struct {
	Language string `json:"language"`
	Genre    string `json:"genre"`
}

// CollectionVectorCreate creates a vector-enabled collection for testing
func CollectionVectorCreate(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	// Create a collection with vector support
	_, err := db.CreateCollection(ctx, vectorCollectionName,
		options.CreateCollection().SetVector(&options.VectorOptions{
			Dimension: ptr.To(vectorDimension),
			Metric:    ptr.To("cosine"),
		}))
	if err != nil {
		return fmt.Errorf("failed to create vector collection: %w", err)
	}

	slog.Info("Created vector-enabled collection", "name", vectorCollectionName, "dimension", vectorDimension)
	return nil
}

// CollectionVectorInsert inserts test documents with vector embeddings
func CollectionVectorInsert(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Sample documents with 3-dimensional vectors
	// These vectors are designed to have different similarities for testing
	docs := []VectorDocument{
		{
			Title:         "The Great Gatsby",
			Rating:        5,
			IsCheckedOut:  false,
			NumberOfPages: 180,
			Vector:        []float32{0.1, 0.2, 0.3},
			Metadata:      Metadata{Language: "English", Genre: "Fiction"},
		},
		{
			Title:         "To Kill a Mockingbird",
			Rating:        5,
			IsCheckedOut:  true,
			NumberOfPages: 281,
			Vector:        []float32{0.15, 0.25, 0.35}, // Similar to first
			Metadata:      Metadata{Language: "English", Genre: "Fiction"},
		},
		{
			Title:         "1984",
			Rating:        4,
			IsCheckedOut:  false,
			NumberOfPages: 328,
			Vector:        []float32{0.9, 0.1, 0.05}, // Different direction
			Metadata:      Metadata{Language: "English", Genre: "Dystopian"},
		},
		{
			Title:         "Pride and Prejudice",
			Rating:        4,
			IsCheckedOut:  false,
			NumberOfPages: 279,
			Vector:        []float32{0.12, 0.22, 0.32}, // Very similar to first
			Metadata:      Metadata{Language: "English", Genre: "Romance"},
		},
		{
			Title:         "The Catcher in the Rye",
			Rating:        3,
			IsCheckedOut:  true,
			NumberOfPages: 234,
			Vector:        []float32{0.5, 0.5, 0.5}, // Middle ground
			Metadata:      Metadata{Language: "English", Genre: "Fiction"},
		},
		{
			Title:         "Don Quixote",
			Rating:        5,
			IsCheckedOut:  false,
			NumberOfPages: 863,
			Vector:        []float32{0.8, 0.2, 0.1}, // Different
			Metadata:      Metadata{Language: "Spanish", Genre: "Adventure"},
		},
	}

	_, err := c.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to insert vector documents: %w", err)
	}

	slog.Info("Inserted vector documents", "count", len(docs))
	return nil
}

// CollectionVectorSearch tests basic vector similarity search
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#example-vector
func CollectionVectorSearch(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Search for documents similar to [0.1, 0.2, 0.3]
	// This should return "The Great Gatsby" first as it has the exact vector
	searchVector := []float32{0.1, 0.2, 0.3}

	cursor, _ := c.Find(filter.F{},
		options.CollectionFind().
			SetSort(sort.Vector(searchVector)).
			SetLimit(3),
	)
	defer cursor.Close()

	var results []VectorDocument
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("vector search failed: %w", err)
	}

	if len(results) == 0 {
		return errors.New("vector search returned no results")
	}

	// The first result should be "The Great Gatsby" (exact match)
	if results[0].Title != "The Great Gatsby" {
		return fmt.Errorf("expected first result to be 'The Great Gatsby', got '%s'", results[0].Title)
	}

	slog.Info("Vector search results",
		"count", len(results),
		"first", results[0].Title,
	)

	return nil
}

// CollectionVectorSearchWithSimilarity tests vector search with similarity scores
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#example-similarity
func CollectionVectorSearchWithSimilarity(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Search with similarity score included
	searchVector := []float32{0.1, 0.2, 0.3}

	cursor, _ := c.Find(filter.F{},
		options.CollectionFind().
			SetSort(sort.Vector(searchVector)).
			SetIncludeSimilarity(true).
			SetLimit(3),
	)
	defer cursor.Close()

	// Use map to capture $similarity field
	var results []map[string]any
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("vector search with similarity failed: %w", err)
	}

	if len(results) == 0 {
		return errors.New("vector search returned no results")
	}

	// Check that similarity scores are present
	for i, doc := range results {
		similarity, ok := doc["$similarity"]
		if !ok {
			return fmt.Errorf("document %d missing $similarity field", i)
		}

		simFloat, ok := similarity.(float64)
		if !ok {
			return fmt.Errorf("$similarity is not a number: %T", similarity)
		}

		// Similarity should be between 0 and 1 for cosine
		if simFloat < 0 || simFloat > 1.0001 { // small epsilon for floating point
			return fmt.Errorf("$similarity out of range [0,1]: %f", simFloat)
		}

		slog.Info("Document with similarity",
			"title", doc["title"],
			"similarity", simFloat,
		)
	}

	// First result should have highest similarity (close to 1.0 for exact match)
	firstSimilarity := results[0]["$similarity"].(float64)
	if firstSimilarity < 0.99 {
		return fmt.Errorf("expected first result to have similarity close to 1.0, got %f", firstSimilarity)
	}

	return nil
}

// CollectionFindWithSort tests ascending/descending sort
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#example-sort
func CollectionFindWithSort(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Sort by rating ascending, then title descending
	cursor, _ := c.Find(filter.Eq("metadata.language", "English"),
		options.CollectionFind().SetSort(sort.Asc("rating").Desc("title")),
	)
	defer cursor.Close()

	var results []VectorDocument
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("sorted find failed: %w", err)
	}

	if len(results) < 2 {
		return fmt.Errorf("expected at least 2 results, got %d", len(results))
	}

	// Verify sorting: ratings should be ascending
	for i := 1; i < len(results); i++ {
		if results[i].Rating < results[i-1].Rating {
			return fmt.Errorf("results not sorted by rating ascending: %d < %d at index %d",
				results[i].Rating, results[i-1].Rating, i)
		}
		// If ratings are equal, titles should be descending
		if results[i].Rating == results[i-1].Rating {
			if results[i].Title > results[i-1].Title {
				return fmt.Errorf("results not sorted by title descending when rating equal")
			}
		}
	}

	slog.Info("Sorted find results", "count", len(results), "first", results[0].Title)
	return nil
}

// CollectionFindWithProjection tests field projection
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#example-include
func CollectionFindWithProjection(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Only include title and is_checked_out fields
	cursor, _ := c.Find(filter.Eq("metadata.language", "English"),
		options.CollectionFind().SetProjection(map[string]any{
			"title":          true,
			"is_checked_out": true,
		}),
	)
	defer cursor.Close()

	var results []map[string]any
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("projected find failed: %w", err)
	}

	if len(results) == 0 {
		return errors.New("projection find returned no results")
	}

	// Verify projection: should have _id, title, is_checked_out but NOT rating, number_of_pages, etc.
	for i, doc := range results {
		// _id is always included unless explicitly excluded
		if _, ok := doc["_id"]; !ok {
			return fmt.Errorf("document %d missing _id field", i)
		}
		if _, ok := doc["title"]; !ok {
			return fmt.Errorf("document %d missing title field", i)
		}
		if _, ok := doc["is_checked_out"]; !ok {
			return fmt.Errorf("document %d missing is_checked_out field", i)
		}
		// These should NOT be present
		if _, ok := doc["rating"]; ok {
			return fmt.Errorf("document %d should not have rating field", i)
		}
		if _, ok := doc["number_of_pages"]; ok {
			return fmt.Errorf("document %d should not have number_of_pages field", i)
		}
	}

	slog.Info("Projection find results", "count", len(results), "fields", "title,is_checked_out")
	return nil
}

// CollectionFindWithLimit tests limiting results
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#example-limit
func CollectionFindWithLimit(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	limit := 2
	cursor, _ := c.Find(filter.Eq("metadata.language", "English"),
		options.CollectionFind().SetLimit(limit),
	)
	defer cursor.Close()

	var results []VectorDocument
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("limited find failed: %w", err)
	}

	if len(results) != limit {
		return fmt.Errorf("expected %d results, got %d", limit, len(results))
	}

	slog.Info("Limited find results", "limit", limit, "count", len(results))
	return nil
}

// CollectionFindWithSkip tests skipping documents
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#example-skip
func CollectionFindWithSkip(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Skip requires an explicit sort criterion
	// First, get all results sorted by rating
	cursorAll, _ := c.Find(filter.Eq("metadata.language", "English"),
		options.CollectionFind().SetSort(sort.Asc("rating").Asc("title")),
	)
	var allResults []VectorDocument
	if err := cursorAll.DecodeAll(ctx, &allResults); err != nil {
		return fmt.Errorf("failed to get all results: %w", err)
	}
	cursorAll.Close()

	if len(allResults) < 3 {
		return fmt.Errorf("need at least 3 documents for skip test, got %d", len(allResults))
	}

	// Now get results with skip=2
	skip := 2
	cursorSkip, _ := c.Find(filter.Eq("metadata.language", "English"),
		options.CollectionFind().
			SetSort(sort.Asc("rating").Asc("title")).
			SetSkip(skip),
	)
	defer cursorSkip.Close()

	var skipResults []VectorDocument
	if err := cursorSkip.DecodeAll(ctx, &skipResults); err != nil {
		return fmt.Errorf("skip find failed: %w", err)
	}

	// Verify that skipResults starts from index 2 of allResults
	expectedCount := len(allResults) - skip
	if len(skipResults) != expectedCount {
		return fmt.Errorf("expected %d results after skip, got %d", expectedCount, len(skipResults))
	}

	// First result of skipResults should match third result of allResults
	if skipResults[0].Title != allResults[skip].Title {
		return fmt.Errorf("skip results don't match: expected '%s', got '%s'",
			allResults[skip].Title, skipResults[0].Title)
	}

	slog.Info("Skip find results", "skip", skip, "totalDocs", len(allResults), "returned", len(skipResults))
	return nil
}

// CollectionFindCombined tests filter, sort, projection, and limit together
// Based on: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/find-many.html#use-filter-sort-and-projection-together
func CollectionFindCombined(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(vectorCollectionName)

	// Find English books with less than 300 pages
	// Sort by rating ascending, title descending
	// Only include title and is_checked_out
	// Limit to 3 results
	cursor, _ := c.Find(
		filter.And(
			filter.Eq("is_checked_out", false),
			filter.Lt("number_of_pages", 300),
		),
		options.CollectionFind().
			SetSort(sort.Asc("rating").Desc("title")).
			SetProjection(map[string]any{
				"title":          true,
				"is_checked_out": true,
			}).
			SetLimit(3),
	)
	defer cursor.Close()

	var results []map[string]any
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("combined find failed: %w", err)
	}

	slog.Info("Combined find results", "count", len(results))

	// Verify results
	for i, doc := range results {
		// Should have limited fields
		if _, ok := doc["rating"]; ok {
			return fmt.Errorf("document %d should not have rating (projection)", i)
		}
		if _, ok := doc["title"]; !ok {
			return fmt.Errorf("document %d missing title field", i)
		}
		slog.Info("Combined result", "index", i, "title", doc["title"])
	}

	// Should be at most 3 results
	if len(results) > 3 {
		return fmt.Errorf("expected at most 3 results, got %d", len(results))
	}

	return nil
}

// CollectionVectorCollectionDrop cleans up the vector test collection
func CollectionVectorCollectionDrop(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	return db.DropCollection(ctx, vectorCollectionName)
}

// #endregion
