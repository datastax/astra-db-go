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

package harness

import (
	"math/big"
	"net"
	"time"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/datastax/astra-db-go/astra/table"
)

type EverythingTable struct {
	Ascii     string             `json:"ascii" astra:"type=ascii"`
	BigInt    int64              `json:"bigint"`
	Blob      []byte             `json:"blob"`
	Boolean   bool               `json:"boolean"`
	Date      datatypes.DateOnly `json:"date"`
	Decimal   big.Float          `json:"decimal"`
	Double    float64            `json:"double"`
	Duration  datatypes.Duration `json:"duration"`
	Float     float32            `json:"float"`
	Int       int                `json:"int"`
	Inet      net.IP             `json:"inet"`
	SmallInt  int16              `json:"smallint"`
	Text      string             `json:"text" astra:"pk"`
	Time      datatypes.TimeOnly `json:"time"`
	Timestamp time.Time          `json:"timestamp"`
	TinyInt   int8               `json:"tinyint"`
	UUID      datatypes.UUID     `json:"uuid"`
	Varint    big.Int            `json:"varint"`
	Vector    datatypes.Vector   `json:"vector" astra:"dim=5"`
	UDT       any                `json:"udt" astra:"type=udt[example_udt]"`
}

type EverythingTableWithVectorize struct {
	Ascii     string             `json:"ascii" astra:"type=ascii"`
	BigInt    int64              `json:"bigint"`
	Blob      []byte             `json:"blob"`
	Boolean   bool               `json:"boolean"`
	Date      datatypes.DateOnly `json:"date"`
	Decimal   big.Float          `json:"decimal"`
	Double    float64            `json:"double"`
	Duration  datatypes.Duration `json:"duration"`
	Float     float32            `json:"float"`
	Int       int                `json:"int"`
	Inet      net.IP             `json:"inet"`
	SmallInt  int16              `json:"smallint"`
	Text      string             `json:"text" astra:"pk"`
	Time      datatypes.TimeOnly `json:"time"`
	Timestamp time.Time          `json:"timestamp"`
	TinyInt   int8               `json:"tinyint"`
	UUID      datatypes.UUID     `json:"uuid"`
	Varint    big.Int            `json:"varint"`
	Vector1   string             `json:"vector1" astra:"vectorize,provider=openai,model=text-embedding-3-small,dim=5"`
	Vector2   string             `json:"vector2" astra:"vectorize,provider=openai,model=text-embedding-3-small,dim=5"`
	UDT       any                `json:"udt" astra:"type=udt[example_udt]"`
}

var EverythingTableSchema = must(table.Infer[EverythingTable]())

var EverythingTableSchemaWithVectorize = must(table.Infer[EverythingTableWithVectorize]())

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
