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
	"encoding/base64"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/table"
)

func TestSerdes_UUID(t *testing.T) {
	type uuidStruct struct {
		UUID datatypes.UUID `json:"uuid"`
	}

	validUUID, _ := datatypes.ParseUUID("123e4567-e89b-12d3-a456-426614174000")
	uuidUntypedRow := astra.NewRow{"id": validUUID}
	uuidUntypedRowCtx := astra.NewRowTargetCtx(table.Columns{{"id", table.UUID()}})
	testStruct := uuidStruct{UUID: validUUID}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Collection Target",
			Target:  serdes.TargetCollection,
			Value:   validUUID,
			Encoded: `{"$uuid":"123e4567-e89b-12d3-a456-426614174000"}`,
		},
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validUUID,
			Encoded: `"123e4567-e89b-12d3-a456-426614174000"`,
		},
		{
			Name:        "Invalid string decode table",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: datatypes.UUID{},
			Encoded:     `"invalid"`,
			DecodeErr:   "invalid UUID string",
		},
		{
			Name:        "Collection missing brace",
			Target:      serdes.TargetCollection,
			SkipEncode:  true,
			DecodeValue: datatypes.UUID{},
			Encoded:     `{"$uuid":"123e4567-e89b-12d3-a456-426614174000"`,
			DecodeErr:   "expected '}'",
		},
		{
			Name:        "Collection invalid key",
			Target:      serdes.TargetCollection,
			SkipEncode:  true,
			DecodeValue: datatypes.UUID{},
			Encoded:     `{"$not":"123e4567-e89b-12d3-a456-426614174000"}`,
			DecodeErr:   "expected \"$uuid\" key",
		},
		{
			Name:        "Collection missing quotes",
			Target:      serdes.TargetCollection,
			SkipEncode:  true,
			DecodeValue: datatypes.UUID{},
			Encoded:     `{"$uuid":123}`,
			DecodeErr:   "expected string",
		},
		{
			Name:      "Untyped Row Table Target",
			Target:    serdes.TargetTable,
			Value:     uuidUntypedRow,
			Ptr:       new(astra.Row),
			TargetCtx: uuidUntypedRowCtx,
			Encoded:   `{"id":"123e4567-e89b-12d3-a456-426614174000"}`,
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"uuid":"123e4567-e89b-12d3-a456-426614174000"}`,
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"uuid":{"$uuid":"123e4567-e89b-12d3-a456-426614174000"}}`,
		},
	})
}

func TestSerdes_ObjectId(t *testing.T) {
	type objectIdStruct struct {
		ObjectId datatypes.ObjectId `json:"objectId"`
	}

	validOid, _ := datatypes.ParseObjectId("507f1f77bcf86cd799439011")
	testStruct := objectIdStruct{ObjectId: validOid}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Collection Target",
			Target:  serdes.TargetCollection,
			Value:   validOid,
			Encoded: `{"$objectId":"507f1f77bcf86cd799439011"}`,
		},
		{
			Name:      "Table Target Err",
			Target:    serdes.TargetTable,
			Value:     validOid,
			EncodeErr: "ObjectId is only supported for collections",
			DecodeErr: "ObjectId is only supported for collections",
		},
		{
			Name:        "Invalid string decode collection",
			Target:      serdes.TargetCollection,
			SkipEncode:  true,
			DecodeValue: datatypes.ObjectId{},
			Encoded:     `{"$objectId":"invalid"}`,
			DecodeErr:   "invalid ObjectId string",
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"objectId":{"$objectId":"507f1f77bcf86cd799439011"}}`,
		},
	})
}

func TestSerdes_Time(t *testing.T) {
	type timeStruct struct {
		Time time.Time `json:"time"`
	}

	validTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	testStruct := timeStruct{Time: validTime}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Collection Target",
			Target:  serdes.TargetCollection,
			Value:   validTime,
			Encoded: `{"$date":1672574400000}`,
		},
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validTime,
			Encoded: `"2023-01-01T12:00:00Z"`,
		},
		{
			Name:        "Invalid string decode table",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: time.Time{},
			Encoded:     `"invalid"`,
			DecodeErr:   "invalid timestamp string",
		},
		{
			Name:        "Invalid number decode collection",
			Target:      serdes.TargetCollection,
			SkipEncode:  true,
			DecodeValue: time.Time{},
			Encoded:     `{"$date":"invalid"}`,
			DecodeErr:   "syntax error",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"time":"2023-01-01T12:00:00Z"}`,
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"time":{"$date":1672574400000}}`,
		},
	})
}

func TestSerdes_DateOnly(t *testing.T) {
	type dateOnlyStruct struct {
		DateOnly datatypes.DateOnly `json:"dateOnly"`
	}

	validDate, _ := datatypes.ParseDateOnly("2023-01-01")
	testStruct := dateOnlyStruct{DateOnly: validDate}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validDate,
			Encoded: `"2023-01-01"`,
		},
		{
			Name:      "Collection Target Err",
			Target:    serdes.TargetCollection,
			Value:     validDate,
			EncodeErr: "DateOnly is not supported for collections",
			DecodeErr: "DateOnly is not supported for collections",
		},
		{
			Name:        "Invalid decode table",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: datatypes.DateOnly{},
			Encoded:     `"invalid"`,
			DecodeErr:   "invalid date string",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"dateOnly":"2023-01-01"}`,
		},
	})
}

func TestSerdes_TimeOnly(t *testing.T) {
	type timeOnlyStruct struct {
		TimeOnly datatypes.TimeOnly `json:"timeOnly"`
	}

	validTimeOnly, _ := datatypes.ParseTimeOnly("12:34:56.123456789")
	testStruct := timeOnlyStruct{TimeOnly: validTimeOnly}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validTimeOnly,
			Encoded: `"12:34:56.123456789"`,
		},
		{
			Name:      "Collection Target Err",
			Target:    serdes.TargetCollection,
			Value:     validTimeOnly,
			EncodeErr: "TimeOnly is not supported for collections",
			DecodeErr: "TimeOnly is not supported for collections",
		},
		{
			Name:        "Invalid decode table",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: datatypes.TimeOnly{},
			Encoded:     `"invalid"`,
			DecodeErr:   "invalid time string",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"timeOnly":"12:34:56.123456789"}`,
		},
	})
}

func TestSerdes_BigInt(t *testing.T) {
	type bigIntStruct struct {
		BigInt big.Int `json:"bigInt"`
	}

	validBigInt := *big.NewInt(123456789)
	testStruct := bigIntStruct{BigInt: validBigInt}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "All Targets",
			Target:  serdes.TargetTable,
			Value:   validBigInt,
			Encoded: `123456789`,
		},
		{
			Name:        "Decode Null",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: big.Int{},
			Encoded:     `null`,
		},
		{
			Name:        "Invalid Decode",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: big.Int{},
			Encoded:     `"invalid"`,
			DecodeErr:   "expected number",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"bigInt":123456789}`,
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"bigInt":123456789}`,
		},
	})
}

func TestSerdes_BigFloat(t *testing.T) {
	type bigFloatStruct struct {
		BigFloat big.Float `json:"bigFloat"`
	}

	validBigFloat, _, _ := big.ParseFloat("123.45", 10, 64, big.ToNearestEven)
	testStruct := bigFloatStruct{BigFloat: *validBigFloat}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "All Targets",
			Target:  serdes.TargetTable,
			Value:   *validBigFloat,
			Encoded: `123.45`,
		},
		{
			Name:        "Decode Null",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: big.Float{},
			Encoded:     `null`,
		},
		{
			Name:        "Invalid Decode",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: big.Float{},
			Encoded:     `"invalid"`,
			DecodeErr:   "expected number",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"bigFloat":123.45}`,
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"bigFloat":123.45}`,
		},
	})
}

func TestSerdes_Vector(t *testing.T) {
	type vectorStruct struct {
		Vector datatypes.Vector `json:"vector"`
	}

	validVector := datatypes.NewVector([]float32{1.0, 2.0, 3.0})
	testStruct := vectorStruct{Vector: validVector}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Encode Base64",
			Target:  serdes.TargetTable,
			Value:   validVector,
			Encoded: `{"$binary":"P4AAAEAAAABAQAAA"}`,
		},
		{
			Name:        "Decode Array",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: validVector,
			Encoded:     `[1.0, 2.0, 3.0]`,
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"vector":{"$binary":"P4AAAEAAAABAQAAA"}}`,
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"vector":{"$binary":"P4AAAEAAAABAQAAA"}}`,
		},
	})
}

func TestSerdes_Binary(t *testing.T) {
	type binaryStruct struct {
		Binary []byte `json:"binary"`
	}

	validBytes := []byte("hello")
	testStruct := binaryStruct{Binary: validBytes}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Collection Target",
			Target:  serdes.TargetCollection,
			Value:   validBytes,
			Encoded: `{"$binary":"` + base64.StdEncoding.EncodeToString(validBytes) + `"}`,
		},
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validBytes,
			Encoded: `{"$binary":"` + base64.StdEncoding.EncodeToString(validBytes) + `"}`,
		},
		{
			Name:    "None Target",
			Target:  serdes.TargetNone,
			Value:   validBytes,
			Encoded: `"` + base64.StdEncoding.EncodeToString(validBytes) + `"`,
		},
		{
			Name:        "Decode Array",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: validBytes,
			Encoded:     `[104, 101, 108, 108, 111]`,
		},
		{
			Name:        "Decode Null",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: []byte(nil),
			Ptr:         new([]byte),
			Encoded:     `null`,
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"binary":{"$binary":"aGVsbG8="}}`,
		},
		{
			Name:    "Struct Collection Target",
			Target:  serdes.TargetCollection,
			Value:   testStruct,
			Encoded: `{"binary":{"$binary":"aGVsbG8="}}`,
		},
	})
}

func TestSerdes_Duration(t *testing.T) {
	type durationStruct struct {
		Duration datatypes.Duration `json:"duration"`
	}

	validDuration, _ := datatypes.ParseDuration("14mo")
	testStruct := durationStruct{Duration: validDuration}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validDuration,
			Encoded: `"14mo"`,
		},
		{
			Name:      "Collection Target Err",
			Target:    serdes.TargetCollection,
			Value:     validDuration,
			EncodeErr: "Duration is not supported for collections",
			DecodeErr: "Duration is not supported for collections",
		},
		{
			Name:        "Invalid decode",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: datatypes.Duration{},
			Encoded:     `"invalid"`,
			DecodeErr:   "invalid standard duration string",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"duration":"14mo"}`,
		},
	})
}

func TestSerdes_IP(t *testing.T) {
	type ipStruct struct {
		IP net.IP `json:"ip"`
	}

	validIP := net.ParseIP("192.168.1.1")
	testStruct := ipStruct{IP: validIP}

	runSerdesTests(t, []serdesTestCase{
		{
			Name:    "Table Target",
			Target:  serdes.TargetTable,
			Value:   validIP,
			Encoded: `"192.168.1.1"`,
		},
		{
			Name:      "Collection Target Err",
			Target:    serdes.TargetCollection,
			Value:     validIP,
			EncodeErr: "net.IP is not supported for collections",
			DecodeErr: "net.IP is not supported for collections",
		},
		{
			Name:        "Invalid decode",
			Target:      serdes.TargetTable,
			SkipEncode:  true,
			DecodeValue: net.IP{},
			Encoded:     `"invalid"`,
			DecodeErr:   "invalid IP string",
		},
		{
			Name:    "Struct Table Target",
			Target:  serdes.TargetTable,
			Value:   testStruct,
			Encoded: `{"ip":"192.168.1.1"}`,
		},
	})
}
