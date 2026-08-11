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
	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/cursors"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/sort"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

// Employee represents a test document with nested fields
type Employee struct {
	ID       any     `json:"_id,omitempty"`
	Key      string  `json:"key,omitempty"`
	Username string  `json:"username,omitempty"`
	Human    *bool   `json:"human,omitempty"`
	Age      *int    `json:"age,omitempty"`
	Password *string `json:"password"`
	Address  *struct {
		Number   *int    `json:"number,omitempty"`
		Street   *string `json:"street"`
		Suburb   *string `json:"suburb"`
		City     *string `json:"city"`
		IsOffice *bool   `json:"is_office,omitempty"`
		Country  *string `json:"country,omitempty"`
	} `json:"address,omitempty"`
}

func createSampleDocWithMultiLevel(key string) astra.NewDocument {
	return astra.NewDocument{
		"key":      key,
		"username": "aaron",
		"human":    true,
		"age":      47,
		"password": nil,
		"address": map[string]any{
			"number":    86,
			"street":    "monkey street",
			"suburb":    "not null",
			"city":      "big banana",
			"is_office": false,
		},
	}
}

func createSampleDoc2WithMultiLevel(key string) astra.NewDocument {
	return astra.NewDocument{
		"key":      key,
		"username": "jim_r",
		"human":    true,
		"age":      52,
		"password": "has_gas==",
		"address": map[string]any{
			"number":    45,
			"street":    "main street",
			"suburb":    nil,
			"city":      "nyc",
			"is_office": true,
			"country":   "usa",
		},
	}
}

func createSampleDoc3WithMultiLevel(key string) astra.NewDocument {
	return astra.NewDocument{
		"key":      key,
		"username": "saml",
		"human":    false,
		"age":      25,
		"password": "jan_k_vans==",
		"address": map[string]any{
			"number":    123,
			"street":    "church street",
			"suburb":    nil,
			"city":      "la",
			"is_office": true,
			"country":   "usa",
		},
	}
}

func init() {
	s := harness.ParallelSuite("finds")
	s.Truncate(harness.SelectCollections, harness.SelectBefore)

	s.Run("should find & findOne document with an empty filter", func(t *harness.T) {
		// Insert a document
		res, err := t.Collection_.InsertOne(t.Ctx, astra.NewDocument{"key": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedID any
		err = res.DecodeID(&insertedID)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with nil filter
		resDoc := t.Collection_.FindOne(t.Ctx, nil)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne with nil filter failed: %v", resDoc.Err())
		var doc1 astra.Document
		err = resDoc.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("_id") != insertedID, "expected _id to match")
		testlib.FailIf(t, doc1.MustGet("key").(string) != t.Key(0), "expected key to match")

		// Test with empty filter.F{}
		resDoc = t.Collection_.FindOne(t.Ctx, filter.F{})
		testlib.FailIfErr(t, resDoc.Err(), "FindOne with empty filter failed: %v", resDoc.Err())
		var doc2 astra.Document
		err = resDoc.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc2.MustGet("_id") != insertedID, "expected _id to match")
		testlib.FailIf(t, doc2.MustGet("key").(string) != t.Key(0), "expected key to match")

		// Test Find with nil filter
		cursor := t.Collection_.Find(nil)
		testlib.FailIf(t, !cursor.Next(t.Ctx), "expected at least one document")
		var doc3 astra.Document
		err = cursor.Decode(&doc3)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc3.MustGet("_id") != insertedID, "expected _id to match")
		testlib.FailIf(t, doc3.MustGet("key").(string) != t.Key(0), "expected key to match")
		testlib.FailIf(t, cursor.Next(t.Ctx), "expected only one document")
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())

		// Test Find with empty filter.F{}
		cursor = t.Collection_.Find(filter.F{})
		testlib.FailIf(t, !cursor.Next(t.Ctx), "expected at least one document")
		var doc4 astra.Document
		err = cursor.Decode(&doc4)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc4.MustGet("_id") != insertedID, "expected _id to match")
		testlib.FailIf(t, doc4.MustGet("key").(string) != t.Key(0), "expected key to match")
		testlib.FailIf(t, cursor.Next(t.Ctx), "expected only one document")
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
	})

	s.Run("should find & findOne document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// FindOne
		f := filter.And(filter.Eq("_id", idToCheck), filter.Eq("key", t.Key(0)))
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())

		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		// Find
		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne document with projection", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		f := filter.And(filter.Eq("_id", idToCheck), filter.Eq("key", t.Key(0)))
		projection := map[string]any{"username": 1}

		// FindOne with projection
		resDoc := t.Collection.FindOne(t.Ctx, f, options.CollectionFindOne().SetProjection(projection))
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())

		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")
		testlib.FailIf(t, found.MustGet("username").(string) != "aaron", "expected username to be 'aaron'")
		_, hasAge := found.Get("age")
		testlib.FailIf(t, hasAge, "expected age to be excluded from projection")

		// Find with projection
		cursor := t.Collection.Find(f, options.CollectionFind().SetProjection(projection))
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
		testlib.FailIf(t, findDocs[0].MustGet("username").(string) != "aaron", "expected username to be 'aaron'")
		_, hasAge = findDocs[0].Get("age")
		testlib.FailIf(t, hasAge, "expected age to be excluded from projection")
	})

	s.Run("should find with sort", func(t *harness.T) {
		// Insert documents
		docs := []astra.NewDocument{
			{"username": "a", "key": t.Key(0)},
			{"username": "c", "key": t.Key(0)},
			{"username": "b", "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Find with ascending sort
		cursor := t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Asc("username")).SetLimit(20),
		)
		var usernames []string
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			usernames = append(usernames, d.MustGet("username").(string))
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		t.NoDiff([]string{"a", "b", "c"}, usernames)

		// Find with descending sort
		cursor = t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Desc("username")).SetLimit(20),
		)
		usernames = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			usernames = append(usernames, d.MustGet("username").(string))
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		t.NoDiff([]string{"c", "b", "a"}, usernames)
	})

	s.Run("should findOne with sort", func(t *harness.T) {
		// Insert documents
		docs := []astra.NewDocument{
			{"username": "a", "key": t.Key(0)},
			{"username": "c", "key": t.Key(0)},
			{"username": "b", "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// FindOne with ascending sort
		resDoc := t.Collection.FindOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOne().SetSort(sort.Asc("username")),
		)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var doc1 astra.Document
		err = resDoc.Decode(&doc1)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc1.MustGet("username").(string) != "a", "expected username to be 'a'")

		// FindOne with descending sort
		resDoc = t.Collection.FindOne(
			t.Ctx,
			filter.Eq("key", t.Key(0)),
			options.CollectionFindOne().SetSort(sort.Desc("username")),
		)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var doc2 astra.Document
		err = resDoc.Decode(&doc2)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, doc2.MustGet("username").(string) != "c", "expected username to be 'c'")
	})

	s.Run("should find with multiple, and different, sorts", func(t *harness.T) {
		// Insert documents
		docs := []astra.NewDocument{
			{"username": "a", "age": 1, "key": t.Key(0)},
			{"username": "a", "age": 3, "key": t.Key(0)},
			{"username": "a", "age": 2, "key": t.Key(0)},
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Test username:1, age:1
		cursor := t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Asc("username").Asc("age")).SetLimit(20),
		)
		var ages []int
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			ages = append(ages, int(d.MustGet("age").(float64)))
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		t.NoDiff([]int{1, 2, 3}, ages)

		// Test username:1, age:-1
		cursor = t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Asc("username").Desc("age")).SetLimit(20),
		)
		ages = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			ages = append(ages, int(d.MustGet("age").(float64)))
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		t.NoDiff([]int{3, 2, 1}, ages)

		// Test username:-1, age:1
		cursor = t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Desc("username").Asc("age")).SetLimit(20),
		)
		ages = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			ages = append(ages, int(d.MustGet("age").(float64)))
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		t.NoDiff([]int{1, 2, 3}, ages)

		// Test username:-1, age:-1
		cursor = t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetSort(sort.Desc("username").Desc("age")).SetLimit(20),
		)
		ages = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			ages = append(ages, int(d.MustGet("age").(float64)))
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		t.NoDiff([]int{3, 2, 1}, ages)
	})

	s.Run("should find & findOne eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with $eq operator
		f := filter.And(filter.Eq("_id", idToCheck), filter.Eq("key", t.Key(0)))

		// FindOne
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		// Find
		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		res1, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck1, idToCheck2 any
		err = res1.DecodeID(&idToCheck1)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with first ID
		f := filter.And(filter.Ne("_id", idToCheck1), filter.Eq("key", t.Key(0)))
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to be second document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to be second document")

		// Test $ne with second ID
		f = filter.And(filter.Ne("_id", idToCheck2), filter.Eq("key", t.Key(0)))
		resDoc = t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck1, "expected _id to be first document")

		cursor = t.Collection.Find(f)
		findDocs = nil
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck1, "expected _id to be first document")
	})

	s.Run("should find & findOne L1 String EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with direct equality (implicit $eq)
		f := filter.And(filter.Eq("username", "aaron"), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 String EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq operator
		f := filter.And(filter.Eq("username", "aaron"), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 String NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		_, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with username
		f := filter.And(filter.Ne("username", "aaron"), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to be second document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to be second document")
	})

	s.Run("should find & findOne L1 Number EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with age field
		f := filter.And(filter.Eq("age", 47), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 Number EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on age
		f := filter.And(filter.Eq("age", 47), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 Number NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		_, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with age
		f := filter.And(filter.Ne("age", 47), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to be second document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to be second document")
	})

	s.Run("should find & findOne L1 Boolean EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with human field
		f := filter.And(filter.Eq("human", true), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 Boolean EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on human
		f := filter.And(filter.Eq("human", true), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 Boolean NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDoc2WithMultiLevel(t.Key(0))
		doc2 := createSampleDoc3WithMultiLevel(t.Key(0))
		res1, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		_, err = t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck1 any
		err = res1.DecodeID(&idToCheck1)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with human=false (should find doc1 which has human=true)
		f := filter.And(filter.Ne("human", false), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck1, "expected _id to be first document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck1, "expected _id to be first document")
	})

	s.Run("should find & findOne L1 Null EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with password=null
		f := filter.And(filter.Eq("password", nil), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 Null EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on password=null
		f := filter.And(filter.Eq("password", nil), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne L1 Null NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		_, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with password=null (should find doc2 which has password="has_gas==")
		f := filter.And(filter.Ne("password", nil), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to be second document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to be second document")
	})

	s.Run("should find & findOne any level String EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with nested field address.street
		f := filter.And(filter.Eq("address.street", "monkey street"), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne any level String EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on nested field
		f := filter.And(filter.Eq("address.street", "monkey street"), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne any level String NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		res1, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		_, err = t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck1 any
		err = res1.DecodeID(&idToCheck1)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with nested field (should find doc1)
		f := filter.And(filter.Ne("address.street", "main street"), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck1, "expected _id to be first document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck1, "expected _id to be first document")
	})

	s.Run("should find & findOne any level Number EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with nested number field
		f := filter.And(filter.Eq("address.number", 86), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should findOne any level Number EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on nested number
		f := filter.And(filter.Eq("address.number", 86), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne any level Number NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		_, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with nested number
		f := filter.And(filter.Ne("address.number", 86), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to be second document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to be second document")
	})

	s.Run("should find & findOne any level Boolean EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with nested boolean field
		f := filter.And(filter.Eq("address.is_office", false), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne any level Boolean EQ $eq document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on nested boolean
		f := filter.And(filter.Eq("address.is_office", false), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne any level Boolean NE $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		_, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with nested boolean
		f := filter.And(filter.Ne("address.is_office", false), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to be second document")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to be second document")
	})

	s.Run("should find & findOne any level Null EQ document", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with nested null field (suburb is "not null" string, not null)
		// We need to use a field that's actually null - let's check the structure
		// address.suburb in doc1 is "not null", but in doc2 it's nil
		// So this test won't find doc1, let's skip or adjust
		// Actually, looking at the TS test, it seems to expect finding the doc
		// Let me check the fixture again - address.suburb is "not null" in doc1
		// The TS test filters by address.suburb which is "not null" (a string)
		// So it's testing equality with the string value, not with null
		// But the test name says "Null EQ" which is confusing
		// Looking more carefully: doc.address?.suburb in TS would be "not null"
		// So the filter { 'address.suburb': doc.address?.suburb } would be { 'address.suburb': "not null" }
		// This is actually testing string equality, not null equality
		// The test name is misleading - it's about the field name containing "suburb" not about null values

		// For now, let's test with a field that's actually null in the fixture
		// Looking at createSampleDocWithMultiLevel, password is nil
		// But that's L1, not nested. Let's check if there's a nested null field
		// Actually, the TS fixture doesn't have a nested null field in doc1
		// Let's use doc2 which has address.suburb: nil
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with nested null field
		f := filter.And(filter.Eq("address.suburb", nil), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to match doc2")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to match doc2")
	})

	s.Run("should find & findOne any level Null EQ $eq document", func(t *harness.T) {
		// Insert doc2 which has address.suburb: nil
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with explicit $eq on nested null
		f := filter.And(filter.Eq("address.suburb", nil), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck2, "expected _id to match doc2")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck2, "expected _id to match doc2")
	})

	s.Run("should find & findOne any level Null EQ $ne document", func(t *harness.T) {
		// Insert two documents
		doc1 := createSampleDocWithMultiLevel(t.Key(0))
		doc2 := createSampleDoc2WithMultiLevel(t.Key(0))
		_, err := t.Collection.InsertOne(t.Ctx, doc1)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)
		res2, err := t.Collection.InsertOne(t.Ctx, doc2)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck2 any
		err = res2.DecodeID(&idToCheck2)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test $ne with nested null (doc1.address.suburb is "not null", doc2 is nil)
		// So $ne nil should find doc1
		f := filter.And(filter.Ne("address.suburb", nil), filter.Eq("key", t.Key(0)))

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		// Should find doc1 which has suburb="not null"
		testlib.FailIf(t, found.MustGet("_id") == idToCheck2, "expected _id to NOT be doc2")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") == idToCheck2, "expected _id to NOT be doc2")
	})

	s.Run("should find & findOne multiple top level conditions", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with multiple top-level conditions
		f := filter.And(
			filter.Eq("age", 47),
			filter.Eq("human", true),
			filter.Eq("password", nil),
			filter.Eq("key", t.Key(0)),
		)

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne multiple level>=2 conditions", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with multiple nested conditions
		f := filter.And(
			filter.Eq("address.number", 86),
			filter.Eq("address.street", "monkey street"),
			filter.Eq("address.is_office", false),
			filter.Eq("key", t.Key(0)),
		)

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne multiple mixed levels conditions", func(t *harness.T) {
		// Insert a document
		doc := createSampleDocWithMultiLevel(t.Key(0))
		res, err := t.Collection.InsertOne(t.Ctx, doc)
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var idToCheck any
		err = res.DecodeID(&idToCheck)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)

		// Test with mixed level conditions
		f := filter.And(
			filter.Eq("age", 47),
			filter.Eq("address.street", "monkey street"),
			filter.Eq("address.is_office", false),
			filter.Eq("key", t.Key(0)),
		)

		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id") != idToCheck, "expected _id to match")

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))
		testlib.FailIf(t, findDocs[0].MustGet("_id") != idToCheck, "expected _id to match")
	})

	s.Run("should find & findOne doc $in test", func(t *harness.T) {
		// Insert 20 documents
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			docs = append(docs, astra.NewDocument{
				"_id":      t.Key(i),
				"username": "id",
				"city":     "nyc",
				"key":      t.Key(0),
			})
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Test $in with multiple IDs
		idsToFind := []any{t.Key(1), t.Key(2), t.Key(3)}
		f := filter.And(filter.In("_id", idsToFind...), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 3, "expected 3 documents, got %d", len(findDocs))

		// Verify all returned IDs are in the input list
		idsSet := make(map[string]bool)
		for _, id := range idsToFind {
			idsSet[id.(string)] = true
		}
		for _, doc := range findDocs {
			id := doc.MustGet("_id").(string)
			testlib.FailIf(t, !idsSet[id], "unexpected _id: %s", id)
		}

		// Test FindOne with single ID
		f = filter.And(filter.In("_id", t.Key(2)), filter.Eq("key", t.Key(0)))
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		testlib.FailIf(t, found.MustGet("_id").(string) != t.Key(2), "expected _id to be %s", t.Key(2))
	})

	s.Run("should find & findOne doc $nin test", func(t *harness.T) {
		// Insert documents with different cities
		var docsNYC []astra.NewDocument
		for i := 0; i < 3; i++ {
			docsNYC = append(docsNYC, astra.NewDocument{
				"city": "nyc" + string(rune('1'+i)),
				"key":  t.Key(0),
			})
		}
		_, err := t.Collection.InsertMany(t.Ctx, docsNYC)
		testlib.FailIfErr(t, err, "InsertMany NYC failed: %v", err)

		var docsSeattle []astra.NewDocument
		for i := 0; i < 2; i++ {
			docsSeattle = append(docsSeattle, astra.NewDocument{
				"city": "seattle" + string(rune('1'+i)),
				"key":  t.Key(0),
			})
		}
		_, err = t.Collection.InsertMany(t.Ctx, docsSeattle)
		testlib.FailIfErr(t, err, "InsertMany Seattle failed: %v", err)

		// Test $nin - exclude NYC cities
		cityArr := []any{"nyc1", "nyc2", "nyc3"}
		f := filter.And(filter.Nin("city", cityArr...), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 2, "expected 2 documents, got %d", len(findDocs))

		// Verify all returned docs have city starting with "seattle"
		for _, doc := range findDocs {
			city := doc.MustGet("city").(string)
			testlib.FailIf(t, len(city) < 7 || city[:7] != "seattle", "expected city to start with 'seattle', got %s", city)
		}
	})

	s.Run("should find & findOne doc $exists true test", func(t *harness.T) {
		// Insert 20 documents with city field
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			docs = append(docs, astra.NewDocument{
				"username": "id",
				"city":     "nyc",
				"key":      t.Key(0),
			})
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Test $exists: true
		f := filter.And(filter.Coll.Exists("city", true), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 20, "expected 20 documents, got %d", len(findDocs))

		// Verify all docs have city field
		for _, doc := range findDocs {
			_, hasCity := doc.Get("city")
			testlib.FailIf(t, !hasCity, "expected document to have city field")
		}

		// Test FindOne
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		_, hasCity := found.Get("city")
		testlib.FailIf(t, !hasCity, "expected document to have city field")
	})

	s.Run("should find & findOne doc $exists false test", func(t *harness.T) {
		// Insert 10 documents with city
		var docsWithCity []astra.NewDocument
		for i := 0; i < 10; i++ {
			docsWithCity = append(docsWithCity, astra.NewDocument{
				"username": "withCity",
				"city":     "nyc",
				"key":      t.Key(0),
			})
		}
		_, err := t.Collection.InsertMany(t.Ctx, docsWithCity)
		testlib.FailIfErr(t, err, "InsertMany with city failed: %v", err)

		// Insert 10 documents without city
		var docsNoCity []astra.NewDocument
		for i := 0; i < 10; i++ {
			docsNoCity = append(docsNoCity, astra.NewDocument{
				"username": "noCity",
				"key":      t.Key(0),
			})
		}
		_, err = t.Collection.InsertMany(t.Ctx, docsNoCity)
		testlib.FailIfErr(t, err, "InsertMany without city failed: %v", err)

		// Test $exists: false
		f := filter.And(filter.Coll.Exists("city", false), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 10, "expected 10 documents, got %d", len(findDocs))

		// Verify all docs do NOT have city field
		for _, doc := range findDocs {
			_, hasCity := doc.Get("city")
			testlib.FailIf(t, hasCity, "expected document to NOT have city field")
		}
	})

	s.Run("should find & findOne doc $all test", func(t *harness.T) {
		// Insert 20 documents, one with specific tags
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"_id":      t.Key(i),
				"username": "id",
				"city":     "nyc",
				"key":      t.Key(0),
			}
			if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3"}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Test $all - find document with all specified tags
		f := filter.And(filter.All("tags", "tag1", "tag2", "tag3"), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))

		// Verify the document has the expected tags
		doc := findDocs[0]
		tags, hasTags := doc.Get("tags")
		testlib.FailIf(t, !hasTags, "expected document to have tags field")
		tagsSlice := tags.([]any)
		testlib.FailIf(t, len(tagsSlice) != 3, "expected 3 tags, got %d", len(tagsSlice))
		testlib.FailIf(t, doc.MustGet("_id").(string) != t.Key(5), "expected _id to be %s", t.Key(5))

		// Test FindOne
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		tags, hasTags = found.Get("tags")
		testlib.FailIf(t, !hasTags, "expected document to have tags field")
		tagsSlice = tags.([]any)
		testlib.FailIf(t, len(tagsSlice) != 3, "expected 3 tags, got %d", len(tagsSlice))
		testlib.FailIf(t, found.MustGet("_id").(string) != t.Key(5), "expected _id to be %s", t.Key(5))
	})

	s.Run("should find & findOne doc $size test", func(t *harness.T) {
		// Insert 20 documents with different tag array sizes
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"_id":      t.Key(i),
				"username": "id",
				"city":     "nyc",
				"key":      t.Key(0),
			}
			if i == 4 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4"}
			} else if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3"}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Test $size: 3
		f := filter.And(filter.Coll.Size("tags", 3), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))

		// Verify the document has exactly 3 tags
		doc := findDocs[0]
		tags := doc.MustGet("tags").([]any)
		testlib.FailIf(t, len(tags) != 3, "expected 3 tags, got %d", len(tags))
		testlib.FailIf(t, doc.MustGet("_id").(string) != t.Key(5), "expected _id to be %s", t.Key(5))

		// Test FindOne
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		tags = found.MustGet("tags").([]any)
		testlib.FailIf(t, len(tags) != 3, "expected 3 tags, got %d", len(tags))
		testlib.FailIf(t, found.MustGet("_id").(string) != t.Key(5), "expected _id to be %s", t.Key(5))
	})

	s.Run("should find & findOne doc $size 0 test", func(t *harness.T) {
		// Insert 20 documents with different tag array sizes
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"_id":      t.Key(i),
				"username": "id",
				"city":     "nyc",
				"key":      t.Key(0),
			}
			if i == 4 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4"}
			} else if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3"}
			} else if i == 6 {
				doc["tags"] = []string{}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Test $size: 0
		f := filter.And(filter.Coll.Size("tags", 0), filter.Eq("key", t.Key(0)))

		cursor := t.Collection.Find(f)
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 1, "expected 1 document, got %d", len(findDocs))

		// Verify the document has empty tags array
		doc := findDocs[0]
		tags := doc.MustGet("tags").([]any)
		testlib.FailIf(t, len(tags) != 0, "expected 0 tags, got %d", len(tags))
		testlib.FailIf(t, doc.MustGet("_id").(string) != t.Key(6), "expected _id to be %s", t.Key(6))

		// Test FindOne
		resDoc := t.Collection.FindOne(t.Ctx, f)
		testlib.FailIfErr(t, resDoc.Err(), "FindOne failed: %v", resDoc.Err())
		var found astra.Document
		err = resDoc.Decode(&found)
		testlib.FailIfErr(t, err, "Decode failed: %v", err)
		tags = found.MustGet("tags").([]any)
		testlib.FailIf(t, len(tags) != 0, "expected 0 tags, got %d", len(tags))
		testlib.FailIf(t, found.MustGet("_id").(string) != t.Key(6), "expected _id to be %s", t.Key(6))
	})

	s.Run("should find doc - return only selected fields (array slice)", func(t *harness.T) {
		// Insert 20 documents with different tag arrays
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"username": "id",
				"address":  map[string]any{"city": "nyc"},
				"key":      t.Key(0),
			}
			if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
			} else if i == 6 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		all, err := cursors.DecodeAll[astra.Document](t.Ctx, t.Collection.Find(
			filter.Eq("key", t.Key(0)),
			options.CollectionFind().SetProjection(map[string]any{
				"username":     1,
				"address.city": true,
				"_id":          0,
				"tags":         map[string]any{"$slice": 1},
			}),
		))
		testlib.FailIfErr(t, err, "DecodeAll failed: %v", err)
		testlib.FailIf(t, len(all) != 20, "expected 20 documents, got %d", len(all))

		// Verify projection results
		for _, doc := range all {
			testlib.FailIf(t, doc.Has("_id"), "expected _id to be excluded")
			testlib.FailIf(t, doc.MustGet("username") == nil, "expected username to be present")

			address := doc.MustGet("address").(map[string]any)
			testlib.FailIf(t, address["city"] == nil, "expected address.city to be present")
			testlib.FailIf(t, address["number"] != nil, "expected address.number to be excluded")

			tags, hasTags := doc.Get("tags")
			if hasTags {
				tagsSlice := tags.([]any)
				testlib.FailIf(t, len(tagsSlice) != 1, "expected 1 tag (sliced), got %d", len(tagsSlice))
				testlib.FailIf(t, tagsSlice[0].(string) != "tag1", "expected first tag to be 'tag1'")
			}
		}
	})

	s.Run("should find doc - return only selected fields (array slice negative)", func(t *harness.T) {
		// Insert 20 documents with different tag arrays
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"username": "id",
				"address":  map[string]any{"city": "nyc"},
				"key":      t.Key(0),
			}
			if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
			} else if i == 6 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)
	})

	s.Run("should find doc - return only selected fields (array slice gt elements)", func(t *harness.T) {
		// Insert 20 documents with different tag arrays
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"username": "id",
				"address":  map[string]any{"city": "nyc"},
				"key":      t.Key(0),
			}
			if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
			} else if i == 6 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Read back with projection including array slice greater than array length
		projection := map[string]any{
			"username":     1,
			"address.city": true,
			"_id":          0,
			"tags":         map[string]any{"$slice": 6},
		}
		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetProjection(projection))
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 20, "expected 20 documents, got %d", len(findDocs))

		// Verify projection results
		for _, resDoc := range findDocs {
			testlib.FailIf(t, resDoc.Has("_id"), "expected _id to be excluded")
			testlib.FailIf(t, resDoc.MustGet("username") == nil, "expected username to be present")

			address := resDoc.MustGet("address").(map[string]any)
			testlib.FailIf(t, address["city"] == nil, "expected address.city to be present")
			testlib.FailIf(t, address["number"] != nil, "expected address.number to be excluded")

			tags, hasTags := resDoc.Get("tags")
			if hasTags {
				tagsSlice := tags.([]any)
				// When slice is greater than array length, return entire array
				// Doc at index 5 has 5 tags, doc at index 6 has 6 tags
				testlib.FailIf(t, len(tagsSlice) != 5 && len(tagsSlice) != 6, "expected 5 or 6 tags, got %d", len(tagsSlice))
			}
		}
	})

	s.Run("should find doc - return only selected fields (array slice gt elements negative)", func(t *harness.T) {
		// Insert 20 documents with different tag arrays
		var docs []astra.NewDocument
		for i := 0; i < 20; i++ {
			doc := astra.NewDocument{
				"username": t.Key(i + 1),
				"address":  map[string]any{"city": "nyc"},
				"key":      t.Key(0),
			}
			if i == 5 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
			} else if i == 6 {
				doc["tags"] = []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"}
			}
			docs = append(docs, doc)
		}
		_, err := t.Collection.InsertMany(t.Ctx, docs)
		testlib.FailIfErr(t, err, "InsertMany failed: %v", err)

		// Read back with projection including negative array slice greater than array length
		projection := map[string]any{
			"username":     1,
			"address.city": true,
			"_id":          0,
			"tags":         map[string]any{"$slice": -6},
		}
		cursor := t.Collection.Find(filter.Eq("key", t.Key(0)), options.CollectionFind().SetProjection(projection))
		var findDocs []astra.Document
		for cursor.Next(t.Ctx) {
			var d astra.Document
			err := cursor.Decode(&d)
			testlib.FailIfErr(t, err, "Decode failed: %v", err)
			findDocs = append(findDocs, d)
		}
		testlib.FailIfErr(t, cursor.Err(), "cursor error: %v", cursor.Err())
		testlib.FailIf(t, len(findDocs) != 20, "expected 20 documents, got %d", len(findDocs))

		// Verify projection results
		for _, resDoc := range findDocs {
			testlib.FailIf(t, resDoc.Has("_id"), "expected _id to be excluded")
			testlib.FailIf(t, resDoc.MustGet("username") == nil, "expected username to be present")

			address := resDoc.MustGet("address").(map[string]any)
			testlib.FailIf(t, address["city"] == nil, "expected address.city to be present")
			testlib.FailIf(t, address["number"] != nil, "expected address.number to be excluded")

			username := resDoc.MustGet("username").(string)
			tags, hasTags := resDoc.Get("tags")
			if hasTags {
				tagsSlice := tags.([]any)
				// When negative slice is greater than array length, return entire array
				if username == t.Key(6) {
					testlib.FailIf(t, len(tagsSlice) != 5, "expected 5 tags for doc 6, got %d", len(tagsSlice))
				} else if username == t.Key(7) {
					testlib.FailIf(t, len(tagsSlice) != 6, "expected 6 tags for doc 7, got %d", len(tagsSlice))
				}
			} else {
				// Documents without tags should not have the field
				testlib.FailIf(t, username == t.Key(6) || username == t.Key(7), "expected tags for doc 6 or 7")
			}
		}
	})

}
