// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
