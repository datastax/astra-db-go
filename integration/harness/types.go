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
