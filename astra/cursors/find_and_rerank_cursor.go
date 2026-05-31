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
	"reflect"

	"github.com/datastax/astra-db-go/astra/serdes"
)

// findAndRerankCursorImpl provides the base implementation for findAndRerank operations.
type findAndRerankCursorImpl struct {
	*findLikeCursorImpl[rawRerankedResult]
}

func newFindAndRerankCursorImpl(source findCursorSource[rawRerankedResult], fetcher findCursorFetcher, target serdes.Target, initPageState *string, err error) *findAndRerankCursorImpl {
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

	// The current document is always at index 0 of the buffer in abstractCursorImpl
	return c.currentPage.Results[0].Scores
}

func (c *findAndRerankCursorImpl) mapPage(resp *findResponse, targetCtx serdes.TargetDecodeCtx) *findPage[rawRerankedResult] {
	results := make([]rawRerankedResult, len(resp.Data.Documents))
	for i := range resp.Data.Documents {
		results[i].Document = resp.Data.Documents[i]
		if resp.Status != nil && len(resp.Status.DocumentResponses) > i {
			results[i].Scores = resp.Status.DocumentResponses[i].Scores
		}
	}

	return &findPage[rawRerankedResult]{
		NextPageState: resp.Data.NextPageState,
		Results:       results,
		SortVector:    resp.Data.SortVector,
		targetCtx:     targetCtx,
	}
}

func (c *findAndRerankCursorImpl) decode(raw rawRerankedResult, result any) error {
	val := reflect.ValueOf(result)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return serdes.Deserialize(raw.Document, result, c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target)
	}

	elem := val.Elem()
	if elem.Kind() == reflect.Struct {
		typ := elem.Type()
		// A RerankedResult[T] always has exactly these two exported fields in this order
		if typ.NumField() == 2 {
			f0 := typ.Field(0)
			f1 := typ.Field(1)
			if f0.Name == "Document" && f1.Name == "Scores" && f1.Type == reflect.TypeOf(map[string]float32(nil)) {
				// 1. Set scores
				scoresField := elem.Field(1)
				if scoresField.CanSet() {
					scoresField.Set(reflect.ValueOf(raw.Scores))
				}

				// 2. Decode the document into the Document field
				docField := elem.Field(0)
				return serdes.Deserialize(raw.Document, docField.Addr().Interface(), c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target)
			}
		}
	}

	// Standard decode into the provided result
	return serdes.Deserialize(raw.Document, result, c.findLikeCursorImpl.currentPage.targetCtx, c.findLikeCursorImpl.target)
}

// findAndRerankPayload is the payload for the findAndRerank command.
type findAndRerankPayload struct {
	Filter     any                   `json:"filter,omitempty"`
	Sort       sort.Sortable         `json:"sort,omitempty"`
	Projection map[string]any        `json:"projection,omitempty"`
	Options    *findAndRerankOptions `json:"options,omitempty"`
}

// findAndRerankOptions contains pagination and result options for findAndRerank operations.
type findAndRerankOptions struct {
	Limit             *int    `json:"limit,omitempty"`
	HybridLimits      any     `json:"hybridLimits,omitempty"`
	IncludeScores     *bool   `json:"includeScores,omitempty"`
	IncludeSortVector *bool   `json:"includeSortVector,omitempty"`
	RerankOn          *string `json:"rerankOn,omitempty"`
	RerankQuery       *string `json:"rerankQuery,omitempty"`
	PageState         *string `json:"pageState,omitempty"`
}
