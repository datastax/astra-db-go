package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DeanPDX/dotconfig"
	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/table"
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

func main() {
	// Load our configuration and create a client and DB handle.
	config, err := dotconfig.FromFileName[Config](".env")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Configuration loaded successfully.")
	client := astra.NewClient(
		options.API().SetToken(config.ApplicationToken),
	)
	db := client.Database(config.DBEndpoint)
	fmt.Printf("Using database endpoint: %s\n", config.DBEndpoint)
	ctx := context.Background()

	tbl := createTable(ctx, db)
	insertRows(ctx, tbl)
	findOneRow(ctx, tbl)
	findAllRows(ctx, tbl)
	createAndListIndexes(ctx, tbl)
	dropTable(ctx, db)
}

func createTable(ctx context.Context, db *astra.Db) *astra.Table {
	logHeader("Creating Table")
	definition := table.Definition{
		Columns: table.Columns{
			{Name: "id", Column: table.Text()},
			{Name: "title", Column: table.Text()},
			{Name: "author", Column: table.Text()},
			{Name: "year", Column: table.Int()},
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy:   []string{"id"},
			PartitionSort: table.PartitionSort{{Name: "year", Order: table.SortDescending}},
		},
	}
	tbl, err := db.CreateTable(ctx, "books", definition)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created table: books")
	return tbl
}

func insertRows(ctx context.Context, tbl *astra.Table) {
	logHeader("Inserting Rows")
	_, err := tbl.InsertOne(ctx, astra.NewRow{
		"id": "1", "title": "The Go Programming Language", "author": "Alan Donovan", "year": 2015,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted: \"The Go Programming Language\" by Alan Donovan (2015)")

	// Insert multiple rows
	_, err = tbl.InsertMany(ctx, []astra.Row{
		astra.NewRow{"id": "2", "title": "Go in Action", "author": "William Kennedy", "year": 2015},
		astra.NewRow{"id": "3", "title": "Concurrency in Go", "author": "Katherine Cox-Buday", "year": 2017},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted 2 more rows via InsertMany.")
}

func findOneRow(ctx context.Context, tbl *astra.Table) {
	logHeader("Finding: id = '1'")
	var row astra.Row
	err := tbl.FindOne(ctx, filter.Eq("id", "1")).Decode(&row)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found: %s by %s (%v)\n", row.MustGet("title"), row.MustGet("author"), row.MustGet("year"))
}

func findAllRows(ctx context.Context, tbl *astra.Table) {
	logHeader("Finding All Rows")
	cursor := tbl.Find(nil)

	var rows []astra.Row
	if err := cursor.DecodeAll(ctx, &rows); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d result(s):\n", len(rows))
	for _, r := range rows {
		fmt.Printf("  - %s (%v)\n", r.MustGet("title"), r.MustGet("year"))
	}
}

func createAndListIndexes(ctx context.Context, tbl *astra.Table) {
	logHeader("Creating Indexes")
	err := tbl.CreateIndex(ctx, "author_idx", "author")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created index: author_idx on column \"author\"")

	logHeader("Listing Indexes")
	indexes, err := tbl.ListIndexes(ctx, options.ListIndexes().SetExplain(true))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d indexes:\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("  - %s (type: %s, columns: %v)\n", idx.Name, idx.IndexType, idx.Definition.Column)
	}
}

func dropTable(ctx context.Context, db *astra.Db) {
	logHeader("Dropping Table")
	err := db.DropTable(ctx, "books")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Dropped table: books")
}

func logHeader(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}
