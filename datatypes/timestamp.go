package datatypes

import (
	"encoding/json"
	"fmt"
	"time"
)

// DataAPITimestamp represents a timestamp with Data API JSON serialization.
// On the wire, timestamps are serialized as extended JSON: {"$date": 1690045891000}.
// The value is Unix epoch milliseconds, matching the TypeScript client's Date.valueOf().
type DataAPITimestamp struct {
	value time.Time
}

// NewDataAPITimestamp creates a DataAPITimestamp from a time.Time.
func NewDataAPITimestamp(t time.Time) DataAPITimestamp {
	return DataAPITimestamp{value: t}
}

// DataAPITimestampNow creates a DataAPITimestamp for the current time.
func DataAPITimestampNow() DataAPITimestamp {
	return DataAPITimestamp{value: time.Now()}
}

// DataAPITimestampFromMillis creates a DataAPITimestamp from Unix epoch milliseconds.
func DataAPITimestampFromMillis(ms int64) DataAPITimestamp {
	return DataAPITimestamp{value: time.UnixMilli(ms)}
}

// Time returns the underlying time.Time value.
func (t DataAPITimestamp) Time() time.Time {
	return t.value
}

// UnixMillis returns the timestamp as Unix epoch milliseconds.
func (t DataAPITimestamp) UnixMillis() int64 {
	return t.value.UnixMilli()
}

// IsZero returns true if the timestamp is the zero value.
func (t DataAPITimestamp) IsZero() bool {
	return t.value.IsZero()
}

// Equals returns true if the two timestamps represent the same millisecond.
func (t DataAPITimestamp) Equals(other DataAPITimestamp) bool {
	return t.value.UnixMilli() == other.value.UnixMilli()
}

// String returns the timestamp in RFC 3339 format for display.
func (t DataAPITimestamp) String() string {
	return t.value.UTC().Format(time.RFC3339Nano)
}

// MarshalJSON produces the Data API extended JSON format: {"$date": 1690045891000}.
func (t DataAPITimestamp) MarshalJSON() ([]byte, error) {
	if t.value.IsZero() {
		return []byte("null"), nil
	}
	// Slight optimization over json.Marshal(map[string]int64{"$date": t.value.UnixMilli()}).
	// In practice, this probably matters little.
	return fmt.Appendf(nil, `{"$date":%d}`, t.value.UnixMilli()), nil
}

// UnmarshalJSON parses the Data API extended JSON format: {"$date": <milliseconds>}.
func (t *DataAPITimestamp) UnmarshalJSON(data []byte) error {
	var wrapper map[string]json.Number
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("datatypes: invalid $date JSON: %w", err)
	}
	v, ok := wrapper["$date"]
	if !ok {
		return fmt.Errorf("datatypes: missing $date key in JSON")
	}
	ms, err := v.Int64()
	if err != nil {
		return fmt.Errorf("datatypes: invalid $date value: %w", err)
	}
	t.value = time.UnixMilli(ms)
	return nil
}
