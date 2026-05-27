package datatypes_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"pgregory.net/rapid"
)

func TestObjectId_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tm := time.Unix(rapid.Int64Range(0, 2147483647).Draw(t, "unix"), 0).UTC()
		oid := datatypes.NewObjectIdAt(tm)

		// String round-trip
		s := oid.String()
		parsed, err := datatypes.ParseObjectId(s)
		if err != nil {
			t.Fatalf("ParseObjectId(%q) failed: %v", s, err)
		}
		if !oid.Equals(parsed) {
			t.Fatalf("String round-trip mismatch: got %v, want %v", parsed, oid)
		}

		// JSON round-trip
		data, err := json.Marshal(oid)
		if err != nil {
			t.Fatal(err)
		}
		var unmarshaled datatypes.ObjectId
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatal(err)
		}
		if !oid.Equals(unmarshaled) {
			t.Fatalf("JSON round-trip mismatch")
		}

		// Text round-trip
		text, err := oid.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var parsedText datatypes.ObjectId
		if err := parsedText.UnmarshalText(text); err != nil {
			t.Fatal(err)
		}
		if !oid.Equals(parsedText) {
			t.Fatalf("Text round-trip mismatch")
		}
	})
}

func TestObjectId_Timestamp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		unix := rapid.Int64Range(0, 2147483647).Draw(t, "unix")
		tm := time.Unix(unix, 0).UTC()
		oid := datatypes.NewObjectIdAt(tm)
		if got := oid.GetTimestamp().Unix(); got != unix {
			t.Fatalf("Timestamp mismatch: got %d, want %d", got, unix)
		}
	})
}

func TestObjectId_Uniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		o1 := datatypes.NewObjectId()
		o2 := datatypes.NewObjectId()
		if o1.Equals(o2) {
			t.Fatalf("Consecutive ObjectIds should be unique: %v", o1)
		}
	})
}

func TestObjectId_CompareTo(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		o1 := datatypes.NewObjectId()
		o2 := datatypes.NewObjectId()
		got := o1.CompareTo(o2)

		b1, b2 := o1.Bytes(), o2.Bytes()
		want := bytes.Compare(b1[:], b2[:])

		if got != want {
			t.Fatalf("CompareTo mismatch: got %d, want %d", got, want)
		}
	})
}

func TestObjectId_ParseErrors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "invalid ObjectId string"},
		{"6672e1cbd7fabb4e549391", "invalid ObjectId string"},
		{"6672e1cbd7fabb4e549391600", "invalid ObjectId string"},
		{"6672e1cbd7fabb4e5493916Z", "invalid ObjectId string"},
		{"not-a-valid-object-id!!!", "invalid ObjectId string"},
	}
	for _, tc := range tests {
		_, err := datatypes.ParseObjectId(tc.input)
		if err == nil {
			t.Errorf("expected error for %q", tc.input)
		}
	}
}

func TestObjectId_JSONErrors(t *testing.T) {
	tests := []string{
		`"not-an-object"`,
		`{"objectId":"6672e1cbd7fabb4e5493916f"}`,
		`{"$objectId":"not-valid"}`,
		`{}`,
	}
	for _, input := range tests {
		var o datatypes.ObjectId
		if err := json.Unmarshal([]byte(input), &o); err == nil {
			t.Errorf("expected JSON unmarshal error for %q", input)
		}
	}
}

func TestObjectId_IsZero(t *testing.T) {
	var zero datatypes.ObjectId
	if !zero.IsZero() {
		t.Fatal("Zero value should be zero")
	}
	if datatypes.NewObjectId().IsZero() {
		t.Fatal("NewObjectId should not be zero")
	}
}
