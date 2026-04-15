package datatypes

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDataAPITimestampJSONMarshalInDocument verifies DataAPITimestamp serialization
// within a document struct, matching the Data API $date format.
//
// From the docs:
//
//	{"date_of_birth": {"$date": 1690045891}}
//
// The wire format uses milliseconds since the Unix epoch.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/collection-dates.html
func TestDataAPITimestampJSONMarshalInDocument(t *testing.T) {
	type doc struct {
		Name        string           `json:"name"`
		DateOfBirth DataAPITimestamp `json:"date_of_birth"`
	}
	// 1690045891 seconds = 1690045891000 milliseconds.
	ts := DataAPITimestampFromMillis(1690045891000)
	d := doc{Name: "example", DateOfBirth: ts}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"name":"example","date_of_birth":{"$date":1690045891000}}`
	if string(got) != expected {
		t.Errorf("document marshal mismatch\n  got:  %s\n  want: %s", string(got), expected)
	}

	// Round-trip
	var decoded doc
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.DateOfBirth.Equals(ts) {
		t.Errorf("round-trip mismatch: %v != %v", decoded.DateOfBirth, ts)
	}
}

// TestDataAPITimestampFromMillis verifies creating a timestamp from milliseconds.
func TestDataAPITimestampFromMillis(t *testing.T) {
	ms := int64(1690045891000) // 2023-07-22T15:31:31Z
	ts := DataAPITimestampFromMillis(ms)

	if ts.UnixMillis() != ms {
		t.Errorf("UnixMillis() = %d, want %d", ts.UnixMillis(), ms)
	}

	expected := time.Unix(1690045891, 0).UTC()
	if !ts.Time().Equal(expected) {
		t.Errorf("Time() = %v, want %v", ts.Time(), expected)
	}
}

// TestDataAPITimestampFromTime verifies creating a timestamp from time.Time.
func TestDataAPITimestampFromTime(t *testing.T) {
	target := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)
	ts := NewDataAPITimestamp(target)

	if !ts.Time().Equal(target) {
		t.Errorf("Time() = %v, want %v", ts.Time(), target)
	}
	if ts.UnixMillis() != target.UnixMilli() {
		t.Errorf("UnixMillis() = %d, want %d", ts.UnixMillis(), target.UnixMilli())
	}
}

// TestDataAPITimestampNow verifies DataAPITimestampNow produces a current-ish timestamp.
func TestDataAPITimestampNow(t *testing.T) {
	before := time.Now()
	ts := DataAPITimestampNow()
	after := time.Now()

	if ts.Time().Before(before) || ts.Time().After(after) {
		t.Errorf("DataAPITimestampNow() = %v, expected between %v and %v", ts.Time(), before, after)
	}
}

// TestDataAPITimestampIsZero verifies IsZero behavior.
func TestDataAPITimestampIsZero(t *testing.T) {
	var zero DataAPITimestamp
	if !zero.IsZero() {
		t.Error("expected zero DataAPITimestamp to be zero")
	}
	ts := DataAPITimestampNow()
	if ts.IsZero() {
		t.Error("expected generated DataAPITimestamp to be non-zero")
	}
}

// TestDataAPITimestampEquals verifies Equals compares at millisecond precision.
func TestDataAPITimestampEquals(t *testing.T) {
	// Same millisecond, different nanoseconds
	t1 := NewDataAPITimestamp(time.Date(2024, 1, 1, 0, 0, 0, 123000000, time.UTC))
	t2 := NewDataAPITimestamp(time.Date(2024, 1, 1, 0, 0, 0, 123999999, time.UTC))
	if !t1.Equals(t2) {
		t.Error("expected timestamps within same millisecond to be equal")
	}

	// Different milliseconds
	t3 := NewDataAPITimestamp(time.Date(2024, 1, 1, 0, 0, 0, 124000000, time.UTC))
	if t1.Equals(t3) {
		t.Error("expected timestamps in different milliseconds to not be equal")
	}
}

// TestDataAPITimestampJSONUnmarshalInvalid verifies error handling for invalid JSON.
func TestDataAPITimestampJSONUnmarshalInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not an object", `"not-an-object"`},
		{"missing $date key", `{"date":1690045891000}`},
		{"non-numeric value", `{"$date":"not-a-number"}`},
		{"empty object", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts DataAPITimestamp
			if err := json.Unmarshal([]byte(tt.input), &ts); err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

// TestDataAPITimestampJSONRoundTrip verifies JSON marshal/unmarshal round-trip.
func TestDataAPITimestampJSONRoundTrip(t *testing.T) {
	original := DataAPITimestampFromMillis(1690045891000)
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var parsed DataAPITimestamp
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if !original.Equals(parsed) {
		t.Errorf("round-trip mismatch: %v != %v", original, parsed)
	}
}

// TestDataAPITimestampString verifies String returns RFC3339Nano format.
func TestDataAPITimestampString(t *testing.T) {
	ts := DataAPITimestampFromMillis(1690045891000)
	s := ts.String()
	expected := "2023-07-22T17:11:31Z"
	if s != expected {
		t.Errorf("\nGOT:\n%q\nWANT:\n%q", s, expected)
	}
}

// TestDataAPITimestampNegativeEpoch verifies handling of pre-epoch timestamps.
func TestDataAPITimestampNegativeEpoch(t *testing.T) {
	// 1969-12-31T23:59:59Z = -1000 milliseconds
	ts := DataAPITimestampFromMillis(-1000)
	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"$date":-1000}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", data, expected)
	}

	var parsed DataAPITimestamp
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if !ts.Equals(parsed) {
		t.Errorf("round-trip mismatch for negative epoch")
	}
}
