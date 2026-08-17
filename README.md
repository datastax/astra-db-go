# astra-db-go

[![License Apache2](https://img.shields.io/hexpm/l/plug.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Reference](https://pkg.go.dev/badge/github.com/datastax/astra-db-go.svg)](https://pkg.go.dev/github.com/datastax/astra-db-go)
[![Documentation](https://img.shields.io/badge/Docs-datastax.com-blueviolet.svg)](https://docs.datastax.com/en/astra-db-serverless/api-reference/dataapiclient.html)

`astra-db-go` is a Go client for interacting with [DataStax Astra DB](https://astra.datastax.com/) and Hyper-Converged Database.

## Setup & requirements

Update to Go 1.23 or higher. [Download it here](https://go.dev/doc/install).

Install the client:

```
go get github.com/datastax/astra-db-go/v2
```

You need an Astra DB database or a Hyper-Converged Database (HCD) for the client to connect to.

## Documentation

For Astra DB Serverless:

- [Quickstart for collections](https://docs.datastax.com/en/astra-db-serverless/get-started/quickstart.html)
- [Quickstart for tables](https://docs.datastax.com/en/astra-db-serverless/get-started/quickstart-tables.html)
- [Get started with the Data API](https://docs.datastax.com/en/astra-db-serverless/api-reference/dataapiclient.html)

For Hyper-Converged Database (HCD):

- [Quickstart for collections](https://docs.datastax.com/en/hyper-converged-database/1.2/api-reference/quickstart.html)
- [Quickstart for tables](https://docs.datastax.com/en/hyper-converged-database/1.2/api-reference/quickstart-tables.html)
- [Get started with the Data API](https://docs.datastax.com/en/hyper-converged-database/1.2/api-reference/dataapiclient.html)

Package-level documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/datastax/astra-db-go/v2).

## At a glance

```go
import (
    "context"
    "fmt"
    "log"

    "github.com/datastax/astra-db-go/v2/astra"
    "github.com/datastax/astra-db-go/v2/astra/filter"
    "github.com/datastax/astra-db-go/v2/astra/options"
    "github.com/datastax/astra-db-go/v2/astra/sort"
)

type MyDocument struct {
    Name        string `json:"name"`
    Price       int    `json:"price"`
    Description string `json:"$vectorize"`
}

func main() {
    ctx := context.Background()

    // Initialize the client
    client := astra.NewClient()
    db := client.Database("<endpoint>", options.API().SetToken("<token>"))

    // Create a collection
    coll, err := db.CreateCollection(ctx, "ExampleCollection")
    if err != nil {
        log.Fatal(err)
    }

    // Insert documents
    docs := []MyDocument{
        {Name: "Apple", Price: 10, Description: "..."},
        {Name: "Peach", Price: 5, Description: "..."},
        {Name: "Walnut", Price: 8, Description: "..."},
    }
    ids, err := coll.InsertMany(ctx, docs)
    if err == nil {
        fmt.Printf("Inserted documents with IDs: %v\n", ids)
    }

    // Find a document with vector search (vectorize)
    var result struct {
        MyDocument
        astra.WithSimilarity
    }

    err = coll.FindOne(ctx, filter.F{},
        options.CollectionFindOne().
            SetSort(sort.Vectorize("<search query>")).
            SetIncludeSimilarity(true),
    ).Decode(&result)

    if err == nil {
        fmt.Printf("Match: '%s' (similarity: %f)\n", result.Name, result.Similarity)
    }
}
```

## Other Data API clients

- [Python](https://github.com/datastax/astrapy)
- [TypeScript](https://github.com/datastax/astra-db-ts)
- [Java](https://github.com/datastax/astra-db-java)
- [.NET](https://github.com/datastax/astra-db-csharp)
