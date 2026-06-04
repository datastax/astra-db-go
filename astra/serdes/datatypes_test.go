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

package serdes_test

import (
	"strings"
	"testing"

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/table"
	"github.com/datastax/astra-db-go/internal/testutils"
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
		target := targetsGen.Draw(t, "Target")

		encoded, err := serdes.Serialize(uuid, target)
		testutils.FailIfErr(t, err, "failed to serialize UUID")

		expected := `"` + uuid.String() + `"`
		if target == serdes.TargetCollection {
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

func TestSerdesUUIDS_Untyped_Table(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		uuid := uuidGen.Draw(t, "uuid")

		row := astra.NewRow{"id": uuid}
		targetCtx := astra.NewRowTargetCtx(table.Columns{{"id", table.UUID()}})

		encoded, err := serdes.Serialize(row, serdes.TargetTable)
		testutils.FailIfErr(t, err, "failed to serialize Row with UUID")

		expected := `{"id":"` + uuid.String() + `"}`
		testutils.FailIf(t, string(encoded) != expected, "unexpected serialized form: got %s, expected %s", encoded, expected)

		var decoded astra.Row
		err = serdes.Deserialize(encoded, &decoded, targetCtx, serdes.TargetTable)
		testutils.FailIfErr(t, err, "failed to deserialize Row with UUID")

		decodedVal, ok := decoded.ToMap()["id"]
		testutils.FailIf(t, !ok, "missing 'id' field after deserialization")
		testutils.FailIf(t, decodedVal != uuid, "mismatch after serdes: original %s, decoded %v", uuid, decodedVal)
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
		target := targetsGen.Filter(func(t serdes.Target) bool { return t != serdes.TargetCollection }).Draw(t, "Target")

		_, err := serdes.Serialize(oid, target)
		testutils.FailIf(t, err == nil, "expected error encoding ObjectId for non-collection Target")

		expectedErrPrefix := "serdes: unsupported value: ObjectId is only supported for collections"
		testutils.FailIf(t, !strings.HasPrefix(err.Error(), expectedErrPrefix), "unexpected error message: %v", err)
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

// ================================
// | Duration
// ================================

// durationGen generates valid Duration values (all components share a sign).
var durationGen = rapid.Custom(func(t *rapid.T) datatypes.Duration {
	negative := rapid.Bool().Draw(t, "negative")
	months := int32(rapid.Int64Range(0, int64(datatypes.MaxMonthsDays)).Draw(t, "months"))
	days := int32(rapid.Int64Range(0, int64(datatypes.MaxMonthsDays)).Draw(t, "days"))
	nanos := rapid.Int64Range(0, datatypes.MaxNanos).Draw(t, "nanos")
	if negative {
		months = -months
		days = -days
		nanos = -nanos
	}
	d, _ := datatypes.NewDuration(months, days, nanos)
	return d
})

func TestSerdesDuration_Typed_Table(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := durationGen.Draw(t, "duration")

		encoded, err := serdes.Serialize(d, serdes.TargetTable)
		testutils.FailIfErr(t, err, "failed to serialize Duration")

		var decoded datatypes.Duration
		err = serdes.Deserialize(encoded, &decoded, nil, serdes.TargetTable)
		testutils.FailIfErr(t, err, "failed to deserialize Duration")

		testutils.FailIf(t, !d.Equals(decoded), "round-trip mismatch: original %+v, decoded %+v", d, decoded)
	})
}

func TestSerdesDuration_Typed_None(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := durationGen.Draw(t, "duration")

		encoded, err := serdes.Serialize(d, serdes.TargetNone)
		testutils.FailIfErr(t, err, "failed to serialize Duration with TargetNone")

		var decoded datatypes.Duration
		err = serdes.Deserialize(encoded, &decoded, nil, serdes.TargetNone)
		testutils.FailIfErr(t, err, "failed to deserialize Duration with TargetNone")

		testutils.FailIf(t, !d.Equals(decoded), "round-trip mismatch: original %+v, decoded %+v", d, decoded)
	})
}

func TestSerdesDuration_Collection_Encode_Error(t *testing.T) {
	d := datatypes.MustParseDuration("1y2mo")
	_, err := serdes.Serialize(d, serdes.TargetCollection)
	testutils.FailIf(t, err == nil, "expected error encoding Duration for TargetCollection")
}

func TestSerdesDuration_Collection_Decode_Error(t *testing.T) {
	encoded, err := serdes.Serialize(datatypes.MustParseDuration("1y2mo"), serdes.TargetTable)
	testutils.FailIfErr(t, err, "failed to pre-encode Duration")

	var decoded datatypes.Duration
	err = serdes.Deserialize(encoded, &decoded, nil, serdes.TargetCollection)
	testutils.FailIf(t, err == nil, "expected error decoding Duration for TargetCollection")
}

func TestSerdesDuration_Struct_Table(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		type row struct {
			ID       string             `json:"id"`
			Duration datatypes.Duration `json:"duration"`
		}

		want := row{ID: "x", Duration: durationGen.Draw(t, "duration")}

		encoded, err := serdes.Serialize(want, serdes.TargetTable)
		testutils.FailIfErr(t, err, "failed to serialize row")

		var got row
		err = serdes.Deserialize(encoded, &got, nil, serdes.TargetTable)
		testutils.FailIfErr(t, err, "failed to deserialize row")

		testutils.FailIf(t, got.ID != want.ID || !got.Duration.Equals(want.Duration),
			"round-trip mismatch: original %+v, decoded %+v", want, got)
	})
}
