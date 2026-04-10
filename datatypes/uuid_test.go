package datatypes

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestJSONMarshalInDocument verifies UUID serialization within a document struct,
// matching the Data API insert format.
//
// From the docs, an insertOne payload with a UUID _id looks like:
//
//	{
//	  "insertOne": {
//	    "document": {
//	      "_id": {"$uuid": "550e8400-e29b-41d4-a716-446655440000"},
//	      "name": "example"
//	    }
//	  }
//	}
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/insert-one.html
func TestJSONMarshalInDocument(t *testing.T) {
	type doc struct {
		ID   UUID   `json:"_id"`
		Name string `json:"name"`
	}
	u, _ := ParseUUID("550e8400-e29b-41d4-a716-446655440000")
	d := doc{ID: u, Name: "example"}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"_id":{"$uuid":"550e8400-e29b-41d4-a716-446655440000"},"name":"example"}`
	if string(got) != expected {
		t.Errorf("document marshal mismatch\n  got:  %s\n  want: %s", string(got), expected)
	}

	// Round-trip
	var decoded doc
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.ID.Equals(u) {
		t.Errorf("round-trip ID mismatch: %s != %s", decoded.ID, u)
	}
}

// TestIsZero verifies IsZero behavior.
func TestIsZero(t *testing.T) {
	var zero UUID
	if !zero.IsZero() {
		t.Error("expected zero UUID to be zero")
	}
	u := NewUUID()
	if u.IsZero() {
		t.Error("expected generated UUID to be non-zero")
	}
}

// TestEquals verifies Equals behavior.
func TestEquals(t *testing.T) {
	u1, _ := ParseUUID("550e8400-e29b-41d4-a716-446655440000")
	u2, _ := ParseUUID("550e8400-e29b-41d4-a716-446655440000")
	u3 := NewUUID()

	if !u1.Equals(u2) {
		t.Error("expected equal UUIDs to be equal")
	}
	if u1.Equals(u3) {
		t.Error("expected different UUIDs to not be equal")
	}
}

// TestUniqueness verifies two consecutive NewUUID calls produce different values.
func TestUniqueness(t *testing.T) {
	u1 := NewUUID()
	u2 := NewUUID()
	if u1.Equals(u2) {
		t.Errorf("expected unique UUIDs, got identical: %s", u1)
	}
}

// TestJSONUnmarshalInvalid verifies error handling for invalid JSON inputs.
func TestJSONUnmarshalInvalid(t *testing.T) {
	// Very similar to TestInvalidUUIDParsing but that tests parse, whereas this
	// tests json.Unmarshal.
	tests := []struct {
		name  string
		input string
	}{
		{"not an object", `"not-an-object"`},
		{"missing $uuid key", `{"uuid":"550e8400-e29b-41d4-a716-446655440000"}`},
		{"invalid uuid value", `{"$uuid":"not-a-uuid"}`},
		{"empty object", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u UUID
			if err := json.Unmarshal([]byte(tt.input), &u); err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

func TestUUIDv7Monotonicity(t *testing.T) {
	// Test that getV7Time produces monotonically increasing values across
	// multiple goroutines.
	const (
		goroutines = 10
		iterations = 10000
	)
	results := make([][]int64, goroutines)
	// Create a wait group and set task counter to number of goroutines.
	var wg sync.WaitGroup
	wg.Add(goroutines)
	// Really hammer our getV7Time function.
	for g := range goroutines {
		results[g] = make([]int64, iterations)
		go func(g int) {
			defer wg.Done()
			for i := range iterations {
				milli, seq := getV7Time()
				results[g][i] = milli<<12 | seq
			}
		}(g)
	}
	wg.Wait()

	// Flatten, sort, and check for duplicates
	all := slices.Concat(results...)
	slices.Sort(all)
	for i := 1; i < len(all); i++ {
		if all[i] <= all[i-1] {
			t.Fatalf("duplicate or out-of-order value at index %d: %d <= %d",
				i, all[i], all[i-1])
		}
	}
}

// Verifies that the timestamp extracted from u is near now.
func verifyTimestampNowIsh(t *testing.T, u UUID, now time.Time) {
	ts, ok := u.Timestamp()
	if !ok {
		t.Fatal("expected Timestamp to return true for v1 UUID")
	}
	diff := ts.Sub(now)
	if diff < 0 {
		diff = -diff
	}
	// This is extremely generous. But also - if there are parsing / round-trip
	// issues, they will be off by more than 100ms.
	if diff > 100*time.Millisecond {
		t.Errorf("timestamp %v differs from now %v by %v", ts, now, diff)
	}
}

// TestUUIDTimestampV6 verifies generating a v6 UUID and extracting its timestamp yields ~now.
func TestUUIDTimestampV6(t *testing.T) {
	verifyTimestampNowIsh(t, NewUUIDv6(), time.Now())
}

// TestUUIDTimestampV1 verifies generating a v1 UUID and extracting its timestamp yields ~now.
func TestUUIDTimestampV1(t *testing.T) {
	// Generate a v1 UUID now, extract timestamp, and verify it round-trips
	// through the Gregorian conversion accurately.
	verifyTimestampNowIsh(t, NewUUIDv1(), time.Now())
}

// TestUUIDv6Ordering verifies that sequentially generated v6 UUIDs sort lexicographically by time.
func TestUUIDv6Ordering(t *testing.T) {
	const n = 100
	uuids := make([]string, n)
	for i := range n {
		uuids[i] = NewUUIDv6().String()
	}
	for i := 1; i < n; i++ {
		if uuids[i] <= uuids[i-1] {
			t.Fatalf("v6 UUIDs not in lexicographic order at index %d:\n  %s\n  %s",
				i, uuids[i-1], uuids[i])
		}
	}
}

// TestUUIDv1Monotonicity verifies concurrent v1 UUID generation produces unique timestamps.
func TestUUIDv1Monotonicity(t *testing.T) {
	const (
		goroutines = 10
		iterations = 10000
	)
	results := make([][]int64, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := range goroutines {
		results[g] = make([]int64, iterations)
		go func(g int) {
			defer wg.Done()
			for i := range iterations {
				ts, _, _ := getGregTime()
				results[g][i] = ts
			}
		}(g)
	}
	wg.Wait()

	all := slices.Concat(results...)
	slices.Sort(all)
	for i := 1; i < len(all); i++ {
		if all[i] <= all[i-1] {
			t.Fatalf("duplicate or out-of-order Gregorian timestamp at index %d: %d <= %d",
				i, all[i], all[i-1])
		}
	}
}

// TestInvalidUUIDParsing tests that invalid strings return errors.
// Inspired by typescript client tests.
func TestInvalidUUIDParsing(t *testing.T) {
	invalidUUIDs := []string{
		"123e4567-e89b-12d3-a456-42661417400",   // too short
		"123e4567-e89b-12d3-a456-4266141740000", // too long
		"123e4567-e89b-12d3-a456-42661417400Z",  // invalid character
		"",                                      // empty string
	}
	for _, s := range invalidUUIDs {
		if _, err := ParseUUID(s); err == nil {
			t.Errorf("expected error for invalid UUID %q, got nil", s)
		}
	}
}

// Just verify this thing looks somewhat valid
func assertLooksLikeValidUUID(u UUID, t *testing.T) {
	if u.IsZero() {
		t.Fatal("NewUUIDv4 returned zero UUID")
	}
	// Variant bits: byte 8 top two bits must be 10
	if u.value[8]>>6 != 0b10 {
		t.Errorf("expected variant bits 10, got %02b", u.value[8]>>6)
	}
	// String format: 8-4-4-4-12
	s := u.String()
	if len(s) != 36 {
		t.Errorf("expected string length 36, got %d", len(s))
	}
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		t.Errorf("expected 5 dash-separated parts, got %d", len(parts))
	}
	expected := []int{8, 4, 4, 4, 12}
	for i, p := range parts {
		if len(p) != expected[i] {
			t.Errorf("part %d: expected length %d, got %d", i, expected[i], len(p))
		}
	}
}

// TestNewUUIDv7At verifies that NewUUIDv7At encodes the given timestamp.
// The extracted timestamp should match the input to millisecond precision.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/insert-one.html
func TestNewUUIDv7At(t *testing.T) {
	target := time.Date(2024, 6, 15, 12, 30, 45, 123456789, time.UTC)
	u := NewUUIDv7At(target)

	assertLooksLikeValidUUID(u, t)
	if u.Version() != 7 {
		t.Fatalf("expected version 7, got %d", u.Version())
	}

	ts, ok := u.Timestamp()
	if !ok {
		t.Fatal("expected Timestamp to return true for v7 UUID")
	}
	// v7 stores millisecond precision
	diff := ts.Sub(target.Truncate(time.Millisecond))
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Millisecond {
		t.Errorf("timestamp mismatch: got %v, want ~%v (diff %v)", ts, target, diff)
	}
}

// TestNewUUIDv7AtUniqueness verifies two UUIDs at the same time have different random bits.
func TestNewUUIDv7AtUniqueness(t *testing.T) {
	target := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	u1 := NewUUIDv7At(target)
	u2 := NewUUIDv7At(target)
	if u1.Equals(u2) {
		t.Errorf("expected unique UUIDs at same timestamp, got identical: %s", u1)
	}
	// Both should encode the same timestamp
	ts1, _ := u1.Timestamp()
	ts2, _ := u2.Timestamp()
	if !ts1.Equal(ts2) {
		t.Errorf("expected same timestamp, got %v and %v", ts1, ts2)
	}
}

// TestNewUUIDv7AtJSON verifies the JSON serialization of a v7 UUID created at a specific time.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/insert-one.html
func TestNewUUIDv7AtJSON(t *testing.T) {
	target := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	u := NewUUIDv7At(target)

	got, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	// Should be {"$uuid":"..."} format
	var wrapper map[string]string
	if err := json.Unmarshal(got, &wrapper); err != nil {
		t.Fatalf("expected JSON object, got: %s", got)
	}
	if _, ok := wrapper["$uuid"]; !ok {
		t.Fatalf("expected $uuid key in JSON, got: %s", got)
	}
	// Round-trip through JSON
	var parsed UUID
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatal(err)
	}
	if !parsed.Equals(u) {
		t.Errorf("JSON round-trip mismatch: %s != %s", parsed, u)
	}
}

// TestNewUUIDv1At verifies that NewUUIDv1At encodes the given timestamp.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/insert-one.html
func TestNewUUIDv1At(t *testing.T) {
	target := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)
	u := NewUUIDv1At(target)

	assertLooksLikeValidUUID(u, t)
	if u.Version() != 1 {
		t.Fatalf("expected version 1, got %d", u.Version())
	}

	ts, ok := u.Timestamp()
	if !ok {
		t.Fatal("expected Timestamp to return true for v1 UUID")
	}
	// Gregorian timestamps have 100ns precision
	diff := ts.Sub(target)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Microsecond {
		t.Errorf("timestamp mismatch: got %v, want ~%v (diff %v)", ts, target, diff)
	}
}

// TestNewUUIDv6At verifies that NewUUIDv6At encodes the given timestamp.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/insert-one.html
func TestNewUUIDv6At(t *testing.T) {
	target := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)
	u := NewUUIDv6At(target)

	assertLooksLikeValidUUID(u, t)
	if u.Version() != 6 {
		t.Fatalf("expected version 6, got %d", u.Version())
	}

	ts, ok := u.Timestamp()
	if !ok {
		t.Fatal("expected Timestamp to return true for v6 UUID")
	}
	diff := ts.Sub(target)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Microsecond {
		t.Errorf("timestamp mismatch: got %v, want ~%v (diff %v)", ts, target, diff)
	}
}

// TestNewUUIDv6AtOrdering verifies v6 UUIDs created at different times sort lexicographically.
func TestNewUUIDv6AtOrdering(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	u1 := NewUUIDv6At(t1)
	u2 := NewUUIDv6At(t2)
	u3 := NewUUIDv6At(t3)

	// v6 UUIDs should sort lexicographically by time
	if u1.String() >= u2.String() {
		t.Errorf("expected %s < %s", u1, u2)
	}
	if u2.String() >= u3.String() {
		t.Errorf("expected %s < %s", u2, u3)
	}
}

// TestStringRoundTripAllVersions generates a UUID of each supported version,
// converts to string, parses back, and verifies equality and version preservation.
// Inspired by typescript client tests.
func TestStringRoundTripAllVersions(t *testing.T) {
	versions := []struct {
		name     string
		original UUID
		version  int
	}{
		{"v1", NewUUIDv1(), 1},
		{"v4", NewUUIDv4(), 4},
		{"v6", NewUUIDv6(), 6},
		{"v7", NewUUIDv7(), 7},
	}
	for _, tt := range versions {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.original
			// Verify the version just for kicks
			if original.Version() != tt.version {
				t.Fatalf("expected version %d, got %d", tt.version, original.Version())
			}
			// Make sure this looks like a valid UUID
			assertLooksLikeValidUUID(original, t)
			// Turn it into string, parse it, and verify nothing has changed.
			s := original.String()
			parsed, err := ParseUUID(s)
			if err != nil {
				t.Fatalf("ParseUUID(%q) error: %v", s, err)
			}
			if !original.Equals(parsed) {
				t.Errorf("round-trip mismatch: original=%s, parsed=%s", original, parsed)
			}
			if parsed.Version() != tt.version {
				t.Errorf("version not preserved: got %d, want %d", parsed.Version(), tt.version)
			}
		})
	}
}
