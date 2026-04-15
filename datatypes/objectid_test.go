package datatypes

import (
	"encoding/json"
	"testing"
	"time"
)

// TestObjectIdJSONMarshalInDocument verifies ObjectId serialization within a document struct,
// matching the Data API insertMany format.
//
// From the docs, an insertMany payload with an ObjectId _id looks like:
//
//	{
//	  "insertMany": {
//	    "documents": [
//	      {
//	        "name": "Melissa",
//	        "_id": {"$objectId": "6672e1cbd7fabb4e5493916f"}
//	      }
//	    ]
//	  }
//	}
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-id.html
func TestObjectIdJSONMarshalInDocument(t *testing.T) {
	type doc struct {
		ID   ObjectId `json:"_id"`
		Name string   `json:"name"`
	}
	oid, _ := ParseObjectId("6672e1cbd7fabb4e5493916f")
	d := doc{ID: oid, Name: "Melissa"}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"_id":{"$objectId":"6672e1cbd7fabb4e5493916f"},"name":"Melissa"}`
	if string(got) != expected {
		t.Errorf("document marshal mismatch\n  got:  %s\n  want: %s", string(got), expected)
	}

	// Round-trip
	var decoded doc
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.ID.Equals(oid) {
		t.Errorf("round-trip ID mismatch: %s != %s", decoded.ID, oid)
	}
}

// TestObjectIdInsertManyExample verifies that a mixed-ID insertMany payload can
// be marshaled and unmarshaled, matching the docs example.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-id.html
func TestObjectIdInsertManyExample(t *testing.T) {
	type doc struct {
		Name string `json:"name"`
		ID   any    `json:"_id"`
	}
	oid, _ := ParseObjectId("6672e1cbd7fabb4e5493916f")
	uid, _ := ParseUUID("1ef2e42c-1fdb-6ad6-aae4-e84679831739")

	// Marshal the ObjectId document
	d := struct {
		Name string   `json:"name"`
		ID   ObjectId `json:"_id"`
	}{Name: "Melissa", ID: oid}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"name":"Melissa","_id":{"$objectId":"6672e1cbd7fabb4e5493916f"}}`
	if string(got) != expected {
		t.Errorf("ObjectId doc mismatch\n  got:  %s\n  want: %s", string(got), expected)
	}

	// Marshal the UUID document
	d2 := struct {
		Name string `json:"name"`
		ID   UUID   `json:"_id"`
	}{Name: "Jess", ID: uid}
	got2, err := json.Marshal(d2)
	if err != nil {
		t.Fatal(err)
	}
	expected2 := `{"name":"Jess","_id":{"$uuid":"1ef2e42c-1fdb-6ad6-aae4-e84679831739"}}`
	if string(got2) != expected2 {
		t.Errorf("UUID doc mismatch\n  got:  %s\n  want: %s", string(got2), expected2)
	}
}

// TestObjectIdParseAndString verifies parse/string round-trip.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-id.html
func TestObjectIdParseAndString(t *testing.T) {
	hex := "6672e1cbd7fabb4e5493916f"
	oid, err := ParseObjectId(hex)
	if err != nil {
		t.Fatalf("ParseObjectId(%q) error: %v", hex, err)
	}
	if oid.String() != hex {
		t.Errorf("String() = %q, want %q", oid.String(), hex)
	}
	// Verify round-trip
	oid2, err := ParseObjectId(oid.String())
	if err != nil {
		t.Fatalf("round-trip parse error: %v", err)
	}
	if !oid.Equals(oid2) {
		t.Errorf("round-trip mismatch: %s != %s", oid, oid2)
	}
}

// TestObjectIdInvalidParsing tests that invalid strings return errors.
func TestObjectIdInvalidParsing(t *testing.T) {
	invalid := []string{
		"",                          // empty
		"6672e1cbd7fabb4e549391",    // too short (22 chars)
		"6672e1cbd7fabb4e549391600", // too long (25 chars)
		"6672e1cbd7fabb4e5493916Z",  // invalid hex character
		"not-a-valid-object-id!!!",  // completely wrong
	}
	for _, s := range invalid {
		if _, err := ParseObjectId(s); err == nil {
			t.Errorf("expected error for invalid ObjectId %q, got nil", s)
		}
	}
}

// TestObjectIdJSONUnmarshalInvalid verifies error handling for invalid JSON inputs.
func TestObjectIdJSONUnmarshalInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not an object", `"not-an-object"`},
		{"missing $objectId key", `{"objectId":"6672e1cbd7fabb4e5493916f"}`},
		{"invalid objectId value", `{"$objectId":"not-valid"}`},
		{"empty object", `{}`},
		{"too short", `{"$objectId":"6672e1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o ObjectId
			if err := json.Unmarshal([]byte(tt.input), &o); err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

// TestObjectIdGeneration verifies NewObjectId produces valid, unique ObjectIds.
func TestObjectIdGeneration(t *testing.T) {
	o1 := NewObjectId()
	o2 := NewObjectId()

	if o1.IsZero() {
		t.Error("expected non-zero ObjectId")
	}
	if o1.Equals(o2) {
		t.Errorf("expected unique ObjectIds, got identical: %s", o1)
	}
	// String should be 24 hex chars
	if len(o1.String()) != 24 {
		t.Errorf("expected 24-char hex string, got %d chars: %s", len(o1.String()), o1)
	}
}

// TestObjectIdGetTimestamp verifies timestamp extraction from generated ObjectIds.
func TestObjectIdGetTimestamp(t *testing.T) {
	now := time.Now()
	o := NewObjectId()
	ts := o.GetTimestamp()
	diff := ts.Sub(now)
	if diff < 0 {
		diff = -diff
	}
	// Timestamps have second precision, so 2s tolerance is generous.
	if diff > 2*time.Second {
		t.Errorf("timestamp %v differs from now %v by %v", ts, now, diff)
	}
}

// TestObjectIdFromTimestamp verifies NewObjectIdFromTimestamp encodes the given time.
func TestObjectIdFromTimestamp(t *testing.T) {
	target := time.Date(2024, 6, 19, 10, 0, 0, 0, time.UTC)
	o := NewObjectIdFromTimestamp(target)

	ts := o.GetTimestamp()
	if ts.Unix() != target.Unix() {
		t.Errorf("timestamp mismatch: got %v (%d), want %v (%d)",
			ts, ts.Unix(), target, target.Unix())
	}
}

// TestObjectIdFromTimestampUniqueness verifies two ObjectIds at the same time differ.
func TestObjectIdFromTimestampUniqueness(t *testing.T) {
	target := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	o1 := NewObjectIdFromTimestamp(target)
	o2 := NewObjectIdFromTimestamp(target)
	if o1.Equals(o2) {
		t.Errorf("expected unique ObjectIds at same timestamp, got identical: %s", o1)
	}
	// Both should encode the same timestamp
	if o1.GetTimestamp().Unix() != o2.GetTimestamp().Unix() {
		t.Errorf("expected same timestamp, got %v and %v", o1.GetTimestamp(), o2.GetTimestamp())
	}
}

// TestObjectIdIsZero verifies IsZero behavior.
func TestObjectIdIsZero(t *testing.T) {
	var zero ObjectId
	if !zero.IsZero() {
		t.Error("expected zero ObjectId to be zero")
	}
	o := NewObjectId()
	if o.IsZero() {
		t.Error("expected generated ObjectId to be non-zero")
	}
}

// TestObjectIdTextRoundTrip verifies MarshalText/UnmarshalText round-trip.
func TestObjectIdTextRoundTrip(t *testing.T) {
	o := NewObjectId()
	text, err := o.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var parsed ObjectId
	if err := parsed.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText error: %v", err)
	}
	if !o.Equals(parsed) {
		t.Errorf("text round-trip mismatch: %s != %s", o, parsed)
	}
}

// TestObjectIdKnownTimestamp verifies that parsing a known ObjectId from the docs
// extracts the expected timestamp.
//
// Doc reference: https://docs.datastax.com/en/astra-db-serverless/api-reference/document-id.html
func TestObjectIdKnownTimestamp(t *testing.T) {
	// "6672e1cb" = first 4 bytes = 0x6672e1cb = 1718812107 seconds
	// = 2024-06-19T14:28:27 UTC
	oid, err := ParseObjectId("6672e1cbd7fabb4e5493916f")
	if err != nil {
		t.Fatal(err)
	}
	ts := oid.GetTimestamp()
	expected := time.Unix(0x6672e1cb, 0)
	if ts.Unix() != expected.Unix() {
		t.Errorf("timestamp = %v, want %v", ts, expected)
	}
}

func TestObjectId_EqualsCaseInsensitive(t *testing.T) {
	// Taken from 'should equal a similar ObjectId' test in ts
	oid, _ := ParseObjectId("507f191e810c19729de860ea")
	other, _ := ParseObjectId("507F191E810C19729DE860EA")
	if !oid.Equals(other) {
		t.Errorf("%s should equal %s", oid, other)
	}
}
