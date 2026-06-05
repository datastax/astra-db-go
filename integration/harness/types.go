package harness

import "github.com/datastax/astra-db-go/astra/table"

var ExampleUDTSchema = table.Definition{
	Columns: table.Columns{
		{"name", table.Text()},
		{"age", table.Varint()},
		{"id", table.UUID()},
	},
}

var EverythingTableSchema = table.Definition{
	Columns: table.Columns{
		{"ascii", table.Ascii()},
		{"bigint", table.BigInt()},
		{"blob", table.Blob()},
		{"boolean", table.Boolean()},
		{"date", table.Date()},
		{"decimal", table.Decimal()},
		{"double", table.Double()},
		{"duration", table.Duration()},
		{"float", table.Float()},
		{"int", table.Int()},
		{"inet", table.Inet()},
		{"smallint", table.SmallInt()},
		{"text", table.Text()},
		{"time", table.Time()},
		{"timestamp", table.Timestamp()},
		{"tinyint", table.TinyInt()},
		{"uuid", table.UUID()},
		{"varint", table.Varint()},
		{"vector", table.Vector(5)},
		{"udt", table.UDT(DefaultUDTName)},
	},
	PrimaryKey: table.PrimaryKey{
		PartitionBy: []string{"text"},
	},
}

var EverythingTableSchemaWithVectorize = table.Definition{
	Columns: table.Columns{
		{"ascii", table.Ascii()},
		{"bigint", table.BigInt()},
		{"blob", table.Blob()},
		{"boolean", table.Boolean()},
		{"date", table.Date()},
		{"decimal", table.Decimal()},
		{"double", table.Double()},
		{"duration", table.Duration()},
		{"float", table.Float()},
		{"int", table.Int()},
		{"inet", table.Inet()},
		{"smallint", table.SmallInt()},
		{"text", table.Text()},
		{"time", table.Time()},
		{"timestamp", table.Timestamp()},
		{"tinyint", table.TinyInt()},
		{"uuid", table.UUID()},
		{"varint", table.Varint()},
		{"vector1", table.VectorWithService(5, &table.VectorService{
			Provider:  "openai",
			ModelName: "text-embedding-3-small",
		})},
		{"vector2", table.VectorWithService(5, &table.VectorService{
			Provider:  "openai",
			ModelName: "text-embedding-3-small",
		})},
		{"udt", table.UDT(DefaultUDTName)},
	},
	PrimaryKey: table.PrimaryKey{
		PartitionBy: []string{"text"},
	},
}
