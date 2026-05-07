package serdes_test

import (
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/internal/testutils"
	"github.com/datastax/astra-db-go/serdes"
	"pgregory.net/rapid"
)

var targetsGen = rapid.SampledFrom([]serdes.Target{serdes.TargetNone, serdes.TargetCollection, serdes.TargetTable})

// ================================
// | UUIDs
// ================================

var uuidGen = rapid.Map(rapid.StringMatching(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), func(s string) datatypes.UUID {
	return datatypes.MustParseUUID(s)
})

func TestSerdesUUIDS_Typed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		uuid := uuidGen.Draw(t, "uuid")
		target := targetsGen.Draw(t, "target")

		encoded, err := serdes.Serialize(uuid, target)
		testutils.FailIfErr(t, err, "failed to serialize UUID")

		expected := `"` + uuid.String() + `"`
		if target.Is(serdes.TargetCollection) {
			expected = `{"$uuid":` + expected + `}`
		}
		testutils.FailIf(t, string(encoded) != expected, "unexpected serialized form: got %s, expected %s", encoded, expected)

		var decoded datatypes.UUID
		err = serdes.Deserialize(encoded, &decoded, nil, target)
		testutils.FailIfErr(t, err, "failed to deserialize UUID")

		testutils.FailIf(t, uuid != decoded, "mismatch after serdes: original %s, decoded %s", uuid, decoded)
	})
}

func TestSerdesUUIDS_Untyped_Collection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		uuid := uuidGen.Draw(t, "uuid")

		var uuidAny any = uuid

		encoded, err := serdes.Serialize(uuidAny, serdes.TargetCollection)
		testutils.FailIfErr(t, err, "failed to serialize UUID")

		expected := `{"$uuid":"` + uuid.String() + `"}`
		testutils.FailIf(t, string(encoded) != expected, "unexpected serialized form: got %s, expected %s", encoded, expected)

		var decoded any
		err = serdes.Deserialize(encoded, &decoded, nil, serdes.TargetCollection)
		testutils.FailIfErr(t, err, "failed to deserialize UUID")

		testutils.FailIf(t, decoded != uuidAny, "mismatch after serdes: original %s, decoded %v", uuid, decoded)
	})
}

// ================================
// | ObjectIDs
// ================================

var oidGen = rapid.Map(rapid.StringMatching(`^[0-9a-f]{24}$`), func(s string) datatypes.ObjectId {
	return datatypes.MustParseObjectId(s)
})

func TestObjectIds_Typed_Collection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		oid := oidGen.Draw(t, "oid")

		encoded, err := serdes.Serialize(oid, serdes.TargetCollection)
		testutils.FailIfErr(t, err, "failed to serialize ObjectId")

		expected := `{"$objectId":"` + oid.String() + `"}`
		testutils.FailIf(t, string(encoded) != expected, "unexpected serialized form: got %s, expected %s", encoded, expected)

		var decoded datatypes.ObjectId
		err = serdes.Deserialize(encoded, &decoded, nil, serdes.TargetCollection)
		testutils.FailIfErr(t, err, "failed to deserialize ObjectId")

		testutils.FailIf(t, oid != decoded, "mismatch after serdes: original %s, decoded %s", oid, decoded)
	})
}

func TestObjectIds_Typed_NonCollection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		oid := oidGen.Draw(t, "oid")
		target := targetsGen.Filter(func(t serdes.Target) bool { return !t.Is(serdes.TargetCollection) }).Draw(t, "target")

		_, err := serdes.Serialize(oid, target)
		testutils.FailIf(t, err == nil, "expected error encoding ObjectId for non-collection target")

		expectedErr := "cannot encode ObjectId in a non-collection"
		testutils.FailIf(t, err.Error() != expectedErr, "unexpected error message: %v", err)
	})
}

func TestObjectIds_Untyped_Collection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		oid := oidGen.Draw(t, "oid")
		var oidAny any = oid

		encoded, err := serdes.Serialize(oidAny, serdes.TargetCollection)
		testutils.FailIfErr(t, err, "failed to serialize ObjectId via any")

		expected := `{"$objectId":"` + oid.String() + `"}`
		testutils.FailIf(t, string(encoded) != expected, "unexpected serialized form: got %s, expected %s", encoded, expected)

		var decoded any
		err = serdes.Deserialize(encoded, &decoded, nil, serdes.TargetCollection)
		testutils.FailIfErr(t, err, "failed to deserialize ObjectId via any")

		testutils.FailIf(t, decoded != oidAny, "mismatch after serdes: original %v, decoded %v", oidAny, decoded)
	})
}
