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

package cursors

import (
	"context"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/sort"
)

// FindAndRerankCursor is a lazy iterable over the results of a findAndRerank operation.
//
// Example usage:
//
//	cursor := collection.FindAndRerank(filter, options.CollectionFindAndRerank().
//	    SetSort(sort.Hybrid("search query")).
//	    SetLimit(10).
//	    SetIncludeScores(true))
//
//	for cursor.Next(ctx) {
//	    var doc MyDocument
//	    if err := cursor.Decode(&doc); err != nil {
//	        // handle error
//	    }
//
//	    // Access reranking scores
//	    scores := cursor.GetScores()
//	}
//
// This type is goroutine safe and may be used concurrently across multiple goroutines.
type FindAndRerankCursor interface {
	AbstractCursor

	// GetScores returns the reranking scores for the current document in the cursor.
	//
	// Scores are only available if IncludeScores was set to true in the options.
	// If scores are not available, this method returns nil.
	GetScores() map[string]float32

	// GetSortVector returns the sort vector used to perform the vector search, if applicable.
	//
	// This method will:
	// 1. Return `nil` if `IncludeSortVector` was not set to `true`
	// 2. Return the original vector if sort.Vector was used
	// 3. Return the generated vector if sort.Vectorize was used
	// 4. Return `nil` if vector search was not used
	//
	// If the sort vector is already in memory, it'll return that; otherwise it'll call the server.
	// If an error occurs while fetching a sort vector, cursor.Err() will be set.
	//
	//  vec := cursor.GetSortVector(ctx)
	//  if vec != nil {
	//    // use the sort vector
	//  }
	GetSortVector(ctx context.Context) *datatypes.Vector

	// Warnings returns all warnings accumulated during cursor operations.
	//
	// Warnings are collected from each page fetch and include any non-fatal issues
	// reported by the server during query execution.
	//
	//  if warnings := cursor.warnings(); len(warnings) > 0 {
	//    // handle warnings
	//  }
	Warnings() results.Warnings
}

// findAndRerankCursorImpl provides the base implementation for findAndRerank operations.
type findAndRerankCursorImpl struct {
	*findLikeCursorImpl[rawRerankedResult]
}

func newFindAndRerankCursorImpl(source findLikeCursorSource[rawRerankedResult], fetcher findLikeCursorFetcher, target serdes.Target, initPageState *string, err error) *findAndRerankCursorImpl {
	impl := &findAndRerankCursorImpl{}
	impl.findLikeCursorImpl = newFindLikeCursorImpl[rawRerankedResult](source, fetcher, target, initPageState, err)
	return impl
}

// GetScores returns the scores for the current document in the cursor.
func (c *findAndRerankCursorImpl) GetScores() map[string]float32 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.currentPage == nil || len(c.currentPage.Results) == 0 {
		return nil
	}

	return c.currentPage.Results[0].Scores
}

// mapPage implements findLikeCursorSource.mapPage
func (c *findAndRerankCursorImpl) mapPage(resp *findResponse, targetCtx serdes.TargetDecodeCtx) *findLikePage[rawRerankedResult] {
	res := make([]rawRerankedResult, len(resp.Data.Documents))
	for i := range resp.Data.Documents {
		res[i].Document = resp.Data.Documents[i]
		if resp.Status != nil && len(resp.Status.DocumentResponses) > i {
			res[i].Scores = resp.Status.DocumentResponses[i].Scores
		}
	}

	return &findLikePage[rawRerankedResult]{
		NextPageState: resp.Data.NextPageState,
		Results:       res,
		SortVector:    resp.Data.SortVector,
		targetCtx:     targetCtx,
	}
}

// mapPage implements findLikeCursorSource.decode
func (c *findAndRerankCursorImpl) decode(raw rawRerankedResult, result any) error {
	flags := c.findLikeCursorImpl.fcs.apiOptions().GetDesFlags()
	if rr, ok := result.(rerankedResultWrapper); ok {
		rr.setScores(raw.Scores)
		return serdes.Deserialize(raw.Document, rr.documentAddr(), c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target, flags)
	}
	return serdes.Deserialize(raw.Document, result, c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target, flags)
}

type findAndRerankPayload struct {
	Filter     any                   `json:"filter,omitempty"`
	Sort       sort.Sortable         `json:"sort,omitempty"`
	Projection map[string]any        `json:"projection,omitempty"`
	Options    *findAndRerankOptions `json:"options,omitempty"`
}

type findAndRerankOptions struct {
	Limit             *int    `json:"limit,omitempty"`
	HybridLimits      any     `json:"hybridLimits,omitempty"`
	IncludeScores     *bool   `json:"includeScores,omitempty"`
	IncludeSortVector *bool   `json:"includeSortVector,omitempty"`
	RerankOn          *string `json:"rerankOn,omitempty"`
	RerankQuery       *string `json:"rerankQuery,omitempty"`
	PageState         *string `json:"pageState,omitempty"`
}
