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
	ID     string `json:"_id"`
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

	coll, err := db.CreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatal(err)
	}

	// Insert a single document
	_, err = coll.InsertOne(ctx, Book{
		ID:     "1",
		Title:  "The Go Programming Language",
		Author: "Alan Donovan",
		Year:   2015,
	})

	// Insert multiple documents
	_, err = coll.InsertMany(ctx, []Book{
		{ID: "2", Title: "Go in Action", Author: "William Kennedy", Year: 2015},
		{ID: "3", Title: "Concurrency in Go", Author: "Katherine Cox-Buday", Year: 2017},
	})

	var result Book
	err = coll.FindOne(ctx, filter.Eq("_id", "1")).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	cursor := coll.Find(ctx, filter.F{"author": "Alan Donovan"})
	defer cursor.Close(ctx)

	var books []Book
	if err = cursor.All(ctx, &books); err != nil {
		log.Fatal(err)
	}

	second := coll.Find(ctx, filter.F{"year": filter.F{"$gte": 2016}})
	defer second.Close(ctx)

	for second.Next(ctx) {
		var book Book
		if err := second.Decode(&book); err != nil {
			log.Fatal(err)
		}
		fmt.Println(book.Title)
	}
	if err := second.Err(); err != nil {
		log.Fatal(err)
	}
	count, err := coll.CountDocuments(ctx, filter.F{}, 1000)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Total documents:", count)
}
