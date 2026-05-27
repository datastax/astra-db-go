package datatypes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

func TestDateOnly_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(-100000, 100000).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")

		daysInMonth := 31
		switch month {
		case 4, 6, 9, 11:
			daysInMonth = 30
		case 2:
			if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
				daysInMonth = 29
			} else {
				daysInMonth = 28
			}
		}
		day := rapid.IntRange(1, daysInMonth).Draw(t, "day")

		d := datatypes.NewDateOnly(year, month, day)
		s := d.String()
		parsed, err := datatypes.ParseDateOnly(s)
		if err != nil {
			t.Fatalf("ParseDateOnly(%q) failed: %v", s, err)
		}

		if diff := cmp.Diff(d, parsed); diff != "" {
			t.Fatalf("round-trip mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestDateOnly_CompareTo(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y1 := rapid.IntRange(-100000, 100000).Draw(t, "t1_y")
		m1 := rapid.IntRange(1, 12).Draw(t, "t1_m")
		d1 := rapid.IntRange(1, 28).Draw(t, "t1_d")

		y2 := rapid.IntRange(-100000, 100000).Draw(t, "t2_y")
		m2 := rapid.IntRange(1, 12).Draw(t, "t2_m")
		d2 := rapid.IntRange(1, 28).Draw(t, "t2_d")

		dt1 := datatypes.NewDateOnly(y1, m1, d1)
		dt2 := datatypes.NewDateOnly(y2, m2, d2)
		tm1 := time.Date(y1, time.Month(m1), d1, 0, 0, 0, 0, time.UTC)
		tm2 := time.Date(y2, time.Month(m2), d2, 0, 0, 0, 0, time.UTC)

		if got, want := dt1.CompareTo(dt2), tm1.Compare(tm2); got != want {
			t.Fatalf("CompareTo mismatch: got %d, want %d for %v vs %v", got, want, dt1, dt2)
		}
	})
}

func TestDateOnly_ToTime(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(-100000, 100000).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		d := datatypes.NewDateOnly(year, month, day)

		tm := d.ToTime()
		if tm.Year() != year || int(tm.Month()) != month || tm.Day() != day {
			t.Fatalf("ToTime component mismatch: got %v, want parts %d-%d-%d", tm, year, month, day)
		}
	})
}

func TestDateOnly_KnownFormats(t *testing.T) {
	tests := []struct {
		input string
		want  datatypes.DateOnly
	}{
		{"2024-01-01", datatypes.NewDateOnly(2024, 1, 1)},
		{"+12345-12-31", datatypes.NewDateOnly(12345, 12, 31)},
		{"-0500-05-05", datatypes.NewDateOnly(-500, 5, 5)},
		{"-0001-01-01", datatypes.NewDateOnly(-1, 1, 1)},
		{"0000-01-01", datatypes.NewDateOnly(0, 1, 1)},
		{"0999-12-31", datatypes.NewDateOnly(999, 12, 31)},
	}
	for _, tc := range tests {
		got, err := datatypes.ParseDateOnly(tc.input)
		if err != nil {
			t.Fatalf("ParseDateOnly(%q) failed: %v", tc.input, err)
		}
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("mismatch for %q (-want +got):\n%s", tc.input, diff)
		}
	}
}

func TestDateOnly_ParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too short", "2024-01"},
		{"invalid separator", "2024/01/01"},
		{"short year", "202-01-01"},
		{"short month", "2024-1-01"},
		{"month 0", "2000-00-01"},
		{"month 13", "2000-13-01"},
		{"day 0", "2000-01-00"},
		{"day 32", "2000-01-32"},
		{"not a leap year", "1999-02-29"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := datatypes.ParseDateOnly(tc.input); err == nil {
				t.Errorf("expected error for %q", tc.input)
			}
		})
	}
}

func TestDateOnly_ParseErrorMessages(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "datatypes: empty date string"},
		{"202-01-01", "datatypes: invalid date string (year part too short)"},
		{"2024/01/01", "datatypes: invalid date string (missing year separator)"},
		{"abcd-01-01", "datatypes: invalid date string (invalid year)"},
		{"2024-ab-01", "datatypes: invalid date string (invalid month)"},
		{"2024-01-ab", "datatypes: invalid date string (invalid day)"},
		{"2024-13-01", "datatypes: invalid date string (month out of range)"},
		{"2024-01-32", "datatypes: invalid date string (day out of range)"},
		{"1999-02-29", "datatypes: invalid date string (day out of range)"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			_, err := datatypes.ParseDateOnly(tc.input)
			if err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error message mismatch for %q: got %q, want it to contain %q", tc.input, err.Error(), tc.want)
			}
		})
	}
}

func TestDateOnly_Padding(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(0, 999).Draw(t, "year")
		d := datatypes.NewDateOnly(year, 1, 1)
		s := d.String()
		if len(s) != 10 {
			t.Fatalf("expected length 10 for year %d, got %q (len %d)", year, s, len(s))
		}
		expected := fmt.Sprintf("%04d-01-01", year)
		if s != expected {
			t.Fatalf("expected %q, got %q", expected, s)
		}
	})
}
