package collection_test

import (
	"fmt"

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/integration/harness"
	"github.com/datastax/astra-db-go/internal/testlib"
)

func init() {
	s := harness.ParallelSuite("collection.insert-one")

	s.Run("should insert a document with IDs of all kinds", func(t harness.T) {
		ids := []any{
			"hi",
			nil,
			3,
			datatypes.NewObjectId(),
			datatypes.NewUUIDv4(),
			datatypes.NewUUIDv7(),
		}

		got := testlib.AwaitAll(t, ids, func(id any) (any, error) {
			res, err := t.Collection.InsertOne(t.Ctx, astra.NewDocument{"_id": id})
			if err != nil {
				return nil, fmt.Errorf("InsertOne failed for id %v: %v", id, err)
			}
			return insertOneDecodeAny(res), nil
		})

		t.NoDiff(ids, got)
	})
}

func insertOneDecodeAny(res *results.InsertOneResult) any {
	var id any
	if err := res.DecodeID(&id); err != nil {
		panic("failed to decode inserted ID: " + err.Error())
	}
	return id
}
