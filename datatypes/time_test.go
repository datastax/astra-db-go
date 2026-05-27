package datatypes_test

import (
	"strings"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

func TestTimeOnly_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		minute := rapid.IntRange(0, 59).Draw(t, "minute")
		second := rapid.IntRange(0, 59).Draw(t, "second")
		nano := rapid.IntRange(0, 999999999).Draw(t, "nano")

		tm := datatypes.NewTimeOnly(hour, minute, second, nano)
		s := tm.String()
		parsed, err := datatypes.ParseTimeOnly(s)
		if err != nil {
			t.Fatalf("ParseTimeOnly(%q) failed: %v", s, err)
		}
		if diff := cmp.Diff(tm, parsed); diff != "" {
			t.Fatalf("round-trip mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestTimeOnly_CompareTo(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		h1 := rapid.IntRange(0, 23).Draw(t, "t1_h")
		m1 := rapid.IntRange(0, 59).Draw(t, "t1_m")
		s1 := rapid.IntRange(0, 59).Draw(t, "t1_s")
		n1 := rapid.IntRange(0, 999999999).Draw(t, "t1_n")

		h2 := rapid.IntRange(0, 23).Draw(t, "t2_h")
		m2 := rapid.IntRange(0, 59).Draw(t, "t2_m")
		s2 := rapid.IntRange(0, 59).Draw(t, "t2_s")
		n2 := rapid.IntRange(0, 999999999).Draw(t, "t2_n")

		dt1 := datatypes.NewTimeOnly(h1, m1, s1, n1)
		dt2 := datatypes.NewTimeOnly(h2, m2, s2, n2)
		tm1 := time.Date(1970, 1, 1, h1, m1, s1, n1, time.UTC)
		tm2 := time.Date(1970, 1, 1, h2, m2, s2, n2, time.UTC)

		if got, want := dt1.CompareTo(dt2), tm1.Compare(tm2); got != want {
			t.Fatalf("CompareTo mismatch: got %d, want %d for %v vs %v", got, want, dt1, dt2)
		}
	})
}

func TestTimeOnly_FromTime(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		now := time.Now()
		tm := datatypes.TimeOnlyFromTime(now)
		if tm.Hour != now.Hour() || tm.Minute != now.Minute() || tm.Second != now.Second() || tm.Nanosecond != now.Nanosecond() {
			t.Fatalf("TimeOnlyFromTime mismatch for %v: got %+v", now, tm)
		}
	})
}

func TestTimeOnly_ParseShortFormats(t *testing.T) {
	tests := []struct {
		input string
		want  datatypes.TimeOnly
	}{
		{"12:30", datatypes.NewTimeOnly(12, 30, 0, 0)},
		{"12:30:45", datatypes.NewTimeOnly(12, 30, 45, 0)},
		{"12:30:45.123", datatypes.NewTimeOnly(12, 30, 45, 123000000)},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := datatypes.ParseTimeOnly(tc.input)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tc.input, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch for %q (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

func TestTimeOnly_ParseErrors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "datatypes: empty time string"},
		{"12", "datatypes: invalid time string format"},
		{"12:3", "datatypes: invalid time string format (HH:MM part length mismatch)"},
		{"12:30:4", "datatypes: invalid time string (second part length mismatch)"},
		{"24:00", "datatypes: invalid time string (hour out of range)"},
		{"12:60", "datatypes: invalid time string (minute out of range)"},
		{"12:30:60", "datatypes: invalid time string (second out of range)"},
		{"ab:cd", "datatypes: invalid time string (hour out of range)"},
		{"12:30:45.", "datatypes: invalid time string (empty fractional seconds)"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			_, err := datatypes.ParseTimeOnly(tc.input)
			if err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error message mismatch for %q: got %q, want it to contain %q", tc.input, err.Error(), tc.want)
			}
		})
	}
}
