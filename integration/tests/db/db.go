package db

import (
	"slices"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/table"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.SequentialSuite("db")
	s.Truncate(harness.SelectCollections, harness.SelectAfter)

	s.Run("(LONG) should create a collection", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "coll_1c", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)
		testlib.FailIf(t, coll == nil, "collection should not be nil")
		testlib.FailIf(t, coll.Name() != "coll_1c", "wrong name")
	})

	s.Run("(LONG) should create a collection in another keyspace", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "coll_2c", options.CreateCollection().SetKeyspace(harness.TestKeyspaces[1]).SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)
		testlib.FailIf(t, coll == nil, "collection should not be nil")
		testlib.FailIf(t, coll.Name() != "coll_2c", "wrong name")
	})

	s.Run("(LONG) should create collections idempotently", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "coll_4c", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)
		testlib.FailIf(t, coll == nil, "collection should not be nil")
		testlib.FailIf(t, coll.Name() != "coll_4c", "wrong name")

		coll2, err := t.Db.CreateCollection(t.Ctx, "coll_4c", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection idempotently: %v", err)
		testlib.FailIf(t, coll2 == nil, "collection2 should not be nil")
		testlib.FailIf(t, coll2.Name() != "coll_4c", "wrong name")
	})

	s.Run("(LONG) should create collections with same options idempotently", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "coll_5c", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)
		testlib.FailIf(t, coll == nil, "collection should not be nil")

		coll2, err := t.Db.CreateCollection(t.Ctx, "coll_5c", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection idempotently: %v", err)
		testlib.FailIf(t, coll2 == nil, "collection2 should not be nil")
	})

	s.Run("(LONG) should fail creating collections with different options", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "coll_6c", options.CreateCollection().SetIndexingDeny("*").SetDefaultIdType(options.CollectionIdTypeUUID))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)
		testlib.FailIf(t, coll == nil, "collection should not be nil")

		_, err = t.Db.CreateCollection(t.Ctx, "coll_6c", options.CreateCollection().SetIndexingDeny("*").SetDefaultIdType(options.CollectionIdTypeUUIDv7))
		testlib.FailIf(t, err == nil, "expected error when creating collection with different options")
	})

	s.Run("(LONG) should create collections with different options in different keyspaces", func(t *harness.T) {
		coll, err := t.Db.CreateCollection(t.Ctx, "coll_7c", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)
		testlib.FailIf(t, coll == nil, "collection should not be nil")

		coll2, err := t.Db.CreateCollection(t.Ctx, "coll_7c", options.CreateCollection().SetKeyspace(harness.TestKeyspaces[1]).SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection in different keyspace: %v", err)
		testlib.FailIf(t, coll2 == nil, "collection2 should not be nil")
	})

	s.Run("(LONG) should drop a collection", func(t *harness.T) {
		_, err := t.Db.CreateCollection(t.Ctx, "coll_1d", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)

		err = t.Db.DropCollection(t.Ctx, "coll_1d")
		testlib.FailIfErr(t, err, "failed to drop collection: %v", err)

		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list collections: %v", err)
		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == "coll_1d"
		})
		testlib.FailIf(t, idx != -1, "collection should not exist")
	})

	s.Run("(LONG) should drop a collection in non-default keyspace", func(t *harness.T) {
		_, err := t.Db.CreateCollection(t.Ctx, "coll_3d", options.CreateCollection().SetKeyspace(harness.TestKeyspaces[1]).SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)

		err = t.Db.DropCollection(t.Ctx, "coll_3d", options.DropCollection().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to drop collection: %v", err)

		collections, err := t.Db.ListCollections(t.Ctx, options.ListCollections().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to list collections: %v", err)
		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == "coll_3d"
		})
		testlib.FailIf(t, idx != -1, "collection should not exist")
	})

	s.Run("(LONG) should not drop a collection in different keyspace", func(t *harness.T) {
		_, err := t.Db.CreateCollection(t.Ctx, "coll_4d", options.CreateCollection().SetIndexingDeny("*"))
		testlib.FailIfErr(t, err, "failed to create collection: %v", err)

		err = t.Db.DropCollection(t.Ctx, "coll_4d", options.DropCollection().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to drop collection: %v", err)

		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list collections: %v", err)
		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == "coll_4d"
		})
		testlib.FailIf(t, idx == -1, "collection should exist")
	})

	s.Run("(LONG) should return a list of just names of collections with nameOnly set to true", func(t *harness.T) {
		names, err := t.Db.ListCollectionNames(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list collection names: %v", err)
		testlib.FailIf(t, !slices.Contains(names, harness.DefaultCollectionName), "should find default collection name")
	})

	s.Run("(LONG) should return a list of collections infos with nameOnly set to false", func(t *harness.T) {
		collections, err := t.Db.ListCollections(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list collections infos: %v", err)
		idx := slices.IndexFunc(collections, func(c results.CollectionDescriptor) bool {
			return c.Name == harness.DefaultCollectionName
		})
		testlib.FailIf(t, idx == -1, "should find default collection info")
		testlib.FailIf(t, collections[idx].Definition.Vector == nil, "vector definition should not be nil")
		testlib.FailIf(t, collections[idx].Definition.Vector.Dimension == nil || *collections[idx].Definition.Vector.Dimension != 5, "vector dimension should be 5")
		testlib.FailIf(t, collections[idx].Definition.Vector.Metric == nil || *collections[idx].Definition.Vector.Metric != string(options.MetricCosine), "vector metric should be cosine")
	})

	s.Run("(LONG) should create a UDT", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_1c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "street", Column: table.Text()},
				{Name: "city", Column: table.Text()},
				{Name: "zip_code", Column: table.Int()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "type_1c"
		})
		testlib.FailIf(t, idx == -1, "type should exist")
	})

	s.Run("(LONG) should create a UDT in another keyspace", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_2c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "name", Column: table.Text()},
				{Name: "age", Column: table.Int()},
			},
		}, options.CreateType().SetKeyspace(harness.TestKeyspaces[1]).SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx, options.ListTypes().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "type_2c"
		})
		testlib.FailIf(t, idx == -1, "type should exist")
	})

	s.Run("(LONG) should create UDTs idempotently with ifNotExists", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_4c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.CreateType(t.Ctx, "type_4c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type again: %v", err)

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		var count int
		for _, u := range types {
			if u.Name == "type_4c" {
				count++
			}
		}
		testlib.FailIf(t, count != 1, "type should exist exactly once")
	})

	s.Run("(LONG) should fail creating UDT with same name without ifNotExists", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_6c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.CreateType(t.Ctx, "type_6c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field2", Column: table.Int()},
			},
		})
		testlib.FailIf(t, err == nil, "expected error creating duplicate type")
	})

	s.Run("(LONG) should create UDTs with different options in different keyspaces", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_7c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.CreateType(t.Ctx, "type_7c", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field2", Column: table.Int()},
			},
		}, options.CreateType().SetKeyspace(harness.TestKeyspaces[1]).SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		time.Sleep(1 * time.Second) // Match TS sleep

		defaultTypes, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		otherTypes, err := t.Db.ListTypes(t.Ctx, options.ListTypes().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to list types: %v", err)

		idxDef := slices.IndexFunc(defaultTypes, func(u results.UDTDescriptor) bool {
			return u.Name == "type_7c"
		})
		idxOth := slices.IndexFunc(otherTypes, func(u results.UDTDescriptor) bool {
			return u.Name == "type_7c"
		})

		testlib.FailIf(t, idxDef == -1, "type should exist in default keyspace")
		testlib.FailIf(t, idxOth == -1, "type should exist in other keyspace")
	})

	s.Run("(LONG) should drop a UDT", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_1d", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		})
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.DropType(t.Ctx, "type_1d")
		testlib.FailIfErr(t, err, "failed to drop type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "type_1d"
		})
		testlib.FailIf(t, idx != -1, "type should not exist")
	})

	s.Run("(LONG) should drop a UDT in non-default keyspace", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_3d", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.DropType(t.Ctx, "type_3d", options.DropType().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to drop type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx, options.ListTypes().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "type_3d"
		})
		testlib.FailIf(t, idx != -1, "type should not exist")
	})

	s.Run("(LONG) should not drop a UDT in different keyspace", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "type_4d", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.DropType(t.Ctx, "type_4d", options.DropType().SetKeyspace(harness.TestKeyspaces[1]).SetIfExists(true))
		if err != nil {
			// ok
		}

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "type_4d"
		})
		testlib.FailIf(t, idx == -1, "type should exist")
	})

	s.Run("(LONG) should return a list of just names of UDTs with nameOnly set to true", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "list_test_type", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		names, err := t.Db.ListTypeNames(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list type names: %v", err)
		testlib.FailIf(t, !slices.Contains(names, "list_test_type"), "should find type name")
	})

	s.Run("(LONG) should return a list of UDT infos with nameOnly set to false", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "list_test_type_2", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "name", Column: table.Text()},
				{Name: "count", Column: table.Int()},
			},
		}, options.CreateType().SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types infos: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "list_test_type_2"
		})
		testlib.FailIf(t, idx == -1, "should find type info")

		u := types[idx]
		testlib.FailIf(t, len(u.Definition.Fields) != 2, "should have 2 fields")
		fieldTypes := make(map[string]string)
		for _, field := range u.Definition.Fields {
			fieldTypes[field.Name] = field.Column.Type
		}
		testlib.FailIf(t, fieldTypes["name"] != table.TypeText, "name should be text")
		testlib.FailIf(t, fieldTypes["count"] != table.TypeInt, "count should be int")
	})

	s.Run("(LONG) should not list UDTs in another keyspace", func(t *harness.T) {
		err := t.Db.CreateType(t.Ctx, "keyspace_specific_type", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetKeyspace(harness.TestKeyspaces[1]).SetIfNotExists(true))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		defaultTypes, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		otherTypes, err := t.Db.ListTypes(t.Ctx, options.ListTypes().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to list types: %v", err)

		idxDef := slices.IndexFunc(defaultTypes, func(u results.UDTDescriptor) bool {
			return u.Name == "keyspace_specific_type"
		})
		idxOth := slices.IndexFunc(otherTypes, func(u results.UDTDescriptor) bool {
			return u.Name == "keyspace_specific_type"
		})

		testlib.FailIf(t, idxDef != -1, "type should not exist in default keyspace")
		testlib.FailIf(t, idxOth == -1, "type should exist in other keyspace")
	})

	s.Run("(LONG) should add fields to UDT", func(t *harness.T) {
		_ = t.Db.DropType(t.Ctx, "alter_test_add", options.DropType().SetIfExists(true))

		err := t.Db.CreateType(t.Ctx, "alter_test_add", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "name", Column: table.Text()},
			},
		})
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.AlterType(t.Ctx, "alter_test_add", table.AddTypeFields{
			Fields: table.Columns{
				{Name: "age", Column: table.Int()},
				{Name: "active", Column: table.Boolean()},
			},
		})
		testlib.FailIfErr(t, err, "failed to alter type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "alter_test_add"
		})
		testlib.FailIf(t, idx == -1, "type should exist")

		u := types[idx]
		fieldTypes := make(map[string]string)
		for _, field := range u.Definition.Fields {
			fieldTypes[field.Name] = field.Column.Type
		}
		testlib.FailIf(t, fieldTypes["name"] != table.TypeText, "name should be text")
		testlib.FailIf(t, fieldTypes["age"] != table.TypeInt, "age should be int")
		testlib.FailIf(t, fieldTypes["active"] != table.TypeBoolean, "active should be boolean")
	})

	s.Run("(LONG) should rename fields in UDT", func(t *harness.T) {
		_ = t.Db.DropType(t.Ctx, "alter_test_rename", options.DropType().SetIfExists(true))

		err := t.Db.CreateType(t.Ctx, "alter_test_rename", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "old_name", Column: table.Text()},
				{Name: "zip_code", Column: table.Int()},
			},
		})
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.AlterType(t.Ctx, "alter_test_rename", table.RenameTypeFields{
			Fields: map[string]string{
				"old_name": "new_name",
				"zip_code": "postal_code",
			},
		})
		testlib.FailIfErr(t, err, "failed to alter type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx)
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "alter_test_rename"
		})
		testlib.FailIf(t, idx == -1, "type should exist")

		u := types[idx]
		fieldNames := make(map[string]bool)
		for _, field := range u.Definition.Fields {
			fieldNames[field.Name] = true
		}
		testlib.FailIf(t, !fieldNames["new_name"], "new_name should exist")
		testlib.FailIf(t, !fieldNames["postal_code"], "postal_code should exist")
		testlib.FailIf(t, fieldNames["old_name"], "old_name should not exist")
		testlib.FailIf(t, fieldNames["zip_code"], "zip_code should not exist")
	})

	s.Run("(LONG) should alter UDT in different keyspace", func(t *harness.T) {
		_ = t.Db.DropType(t.Ctx, "alter_test_keyspace", options.DropType().SetKeyspace(harness.TestKeyspaces[1]).SetIfExists(true))

		err := t.Db.CreateType(t.Ctx, "alter_test_keyspace", table.UDTDefinition{
			Fields: table.Columns{
				{Name: "field1", Column: table.Text()},
			},
		}, options.CreateType().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to create type: %v", err)

		err = t.Db.AlterType(t.Ctx, "alter_test_keyspace", table.AddTypeFields{
			Fields: table.Columns{
				{Name: "field2", Column: table.Int()},
			},
		}, options.AlterType().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to alter type: %v", err)

		types, err := t.Db.ListTypes(t.Ctx, options.ListTypes().SetKeyspace(harness.TestKeyspaces[1]))
		testlib.FailIfErr(t, err, "failed to list types: %v", err)
		idx := slices.IndexFunc(types, func(u results.UDTDescriptor) bool {
			return u.Name == "alter_test_keyspace"
		})
		testlib.FailIf(t, idx == -1, "type should exist")

		u := types[idx]
		fieldNames := make(map[string]bool)
		for _, field := range u.Definition.Fields {
			fieldNames[field.Name] = true
		}
		testlib.FailIf(t, !fieldNames["field2"], "field2 should exist")
	})
}
