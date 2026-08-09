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

package documents_tests

import (
	"slices"
	"sync"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.SequentialSuite("ids")

	// Create collections with different default ID types
	var collUUID *astra.Collection
	var collUUIDv6 *astra.Collection
	var collUUIDv7 *astra.Collection
	var collObjectId *astra.Collection

	type spec struct {
		suffix string
		idType options.CollectionIdType
		dst    **astra.Collection
	}

	specs := []spec{
		{"_uuid", options.CollectionIdTypeUUID, &collUUID},
		{"_uuidv6", options.CollectionIdTypeUUIDv6, &collUUIDv6},
		{"_uuidv7", options.CollectionIdTypeUUIDv7, &collUUIDv7},
		{"_objectId", options.CollectionIdTypeObjectId, &collObjectId},
	}

	foreachCollAsync := func(f func(coll *spec)) {
		var wg sync.WaitGroup

		for _, s := range specs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				f(&s)
			}()
		}

		wg.Wait()
	}

	s.Before(func(t *harness.T) {
		foreachCollAsync(func(s *spec) {
			var err error
			*s.dst, err = t.Db.CreateCollection(t.Ctx, harness.DefaultCollectionName+s.suffix, options.CreateCollection().SetDefaultIdType(s.idType))
			testlib.FailIfErr(t, err, "CreateCollection %s failed: %v", s.suffix, err)
		})
	})

	s.After(func(t *harness.T) {
		foreachCollAsync(func(s *spec) {
			err := (*s.dst).Drop(t.Ctx)
			testlib.FailIfErr(t, err, "DropCollection %s failed: %v", s.suffix, err)
		})
	})

	s.Run("default id is not in listCollections", func(t *harness.T) {
		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "ListCollections failed: %v", err)

		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == harness.DefaultCollectionName
		})
		testlib.FailIf(t, idx == -1, "collection %s not found", harness.DefaultCollectionName)

		collection := collections[idx]
		testlib.FailIf(t, collection.Definition.DefaultId != nil, "expected defaultId to be nil, got %v", collection.Definition.DefaultId)
	})

	s.Run("default id is set as the default id", func(t *harness.T) {
		res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"name": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedId string
		err = res.DecodeID(&insertedId)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		testlib.FailIf(t, insertedId == "", "expected non-empty insertedId")

		// Verify it's a valid UUID string
		_, err = datatypes.ParseUUID(insertedId)
		testlib.FailIfErr(t, err, "insertedId is not a valid UUID: %v", err)

		var found astra.Document
		err = t.Collection.FindOne(t.Ctx, filter.Eq("name", t.Key(0))).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		id := found.MustGet("_id").(string)
		testlib.FailIf(t, id != insertedId, "expected _id %s, got %s", insertedId, id)

		// Verify the found ID is also a valid UUID
		_, err = datatypes.ParseUUID(id)
		testlib.FailIfErr(t, err, "found _id is not a valid UUID: %v", err)
	})

	s.Run("uuid is set in listCollections", func(t *harness.T) {
		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "ListCollections failed: %v", err)

		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == harness.DefaultCollectionName+"_uuid"
		})
		testlib.FailIf(t, idx == -1, "collection %s_uuid not found", harness.DefaultCollectionName)

		collection := collections[idx]
		testlib.FailIf(t, collection.Definition.DefaultId == nil, "expected defaultId to be set")
		testlib.FailIf(t, *collection.Definition.DefaultId.Type != results.CollectionIdTypeUUID, "expected defaultId type 'uuid', got %v", *collection.Definition.DefaultId.Type)
	})

	s.Run("uuid sets it as the default id", func(t *harness.T) {
		res, err := collUUID.InsertOne(t.Ctx, astra.NewDocument{"name": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedId datatypes.UUID
		err = res.DecodeID(&insertedId)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		testlib.FailIf(t, insertedId.Version() != 4, "expected UUID version 4, got %d", insertedId.Version())

		var found astra.Document
		err = collUUID.FindOne(t.Ctx, filter.Eq("name", t.Key(0))).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		id := found.MustGet("_id").(datatypes.UUID)
		testlib.FailIf(t, id.Version() != 4, "expected UUID version 4, got %d", id.Version())
		testlib.FailIf(t, id.String() != insertedId.String(), "expected _id %s, got %s", insertedId, id)
	})

	s.Run("uuidv6 is set in listCollections", func(t *harness.T) {
		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "ListCollections failed: %v", err)

		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == harness.DefaultCollectionName+"_uuidv6"
		})
		testlib.FailIf(t, idx == -1, "collection %s_uuidv6 not found", harness.DefaultCollectionName)

		collection := collections[idx]
		testlib.FailIf(t, collection.Definition.DefaultId == nil, "expected defaultId to be set")
		testlib.FailIf(t, *collection.Definition.DefaultId.Type != results.CollectionIdTypeUUIDv6, "expected defaultId type 'uuidv6', got %v", *collection.Definition.DefaultId.Type)
	})

	s.Run("uuidv6 sets it as the default id", func(t *harness.T) {
		res, err := collUUIDv6.InsertOne(t.Ctx, astra.NewDocument{"name": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedId datatypes.UUID
		err = res.DecodeID(&insertedId)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		testlib.FailIf(t, insertedId.Version() != 6, "expected UUID version 6, got %d", insertedId.Version())

		var found astra.Document
		err = collUUIDv6.FindOne(t.Ctx, filter.Eq("name", t.Key(0))).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		id := found.MustGet("_id").(datatypes.UUID)
		testlib.FailIf(t, id.Version() != 6, "expected UUID version 6, got %d", id.Version())
		testlib.FailIf(t, id.String() != insertedId.String(), "expected _id %s, got %s", insertedId, id)
	})

	s.Run("uuidv7 is set in listCollections", func(t *harness.T) {
		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "ListCollections failed: %v", err)

		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == harness.DefaultCollectionName+"_uuidv7"
		})
		testlib.FailIf(t, idx == -1, "collection %s_uuidv7 not found", harness.DefaultCollectionName)

		collection := collections[idx]
		testlib.FailIf(t, collection.Definition.DefaultId == nil, "expected defaultId to be set")
		testlib.FailIf(t, *collection.Definition.DefaultId.Type != results.CollectionIdTypeUUIDv7, "expected defaultId type 'uuidv7', got %v", *collection.Definition.DefaultId.Type)
	})

	s.Run("uuidv7 sets it as the default id", func(t *harness.T) {
		res, err := collUUIDv7.InsertOne(t.Ctx, astra.NewDocument{"name": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedId datatypes.UUID
		err = res.DecodeID(&insertedId)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		testlib.FailIf(t, insertedId.Version() != 7, "expected UUID version 7, got %d", insertedId.Version())

		var found astra.Document
		err = collUUIDv7.FindOne(t.Ctx, filter.Eq("name", t.Key(0))).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		id := found.MustGet("_id").(datatypes.UUID)
		testlib.FailIf(t, id.Version() != 7, "expected UUID version 7, got %d", id.Version())
		testlib.FailIf(t, id.String() != insertedId.String(), "expected _id %s, got %s", insertedId, id)
	})

	s.Run("objectId is set in listCollections", func(t *harness.T) {
		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "ListCollections failed: %v", err)

		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == harness.DefaultCollectionName+"_objectId"
		})
		testlib.FailIf(t, idx == -1, "collection %s_objectId not found", harness.DefaultCollectionName)

		collection := collections[idx]
		testlib.FailIf(t, collection.Definition.DefaultId == nil, "expected defaultId to be set")
		testlib.FailIf(t, *collection.Definition.DefaultId.Type != results.CollectionIdTypeObjectId, "expected defaultId type 'objectId', got %v", *collection.Definition.DefaultId.Type)
	})

	s.Run("objectId sets it as the default id", func(t *harness.T) {
		res, err := collObjectId.InsertOne(t.Ctx, astra.NewDocument{"name": t.Key(0)})
		testlib.FailIfErr(t, err, "InsertOne failed: %v", err)

		var insertedId datatypes.ObjectId
		err = res.DecodeID(&insertedId)
		testlib.FailIfErr(t, err, "DecodeID failed: %v", err)
		testlib.FailIf(t, insertedId.String() == "", "expected non-empty ObjectId")

		var found astra.Document
		err = collObjectId.FindOne(t.Ctx, filter.Eq("name", t.Key(0))).Decode(&found)
		testlib.FailIfErr(t, err, "FindOne failed: %v", err)

		id := found.MustGet("_id").(datatypes.ObjectId)
		testlib.FailIf(t, id.String() != insertedId.String(), "expected _id %s, got %s", insertedId, id)
	})
}
