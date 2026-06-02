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

// DataAPIDuration represents a Cassandra duration type stored as months, days, and nanoseconds.
// These three components are not directly convertible to each other (months are calendar months,
// days are calendar days, and nanoseconds are a fixed-length time unit).
// All three components must share the same sign.
type DataAPIDuration struct {
	Months      int32
	Days        int32
	Nanoseconds int64
}

// NewDataAPIDuration creates a DataAPIDuration from months, days, and nanoseconds.
// All components must share the same sign and be within valid ranges:
// months and days in [-2147483647, 2147483647], nanoseconds in [-9223372036854775807, 9223372036854775807].
func NewDataAPIDuration(months int32, days int32, nanoseconds int64) (DataAPIDuration, error) {
	if err := validateDuration(months, days, nanoseconds); err != nil {
		return DataAPIDuration{}, err
	}
	return DataAPIDuration{Months: months, Days: days, Nanoseconds: nanoseconds}, nil
}

// MustParseDuration parses a duration string and panics on error.
// Intended for use in tests and with known-valid inputs.
func MustParseDuration(s string) DataAPIDuration {
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
func ParseDuration(s string) (DataAPIDuration, error) {
	if s == "" {
		return DataAPIDuration{}, fmt.Errorf("invalid duration: empty string")
	}

	negative := s[0] == '-'
	rest := s
	if negative {
		rest = s[1:]
	}

	if rest == "" {
		return DataAPIDuration{}, fmt.Errorf("invalid duration: empty after sign")
	}

	var d DataAPIDuration
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
		return DataAPIDuration{}, err
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

func parseBasicDuration(s, original string) (DataAPIDuration, error) {
	lower := strings.ToLower(s)
	var months int32
	var days int32
	var nanos int64
	lastIdx := -1
	hasAny := false

	for i := 0; i < len(lower); {
		m := basicDurationPartRe.FindStringSubmatch(lower[i:])
		if m == nil {
			return DataAPIDuration{}, fmt.Errorf("invalid standard duration string: %q", original)
		}

		num, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return DataAPIDuration{}, fmt.Errorf("invalid duration number in %q: %w", original, err)
		}

		unit := m[2]
		idx := basicUnitOrderIndex[unit]

		if idx <= lastIdx {
			return DataAPIDuration{}, fmt.Errorf("invalid standard duration string %q: units must appear in order and at most once", original)
		}
		lastIdx = idx

		switch unit {
		case "y":
			if num > int64(MaxMonthsDays)/12 {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: months component out of range", original)
			}
			months += int32(num * 12)
		case "mo":
			v := int64(months) + num
			if v > int64(MaxMonthsDays) {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: months component out of range", original)
			}
			months = int32(v)
		case "w":
			if num > int64(MaxMonthsDays)/7 {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: days component out of range", original)
			}
			days += int32(num * 7)
		case "d":
			v := int64(days) + num
			if v > int64(MaxMonthsDays) {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: days component out of range", original)
			}
			days = int32(v)
		case "h":
			if num > (MaxNanos-nanos)/NSPerHour {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: nanoseconds component out of range", original)
			}
			nanos += num * NSPerHour
		case "m":
			if num > (MaxNanos-nanos)/NSPerMin {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: nanoseconds component out of range", original)
			}
			nanos += num * NSPerMin
		case "s":
			if num > (MaxNanos-nanos)/NSPerSec {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: nanoseconds component out of range", original)
			}
			nanos += num * NSPerSec
		case "ms":
			if num > (MaxNanos-nanos)/NSPerMS {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: nanoseconds component out of range", original)
			}
			nanos += num * NSPerMS
		case "us", "µs":
			if num > (MaxNanos-nanos)/NSPerUS {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: nanoseconds component out of range", original)
			}
			nanos += num * NSPerUS
		case "ns":
			if num > MaxNanos-nanos {
				return DataAPIDuration{}, fmt.Errorf("invalid duration %q: nanoseconds component out of range", original)
			}
			nanos += num
		}

		hasAny = true
		i += len(m[0])
	}

	if !hasAny {
		return DataAPIDuration{}, fmt.Errorf("invalid standard duration string: %q", original)
	}

	return DataAPIDuration{Months: months, Days: days, Nanoseconds: nanos}, nil
}

// isoStandardDurationRe matches ISO 8601 standard duration: P[nY][nM][nD][T[nH][nM][n[.f]S]]
var isoStandardDurationRe = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)(?:\.(\d+))?S)?)?$`)

func parseISOStandardDuration(s string) (DataAPIDuration, error) {
	m := isoStandardDurationRe.FindStringSubmatch(s)
	if m == nil {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 standard duration string: %q", s)
	}

	var months int32
	var days int32
	var nanos int64

	if m[1] != "" {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		if n > int64(MaxMonthsDays)/12 {
			return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 duration %q: months out of range", s)
		}
		months += int32(n * 12)
	}
	if m[2] != "" {
		n, _ := strconv.ParseInt(m[2], 10, 64)
		v := int64(months) + n
		if v > int64(MaxMonthsDays) {
			return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 duration %q: months out of range", s)
		}
		months = int32(v)
	}
	if m[3] != "" {
		n, _ := strconv.ParseInt(m[3], 10, 64)
		v := int64(days) + n
		if v > int64(MaxMonthsDays) {
			return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 duration %q: days out of range", s)
		}
		days = int32(v)
	}
	if m[4] != "" {
		n, _ := strconv.ParseInt(m[4], 10, 64)
		if n > (MaxNanos-nanos)/NSPerHour {
			return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 duration %q: nanoseconds out of range", s)
		}
		nanos += n * NSPerHour
	}
	if m[5] != "" {
		n, _ := strconv.ParseInt(m[5], 10, 64)
		if n > (MaxNanos-nanos)/NSPerMin {
			return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 duration %q: nanoseconds out of range", s)
		}
		nanos += n * NSPerMin
	}
	if m[6] != "" {
		n, _ := strconv.ParseInt(m[6], 10, 64)
		if n > (MaxNanos-nanos)/NSPerSec {
			return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 duration %q: nanoseconds out of range", s)
		}
		nanos += n * NSPerSec
	}
	if m[7] != "" {
		frac, _ := strconv.ParseInt(m[7], 10, 64)
		nanos += int64(float64(frac) * math.Pow10(9-len(m[7])))
	}

	return DataAPIDuration{Months: months, Days: days, Nanoseconds: nanos}, nil
}

// isoWeekDurationRe matches ISO 8601 week duration: PnW
var isoWeekDurationRe = regexp.MustCompile(`^P(\d+)W$`)

func parseISOWeekDuration(s string) (DataAPIDuration, error) {
	m := isoWeekDurationRe.FindStringSubmatch(s)
	if m == nil {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 week duration string: %q", s)
	}

	weeks, _ := strconv.ParseInt(m[1], 10, 64)
	if weeks > int64(MaxMonthsDays)/7 {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 week duration %q: days out of range", s)
	}

	return DataAPIDuration{Days: int32(weeks * 7)}, nil
}

// isoAlternateDurationRe matches ISO 8601 alternate duration: PYYYY-MM-DDThh:mm:ss
var isoAlternateDurationRe = regexp.MustCompile(`^P(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})$`)

func parseISOAlternateDuration(s string) (DataAPIDuration, error) {
	m := isoAlternateDurationRe.FindStringSubmatch(s)
	if m == nil {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 alternate duration string: %q", s)
	}

	years, _ := strconv.ParseInt(m[1], 10, 64)
	mons, _ := strconv.ParseInt(m[2], 10, 64)
	days, _ := strconv.ParseInt(m[3], 10, 64)
	hours, _ := strconv.ParseInt(m[4], 10, 64)
	mins, _ := strconv.ParseInt(m[5], 10, 64)
	secs, _ := strconv.ParseInt(m[6], 10, 64)

	totalMonths := years*12 + mons
	if years > int64(MaxMonthsDays)/12 || totalMonths > int64(MaxMonthsDays) {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 alternate duration %q: months out of range", s)
	}
	if days > int64(MaxMonthsDays) {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 alternate duration %q: days out of range", s)
	}
	totalNanos := hours*NSPerHour + mins*NSPerMin + secs*NSPerSec
	if totalNanos > MaxNanos {
		return DataAPIDuration{}, fmt.Errorf("invalid ISO 8601 alternate duration %q: nanoseconds out of range", s)
	}

	return DataAPIDuration{
		Months:      int32(totalMonths),
		Days:        int32(days),
		Nanoseconds: totalNanos,
	}, nil
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
func (d DataAPIDuration) Equals(other DataAPIDuration) bool {
	return d.Months == other.Months && d.Days == other.Days && d.Nanoseconds == other.Nanoseconds
}

// HasDayPrecision returns true if the nanoseconds component is zero (no sub-day precision).
func (d DataAPIDuration) HasDayPrecision() bool {
	return d.Nanoseconds == 0
}

// HasMillisecondPrecision returns true if the nanoseconds component is a multiple of 1,000,000.
func (d DataAPIDuration) HasMillisecondPrecision() bool {
	return d.Nanoseconds%NSPerMS == 0
}

// IsNegative returns true if any component is negative.
func (d DataAPIDuration) IsNegative() bool {
	return d.Months < 0 || d.Days < 0 || d.Nanoseconds < 0
}

// IsZero returns true if all components are zero.
func (d DataAPIDuration) IsZero() bool {
	return d.Months == 0 && d.Days == 0 && d.Nanoseconds == 0
}

// Plus returns the component-wise sum of two durations.
// Returns (result, true) on success, or (zero, false) if the durations have opposite signs.
func (d DataAPIDuration) Plus(other DataAPIDuration) (DataAPIDuration, bool) {
	if d.IsNegative() != other.IsNegative() {
		return DataAPIDuration{}, false
	}
	return DataAPIDuration{
		Months:      d.Months + other.Months,
		Days:        d.Days + other.Days,
		Nanoseconds: d.Nanoseconds + other.Nanoseconds,
	}, true
}

// Negate returns a new DataAPIDuration with all components negated.
func (d DataAPIDuration) Negate() DataAPIDuration {
	return DataAPIDuration{Months: -d.Months, Days: -d.Days, Nanoseconds: -d.Nanoseconds}
}

// Abs returns the duration with a positive sign, negating if needed.
func (d DataAPIDuration) Abs() DataAPIDuration {
	if d.IsNegative() {
		return d.Negate()
	}
	return d
}

// ToYears returns the number of whole years derived from the months component only.
func (d DataAPIDuration) ToYears() int32 {
	return d.Months / 12
}

// ToHours returns the number of whole hours derived from the nanoseconds component only.
func (d DataAPIDuration) ToHours() int64 {
	return d.Nanoseconds / NSPerHour
}

// ToMinutes returns the number of whole minutes derived from the nanoseconds component only.
func (d DataAPIDuration) ToMinutes() int64 {
	return d.Nanoseconds / NSPerMin
}

// ToSeconds returns the number of whole seconds derived from the nanoseconds component only.
func (d DataAPIDuration) ToSeconds() int64 {
	return d.Nanoseconds / NSPerSec
}

// ToMillis returns the number of whole milliseconds derived from the nanoseconds component only.
func (d DataAPIDuration) ToMillis() int64 {
	return d.Nanoseconds / NSPerMS
}

// ToMicros returns the number of whole microseconds derived from the nanoseconds component only.
func (d DataAPIDuration) ToMicros() int64 {
	return d.Nanoseconds / NSPerUS
}

// String returns the human-readable long-form representation (e.g. "1y3mo25d5h6m7s8ms9us10ns").
func (d DataAPIDuration) String() string {
	return durationToLongString(d)
}

// AppendShortString appends the compact API wire format to dst.
// Months, days, and nanoseconds are each expressed as a raw count with a single unit suffix.
// Example: DataAPIDuration{14, 3, 3_600_000_000_000} → "14mo3d3600000000000ns"
func (d DataAPIDuration) AppendShortString(dst []byte) []byte {
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

func durationToLongString(d DataAPIDuration) string {
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
func (d DataAPIDuration) MarshalJSON() ([]byte, error) {
	s := string(d.AppendShortString(nil))
	return json.Marshal(s)
}

// UnmarshalJSON implements json.Unmarshaler, parsing the duration from a string.
func (d *DataAPIDuration) UnmarshalJSON(data []byte) error {
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

// DurationBuilder builds a DataAPIDuration incrementally.
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

// NewDurationBuilderFrom returns a DurationBuilder initialized from an existing DataAPIDuration.
func NewDurationBuilderFrom(base DataAPIDuration) *DurationBuilder {
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
func (b *DurationBuilder) AddYears(n int32) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.months) + int64(n)*12
	if v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total months %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.months = int32(v)
	return b
}

// AddMonths adds n months.
func (b *DurationBuilder) AddMonths(n int32) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.months) + int64(n)
	if v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total months %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.months = int32(v)
	return b
}

// AddWeeks adds n weeks (converted to days: 1 week = 7 days).
func (b *DurationBuilder) AddWeeks(n int32) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.days) + int64(n)*7
	if v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total days %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.days = int32(v)
	return b
}

// AddDays adds n days.
func (b *DurationBuilder) AddDays(n int32) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := int64(b.days) + int64(n)
	if v > int64(MaxMonthsDays) || v < 0 {
		b.err = fmt.Errorf("invalid duration: total days %d out of range [0, %d]", v, MaxMonthsDays)
		return b
	}
	b.days = int32(v)
	return b
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

func (b *DurationBuilder) addNanos(n, nsPerUnit int64, name string) *DurationBuilder {
	if b.err != nil {
		return b
	}
	v := b.nanoseconds + n*nsPerUnit
	if v > MaxNanos || v < 0 {
		b.err = fmt.Errorf("invalid duration: total nanoseconds %d out of range [0, %d] (tried to add %d %s)", v, MaxNanos, n, name)
		return b
	}
	b.nanoseconds = v
	return b
}

// Build returns the constructed DataAPIDuration, or an error if any Add* call failed.
func (b *DurationBuilder) Build() (DataAPIDuration, error) {
	if b.err != nil {
		return DataAPIDuration{}, b.err
	}
	if b.negative {
		return DataAPIDuration{Months: -b.months, Days: -b.days, Nanoseconds: -b.nanoseconds}, nil
	}
	return DataAPIDuration{Months: b.months, Days: b.days, Nanoseconds: b.nanoseconds}, nil
}
