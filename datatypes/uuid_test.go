package datatypes_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/datatypes"
	"pgregory.net/rapid"
)

func TestUUID_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := rapid.SampledFrom([]func() datatypes.UUID{
			datatypes.NewUUIDv1,
			datatypes.NewUUIDv4,
			datatypes.NewUUIDv6,
			datatypes.NewUUIDv7,
		})
		original := gen.Draw(t, "uuid_gen")()

		// String round-trip
		s := original.String()
		parsed, err := datatypes.ParseUUID(s)
		if err != nil {
			t.Fatalf("ParseUUID(%q) failed: %v", s, err)
		}
		if !original.Equals(parsed) {
			t.Fatalf("String round-trip mismatch: got %v, want %v", parsed, original)
		}

		// JSON round-trip
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		var unmarshaled datatypes.UUID
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatal(err)
		}
		if !original.Equals(unmarshaled) {
			t.Fatal("JSON round-trip mismatch")
		}

		// Text round-trip
		text, err := original.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var parsedText datatypes.UUID
		if err := parsedText.UnmarshalText(text); err != nil {
			t.Fatal(err)
		}
		if !original.Equals(parsedText) {
			t.Fatal("Text round-trip mismatch")
		}
	})
}

func TestUUID_Timestamp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// v7 uses milliseconds, v1/v6 use 100ns. Use a range safe for both.
		unix := rapid.Int64Range(0, 2147483647).Draw(t, "unix")
		tm := time.Unix(unix, 0).UTC()

		// v1
		u1 := datatypes.NewUUIDv1At(tm)
		ts1, ok1 := u1.Timestamp()
		if !ok1 || ts1.Unix() != unix {
			t.Fatalf("v1 timestamp mismatch: got %v, want %v", ts1, tm)
		}

		// v6
		u6 := datatypes.NewUUIDv6At(tm)
		ts6, ok6 := u6.Timestamp()
		if !ok6 || ts6.Unix() != unix {
			t.Fatalf("v6 timestamp mismatch: got %v, want %v", ts6, tm)
		}

		// v7
		u7 := datatypes.NewUUIDv7At(tm)
		ts7, ok7 := u7.Timestamp()
		if !ok7 || ts7.Unix() != unix {
			t.Fatalf("v7 timestamp mismatch: got %v, want %v", ts7, tm)
		}
	})
}

func TestUUID_CompareTo(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tm1 := time.Unix(rapid.Int64Range(0, 2147483647).Draw(t, "t1"), 0).UTC()
		tm2 := time.Unix(rapid.Int64Range(0, 2147483647).Draw(t, "t2"), 0).UTC()

		u1 := datatypes.NewUUIDv7At(tm1)
		u2 := datatypes.NewUUIDv7At(tm2)

		got := u1.CompareTo(u2)

		want := tm1.Compare(tm2)
		if want == 0 {
			// If timestamps are equal, fallback to bytes.Compare
			b1, b2 := u1.Bytes(), u2.Bytes()
			want = bytes.Compare(b1[:], b2[:])
		}

		if got != want {
			t.Fatalf("CompareTo mismatch: got %d, want %d for %v vs %v", got, want, u1, u2)
		}
	})
}

func TestUUID_Version(t *testing.T) {
	if datatypes.NewUUIDv1().Version() != 1 {
		t.Fatal("v1 version mismatch")
	}
	if datatypes.NewUUIDv4().Version() != 4 {
		t.Fatal("v4 version mismatch")
	}
	if datatypes.NewUUIDv6().Version() != 6 {
		t.Fatal("v6 version mismatch")
	}
	if datatypes.NewUUIDv7().Version() != 7 {
		t.Fatal("v7 version mismatch")
	}
}

func TestUUID_ParseErrors(t *testing.T) {
	tests := []string{
		"123e4567-e89b-12d3-a456-42661417400",
		"123e4567-e89b-12d3-a456-4266141740000",
		"123e4567-e89b-12d3-a456-42661417400Z",
		"",
	}
	for _, s := range tests {
		if _, err := datatypes.ParseUUID(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestUUID_IsZero(t *testing.T) {
	var zero datatypes.UUID
	if !zero.IsZero() {
		t.Fatal("Zero value should be zero")
	}
	if datatypes.NewUUID().IsZero() {
		t.Fatal("NewUUID should not be zero")
	}
}
