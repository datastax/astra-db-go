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
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Nanosecond conversion constants.
const (
	NSPerHour = int64(3_600_000_000_000)
	NSPerMin  = int64(60_000_000_000)
	NSPerSec  = int64(1_000_000_000)
	NSPerMS   = int64(1_000_000)
	NSPerUS   = int64(1_000)
)

const (
	MaxMonthsDays = int32(2_147_483_647)
	MaxNanos      = int64(9_223_372_036_854_775_807)
)

// Duration represents a Cassandra duration type stored as months, days, and nanoseconds.
// These three components are not directly convertible to each other (months are calendar months,
// days are calendar days, and nanoseconds are a fixed-length time unit).
// All three components must share the same sign.
type Duration struct {
	months      int32
	days        int32
	nanoseconds int64
}

// NewDuration creates a Duration from months, days, and nanoseconds.
// All components must share the same sign and be within valid ranges:
// months and days in [-2147483647, 2147483647], nanoseconds in [-9223372036854775807, 9223372036854775807].
func NewDuration(months int32, days int32, nanoseconds int64) (Duration, error) {
	if err := validateDuration(months, days, nanoseconds); err != nil {
		return Duration{}, err
	}
	return Duration{months, days, nanoseconds}, nil
}

// Months returns the months component of the duration.
func (d Duration) Months() int32 {
	return d.months
}

// Days returns the days component of the duration.
func (d Duration) Days() int32 {
	return d.days
}

// Nanoseconds returns the nanoseconds component of the duration.
func (d Duration) Nanoseconds() int64 {
	return d.nanoseconds
}

// MustParseDuration parses a duration string and panics on error.
// Intended for use in tests and with known-valid inputs.
func MustParseDuration(s string) Duration {
	d, err := ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("MustParseDuration(%q): %v", s, err))
	}
	return d
}

// ParseDuration parses a duration string in one of four formats:
//
//   - Standard format: -?(<number><unit>)+
//     Units (case-insensitive): y, mo, w, d, h, m, s, ms, us, µs, ns
//     Example: "1y2mo3w4d5h6m7s8ms9us10ns"
//
//   - ISO 8601 standard: -?P[nY][nM][nD][T[nH][nM][n[.f]S]]
//     Example: "P1Y2M3DT4H5M6.007S"
//
//   - ISO 8601 week: -?PnW
//     Example: "P2W"
//
//   - ISO 8601 alternate: -?PYYYY-MM-DDThh:mm:ss
//     Example: "P0001-02-03T04:05:06"
func ParseDuration(s string) (Duration, error) {
	if s == "" {
		return Duration{}, fmt.Errorf("invalid duration: empty string")
	}

	negative := s[0] == '-'
	rest := s
	if negative {
		rest = s[1:]
	}

	if rest == "" {
		return Duration{}, fmt.Errorf("invalid duration: empty after sign")
	}

	var d Duration
	var err error

	if rest[0] == 'P' {
		switch {
		case strings.HasSuffix(rest, "W"):
			d, err = parseISOWeekDuration(rest)
		case strings.Contains(rest, "-"):
			d, err = parseISOAlternateDuration(rest)
		default:
			d, err = parseISOStandardDuration(rest)
		}
	} else {
		d, err = parseBasicDuration(rest, s)
	}

	if err != nil {
		return Duration{}, err
	}

	if negative {
		d = d.Negate()
	}
	return d, nil
}

// basicDurationPartRe matches a single number+unit pair in the standard duration format.
// mo and ms must appear before m to avoid a prefix match cutting them short.
var basicDurationPartRe = regexp.MustCompile(`^(\d+)(y|mo|w|d|h|s|ms|us|µs|ns|m)`)

var basicUnitOrderIndex = map[string]int{
	"y": 0, "mo": 1, "w": 2, "d": 3,
	"h": 4, "m": 5, "s": 6, "ms": 7, "us": 8, "µs": 8, "ns": 9,
}

func parseBasicDuration(s, original string) (Duration, error) {
	lower := strings.ToLower(s)
	b := NewDurationBuilder()
	lastIdx := -1
	hasAny := false

	for i := 0; i < len(lower); {
		m := basicDurationPartRe.FindStringSubmatch(lower[i:])
		if m == nil {
			return Duration{}, fmt.Errorf("invalid standard duration string: %q", original)
		}

		num, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return Duration{}, fmt.Errorf("invalid duration number in %q: %w", original, err)
		}

		unit := m[2]
		idx := basicUnitOrderIndex[unit]

		if idx <= lastIdx {
			return Duration{}, fmt.Errorf("invalid standard duration string %q: units must appear in order and at most once", original)
		}
		lastIdx = idx

		switch unit {
		case "y":
			b.AddYears(int32(num))
		case "mo":
			b.AddMonths(int32(num))
		case "w":
			b.AddWeeks(int32(num))
		case "d":
			b.AddDays(int32(num))
		case "h":
			b.AddHours(num)
		case "m":
			b.AddMinutes(num)
		case "s":
			b.AddSeconds(num)
		case "ms":
			b.AddMillis(num)
		case "us", "µs":
			b.AddMicros(num)
		case "ns":
			b.AddNanos(num)
		}

		hasAny = true
		i += len(m[0])
	}

	if !hasAny {
		return Duration{}, fmt.Errorf("invalid standard duration string: %q", original)
	}

	d, err := b.Build()
	if err != nil {
		return Duration{}, fmt.Errorf("invalid duration %q: %w", original, err)
	}
	return d, nil
}

// isoStandardDurationRe matches ISO 8601 standard duration: P[nY][nM][nD][T[nH][nM][n[.f]S]]
var isoStandardDurationRe = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)(?:\.(\d+))?S)?)?$`)

func parseISOStandardDuration(s string) (Duration, error) {
	m := isoStandardDurationRe.FindStringSubmatch(s)
	if m == nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 standard duration string: %q", s)
	}

	b := NewDurationBuilder()
	if m[1] != "" {
		n, _ := strconv.ParseInt(m[1], 10, 32)
		b.AddYears(int32(n))
	}
	if m[2] != "" {
		n, _ := strconv.ParseInt(m[2], 10, 32)
		b.AddMonths(int32(n))
	}
	if m[3] != "" {
		n, _ := strconv.ParseInt(m[3], 10, 32)
		b.AddDays(int32(n))
	}

	if m[4] != "" {
		n, _ := strconv.ParseInt(m[4], 10, 64)
		b.AddHours(n)
	}
	if m[5] != "" {
		n, _ := strconv.ParseInt(m[5], 10, 64)
		b.AddMinutes(n)
	}
	if m[6] != "" {
		n, _ := strconv.ParseInt(m[6], 10, 64)
		b.AddSeconds(n)
	}
	if m[7] != "" {
		fracStr := m[7]
		if len(fracStr) > 9 {
			fracStr = fracStr[:9]
		}
		frac, _ := strconv.ParseInt(fracStr, 10, 32)
		b.AddNanos(frac * int64(math.Pow10(9-len(fracStr))))
	}

	d, err := b.Build()
	if err != nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 duration %q: %w", s, err)
	}
	return d, nil
}

// isoWeekDurationRe matches ISO 8601 week duration: PnW
var isoWeekDurationRe = regexp.MustCompile(`^P(\d+)W$`)

func parseISOWeekDuration(s string) (Duration, error) {
	m := isoWeekDurationRe.FindStringSubmatch(s)
	if m == nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 week duration string: %q", s)
	}

	weeks, _ := strconv.ParseInt(m[1], 10, 32)
	b := NewDurationBuilder()
	b.AddWeeks(int32(weeks))

	d, err := b.Build()
	if err != nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 week duration %q: %w", s, err)
	}
	return d, nil
}

// isoAlternateDurationRe matches ISO 8601 alternate duration: PYYYY-MM-DDThh:mm:ss
var isoAlternateDurationRe = regexp.MustCompile(`^P(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})$`)

func parseISOAlternateDuration(s string) (Duration, error) {
	m := isoAlternateDurationRe.FindStringSubmatch(s)
	if m == nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 alternate duration string: %q", s)
	}

	years, _ := strconv.ParseInt(m[1], 10, 32)
	mons, _ := strconv.ParseInt(m[2], 10, 32)
	days, _ := strconv.ParseInt(m[3], 10, 32)
	hours, _ := strconv.ParseInt(m[4], 10, 64)
	mins, _ := strconv.ParseInt(m[5], 10, 64)
	secs, _ := strconv.ParseInt(m[6], 10, 64)

	b := NewDurationBuilder()
	b.AddYears(int32(years)).AddMonths(int32(mons)).AddDays(int32(days)).
		AddHours(hours).
		AddMinutes(mins).
		AddSeconds(secs)

	d, err := b.Build()
	if err != nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 alternate duration %q: %w", s, err)
	}
	return d, nil
}

func validateDuration(months int32, days int32, nanoseconds int64) error {
	if months < -MaxMonthsDays || months > MaxMonthsDays {
		return fmt.Errorf("invalid duration months %d: out of range [-%d, %d]", months, MaxMonthsDays, MaxMonthsDays)
	}
	if days < -MaxMonthsDays || days > MaxMonthsDays {
		return fmt.Errorf("invalid duration days %d: out of range [-%d, %d]", days, MaxMonthsDays, MaxMonthsDays)
	}
	if nanoseconds < -MaxNanos || nanoseconds > MaxNanos {
		return fmt.Errorf("invalid duration nanoseconds %d: out of range [-%d, %d]", nanoseconds, MaxNanos, MaxNanos)
	}

	allNonNeg := months >= 0 && days >= 0 && nanoseconds >= 0
	allNonPos := months <= 0 && days <= 0 && nanoseconds <= 0
	if !allNonNeg && !allNonPos {
		return fmt.Errorf("invalid duration (%d, %d, %d): all components must share the same sign", months, days, nanoseconds)
	}
	return nil
}

// Equals returns true if both durations have identical component values.
func (d Duration) Equals(other Duration) bool {
	return d.months == other.months && d.days == other.days && d.nanoseconds == other.nanoseconds
}

// HasDayPrecision returns true if the nanoseconds component is zero (no sub-day precision).
func (d Duration) HasDayPrecision() bool {
	return d.nanoseconds == 0
}

// HasMillisecondPrecision returns true if the nanoseconds component is a multiple of 1,000,000.
func (d Duration) HasMillisecondPrecision() bool {
	return d.nanoseconds%NSPerMS == 0
}

// IsNegative returns true if any component is negative.
func (d Duration) IsNegative() bool {
	return d.months < 0 || d.days < 0 || d.nanoseconds < 0
}

// IsZero returns true if all components are zero.
func (d Duration) IsZero() bool {
	return d.months == 0 && d.days == 0 && d.nanoseconds == 0
}

// Plus returns the component-wise sum of two durations.
// Returns (result, true) on success, or (zero, false) if the durations have opposite signs or the result is invalid (e.g. overflow).
func (d Duration) Plus(other Duration) (Duration, bool) {
	if d.IsZero() {
		return other, true
	}
	if other.IsZero() {
		return d, true
	}
	if d.IsNegative() != other.IsNegative() {
		return Duration{}, false
	}

	resMonths := d.months + other.months
	resDays := d.days + other.days
	resNanos := d.nanoseconds + other.nanoseconds

	// Overflow check: if operands have same sign, result must have same sign (or be zero)
	isNeg := d.IsNegative()
	if (isNeg && (resMonths > 0 || resDays > 0 || resNanos > 0)) ||
		(!isNeg && (resMonths < 0 || resDays < 0 || resNanos < 0)) {
		return Duration{}, false
	}

	if err := validateDuration(resMonths, resDays, resNanos); err != nil {
		return Duration{}, false
	}

	return Duration{
		months:      resMonths,
		days:        resDays,
		nanoseconds: resNanos,
	}, true
}

// Negate returns a new Duration with all components negated.
func (d Duration) Negate() Duration {
	return Duration{months: -d.months, days: -d.days, nanoseconds: -d.nanoseconds}
}

// Abs returns the duration with a positive sign, negating if needed.
func (d Duration) Abs() Duration {
	if d.IsNegative() {
		return d.Negate()
	}
	return d
}

// String returns the human-readable long-form representation (e.g. "1y3mo25d5h6m7s8ms9us10ns").
func (d Duration) String() string {
	return durationToLongString(d)
}

// AppendShortString appends the compact API wire format to dst.
// Months, days, and nanoseconds are each expressed as a raw count with a single unit suffix.
// Example: Duration{14, 3, 3_600_000_000_000} → "14mo3d3600000000000ns"
func (d Duration) AppendShortString(dst []byte) []byte {
	if d.IsZero() {
		return append(dst, "0s"...)
	}

	if d.IsNegative() {
		dst = append(dst, '-')
	}

	m := d.months
	if m < 0 {
		m = -m
	}
	if m != 0 {
		dst = strconv.AppendInt(dst, int64(m), 10)
		dst = append(dst, "mo"...)
	}

	dy := d.days
	if dy < 0 {
		dy = -dy
	}
	if dy != 0 {
		dst = strconv.AppendInt(dst, int64(dy), 10)
		dst = append(dst, 'd')
	}

	ns := d.nanoseconds
	if ns < 0 {
		ns = -ns
	}
	if ns != 0 {
		dst = strconv.AppendInt(dst, ns, 10)
		dst = append(dst, "ns"...)
	}

	return dst
}

func durationToLongString(d Duration) string {
	if d.IsZero() {
		return "0s"
	}

	var sb strings.Builder

	if d.IsNegative() {
		sb.WriteByte('-')
	}

	months := d.months
	if months < 0 {
		months = -months
	}
	if months != 0 {
		if years := months / 12; years != 0 {
			sb.WriteString(strconv.FormatInt(int64(years), 10))
			sb.WriteByte('y')
		}
		if rem := months % 12; rem != 0 {
			sb.WriteString(strconv.FormatInt(int64(rem), 10))
			sb.WriteString("mo")
		}
	}

	days := d.days
	if days < 0 {
		days = -days
	}
	if days != 0 {
		sb.WriteString(strconv.FormatInt(int64(days), 10))
		sb.WriteByte('d')
	}

	nanos := d.nanoseconds
	if nanos < 0 {
		nanos = -nanos
	}
	if nanos != 0 {
		nanos = appendNanoUnit(&sb, nanos, NSPerHour, "h")
		nanos = appendNanoUnit(&sb, nanos, NSPerMin, "m")
		nanos = appendNanoUnit(&sb, nanos, NSPerSec, "s")
		nanos = appendNanoUnit(&sb, nanos, NSPerMS, "ms")
		nanos = appendNanoUnit(&sb, nanos, NSPerUS, "us")
		appendNanoUnit(&sb, nanos, 1, "ns")
	}

	return sb.String()
}

func appendNanoUnit(sb *strings.Builder, value, unitSize int64, unit string) int64 {
	if value >= unitSize {
		sb.WriteString(strconv.FormatInt(value/unitSize, 10))
		sb.WriteString(unit)
		return value % unitSize
	}
	return value
}

// MarshalJSON implements json.Marshaler using the compact string wire format.
func (d Duration) MarshalJSON() ([]byte, error) {
	s := string(d.AppendShortString(nil))
	return json.Marshal(s)
}

// UnmarshalJSON implements json.Unmarshaler, parsing the duration from a string.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("datatypes: invalid duration JSON: %w", err)
	}
	parsed, err := ParseDuration(s)
	if err != nil {
		return fmt.Errorf("datatypes: %w", err)
	}
	*d = parsed
	return nil
}

// ================================
// | DurationBuilder
// ================================

// DurationBuilder builds a Duration incrementally.
// Use NewDurationBuilder or NewDurationBuilderFrom to create one.
// Errors from Add* methods accumulate and are returned on Build.
type DurationBuilder struct {
	months      int32
	days        int32
	nanoseconds int64
	negative    bool
	err         error
}

// NewDurationBuilder returns a new empty DurationBuilder.
func NewDurationBuilder() *DurationBuilder {
	return &DurationBuilder{}
}

// NewDurationBuilderFrom returns a DurationBuilder initialized from an existing Duration.
func NewDurationBuilderFrom(base Duration) *DurationBuilder {
	abs := base.Abs()
	return &DurationBuilder{
		months:      abs.months,
		days:        abs.days,
		nanoseconds: abs.nanoseconds,
		negative:    base.IsNegative(),
	}
}

// Negate toggles whether the built duration is negative.
func (b *DurationBuilder) Negate() *DurationBuilder {
	b.negative = !b.negative
	return b
}

// SetNegative explicitly sets the sign of the built duration.
func (b *DurationBuilder) SetNegative(negative bool) *DurationBuilder {
	b.negative = negative
	return b
}

func (b *DurationBuilder) AddYears(n int32) *DurationBuilder {
	return b.addMonths(n, 12, "years")
}

func (b *DurationBuilder) AddMonths(n int32) *DurationBuilder {
	return b.addMonths(n, 1, "months")
}

func (b *DurationBuilder) AddWeeks(n int32) *DurationBuilder {
	return b.addDays(n, 7, "weeks")
}

func (b *DurationBuilder) AddDays(n int32) *DurationBuilder {
	return b.addDays(n, 1, "days")
}

// AddHours adds n hours (converted to nanoseconds).
func (b *DurationBuilder) AddHours(n int64) *DurationBuilder {
	return b.addNanos(n, NSPerHour, "hours")
}

// AddMinutes adds n minutes (converted to nanoseconds).
func (b *DurationBuilder) AddMinutes(n int64) *DurationBuilder {
	return b.addNanos(n, NSPerMin, "minutes")
}

// AddSeconds adds n seconds (converted to nanoseconds).
func (b *DurationBuilder) AddSeconds(n int64) *DurationBuilder {
	return b.addNanos(n, NSPerSec, "seconds")
}

// AddMillis adds n milliseconds (converted to nanoseconds).
func (b *DurationBuilder) AddMillis(n int64) *DurationBuilder {
	return b.addNanos(n, NSPerMS, "milliseconds")
}

// AddMicros adds n microseconds (converted to nanoseconds).
func (b *DurationBuilder) AddMicros(n int64) *DurationBuilder {
	return b.addNanos(n, NSPerUS, "microseconds")
}

// AddNanos adds n nanoseconds.
func (b *DurationBuilder) AddNanos(n int64) *DurationBuilder {
	return b.addNanos(n, 1, "nanoseconds")
}

func (b *DurationBuilder) addMonths(n int32, monthsPerUnit int64, name string) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.months) + int64(n)*monthsPerUnit
	if v < -int64(MaxMonthsDays) || int64(MaxMonthsDays) < v {
		b.err = fmt.Errorf("invalid duration: total months %d out of range [-%d, %d] (tried to add %d %s)", v, MaxMonthsDays, MaxMonthsDays, n, name)
		return b
	}
	b.months = int32(v)
	return b
}

func (b *DurationBuilder) addDays(n int32, daysPerUnit int64, name string) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.days) + int64(n)*daysPerUnit
	if v < -int64(MaxMonthsDays) || int64(MaxMonthsDays) < v {
		b.err = fmt.Errorf("invalid duration: total days %d out of range [-%d, %d] (tried to add %d %s)", v, MaxMonthsDays, MaxMonthsDays, n, name)
		return b
	}
	b.days = int32(v)
	return b
}

func (b *DurationBuilder) addNanos(n, nsPerUnit int64, name string) *DurationBuilder {
	if b.err != nil {
		return b
	}

	if n > 0 && n > (MaxNanos-b.nanoseconds)/nsPerUnit {
		b.err = fmt.Errorf("invalid duration: total nanoseconds out of range [-%d, %d] (tried to add %d %s)", MaxNanos, MaxNanos, n, name)
		return b
	}
	if n < 0 && n < (-MaxNanos-b.nanoseconds)/nsPerUnit {
		b.err = fmt.Errorf("invalid duration: total nanoseconds out of range [-%d, %d] (tried to add %d %s)", MaxNanos, MaxNanos, n, name)
		return b
	}

	b.nanoseconds += n * nsPerUnit
	return b
}

// Build returns the constructed Duration, or an error if any Add* call failed.
func (b *DurationBuilder) Build() (Duration, error) {
	if b.err != nil {
		return Duration{}, b.err
	}

	months, days, nanos := b.months, b.days, b.nanoseconds
	if b.negative {
		months, days, nanos = -months, -days, -nanos
	}

	if err := validateDuration(months, days, nanos); err != nil {
		return Duration{}, err
	}

	return Duration{months, days, nanos}, nil
}
