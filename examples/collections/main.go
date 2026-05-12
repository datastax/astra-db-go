package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

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
	// Parse command-line flags
	readonly := flag.Bool("readonly", false, "Use existing collection in read-only mode instead of creating/populating/dropping a new one")
	debug := flag.Bool("debug", false, "Verbose debug logs")
	flag.Parse()

	if *debug {
		// Set log level to DEBUG
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

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

	var coll *astradb.Collection
	if *readonly {
		coll = getCollection(ctx, db)
	} else {
		coll = createCollection(ctx, db)
		insertDocuments(ctx, coll)
	}

	findOneDocument(ctx, coll)
	findAllDocuments(ctx, coll)
	filterDocumentsByYear(ctx, coll)
	countDocuments(ctx, coll)
	if !(*readonly) {
		dropCollection(ctx, db)
	}
}

func createCollection(ctx context.Context, db *astradb.Db) *astradb.Collection {
	logHeader("Creating Collection")
	coll, err := db.CreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created collection: my_collection")
	return coll
}

func getCollection(ctx context.Context, db *astradb.Db) *astradb.Collection {
	logHeader("Getting Collection")
	coll := db.Collection("my_collection")
	logHeader("Gotten collection")
	return coll
}

func insertDocuments(ctx context.Context, coll *astradb.Collection) {
	logHeader("Inserting Documents")
	resp, err := coll.InsertOne(ctx, Book{
		Title:  "The Go Programming Language",
		Author: "Alan Donovan",
		Year:   2015,
	})
	if err != nil {
		log.Fatal(err)
	}
	var insertedId string
	if err := resp.DecodeID(&insertedId); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted: \"The Go Programming Language\" by Alan Donovan (2015)")
	fmt.Println("Inserted ID:", insertedId)

	// Insert multiple documents
	respMany, err := coll.InsertMany(ctx, []Book{
		{Title: "Go in Action", Author: "William Kennedy", Year: 2015},
		{Title: "Concurrency in Go", Author: "Katherine Cox-Buday", Year: 2017},
		{Title: "Learning Go: An Idiomatic Approach to Real-World Go Programming", Author: "Jon Bodner", Year: 2024},
	})
	if err != nil {
		log.Fatal(err)
	}
	var insertedIds []string
	if err := respMany.DecodeIDs(&insertedIds); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted %d more documents via InsertMany:\n", respMany.InsertedCount())
	for _, id := range insertedIds {
		fmt.Printf("  - %s\n", id)
	}
}

func findOneDocument(ctx context.Context, coll *astradb.Collection) {
	logHeader("Finding: title = 'The Go Programming Language'")
	var result Book
	err := coll.FindOne(ctx, filter.Eq("title", "The Go Programming Language")).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found: %s by %s (%d)\n", result.Title, result.Author, result.Year)
}

func findAllDocuments(ctx context.Context, coll *astradb.Collection) {
	logHeader("Finding All Documents")
	cursor := coll.Find(filter.F{})

	var books []Book
	if err := cursor.DecodeAll(ctx, &books); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d result(s):\n", len(books))
	for _, b := range books {
		fmt.Printf("  - %s (%d)\n", b.Title, b.Year)
	}
}

func filterDocumentsByYear(ctx context.Context, coll *astradb.Collection) {
	logHeader("Filtering: year >= 2016")
	cursor := coll.Find(filter.F{"year": filter.F{"$gte": 2016}})

	for cursor.Next(ctx) {
		var book Book
		if err := cursor.Decode(&book); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  - %s by %s (%d)\n", book.Title, book.Author, book.Year)
	}
	if err := cursor.Err(); err != nil {
		log.Fatal(err)
	}
}

func countDocuments(ctx context.Context, coll *astradb.Collection) {
	logHeader("Counting Documents")
	count, err := coll.CountDocuments(ctx, filter.F{}, 1000)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Total documents:", count)
}

func dropCollection(ctx context.Context, db *astradb.Db) {
	logHeader("Dropping Collection")
	if err := db.DropCollection(ctx, "my_collection"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Collection dropped.")
}

func logHeader(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}
