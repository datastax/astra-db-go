# Astra DB Go Client

[![License Apache2](https://img.shields.io/hexpm/l/plug.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/datastax/astra-db-go)](https://goreportcard.com/report/github.com/datastax/astra-db-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/datastax/astra-db-go.svg)](https://pkg.go.dev/github.com/datastax/astra-db-go)
[![Documentation](https://img.shields.io/badge/Docs-datastax.com-blueviolet.svg)](https://docs.datastax.com/en/astra-db-serverless/api-reference/dataapiclient.html)

The official Go client for [Astra DB](https://www.ibm.com/products/datastax), DataStax's cloud-native database. It provides access to both the [Astra DB Data API](https://docs.datastax.com/en/astra-db-serverless/api-reference/dataapiclient.html) (collections and tables) and the [Astra DevOps API](https://docs.datastax.com/en/astra-api-docs/_attachments/devops-api/index.html) (database administration).

> **Note:** This client is in active development. Until it reaches a stable v1.0, it will contain breaking changes.

## Requirements

- Go 1.24 or higher. [Download it here](https://go.dev/doc/install).
- An [Astra DB](https://astra.datastax.com/) account with an Application Token. Don't have an account? [Create one](https://astra.datastax.com/signup).

## Quickstart

In a folder with a [Go module](https://go.dev/blog/using-go-modules), use `go get` to add the client to your project:

```bash
go get github.com/datastax/astra-db-go
```

## Usage

You will need a token to test with. From the [Astra Console](https://astra.datastax.com/), go to `Settings` > `Tokens`. Select a role, and click "Generate token". In these contrived examples, the token is represented as a string. **Never check the token in to source control or share it**. Use a secret manager to expose the token to your client.

Check out [./examples](./examples/) for runnable examples and instructions on how to run them.

### Creating a Client

Import the package and create a client with your Application Token:

```go
import (
    "github.com/datastax/astra-db-go/v2/astra"
    "github.com/datastax/astra-db-go/v2/astra/options"
)

client := astra.NewClient(
    options.WithToken("AstraCS:..."),
)
```

Get a handle to a database using its API endpoint:

```go
db := client.Database("https://<ASTRA_DB_ID>-<REGION>.apps.astra.datastax.com")
```

### Collections (Document API)

Collections store schema-flexible JSON documents, similar to MongoDB.

#### Creating a Collection

```go
ctx := context.Background()

coll, err := db.CreateCollection(ctx, "my_collection")
if err != nil {
    log.Fatal(err)
}
```

#### Inserting Documents

```go
type Book struct {
    ID     string `json:"_id"`
    Title  string `json:"title"`
    Author string `json:"author"`
    Year   int    `json:"year"`
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
```

#### Finding Documents

Use `FindOne` to retrieve a single document:

```go
import "github.com/datastax/astra-db-go/v2/astra/filter"

var result Book
err = coll.FindOne(ctx, filter.Eq("_id", "1")).Decode(&result)
if err != nil {
    log.Fatal(err)
}
```

Use `Find` to retrieve multiple documents and iterate with a cursor:

```go
cursor := coll.Find(filter.F{"author": "Alan Donovan"})
defer cursor.Close()

var books []Book
if err = cursor.All(ctx, &books); err != nil {
    log.Fatal(err)
}
```

Or iterate document by document:

```go
cursor := coll.Find(filter.F{"year": filter.F{"$gte": 2016}})
defer cursor.Close()

for cursor.Next(ctx) {
    var book Book
    if err := cursor.Decode(&book); err != nil {
        log.Fatal(err)
    }
    fmt.Println(book.Title)
}
if err := cursor.Err(); err != nil {
    log.Fatal(err)
}
```

#### Counting Documents

```go
count, err := coll.CountDocuments(ctx, filter.F{}, 1000)
```

### Filtering

The `filter` package provides a composable set of operators for querying collections and tables.

```go
import "github.com/datastax/astra-db-go/v2/astra/filter"

// Equality
filter.Eq("status", "active")

// Comparison
filter.Lt("year", 2020)
filter.Gte("rating", 4.5)

// Membership
filter.In("genre", "fiction", "mystery", "sci-fi")

// Logical composition
filter.And(
    filter.Eq("status", "active"),
    filter.Gte("year", 2000),
)

filter.Or(
    filter.Eq("genre", "fiction"),
    filter.Eq("genre", "mystery"),
)

// Raw map syntax
filter.F{"title": "The Go Programming Language"}
```

### Tables (Table API)

Tables provide a structured, schema-enforced data model backed by Cassandra's CQL engine.

#### Defining and Creating a Table

```go
import (
    "github.com/datastax/astra-db-go/v2/astra/table"
)

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

tbl, err := db.CreateTable(ctx, "books", definition, nil)
if err != nil {
    log.Fatal(err)
}
```

#### Inserting Rows

```go
type BookRow struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Author string `json:"author"`
    Year   int    `json:"year"`
}

_, err = tbl.InsertOne(ctx, BookRow{
    ID: "1", Title: "The Go Programming Language", Author: "Alan Donovan", Year: 2015,
})
```

#### Finding Rows

```go
var row BookRow
err = tbl.FindOne(ctx, filter.Eq("id", "1")).Decode(&row)

cursor := tbl.Find(filter.F{"author": "Alan Donovan"})
defer cursor.Close()

var rows []BookRow
if err = cursor.All(ctx, &rows); err != nil {
    log.Fatal(err)
}
```

#### Creating Indexes

```go
// Standard index on a column
err = tbl.CreateIndex(ctx, "author_idx", "author")

// Vector index for similarity search
err = tbl.CreateVectorIndex(ctx, "embedding_idx", "embedding")

// List existing indexes
indexes, err := tbl.ListIndexes(ctx)
```

### Admin Operations

#### Astra Admin

```go
admin, err := client.Admin()
if err != nil {
    log.Fatal(err)
}

// List all databases
databases, err := admin.ListDatabases(ctx)

// Get a specific database
db, err := admin.GetDatabase(ctx, "my-database-id")

// Find available regions
regions, err := admin.FindAvailableRegions(ctx)
```

#### Database Admin

```go
dbAdmin := admin.DatabaseAdmin("my-database-id")

// Manage keyspaces
keyspaces, err := dbAdmin.ListKeyspaces(ctx)
err = dbAdmin.CreateKeyspace(ctx, "my_keyspace")
err = dbAdmin.DropKeyspace(ctx, "old_keyspace")

// Database info and lifecycle
info, err := dbAdmin.Info(ctx)
err = dbAdmin.Drop(ctx)
```

You can also get a `DatabaseAdmin` from an existing `Db` handle:

```go
db := client.Database("https://...")
dbAdmin, err := db.DatabaseAdmin()
```

### Options and Configuration

Options can be specified at the client, database, collection, or table level. More specific options override broader ones.

```go
import "github.com/datastax/astra-db-go/v2/astra/options"

// Client-level defaults
client := astra.NewClient(
    options.WithToken("AstraCS:..."),
    options.WithTimeout(30 * time.Second),
    options.WithKeyspace("my_keyspace"),
)

// Override at the database level
db := client.Database(endpoint,
    options.WithKeyspace("another_keyspace"),
)

// Custom HTTP client
httpClient := &http.Client{Timeout: 60 * time.Second}
client := astra.NewClient(
    options.WithToken("AstraCS:..."),
    options.WithHTTPClient(httpClient),
)

// Warning handler
client := astra.NewClient(
    options.WithToken("AstraCS:..."),
    options.WithWarningHandler(func(warnings results.Warnings) {
        for _, w := range warnings {
            log.Printf("warning: %s", w.Message)
        }
    }),
)
```

## Development and Testing

### Unit Tests

Run the standard Go test suite:

```bash
go test ./...
```

If your changes need to change generated code:

```bash
go generate ./...
```

### Integration Tests

Integration tests run against a live Astra DB instance. See the [Integration Tests README](./internal/integrationtests/README.md) for full setup instructions.

## Other Astra DB Clients

- [Python](https://github.com/datastax/astrapy)
- [TypeScript](https://github.com/datastax/astra-db-ts)
- [.NET](https://github.com/datastax/astra-db-csharp)
- [Java](https://github.com/datastax/astra-db-java)