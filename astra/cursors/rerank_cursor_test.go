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
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/serdes"
)

func TestFindAndRerankCursorImpl_MapPage(t *testing.T) {
	impl := newFindAndRerankCursorImpl(nil, nil, serdes.TargetCollection, nil, nil)
	resp := &findResponse{}
	resp.Data.Documents = []json.RawMessage{
		json.RawMessage(`{"_id": "1"}`),
		json.RawMessage(`{"_id": "2"}`),
	}
	resp.Status = &struct {
		DocumentResponses []struct {
			Scores map[string]float32 `json:"scores"`
		} `json:"documentResponses"`
	}{
		DocumentResponses: []struct {
			Scores map[string]float32 `json:"scores"`
		}{
			{Scores: map[string]float32{"$reranker": 0.9}},
			{Scores: map[string]float32{"$reranker": 0.8}},
		},
	}

	page := impl.mapPage(resp, nil)

	if len(page.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(page.Results))
	}

	if string(page.Results[0].Document) != `{"_id": "1"}` {
		t.Errorf("expected doc 1, got %s", string(page.Results[0].Document))
	}
	if page.Results[0].Scores["$reranker"] != 0.9 {
		t.Errorf("expected score 0.9, got %v", page.Results[0].Scores["$reranker"])
	}

	if string(page.Results[1].Document) != `{"_id": "2"}` {
		t.Errorf("expected doc 2, got %s", string(page.Results[1].Document))
	}
	if page.Results[1].Scores["$reranker"] != 0.8 {
		t.Errorf("expected score 0.8, got %v", page.Results[1].Scores["$reranker"])
	}
}

func TestFindAndRerankCursorImpl_GetScores(t *testing.T) {
	mkImpl := func() *findAndRerankCursorImpl {
		return newFindAndRerankCursorImpl(nil, nil, serdes.TargetCollection, nil, nil)
	}

	impl := mkImpl()

	// Test nil page
	if scores := impl.GetScores(); scores != nil {
		t.Errorf("expected nil scores for nil page")
	}

	// Test empty page
	impl.currentPage = &findLikePage[rawRerankedResult]{Results: []rawRerankedResult{}}
	if scores := impl.GetScores(); scores != nil {
		t.Errorf("expected nil scores for empty page")
	}

	// Test page with results
	impl.currentPage = &findLikePage[rawRerankedResult]{
		Results: []rawRerankedResult{
			{Scores: map[string]float32{"$reranker": 0.95}},
		},
	}
	scores := impl.GetScores()
	if scores == nil || scores["$reranker"] != 0.95 {
		t.Errorf("expected 0.95 score, got %v", scores)
	}
}

type mockFindLikeCursorSource struct {
	findLikeCursorSource[rawRerankedResult]
	opts *options.APIOptions
}

func (m *mockFindLikeCursorSource) apiOptions() *options.APIOptions {
	return m.opts
}

func TestFindAndRerankCursorImpl_Decode(t *testing.T) {
	mockSource := &mockFindLikeCursorSource{
		opts: &options.APIOptions{},
	}
	impl := &findAndRerankCursorImpl{
		findLikeCursorImpl: &findLikeCursorImpl[rawRerankedResult]{
			fcs:         mockSource,
			target:      serdes.TargetCollection,
			currentPage: &findLikePage[rawRerankedResult]{},
		},
	}

	raw := rawRerankedResult{
		Document: json.RawMessage(`{"name": "test"}`),
		Scores:   map[string]float32{"$reranker": 0.99},
	}

	type Doc struct {
		Name string `json:"name"`
	}

	t.Run("Standard struct", func(t *testing.T) {
		var doc Doc
		if err := impl.decode(raw, &doc); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if doc.Name != "test" {
			t.Errorf("expected test, got %s", doc.Name)
		}
	})

	t.Run("RerankedResult", func(t *testing.T) {
		var rr RerankedResult[Doc]
		if err := impl.decode(raw, &rr); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if rr.Document.Name != "test" {
			t.Errorf("expected test in Document, got %s", rr.Document.Name)
		}
		if rr.Scores["$reranker"] != 0.99 {
			t.Errorf("expected 0.99 in Scores, got %v", rr.Scores["$reranker"])
		}
	})

	t.Run("Invalid target (not a pointer)", func(t *testing.T) {
		var doc Doc
		// Should just delegate to serdes which will return an error
		err := impl.decode(raw, doc)
		if err == nil {
			t.Error("expected error for non-pointer target")
		}
	})
}
