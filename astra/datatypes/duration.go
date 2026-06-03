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
	Months      int32
	Days        int32
	Nanoseconds int64
}

// NewDuration creates a Duration from months, days, and nanoseconds.
// All components must share the same sign and be within valid ranges:
// months and days in [-2147483647, 2147483647], nanoseconds in [-9223372036854775807, 9223372036854775807].
func NewDuration(months int32, days int32, nanoseconds int64) (Duration, error) {
	if err := validateDuration(months, days, nanoseconds); err != nil {
		return Duration{}, err
	}
	return Duration{Months: months, Days: days, Nanoseconds: nanoseconds}, nil
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
			b.addYears(num)
		case "mo":
			b.addMonths(num)
		case "w":
			b.addWeeks(num)
		case "d":
			b.addDays(num)
		case "h":
			b.addNanos(num, NSPerHour, "hours")
		case "m":
			b.addNanos(num, NSPerMin, "minutes")
		case "s":
			b.addNanos(num, NSPerSec, "seconds")
		case "ms":
			b.addNanos(num, NSPerMS, "milliseconds")
		case "us", "µs":
			b.addNanos(num, NSPerUS, "microseconds")
		case "ns":
			b.addNanos(num, 1, "nanoseconds")
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
		n, _ := strconv.ParseInt(m[1], 10, 64)
		b.addYears(n)
	}
	if m[2] != "" {
		n, _ := strconv.ParseInt(m[2], 10, 64)
		b.addMonths(n)
	}
	if m[3] != "" {
		n, _ := strconv.ParseInt(m[3], 10, 64)
		b.addDays(n)
	}
	if m[4] != "" {
		n, _ := strconv.ParseInt(m[4], 10, 64)
		b.addNanos(n, NSPerHour, "hours")
	}
	if m[5] != "" {
		n, _ := strconv.ParseInt(m[5], 10, 64)
		b.addNanos(n, NSPerMin, "minutes")
	}
	if m[6] != "" {
		n, _ := strconv.ParseInt(m[6], 10, 64)
		b.addNanos(n, NSPerSec, "seconds")
	}
	if m[7] != "" {
		frac, _ := strconv.ParseInt(m[7], 10, 64)
		b.addNanos(int64(float64(frac)*math.Pow10(9-len(m[7]))), 1, "nanoseconds")
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

	weeks, _ := strconv.ParseInt(m[1], 10, 64)
	b := NewDurationBuilder()
	b.addWeeks(weeks)

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

	years, _ := strconv.ParseInt(m[1], 10, 64)
	mons, _ := strconv.ParseInt(m[2], 10, 64)
	days, _ := strconv.ParseInt(m[3], 10, 64)
	hours, _ := strconv.ParseInt(m[4], 10, 64)
	mins, _ := strconv.ParseInt(m[5], 10, 64)
	secs, _ := strconv.ParseInt(m[6], 10, 64)

	b := NewDurationBuilder()
	b.addYears(years).addMonths(mons).addDays(days).
		addNanos(hours, NSPerHour, "hours").
		addNanos(mins, NSPerMin, "minutes").
		addNanos(secs, NSPerSec, "seconds")

	d, err := b.Build()
	if err != nil {
		return Duration{}, fmt.Errorf("invalid ISO 8601 alternate duration %q: %w", s, err)
	}
	return d, nil
}

func validateDuration(months int32, days int32, nanoseconds int64) error {
	allNonNeg := months >= 0 && days >= 0 && nanoseconds >= 0
	allNonPos := months <= 0 && days <= 0 && nanoseconds <= 0
	if !allNonNeg && !allNonPos {
		return fmt.Errorf("invalid duration (%d, %d, %d): all components must share the same sign", months, days, nanoseconds)
	}
	if months < -MaxMonthsDays || months > MaxMonthsDays {
		return fmt.Errorf("invalid duration: months %d out of range [%d, %d]", months, -MaxMonthsDays, MaxMonthsDays)
	}
	if days < -MaxMonthsDays || days > MaxMonthsDays {
		return fmt.Errorf("invalid duration: days %d out of range [%d, %d]", days, -MaxMonthsDays, MaxMonthsDays)
	}
	if nanoseconds < -MaxNanos || nanoseconds > MaxNanos {
		return fmt.Errorf("invalid duration: nanoseconds %d out of range [%d, %d]", nanoseconds, -MaxNanos, MaxNanos)
	}
	return nil
}

// Equals returns true if both durations have identical component values.
func (d Duration) Equals(other Duration) bool {
	return d.Months == other.Months && d.Days == other.Days && d.Nanoseconds == other.Nanoseconds
}

// HasDayPrecision returns true if the nanoseconds component is zero (no sub-day precision).
func (d Duration) HasDayPrecision() bool {
	return d.Nanoseconds == 0
}

// HasMillisecondPrecision returns true if the nanoseconds component is a multiple of 1,000,000.
func (d Duration) HasMillisecondPrecision() bool {
	return d.Nanoseconds%NSPerMS == 0
}

// IsNegative returns true if any component is negative.
func (d Duration) IsNegative() bool {
	return d.Months < 0 || d.Days < 0 || d.Nanoseconds < 0
}

// IsZero returns true if all components are zero.
func (d Duration) IsZero() bool {
	return d.Months == 0 && d.Days == 0 && d.Nanoseconds == 0
}

// Plus returns the component-wise sum of two durations.
// Returns (result, true) on success, or (zero, false) if the durations have opposite signs.
func (d Duration) Plus(other Duration) (Duration, bool) {
	if d.IsNegative() != other.IsNegative() {
		return Duration{}, false
	}
	return Duration{
		Months:      d.Months + other.Months,
		Days:        d.Days + other.Days,
		Nanoseconds: d.Nanoseconds + other.Nanoseconds,
	}, true
}

// Negate returns a new Duration with all components negated.
func (d Duration) Negate() Duration {
	return Duration{Months: -d.Months, Days: -d.Days, Nanoseconds: -d.Nanoseconds}
}

// Abs returns the duration with a positive sign, negating if needed.
func (d Duration) Abs() Duration {
	if d.IsNegative() {
		return d.Negate()
	}
	return d
}

// ToYears returns the number of whole years derived from the months component only.
func (d Duration) ToYears() int32 {
	return d.Months / 12
}

// ToHours returns the number of whole hours derived from the nanoseconds component only.
func (d Duration) ToHours() int64 {
	return d.Nanoseconds / NSPerHour
}

// ToMinutes returns the number of whole minutes derived from the nanoseconds component only.
func (d Duration) ToMinutes() int64 {
	return d.Nanoseconds / NSPerMin
}

// ToSeconds returns the number of whole seconds derived from the nanoseconds component only.
func (d Duration) ToSeconds() int64 {
	return d.Nanoseconds / NSPerSec
}

// ToMillis returns the number of whole milliseconds derived from the nanoseconds component only.
func (d Duration) ToMillis() int64 {
	return d.Nanoseconds / NSPerMS
}

// ToMicros returns the number of whole microseconds derived from the nanoseconds component only.
func (d Duration) ToMicros() int64 {
	return d.Nanoseconds / NSPerUS
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

	m := d.Months
	if m < 0 {
		m = -m
	}
	if m != 0 {
		dst = strconv.AppendInt(dst, int64(m), 10)
		dst = append(dst, "mo"...)
	}

	dy := d.Days
	if dy < 0 {
		dy = -dy
	}
	if dy != 0 {
		dst = strconv.AppendInt(dst, int64(dy), 10)
		dst = append(dst, 'd')
	}

	ns := d.Nanoseconds
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

	months := d.Months
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

	days := d.Days
	if days < 0 {
		days = -days
	}
	if days != 0 {
		sb.WriteString(strconv.FormatInt(int64(days), 10))
		sb.WriteByte('d')
	}

	nanos := d.Nanoseconds
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
		months:      abs.Months,
		days:        abs.Days,
		nanoseconds: abs.Nanoseconds,
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

// AddYears adds n years (converted to months: 1 year = 12 months).
func (b *DurationBuilder) AddYears(n int) *DurationBuilder {
	return b.addYears(int64(n))
}

// AddMonths adds n months.
func (b *DurationBuilder) AddMonths(n int) *DurationBuilder {
	return b.addMonths(int64(n))
}

// AddWeeks adds n weeks (converted to days: 1 week = 7 days).
func (b *DurationBuilder) AddWeeks(n int) *DurationBuilder {
	return b.addWeeks(int64(n))
}

// AddDays adds n days.
func (b *DurationBuilder) AddDays(n int) *DurationBuilder {
	return b.addDays(int64(n))
}

func (b *DurationBuilder) addYears(n int64) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.months) + n*12
	if n > int64(MaxMonthsDays)/12 || v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total months %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.months = int32(v)
	return b
}

func (b *DurationBuilder) addMonths(n int64) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.months) + n
	if n > int64(MaxMonthsDays) || v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total months %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.months = int32(v)
	return b
}

func (b *DurationBuilder) addWeeks(n int64) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.days) + n*7
	if n > int64(MaxMonthsDays)/7 || v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total days %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.days = int32(v)
	return b
}

func (b *DurationBuilder) addDays(n int64) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.days) + n
	if n > int64(MaxMonthsDays) || v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total days %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.days = int32(v)
	return b
}

// AddHours adds n hours (converted to nanoseconds).
func (b *DurationBuilder) AddHours(n int) *DurationBuilder {
	return b.addNanos(int64(n), NSPerHour, "hours")
}

// AddMinutes adds n minutes (converted to nanoseconds).
func (b *DurationBuilder) AddMinutes(n int) *DurationBuilder {
	return b.addNanos(int64(n), NSPerMin, "minutes")
}

// AddSeconds adds n seconds (converted to nanoseconds).
func (b *DurationBuilder) AddSeconds(n int) *DurationBuilder {
	return b.addNanos(int64(n), NSPerSec, "seconds")
}

// AddMillis adds n milliseconds (converted to nanoseconds).
func (b *DurationBuilder) AddMillis(n int) *DurationBuilder {
	return b.addNanos(int64(n), NSPerMS, "milliseconds")
}

// AddMicros adds n microseconds (converted to nanoseconds).
func (b *DurationBuilder) AddMicros(n int) *DurationBuilder {
	return b.addNanos(int64(n), NSPerUS, "microseconds")
}

// AddNanos adds n nanoseconds.
func (b *DurationBuilder) AddNanos(n int) *DurationBuilder {
	return b.addNanos(int64(n), 1, "nanoseconds")
}

func (b *DurationBuilder) addNanos(n, nsPerUnit int64, name string) *DurationBuilder {
	if b.err != nil {
		return b
	}
	if n > (MaxNanos-b.nanoseconds)/nsPerUnit {
		b.err = fmt.Errorf("invalid duration: total nanoseconds out of range [0, %d] (tried to add %d %s)", MaxNanos, n, name)
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
	if b.negative {
		return Duration{Months: -b.months, Days: -b.days, Nanoseconds: -b.nanoseconds}, nil
	}
	return Duration{Months: b.months, Days: b.days, Nanoseconds: b.nanoseconds}, nil
}
