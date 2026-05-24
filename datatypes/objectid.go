package datatypes

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ObjectId represents an ObjectId that can be used as an _id in the DataAPI.
// json.Marshals to {"$objectId": "6672e1cbd7fabb4e5493916f"}.
type ObjectId struct {
	// An ObjectId is 12 bytes: 4 bytes timestamp (seconds) + 5 bytes random + 3 bytes counter.
	value [12]byte
}

// CompareTo implements the Comparable interface.
func (o ObjectId) CompareTo(other any) int {
	otherOid := other.(ObjectId)
	return bytes.Compare(o.value[:], otherOid.value[:])
}

var (
	oidOnce    sync.Once
	oidRandID  [5]byte // 3 bytes machine ID + 2 bytes PID-like
	oidCounter uint32
)

func initOid() {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// See note in DataAPIVector but this PROBABLY can never happen. But just in case...
		panic(fmt.Sprintf("datatypes: crypto/rand failed: %v", err))
	}
	copy(oidRandID[:], buf[:5])
	// Initialize counter to a random 24-bit value.
	oidCounter = uint32(buf[5])<<16 | uint32(buf[6])<<8 | uint32(buf[7])
}

// NewObjectId generates a new ObjectId based on the current time.
func NewObjectId() ObjectId {
	return newObjectId(time.Now())
}

// NewObjectIdAt generates a new ObjectId encoding the given timestamp.
// The random and counter components are still generated normally.
func NewObjectIdAt(t time.Time) ObjectId {
	return newObjectId(t)
}

func newObjectId(t time.Time) ObjectId {
	oidOnce.Do(initOid)

	var o ObjectId
	// Bytes 0–3: Unix timestamp in seconds (big-endian).
	ts := uint32(t.Unix())
	o.value[0] = byte(ts >> 24)
	o.value[1] = byte(ts >> 16)
	o.value[2] = byte(ts >> 8)
	o.value[3] = byte(ts)
	// Bytes 4–8: random machine ID + PID-like value (set once per process).
	copy(o.value[4:9], oidRandID[:])
	// Bytes 9–11: auto-incrementing 24-bit counter.
	c := atomic.AddUint32(&oidCounter, 1) % 0x1000000
	o.value[9] = byte(c >> 16)
	o.value[10] = byte(c >> 8)
	o.value[11] = byte(c)

	return o
}

// ParseObjectId parses an ObjectId from its 24-character hex string representation.
func ParseObjectId(s string) (ObjectId, error) {
	if len(s) != 24 {
		return ObjectId{}, fmt.Errorf("datatypes: invalid ObjectId string: %q (expected 24 hex characters)", s)
	}
	var o ObjectId
	if _, err := hex.Decode(o.value[:], []byte(s)); err != nil {
		return ObjectId{}, fmt.Errorf("datatypes: invalid ObjectId string: %q: %w", s, err)
	}
	return o, nil
}

func MustParseObjectId(s string) ObjectId {
	o, err := ParseObjectId(s)
	if err != nil {
		panic(fmt.Sprintf("datatypes: invalid ObjectId string: %q: %v", s, err))
	}
	return o
}

// String returns the 24-character lowercase hex representation of the ObjectId.
func (o ObjectId) String() string {
	return hex.EncodeToString(o.value[:])
}

// GetTimestamp extracts the creation timestamp from the ObjectId.
// The first 4 bytes encode Unix seconds.
func (o ObjectId) GetTimestamp() time.Time {
	secs := int64(o.value[0])<<24 | int64(o.value[1])<<16 | int64(o.value[2])<<8 | int64(o.value[3])
	return time.Unix(secs, 0)
}

// IsZero returns true if the ObjectId is the zero value (all bytes are 0).
func (o ObjectId) IsZero() bool {
	return o.value == [12]byte{}
}

// Equals returns true if the two ObjectIds have identical bytes.
func (o ObjectId) Equals(other ObjectId) bool {
	return o.value == other.value
}

// Bytes returns the raw 12-byte ObjectId value.
func (o ObjectId) Bytes() [12]byte {
	return o.value
}

// MarshalJSON produces the Data API extended JSON format: {"$objectId": "6672e1cbd7fabb4e5493916f"}.
func (o ObjectId) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"$objectId": o.String()})
}

// UnmarshalJSON parses the Data API extended JSON format: {"$objectId": "..."}.
func (o *ObjectId) UnmarshalJSON(data []byte) error {
	var wrapper map[string]string
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("datatypes: invalid ObjectId JSON: %w", err)
	}
	s, ok := wrapper["$objectId"]
	if !ok {
		return fmt.Errorf("datatypes: missing $objectId key in JSON")
	}
	parsed, err := ParseObjectId(s)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler. It returns the 24-character
// lowercase hex string.
func (o ObjectId) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. It parses a 24-character hex string.
func (o *ObjectId) UnmarshalText(data []byte) error {
	parsed, err := ParseObjectId(string(data))
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}
