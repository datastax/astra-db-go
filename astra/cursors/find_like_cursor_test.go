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
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

func mkFindResponse(docs []string, sortVec []float32, nextPageState *string) []byte {
	type statusBlock struct {
		SortVector *datatypes.Vector `json:"sortVector,omitempty"`
	}
	type dataBlock struct {
		Documents     []json.RawMessage `json:"documents"`
		NextPageState *string           `json:"nextPageState,omitempty"`
	}
	type resp struct {
		Data   dataBlock    `json:"data"`
		Status *statusBlock `json:"status,omitempty"`
	}

	r := resp{}
	for _, d := range docs {
		r.Data.Documents = append(r.Data.Documents, json.RawMessage(d))
	}
	r.Data.NextPageState = nextPageState
	if sortVec != nil {
		v := datatypes.NewVector(sortVec)
		r.Status = &statusBlock{SortVector: &v}
	}
	b, _ := json.Marshal(r)
	return b
}

func TestGetSortVector_ThenNext_NoItemsSkipped(t *testing.T) {
	docs := []string{`{"_id":"1"}`, `{"_id":"2"}`, `{"_id":"3"}`}
	page := mkFindResponse(docs, []float32{0.1, 0.2, 0.3}, nil)
	fetcher := func(_ context.Context, _ any, _ *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		return page, nil, nil, nil
	}
	opts := options.CollectionFindOptions{
		APIOptions:        &options.APIOptions{},
		IncludeSortVector: ptr.To(true),
	}
	cursor := NewCollectionFindCursor(nil, &opts, fetcher, nil)

	sv := cursor.GetSortVector(context.Background())
	if sv == nil {
		t.Fatal("expected sort vector, got nil")
	}
	if cursor.State() != CursorStateStarted {
		t.Fatalf("expected CursorStateStarted after GetSortVector, got %v", cursor.State())
	}

	var collected []string
	for cursor.Next(context.Background()) {
		var doc map[string]string
		if err := cursor.Decode(&doc); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		collected = append(collected, doc["_id"])
	}

	if len(collected) != len(docs) {
		t.Fatalf("expected %d docs, got %d: %v", len(docs), len(collected), collected)
	}
	for i, id := range []string{"1", "2", "3"} {
		if collected[i] != id {
			t.Errorf("doc[%d] = %q, want %q", i, collected[i], id)
		}
	}
}

func TestGetSortVector_WithoutIncludeSortVector_DoesNotFetch(t *testing.T) {
	fetched := false
	fetcher := func(_ context.Context, _ any, _ *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		fetched = true
		return mkFindResponse([]string{`{"_id":"1"}`}, nil, nil), nil, nil, nil
	}
	opts := options.CollectionFindOptions{
		APIOptions:        &options.APIOptions{},
		IncludeSortVector: ptr.To(false),
	}
	cursor := NewCollectionFindCursor(nil, &opts, fetcher, nil)

	if sv := cursor.GetSortVector(context.Background()); sv != nil {
		t.Errorf("expected nil sort vector, got %v", sv)
	}
	if fetched {
		t.Error("expected no fetch when IncludeSortVector=false")
	}
	if cursor.State() != CursorStateIdle {
		t.Errorf("expected CursorStateIdle, got %v", cursor.State())
	}
}

func TestGetSortVector_AlreadyStarted_DoesNotRefetch(t *testing.T) {
	fetchCount := 0
	fetcher := func(_ context.Context, _ any, _ *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		fetchCount++
		return mkFindResponse([]string{`{"_id":"1"}`}, []float32{0.1, 0.2}, nil), nil, nil, nil
	}
	opts := options.CollectionFindOptions{
		APIOptions:        &options.APIOptions{},
		IncludeSortVector: ptr.To(true),
	}
	cursor := NewCollectionFindCursor(nil, &opts, fetcher, nil)

	if !cursor.Next(context.Background()) {
		t.Fatal("expected Next() = true")
	}
	if fetchCount != 1 {
		t.Fatalf("expected 1 fetch after Next(), got %d", fetchCount)
	}

	sv := cursor.GetSortVector(context.Background())
	if sv == nil {
		t.Fatal("expected sort vector, got nil")
	}
	if fetchCount != 1 {
		t.Errorf("expected no additional fetch from GetSortVector, got %d total", fetchCount)
	}
}

func TestGetSortVector_Cached(t *testing.T) {
	fetchCount := 0
	fetcher := func(_ context.Context, _ any, _ *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		fetchCount++
		return mkFindResponse([]string{`{"_id":"1"}`}, []float32{0.5, 0.6}, nil), nil, nil, nil
	}
	opts := options.CollectionFindOptions{
		APIOptions:        &options.APIOptions{},
		IncludeSortVector: ptr.To(true),
	}
	cursor := NewCollectionFindCursor(nil, &opts, fetcher, nil)

	sv1 := cursor.GetSortVector(context.Background())
	sv2 := cursor.GetSortVector(context.Background())

	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}
	if sv1 == nil || sv2 == nil || sv1 != sv2 {
		t.Errorf("expected same non-nil sort vector both times")
	}
}
