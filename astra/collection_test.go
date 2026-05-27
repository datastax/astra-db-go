package astra_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/serdes"
	"github.com/datastax/astra-db-go/astra/update"
)

// TODO: I captured these responses for future tests but if we don't end up using them, we should remove them.

// Example response from insertMany
const insertManyResponse = "{\"status\":{\"insertedIds\":[\"61c2a03a-c09e-42f5-82a0-3ac09e42f503\",\"c9a9cfc6-be22-408b-a9cf-c6be22908b11\",\"bacd85a8-8ed2-47f0-8d85-a88ed2b7f0d3\",\"5a1b4c50-96e1-4604-9b4c-5096e19604af\",\"95520407-62cb-446a-9204-0762cb946a48\",\"b5bf00a6-c6bd-4151-bf00-a6c6bdb151e9\",\"37155935-825c-4031-9559-35825cf03118\",\"80add809-295c-4d82-add8-09295ced82f3\",\"d96710b5-21f9-4eb8-a710-b521f90eb883\",\"3e58aad9-07a3-4b9b-98aa-d907a30b9b32\",\"60bc0e68-2b55-4ff8-bc0e-682b556ff888\",\"59bdd2e1-8d90-425c-bdd2-e18d90425c7b\",\"c5f4ee93-ed02-42e4-b4ee-93ed0212e4db\",\"2664f4d7-0b64-4678-a4f4-d70b64e678bb\",\"7551ba1f-cc20-4c32-91ba-1fcc200c3226\",\"b50ef75e-1052-4e1d-8ef7-5e10526e1d82\",\"8c6d94ed-41c6-4355-ad94-ed41c613550b\",\"05d47ef5-7544-4fe8-947e-f575440fe861\",\"7931a48a-e575-49e8-b1a4-8ae57559e82c\",\"dc62f18a-da74-4f5f-a2f1-8ada741f5f1b\",\"b230601c-8d84-446a-b060-1c8d84546afa\",\"7ffca2e2-20cb-4336-bca2-e220cb6336dc\",\"7ac46bc2-ae92-4d71-846b-c2ae92fd71cb\",\"fbdb4808-6b9c-4eef-9b48-086b9caeef19\",\"ff51f284-1841-459b-91f2-841841959b25\",\"6b658ccd-2fd0-4141-a58c-cd2fd0914129\",\"df09fcb5-51d0-4e21-89fc-b551d0fe2113\",\"9751e7bc-ce21-4205-91e7-bcce21020524\",\"82ce476e-df90-4605-8e47-6edf90760540\",\"64b511e9-156e-4731-b511-e9156e87314c\"]}}"

// Example response when create/delete happens
const createDeleteResponse = "{\"status\":{\"ok\":1}}"

// TestDeleteOnePayloadSerialization verifies the deleteOne command payload
// matches the expected Data API JSON format from the docs:
//
//	curl -s --location -X POST "..." --header "Token: ..." --header "Content-Type: application/json" --data '{
//	  "deleteOne": {
//	    "filter": {"status": "inactive"},
//	    "sort": {"timestamp": 1}
//	  }
//	}'
func TestDeleteOnePayloadSerialization(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.F
		sort     map[string]any
		expected string
	}{
		{
			name:     "filter only",
			filter:   filter.F{"status": "inactive"},
			expected: `{"deleteOne":{"filter":{"status":"inactive"}}}`,
		},
		{
			name:     "filter with sort",
			filter:   filter.F{"status": "inactive"},
			sort:     map[string]any{"timestamp": 1},
			expected: `{"deleteOne":{"filter":{"status":"inactive"},"sort":{"timestamp":1}}}`,
		},
		{
			name:     "empty filter",
			filter:   filter.F{},
			expected: `{"deleteOne":{"filter":{}}}`,
		},
		{
			name:     "vector sort",
			filter:   filter.F{"status": "active"},
			sort:     map[string]any{"$vector": []float32{0.1, 0.2, 0.3}},
			expected: `{"deleteOne":{"filter":{"status":"active"},"sort":{"$vector":[0.1,0.2,0.3]}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the same payload structure used by DeleteOne
			type deleteOnePayload struct {
				Filter any            `json:"filter,omitempty"`
				Sort   map[string]any `json:"sort,omitempty"`
			}
			payload := deleteOnePayload{
				Filter: tt.filter,
				Sort:   tt.sort,
			}
			wrapped := map[string]any{"deleteOne": payload}
			got, err := serdes.Serialize(wrapped, serdes.TargetCollection)
			if err != nil {
				t.Fatalf("serdes.Serialize error: %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("payload mismatch\n  got:  %s\n  want: %s", string(got), tt.expected)
			}
		})
	}
}

// TestDeleteOneResponseDeserialization verifies we correctly parse the deleteOne response.
func TestDeleteOneResponseDeserialization(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		deletedCount int
	}{
		{
			name:         "one deleted",
			response:     `{"status":{"deletedCount":1}}`,
			deletedCount: 1,
		},
		{
			name:         "none deleted",
			response:     `{"status":{"deletedCount":0}}`,
			deletedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type deleteOneResponse struct {
				Status struct {
					DeletedCount int `json:"deletedCount"`
				} `json:"status"`
			}
			var resp deleteOneResponse
			if err := serdes.Deserialize([]byte(tt.response), &resp, nil, serdes.TargetCollection); err != nil {
				t.Fatalf("serdes.Deserialize error: %v", err)
			}
			if resp.Status.DeletedCount != tt.deletedCount {
				t.Errorf("deletedCount = %d, want %d", resp.Status.DeletedCount, tt.deletedCount)
			}
		})
	}
}

// TestDeleteManyPayloadSerialization verifies the deleteMany command payload
// matches the expected Data API JSON format from the docs:
//
//	"deleteMany": {
//	  "filter": {"$and": [
//	    {"is_checked_out": false},
//	    {"number_of_pages": {"$lt": 300}}
//	  ]}
//	}
func TestDeleteManyPayloadSerialization(t *testing.T) {
	// TODO: could probably make this work with testutils helpers. But - it's just different
	// enough with the wrapper, etc., that leaving it separate for now.
	tests := []struct {
		name     string
		filter   astra.CollectionFilter
		expected string
	}{
		{
			name:     "filter docs example - raw",
			filter:   filter.F{"$and": filter.A{filter.F{"is_checked_out": false}, filter.F{"number_of_pages": filter.F{"$lt": 300}}}},
			expected: `{"deleteMany":{"filter":{"$and":[{"is_checked_out":false},{"number_of_pages":{"$lt":300}}]}}}`,
		},
		{
			name:     "filter docs example - fluent",
			filter:   filter.And(filter.Eq("is_checked_out", false), filter.Lt("number_of_pages", 300)),
			expected: `{"deleteMany":{"filter":{"$and":[{"is_checked_out":false},{"number_of_pages":{"$lt":300}}]}}}`,
		},
		{
			name:     "empty filter deletes all",
			filter:   filter.F{},
			expected: `{"deleteMany":{"filter":{}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type deleteManyPayload struct {
				Filter any `json:"filter,omitempty"`
			}
			payload := deleteManyPayload{
				Filter: tt.filter,
			}
			wrapped := map[string]any{"deleteMany": payload}
			got, err := serdes.Serialize(wrapped, serdes.TargetCollection)
			if err != nil {
				t.Fatalf("serdes.Serialize error: %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("payload mismatch\n  got:  %s\n  want: %s", string(got), tt.expected)
			}
		})
	}
}

// TestDeleteManyResponseDeserialization verifies we correctly parse the deleteMany response,
// including the moreData pagination field.
func TestDeleteManyResponseDeserialization(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		deletedCount int
		moreData     bool
	}{
		{
			name:         "partial page with more data",
			response:     `{"status":{"deletedCount":20,"moreData":true}}`,
			deletedCount: 20,
			moreData:     true,
		},
		{
			name:         "final page",
			response:     `{"status":{"deletedCount":5,"moreData":false}}`,
			deletedCount: 5,
			moreData:     false,
		},
		{
			name:         "none deleted",
			response:     `{"status":{"deletedCount":0}}`,
			deletedCount: 0,
			moreData:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type deleteManyResponse struct {
				Status struct {
					DeletedCount int  `json:"deletedCount"`
					MoreData     bool `json:"moreData"`
				} `json:"status"`
			}
			var resp deleteManyResponse
			if err := serdes.Deserialize([]byte(tt.response), &resp, nil, serdes.TargetCollection); err != nil {
				t.Fatalf("serdes.Deserialize error: %v", err)
			}
			if resp.Status.DeletedCount != tt.deletedCount {
				t.Errorf("deletedCount = %d, want %d", resp.Status.DeletedCount, tt.deletedCount)
			}
			if resp.Status.MoreData != tt.moreData {
				t.Errorf("moreData = %v, want %v", resp.Status.MoreData, tt.moreData)
			}
		})
	}
}

func TestCollectionDeleteManyEnforceNonNilFilters(t *testing.T) {
	// Just make sure users cannot pass a nil filter. "Empty" filter feels
	// more intentional.
	coll := &astra.Collection{}
	_, err := coll.DeleteMany(context.Background(), nil)
	if err == nil {
		t.Errorf("Expected error when filter is nil. Got %v", err)
	}
	if err != astra.ErrNilFilter {
		t.Errorf("Expected ErrNilFilter when filter is nil. Got %v", err)
	}
}

// newTestCollection creates a Collection backed by the given httptest.Server.
func newTestCollection(ts *httptest.Server, apiOpts ...options.APIOption) *astra.Collection {
	allOpts := append([]options.APIOption{options.WithToken("test-token")}, apiOpts...)
	client := astra.NewClient(allOpts...)
	db := client.Database(ts.URL)
	return db.Collection("test_coll")
}

func TestDeleteManyTimeout(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// Simulate a slow paginated response — always returns moreData=true with a delay
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"deletedCount":20,"moreData":true}}`)
	}))
	defer ts.Close()

	coll := newTestCollection(ts)
	ctx := context.Background()

	_, err := coll.DeleteMany(ctx, filter.F{"status": "old"},
		options.CollectionDeleteMany().SetTimeout(250*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	// Should have made at least 2 calls before timing out
	if c := calls.Load(); c < 2 {
		t.Errorf("expected at least 2 calls before timeout, got %d", c)
	}
}

func TestUpdateManyTimeout(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"matchedCount":20,"modifiedCount":20,"moreData":true}}`)
	}))
	defer ts.Close()

	coll := newTestCollection(ts)
	ctx := context.Background()

	_, err := coll.UpdateMany(ctx, filter.F{"status": "old"}, update.Coll().Set("status", "archived"),
		options.CollectionUpdateMany().SetTimeout(250*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if c := calls.Load(); c < 2 {
		t.Errorf("expected at least 2 calls before timeout, got %d", c)
	}
}

func TestDeleteManyHierarchyTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"deletedCount":20,"moreData":true}}`)
	}))
	defer ts.Close()

	// Set GeneralMethod timeout at the client level
	coll := newTestCollection(ts, options.WithGeneralMethodTimeout(250*time.Millisecond))
	ctx := context.Background()

	_, err := coll.DeleteMany(ctx, filter.F{"status": "old"})
	if err == nil {
		t.Fatal("expected timeout error from hierarchy timeout, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestDeleteManyMethodTimeoutOverridesHierarchy(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		// Return moreData=false on first call so it completes
		fmt.Fprint(w, `{"status":{"deletedCount":5,"moreData":false}}`)
	}))
	defer ts.Close()

	// Hierarchy has a very short timeout that would expire
	coll := newTestCollection(ts, options.WithGeneralMethodTimeout(1*time.Millisecond))
	ctx := context.Background()

	// Method-level timeout is generous enough to succeed
	_, err := coll.DeleteMany(ctx, filter.F{"status": "old"},
		options.CollectionDeleteMany().SetTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("expected success with method-level override, got: %v", err)
	}
	if c := calls.Load(); c != 1 {
		t.Errorf("expected 1 call, got %d", c)
	}
}

func TestDeleteManyAPIOptionsOverrideToken(t *testing.T) {
	var receivedToken atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken.Store(r.Header.Get("Token"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"deletedCount":1,"moreData":false}}`)
	}))
	defer ts.Close()

	coll := newTestCollection(ts) // uses "test-token" at client level
	ctx := context.Background()

	_, err := coll.DeleteMany(ctx, filter.F{"x": 1},
		options.CollectionDeleteMany().
			SetAPIOptions(options.API().SetToken("override-token")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := receivedToken.Load().(string); got != "override-token" {
		t.Errorf("expected token 'override-token' in request header, got %q", got)
	}
}

func TestDeleteOneAPIOptionsOverrideToken(t *testing.T) {
	var receivedToken atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken.Store(r.Header.Get("Token"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"deletedCount":1}}`)
	}))
	defer ts.Close()

	coll := newTestCollection(ts)
	ctx := context.Background()

	_, err := coll.DeleteOne(ctx, filter.F{"x": 1},
		options.CollectionDeleteOne().
			SetAPIOptions(options.API().SetToken("override-token")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := receivedToken.Load().(string); got != "override-token" {
		t.Errorf("expected token 'override-token' in request header, got %q", got)
	}
}

func TestUpdateOneAPIOptionsOverrideToken(t *testing.T) {
	var receivedToken atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken.Store(r.Header.Get("Token"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"matchedCount":1,"modifiedCount":1}}`)
	}))
	defer ts.Close()

	coll := newTestCollection(ts)
	ctx := context.Background()

	_, err := coll.UpdateOne(ctx, filter.F{"x": 1}, update.Coll().Set("x", 2),
		options.CollectionUpdateOne().
			SetAPIOptions(options.API().SetToken("override-token")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := receivedToken.Load().(string); got != "override-token" {
		t.Errorf("expected token 'override-token' in request header, got %q", got)
	}
}

func TestResolveGeneralMethodTimeoutFromAPIOverride(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":{"deletedCount":20,"moreData":true}}`)
	}))
	defer ts.Close()

	// No GeneralMethod timeout at client level
	coll := newTestCollection(ts)
	ctx := context.Background()

	// Set GeneralMethod timeout via APIOptions override
	_, err := coll.DeleteMany(ctx, filter.F{"status": "old"},
		options.CollectionDeleteMany().
			SetAPIOptions(
				options.API().SetTimeout(
					options.Timeout().SetGeneralMethod(250*time.Millisecond),
				),
			),
	)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestNilDB(t *testing.T) {
	var db *astra.Db = nil
	c := db.Collection("nildb")
	_, err := c.CountDocuments(context.Background(), nil, 100)
	if err == nil {
		t.Errorf("Expected error. Got %v", err)
	}
}
