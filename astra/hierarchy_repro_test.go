package astra

import (
	"fmt"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/options"
)

func TestHierarchy(t *testing.T) {
	client := NewClient(
		options.API().
			SetToken("abc").
			AddHeader("k1", "v1").
			UpdateTimeout(options.Timeout().SetConnection(1 * time.Minute)),
	)

	db := client.Database("https://example.com", options.API().AddHeader("k2", "v2").UpdateTimeout(options.Timeout().SetRequest(2*time.Minute)))

	fmt.Println(db.ClientOptions().Headers)
	fmt.Println(db.ClientOptions().Timeout)

	coll := db.Collection("my_collection", options.GetCollection().SetAPIOptions(options.API().AddHeader("k3", "v3").UpdateTimeout(options.Timeout().SetRequest(2*time.Minute))))

	fmt.Println(coll.ClientOptions().Headers)
	fmt.Println(coll.ClientOptions().Timeout)
}
