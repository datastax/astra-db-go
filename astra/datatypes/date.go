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
	"cmp"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/datastax/astra-db-go/v2/internal/utils"
)

// DateOnly represents a date (year, month, day) without a time component,
// used for Data API 'date' columns in tables.
type DateOnly struct {
	Year  int
	Month int
	Day   int
}

// CompareTo implements the Comparable interface.
func (d DateOnly) CompareTo(other any) int {
	o := other.(DateOnly)
	if c := cmp.Compare(d.Year, o.Year); c != 0 {
		return c
	}
	if c := cmp.Compare(d.Month, o.Month); c != 0 {
		return c
	}
	return cmp.Compare(d.Day, o.Day)
}

// NewDateOnly creates a new DateOnly.
func NewDateOnly(year, month, day int) DateOnly {
	return DateOnly{Year: year, Month: month, Day: day}
}

// DateOnlyFromTime creates a DateOnly from a time.Time (local time).
func DateOnlyFromTime(t time.Time) DateOnly {
	return DateOnly{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

// DateOnlyNow creates a DateOnly for the current local date.
func DateOnlyNow() DateOnly {
	return DateOnlyFromTime(time.Now())
}

// DateOnlyUTCNow creates a DateOnly for the current UTC date.
func DateOnlyUTCNow() DateOnly {
	return DateOnlyFromTime(time.Now().UTC())
}

// ToTime converts the DateOnly to a time.Time at midnight UTC.
func (d DateOnly) ToTime() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

// String returns the date in YYYY-MM-DD format.
// Years < 0 or >= 10000 are prefixed with a sign.
func (d DateOnly) String() string {
	sign := ""
	year := d.Year
	if year < 0 {
		sign = "-"
		year = -year
	} else if year >= 10000 {
		sign = "+"
	}
	return fmt.Sprintf("%s%04d-%02d-%02d", sign, year, d.Month, d.Day)
}

// ParseDateOnly parses a date string in [+-]?YYYY-MM-DD format.
func ParseDateOnly(s string) (DateOnly, error) {
	if len(s) == 0 {
		return DateOnly{}, fmt.Errorf("datatypes: empty date string")
	}

	var sign = 1
	var yearStr string
	var remaining string

	if s[0] == '-' {
		sign = -1
		remaining = s[1:]
	} else if s[0] == '+' {
		remaining = s[1:]
	} else {
		remaining = s
	}

	idx1 := strings.IndexByte(remaining, '-')
	if idx1 == -1 {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (missing year separator): %q", s)
	}
	if idx1 < 4 { // At least 4 digits for year
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (year part too short): %q", s)
	}
	yearStr = remaining[:idx1]
	remaining = remaining[idx1+1:]

	if len(remaining) != 5 || remaining[2] != '-' {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (invalid month/day format): %q", s)
	}

	if len(s) < 10 {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (too short): %q", s)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (invalid year): %q", s)
	}

	month, err := strconv.Atoi(remaining[:2])
	if err != nil {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (invalid month): %q", s)
	}

	day, err := strconv.Atoi(remaining[3:])
	if err != nil {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (invalid day): %q", s)
	}

	y := sign * year
	if month < 1 || month > 12 {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (month out of range): %q", s)
	}

	daysInMonth := 31
	switch month {
	case 4, 6, 9, 11:
		daysInMonth = 30
	case 2:
		if (y%4 == 0 && y%100 != 0) || (y%400 == 0) {
			daysInMonth = 29
		} else {
			daysInMonth = 28
		}
	}

	if day < 1 || day > daysInMonth {
		return DateOnly{}, fmt.Errorf("datatypes: invalid date string (day out of range): %q", s)
	}

	return DateOnly{Year: y, Month: month, Day: day}, nil
}

// MustParseDateOnly is like ParseDateOnly but panics on error.
func MustParseDateOnly(s string) DateOnly {
	return utils.Must(ParseDateOnly(s))
}
