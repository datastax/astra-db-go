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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/datastax/astra-db-go/astra/internal/utils"
)

// TimeOnly represents a time (hour, minute, second, nanosecond) without a date component,
// used for Data API 'time' columns in tables.
type TimeOnly struct {
	Hour       int
	Minute     int
	Second     int
	Nanosecond int
}

// CompareTo implements the Comparable interface.
func (t TimeOnly) CompareTo(other any) int {
	o := other.(TimeOnly)
	if c := cmp.Compare(t.Hour, o.Hour); c != 0 {
		return c
	}
	if c := cmp.Compare(t.Minute, o.Minute); c != 0 {
		return c
	}
	if c := cmp.Compare(t.Second, o.Second); c != 0 {
		return c
	}
	return cmp.Compare(t.Nanosecond, o.Nanosecond)
}

// NewTimeOnly creates a new TimeOnly.
func NewTimeOnly(hour, minute, second, nanosecond int) TimeOnly {
	return TimeOnly{Hour: hour, Minute: minute, Second: second, Nanosecond: nanosecond}
}

// TimeOnlyFromTime creates a TimeOnly from a time.Time (local time).
func TimeOnlyFromTime(t time.Time) TimeOnly {
	return TimeOnly{Hour: t.Hour(), Minute: t.Minute(), Second: t.Second(), Nanosecond: t.Nanosecond()}
}

// TimeOnlyNow creates a TimeOnly for the current local time.
func TimeOnlyNow() TimeOnly {
	return TimeOnlyFromTime(time.Now())
}

// TimeOnlyUTCNow creates a TimeOnly for the current UTC time.
func TimeOnlyUTCNow() TimeOnly {
	return TimeOnlyFromTime(time.Now().UTC())
}

// String returns the time in HH:MM:SS.NNNNNNNNN format.
func (t TimeOnly) String() string {
	return fmt.Sprintf("%02d:%02d:%02d.%09d", t.Hour, t.Minute, t.Second, t.Nanosecond)
}

// ParseTimeOnly parses a time string in HH:MM[:SS[.NNNNNNNNN]] format.
func ParseTimeOnly(s string) (TimeOnly, error) {
	if len(s) == 0 {
		return TimeOnly{}, fmt.Errorf("datatypes: empty time string")
	}

	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return TimeOnly{}, fmt.Errorf("datatypes: invalid time string format (expected HH:MM[:SS]): %q", s)
	}

	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		return TimeOnly{}, fmt.Errorf("datatypes: invalid time string format (HH:MM part length mismatch): %q", s)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return TimeOnly{}, fmt.Errorf("datatypes: invalid time string (hour out of range): %q", s)
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return TimeOnly{}, fmt.Errorf("datatypes: invalid time string (minute out of range): %q", s)
	}

	second := 0
	nanosecond := 0

	if len(parts) == 3 {
		secParts := strings.Split(parts[2], ".")
		if len(secParts[0]) != 2 {
			return TimeOnly{}, fmt.Errorf("datatypes: invalid time string (second part length mismatch): %q", s)
		}

		second, err = strconv.Atoi(secParts[0])
		if err != nil || second < 0 || second > 59 {
			return TimeOnly{}, fmt.Errorf("datatypes: invalid time string (second out of range): %q", s)
		}

		if len(secParts) == 2 {
			nanoStr := secParts[1]
			if nanoStr == "" {
				return TimeOnly{}, fmt.Errorf("datatypes: invalid time string (empty fractional seconds): %q", s)
			}
			if len(nanoStr) > 9 {
				nanoStr = nanoStr[:9]
			}
			nanosecond, err = strconv.Atoi(nanoStr)
			if err != nil || nanosecond < 0 {
				return TimeOnly{}, fmt.Errorf("datatypes: invalid time string (invalid fractional seconds): %q", s)
			}
			nanosecond *= int(math.Pow10(9 - len(nanoStr)))
		}
	}

	return TimeOnly{Hour: hour, Minute: minute, Second: second, Nanosecond: nanosecond}, nil
}

// MustParseTimeOnly is like ParseTimeOnly but panics on error.
func MustParseTimeOnly(s string) TimeOnly {
	return utils.Must(ParseTimeOnly(s))
}
