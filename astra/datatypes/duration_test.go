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

package datatypes

import (
	"encoding/json"
	"math"
	"testing"
)

// ================================
// | ParseDuration – standard format
// ================================

func TestParseDuration_Standard(t *testing.T) {
	tests := []struct {
		input string
		want  Duration
	}{
		{"0s", Duration{0, 0, 0}},
		{"1y", Duration{12, 0, 0}},
		{"2mo", Duration{2, 0, 0}},
		{"3w", Duration{0, 21, 0}},
		{"4d", Duration{0, 4, 0}},
		{"5h", Duration{0, 0, 5 * NSPerHour}},
		{"6m", Duration{0, 0, 6 * NSPerMin}},
		{"7s", Duration{0, 0, 7 * NSPerSec}},
		{"8ms", Duration{0, 0, 8 * NSPerMS}},
		{"9us", Duration{0, 0, 9 * NSPerUS}},
		{"9µs", Duration{0, 0, 9 * NSPerUS}},
		{"10ns", Duration{0, 0, 10}},
		{"1y2mo3w4d5h6m7s8ms9us10ns", Duration{
			months:      12 + 2,
			days:        21 + 4,
			nanoseconds: 5*NSPerHour + 6*NSPerMin + 7*NSPerSec + 8*NSPerMS + 9*NSPerUS + 10,
		}},
		// case-insensitive
		{"1Y2MO3W4D5H6M7S8MS9US10NS", Duration{
			months:      14,
			days:        25,
			nanoseconds: 5*NSPerHour + 6*NSPerMin + 7*NSPerSec + 8*NSPerMS + 9*NSPerUS + 10,
		}},
		// negative
		{"-1y", Duration{-12, 0, 0}},
		{"-2w", Duration{0, -14, 0}},
		{"-5h6m", Duration{0, 0, -(5*NSPerHour + 6*NSPerMin)}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDuration_Standard_Invalid(t *testing.T) {
	tests := []string{
		"",
		"-",
		"abc",
		"1x",    // unknown unit
		"1y2y",  // repeated unit
		"2mo1y", // wrong order (mo before y)
		"1ms1s", // wrong order (ms before s)
	}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			_, err := ParseDuration(s)
			if err == nil {
				t.Errorf("ParseDuration(%q) expected error, got nil", s)
			}
		})
	}
}

// ================================
// | ParseDuration – ISO 8601 standard
// ================================

func TestParseDuration_ISOStandard(t *testing.T) {
	tests := []struct {
		input string
		want  Duration
	}{
		{"P1Y", Duration{12, 0, 0}},
		{"P2M", Duration{2, 0, 0}},
		{"P3D", Duration{0, 3, 0}},
		{"PT4H", Duration{0, 0, 4 * NSPerHour}},
		{"PT5M", Duration{0, 0, 5 * NSPerMin}},
		{"PT6S", Duration{0, 0, 6 * NSPerSec}},
		{"PT0S", Duration{0, 0, 0}},
		{"P1Y2M3DT4H5M6S", Duration{14, 3, 4*NSPerHour + 5*NSPerMin + 6*NSPerSec}},
		// fractional seconds
		{"PT6.007S", Duration{0, 0, 6*NSPerSec + 7*NSPerMS}},
		{"PT1.000000001S", Duration{0, 0, NSPerSec + 1}},
		{"PT0.5S", Duration{0, 0, 500 * NSPerMS}},
		// >9 fractional digits: sub-nanosecond precision truncated
		{"PT1.0000000001S", Duration{0, 0, NSPerSec}},
		{"PT0.9999999999S", Duration{0, 0, 999_999_999}},
		// negative
		{"-P7D", Duration{0, -7, 0}},
		{"-P1Y2M3DT4H5M6.007S", Duration{
			months:      -(12 + 2),
			days:        -3,
			nanoseconds: -(4*NSPerHour + 5*NSPerMin + 6*NSPerSec + 7*NSPerMS),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// ================================
// | ParseDuration – ISO 8601 week
// ================================

func TestParseDuration_ISOWeek(t *testing.T) {
	tests := []struct {
		input string
		want  Duration
	}{
		{"P0W", Duration{0, 0, 0}},
		{"P1W", Duration{0, 7, 0}},
		{"P2W", Duration{0, 14, 0}},
		{"-P2W", Duration{0, -14, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// ================================
// | ParseDuration – ISO 8601 alternate
// ================================

func TestParseDuration_ISOAlternate(t *testing.T) {
	tests := []struct {
		input string
		want  Duration
	}{
		{"P0001-02-03T04:05:06", Duration{
			months:      12 + 2,
			days:        3,
			nanoseconds: 4*NSPerHour + 5*NSPerMin + 6*NSPerSec,
		}},
		{"P0000-00-00T00:00:00", Duration{0, 0, 0}},
		{"-P0001-02-03T04:05:06", Duration{
			months:      -(12 + 2),
			days:        -3,
			nanoseconds: -(4*NSPerHour + 5*NSPerMin + 6*NSPerSec),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// ================================
// | NewDuration validation
// ================================

func TestNewDuration(t *testing.T) {
	t.Run("valid positive", func(t *testing.T) {
		d, err := NewDuration(12, 3, NSPerHour)
		if err != nil {
			t.Fatal(err)
		}
		if d.Months() != 12 || d.Days() != 3 || d.Nanoseconds() != NSPerHour {
			t.Errorf("unexpected: %+v", d)
		}
	})

	t.Run("valid negative", func(t *testing.T) {
		d, err := NewDuration(-12, -3, -NSPerHour)
		if err != nil {
			t.Fatal(err)
		}
		if d.Months() != -12 {
			t.Errorf("unexpected months: %d", d.Months())
		}
	})

	t.Run("valid zero", func(t *testing.T) {
		d, err := NewDuration(0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if !d.IsZero() {
			t.Error("expected zero")
		}
	})

	t.Run("mixed signs rejected", func(t *testing.T) {
		_, err := NewDuration(1, -1, 0)
		if err == nil {
			t.Error("expected error for mixed signs")
		}
	})

	t.Run("mixed signs rejected 2", func(t *testing.T) {
		_, err := NewDuration(0, 1, -NSPerSec)
		if err == nil {
			t.Error("expected error for mixed signs")
		}
	})

	t.Run("out of range months rejected", func(t *testing.T) {
		_, err := NewDuration(math.MinInt32, 0, 0)
		if err == nil {
			t.Error("expected error for math.MinInt32 months")
		}
	})

	t.Run("out of range nanos rejected", func(t *testing.T) {
		_, err := NewDuration(0, 0, math.MinInt64)
		if err == nil {
			t.Error("expected error for math.MinInt64 nanos")
		}
	})
}

// ================================
// | Methods
// ================================

func TestDuration_Equals(t *testing.T) {
	d1 := MustParseDuration("1y2d")
	d2 := MustParseDuration("P1Y2D")
	d3 := MustParseDuration("-7d")
	d4 := MustParseDuration("-P1W")

	if !d1.Equals(d2) {
		t.Errorf("1y2d should equal P1Y2D")
	}
	if !d3.Equals(d4) {
		t.Errorf("-7d should equal -P1W")
	}
	if d1.Equals(d3) {
		t.Errorf("1y2d should not equal -7d")
	}
}

func TestDuration_IsNegative(t *testing.T) {
	if MustParseDuration("1y").IsNegative() {
		t.Error("1y should not be negative")
	}
	if !MustParseDuration("-1y").IsNegative() {
		t.Error("-1y should be negative")
	}
	if MustParseDuration("0s").IsNegative() {
		t.Error("0s should not be negative")
	}
}

func TestDuration_IsZero(t *testing.T) {
	if !MustParseDuration("0s").IsZero() {
		t.Error("0s should be zero")
	}
	if !MustParseDuration("PT0S").IsZero() {
		t.Error("PT0S should be zero")
	}
	if MustParseDuration("1ns").IsZero() {
		t.Error("1ns should not be zero")
	}
}

func TestDuration_HasDayPrecision(t *testing.T) {
	if !MustParseDuration("1y2d").HasDayPrecision() {
		t.Error("1y2d has day precision")
	}
	if MustParseDuration("1h").HasDayPrecision() {
		t.Error("1h does not have day precision")
	}
}

func TestDuration_HasMillisecondPrecision(t *testing.T) {
	if !MustParseDuration("1s").HasMillisecondPrecision() {
		t.Error("1s has millisecond precision")
	}
	if !MustParseDuration("5ms").HasMillisecondPrecision() {
		t.Error("5ms has millisecond precision")
	}
	if MustParseDuration("1us").HasMillisecondPrecision() {
		t.Error("1us does not have millisecond precision")
	}
}

func TestDuration_Plus(t *testing.T) {
	a := MustParseDuration("1y")
	b := MustParseDuration("1mo1s")
	got, ok := a.Plus(b)
	if !ok {
		t.Fatal("Plus returned false for same-sign durations")
	}
	want := Duration{months: 13, days: 0, nanoseconds: NSPerSec}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	neg := MustParseDuration("-1mo")
	_, ok = a.Plus(neg)
	if ok {
		t.Error("Plus should return false for opposite signs")
	}

	t.Run("overflow months", func(t *testing.T) {
		d1 := Duration{months: MaxMonthsDays, days: 0, nanoseconds: 0}
		d2 := Duration{months: 1, days: 0, nanoseconds: 0}
		if _, ok := d1.Plus(d2); ok {
			t.Error("Plus should return false on month overflow")
		}
	})

	t.Run("overflow nanos", func(t *testing.T) {
		d1 := Duration{months: 0, days: 0, nanoseconds: MaxNanos}
		d2 := Duration{months: 0, days: 0, nanoseconds: 1}
		if _, ok := d1.Plus(d2); ok {
			t.Error("Plus should return false on nanos overflow")
		}
	})
}

func TestDuration_Negate(t *testing.T) {
	d := MustParseDuration("1y")
	neg := d.Negate()
	if !neg.IsNegative() {
		t.Error("negated duration should be negative")
	}
	if neg.Months() != -12 {
		t.Errorf("expected months -12, got %d", neg.Months())
	}
	if !neg.Negate().Equals(d) {
		t.Error("double negation should return original")
	}
}

func TestDuration_Abs(t *testing.T) {
	pos := MustParseDuration("1y")
	neg := MustParseDuration("-1y")
	if !neg.Abs().Equals(pos) {
		t.Error("Abs of -1y should equal 1y")
	}
	if !pos.Abs().Equals(pos) {
		t.Error("Abs of 1y should equal 1y")
	}
}

// ================================
// | String representation
// ================================

func TestDuration_String(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"15mo", "1y3mo"},
		{"1y", "1y"},
		{"25d", "25d"},
		{"1y2mo3w4d5h6m7s8ms9us10ns", "1y2mo25d5h6m7s8ms9us10ns"},
		{"-5ms10000us", "-15ms"},
		{"0s", "0s"},
		{"PT0S", "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := MustParseDuration(tt.input)
			got := d.String()
			if got != tt.want {
				t.Errorf("ParseDuration(%q).String() = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDuration_AppendShortString(t *testing.T) {
	tests := []struct {
		d    Duration
		want string
	}{
		{Duration{0, 0, 0}, "0s"},
		{Duration{14, 3, 3_600_000_000_000}, "14mo3d3600000000000ns"},
		{Duration{-14, -3, -3_600_000_000_000}, "-14mo3d3600000000000ns"},
		{Duration{12, 0, 0}, "12mo"},
		{Duration{0, 7, 0}, "7d"},
		{Duration{0, 0, 1}, "1ns"},
	}

	for _, tt := range tests {
		got := string(tt.d.AppendShortString(nil))
		if got != tt.want {
			t.Errorf("AppendShortString(%+v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// ================================
// | JSON round-trip
// ================================

func TestDuration_JSONRoundTrip(t *testing.T) {
	original := MustParseDuration("1y2mo3w4d5h6m7s8ms9us10ns")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Duration
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !original.Equals(parsed) {
		t.Errorf("round-trip mismatch: %+v != %+v", original, parsed)
	}
}

func TestDuration_JSONInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not a string", `123`},
		{"invalid duration", `"notaduration"`},
		{"unknown unit", `"1x"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			if err := json.Unmarshal([]byte(tt.input), &d); err == nil {
				t.Errorf("expected error for %q, got nil", tt.input)
			}
		})
	}
}

// ================================
// | DurationBuilder
// ================================

func TestDurationBuilder(t *testing.T) {
	t.Run("basic build", func(t *testing.T) {
		d, err := NewDurationBuilder().
			AddYears(1).
			AddDays(3).
			AddSeconds(5).
			Build()
		if err != nil {
			t.Fatal(err)
		}
		want := Duration{months: 12, days: 3, nanoseconds: 5 * NSPerSec}
		if d != want {
			t.Errorf("got %+v, want %+v", d, want)
		}
	})

	t.Run("negate", func(t *testing.T) {
		d, err := NewDurationBuilder().AddHours(10).Negate().Build()
		if err != nil {
			t.Fatal(err)
		}
		if d.Nanoseconds() != -10*NSPerHour {
			t.Errorf("unexpected nanos: %d", d.Nanoseconds())
		}
	})

	t.Run("from base", func(t *testing.T) {
		base := MustParseDuration("1y")
		d, err := NewDurationBuilderFrom(base).AddDays(3).Build()
		if err != nil {
			t.Fatal(err)
		}
		want := Duration{months: 12, days: 3, nanoseconds: 0}
		if d != want {
			t.Errorf("got %+v, want %+v", d, want)
		}
	})

	t.Run("overflow months", func(t *testing.T) {
		_, err := NewDurationBuilder().AddMonths(MaxMonthsDays).AddYears(1).Build()
		if err == nil {
			t.Error("expected overflow error")
		}
	})

	t.Run("symmetry math.MinInt32 rejected", func(t *testing.T) {
		_, err := NewDurationBuilder().AddMonths(math.MinInt32).Build()
		if err == nil {
			t.Error("expected error for math.MinInt32 months")
		}
	})

	t.Run("all time units", func(t *testing.T) {
		d, err := NewDurationBuilder().
			AddHours(1).
			AddMinutes(1).
			AddSeconds(1).
			AddMillis(1).
			AddMicros(1).
			AddNanos(1).
			Build()
		if err != nil {
			t.Fatal(err)
		}
		want := NSPerHour + NSPerMin + NSPerSec + NSPerMS + NSPerUS + 1
		if d.Nanoseconds() != want {
			t.Errorf("got nanos %d, want %d", d.Nanoseconds(), want)
		}
	})
}

// ================================
// | Cross-format equivalence
// ================================

func TestParseDuration_CrossFormat(t *testing.T) {
	// Same duration expressed in different formats should produce identical results
	pairs := [][2]string{
		{"7d", "P1W"},
		{"14d", "P2W"},
		{"1y", "P1Y"},
		{"12mo", "P1Y"},
		{"1y", "12mo"},
		{"-7d", "-P1W"},
	}
	for _, pair := range pairs {
		a, err := ParseDuration(pair[0])
		if err != nil {
			t.Fatalf("ParseDuration(%q): %v", pair[0], err)
		}
		b, err := ParseDuration(pair[1])
		if err != nil {
			t.Fatalf("ParseDuration(%q): %v", pair[1], err)
		}
		if !a.Equals(b) {
			t.Errorf("ParseDuration(%q) = %+v != ParseDuration(%q) = %+v", pair[0], a, pair[1], b)
		}
	}
}
