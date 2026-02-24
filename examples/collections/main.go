package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DeanPDX/dotconfig"
	astradb "github.com/datastax/astra-db-go"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/options"
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

// Book represents a document in the collection.
type Book struct {
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
	fmt.Println("Configuration loaded successfully.")
	client := astradb.NewClient(
		options.WithToken(config.ApplicationToken),
	)
	db := client.Database(config.DBEndpoint)
	fmt.Printf("Using database endpoint: %s\n", config.DBEndpoint)
	ctx := context.Background()

	logHeader("Creating Collection")
	coll, err := db.CreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created collection: my_collection")

	// Insert a single document
	logHeader("Inserting Documents")
	resp, err := coll.InsertOne(ctx, Book{
		Title:  "The Go Programming Language",
		Author: "Alan Donovan",
		Year:   2015,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted: \"The Go Programming Language\" by Alan Donovan (2015)")
	fmt.Println("Inserted IDs", resp.Status.InsertedIds)

	// Insert multiple documents
	resp, err = coll.InsertMany(ctx, []Book{
		{Title: "Go in Action", Author: "William Kennedy", Year: 2015},
		{Title: "Concurrency in Go", Author: "Katherine Cox-Buday", Year: 2017},
		{Title: "Learning Go: An Idiomatic Approach to Real-World Go Programming", Author: "Jon Bodner", Year: 2024},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted %d more documents via InsertMany:\n", len(resp.Status.InsertedIds))
	for _, id := range resp.Status.InsertedIds {
		fmt.Printf("  - %s\n", id)
	}

	logHeader("Finding Documents")
	var result Book
	err = coll.FindOne(ctx, filter.Eq("title", "The Go Programming Language")).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found: %s by %s (%d)\n", result.Title, result.Author, result.Year)

	cursor := coll.Find(ctx, filter.F{})
	defer cursor.Close(ctx)

	var books []Book
	if err = cursor.All(ctx, &books); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d result(s):\n", len(books))
	for _, b := range books {
		fmt.Printf("  - %s (%d)\n", b.Title, b.Year)
	}

	logHeader("Filtering: year >= 2016")
	second := coll.Find(ctx, filter.F{"year": filter.F{"$gte": 2016}})
	defer second.Close(ctx)

	for second.Next(ctx) {
		var book Book
		if err := second.Decode(&book); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  - %s by %s (%d)\n", book.Title, book.Author, book.Year)
	}
	if err := second.Err(); err != nil {
		log.Fatal(err)
	}

	logHeader("Counting Documents")
	count, err := coll.CountDocuments(ctx, filter.F{}, 1000)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Total documents:", count)

	logHeader("Dropping Collection")
	if err = db.DropCollection(ctx, "my_collection"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Collection dropped.")
}

func logHeader(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}
