package tables

import (
	"context"
	"fmt"
	"log"

	"github.com/DeanPDX/dotconfig"
	astradb "github.com/datastax/astra-db-go"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/table"
)

// Config is a struct for retrieving configuration from environment variables.
// The `env` tags specify the environment variable names to look for when
// populating this struct. If you have the [Astra CLI] installed, you can run
// the following command to generate a .env file with the necessary variables:
//
//	astra db create-dotenv mydatabase
//
// [Astra CLI]: https://docs.datastax.com/en/astra-cli/index.html
type Config struct {
	DBEndpoint       string `env:"ASTRA_DB_API_ENDPOINT"`
	ApplicationToken string `env:"ASTRA_DB_APPLICATION_TOKEN"`
}

// BookRow represents a row in the "books" table.
type BookRow struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

func main() {
	// Load our configuration and create a client and DB handle.
	config, err := dotconfig.FromFileName[Config](".env")
	if err != nil {
		log.Fatal(err)
	}
	client := astradb.NewClient(
		options.WithToken(config.ApplicationToken),
	)
	db := client.Database(config.DBEndpoint)
	ctx := context.Background()

	// Define our table schema and create the table.
	definition := table.Definition{
		Columns: map[string]table.Column{
			"id":     table.Text(),
			"title":  table.Text(),
			"author": table.Text(),
			"year":   table.Int(),
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy:   []string{"id"},
			PartitionSort: map[string]int{"year": -1},
		},
	}
	tbl, err := db.CreateTable(ctx, "books", definition, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Insert a single row
	_, err = tbl.InsertOne(ctx, BookRow{
		ID: "1", Title: "The Go Programming Language", Author: "Alan Donovan", Year: 2015,
	})

	// Insert multiple rows
	_, err = tbl.InsertMany(ctx, []BookRow{
		{ID: "2", Title: "Go in Action", Author: "William Kennedy", Year: 2015},
		{ID: "3", Title: "Concurrency in Go", Author: "Katherine Cox-Buday", Year: 2017},
	})

	// Find Rows
	var row BookRow
	err = tbl.FindOne(ctx, filter.Eq("id", "1")).Decode(&row)

	cursor := tbl.Find(ctx, filter.F{"author": "Alan Donovan"})
	defer cursor.Close(ctx)

	var rows []BookRow
	if err = cursor.All(ctx, &rows); err != nil {
		log.Fatal(err)
	}

	// Creating Indexes

	// Standard index on a column
	err = tbl.CreateIndex(ctx, "author_idx", "author")

	// Vector index for similarity search
	err = tbl.CreateVectorIndex(ctx, "embedding_idx", "embedding")

	// List existing indexes
	indexes, err := tbl.ListIndexes(ctx)

	fmt.Println(indexes)

}
