package astra

// WithSimilarity is a struct that can be embedded in query result structs to include a similarity score for each result.
//
// Example usage:
//
//	type MyDocument struct {
//	    ID   string `json:"id"`
//	    Name string `json:"name"`
//	}
//
//	var result struct {
//	    MyDocument
//	    astra.WithSimilarity
//	}
//
//	err := coll.FindOne(ctx, filter.F{},
//	    options.CollectionFindOne().
//	        SetSort(sort.Vectorize("<search query>")).
//	        SetIncludeSimilarity(true),
//	).Decode(&result)
//
//	if err == nil {
//	    fmt.Printf("Match: '%s' (similarity: %f)\n", result.Name, result.Similarity)
//	}
type WithSimilarity struct {
	Similarity float64 `json:"$similarity"`
}
