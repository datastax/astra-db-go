package datatypes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// UUID represents a universally unique identifier with Data API JSON serialization.
// On the wire, UUIDs are serialized as extended JSON: {"$uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}.
type UUID struct {
	value [16]byte
}

// NewUUID generates a new v4 (random) UUID. Implementations SHOULD utilize
// UUIDv7 instead of UUIDv1 and UUIDv6 if possible.
func NewUUID() UUID {
	return NewUUIDv4()
}

// NewUUIDv4 generates a new v4 (random) UUID using crypto/rand. Implementations SHOULD utilize
// UUIDv7 instead of UUIDv1 and UUIDv6 if possible.
func NewUUIDv4() UUID {
	var u UUID
	if _, err := rand.Read(u.value[:]); err != nil {
		panic(fmt.Sprintf("datatypes: crypto/rand failed: %v", err))
	}
	u.value[6] = (u.value[6] & 0x0f) | 0x40 // version 4
	u.value[8] = (u.value[8] & 0x3f) | 0x80 // variant 10
	return u
}

// gregorianUnixOffset is the number of 100-nanosecond intervals between the
// UUID Gregorian epoch (1582-10-15) and the Unix epoch (1970-01-01).
// Also called MAGIC_NUMBER in ts client.
const gregorianUnixOffset = 122192928000000000

var (
	gregMu       sync.Mutex
	gregLastTime int64   // last 60-bit Gregorian timestamp
	gregClockSeq uint16  // 14-bit clock sequence
	gregNodeID   [6]byte // random node ID (multicast bit set)
	gregInited   bool
)

// randomClockSeqAndNode generates a random 14-bit clock sequence and 6-byte
// node ID (with multicast bit set). Used by the "At" UUID constructors.
func randomClockSeqAndNode() (uint16, [6]byte) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(fmt.Sprintf("datatypes: crypto/rand failed: %v", err))
	}
	clockSeq := (uint16(buf[0])<<8 | uint16(buf[1])) & 0x3FFF
	var node [6]byte
	copy(node[:], buf[2:8])
	node[0] |= 0x01 // set multicast bit
	return clockSeq, node
}

// initGreg initializes the shared Gregorian state. Must be called under gregMu.
func initGreg() {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(fmt.Sprintf("datatypes: crypto/rand failed: %v", err))
	}
	copy(gregNodeID[:], buf[:6])
	gregNodeID[0] |= 0x01 // set multicast bit
	gregClockSeq = uint16(buf[6])<<8 | uint16(buf[7])
	gregClockSeq &= 0x3FFF // mask to 14 bits
	gregInited = true
}

// getGregTime returns a monotonically increasing 60-bit Gregorian timestamp,
// a 14-bit clock sequence, and the random node ID.
func getGregTime() (timestamp int64, clockSeq uint16, node [6]byte) {
	gregMu.Lock()
	defer gregMu.Unlock()

	if !gregInited {
		initGreg()
	}

	now := time.Now().UnixNano()/100 + gregorianUnixOffset
	if now < gregLastTime {
		// Actual clock rollback — increment clock sequence per RFC 4122 §4.2.1
		gregClockSeq = (gregClockSeq + 1) & 0x3FFF
	}
	if now <= gregLastTime {
		now = gregLastTime + 1
	}
	gregLastTime = now
	return now, gregClockSeq, gregNodeID
}

// gregorianToTime converts a 60-bit Gregorian timestamp (100ns ticks since
// 1582-10-15) to a time.Time with sub-millisecond precision.
func gregorianToTime(ticks uint64) time.Time {
	unixHundredNanos := int64(ticks) - gregorianUnixOffset
	return time.Unix(unixHundredNanos/10_000_000, (unixHundredNanos%10_000_000)*100)
}

// NewUUIDv1 generates a new v1 (Gregorian time + random node) UUID.
// The timestamp bits are scattered across the first 8 bytes per RFC 4122.
func NewUUIDv1() UUID {
	ts, clockSeq, node := getGregTime()
	return buildV1(ts, clockSeq, node)
}

// NewUUIDv1At generates a v1 UUID encoding the given timestamp.
// Clock sequence and node ID are random. This does not participate in
// monotonic ordering with NewUUIDv1 calls.
func NewUUIDv1At(t time.Time) UUID {
	ts := t.UnixNano()/100 + gregorianUnixOffset
	clockSeq, node := randomClockSeqAndNode()
	return buildV1(ts, clockSeq, node)
}

func buildV1(ts int64, clockSeq uint16, node [6]byte) UUID {
	var u UUID
	// time_low: bits 0–31 (bytes 0–3)
	u.value[0] = byte(ts >> 24)
	u.value[1] = byte(ts >> 16)
	u.value[2] = byte(ts >> 8)
	u.value[3] = byte(ts)
	// time_mid: bits 32–47 (bytes 4–5)
	u.value[4] = byte(ts >> 40)
	u.value[5] = byte(ts >> 32)
	// time_hi_and_version: version(1)<<12 | bits 48–59 (bytes 6–7)
	u.value[6] = 0x10 | byte((ts>>56)&0x0F)
	u.value[7] = byte(ts >> 48)
	// clock_seq_hi_and_variant: variant(10)<<6 | upper 6 bits of clock_seq
	u.value[8] = 0x80 | byte(clockSeq>>8)&0x3F
	// clock_seq_low: lower 8 bits
	u.value[9] = byte(clockSeq)
	// node: bytes 10–15
	copy(u.value[10:], node[:])
	return u
}

// NewUUIDv6 generates a new v6 (reordered Gregorian time) UUID.
// V6 rearranges the v1 timestamp bits into big-endian order for lexicographic sorting.
func NewUUIDv6() UUID {
	ts, clockSeq, node := getGregTime()
	return buildV6(ts, clockSeq, node)
}

// NewUUIDv6At generates a v6 UUID encoding the given timestamp.
// Clock sequence and node ID are random. This does not participate in
// monotonic ordering with NewUUIDv6 calls.
func NewUUIDv6At(t time.Time) UUID {
	ts := t.UnixNano()/100 + gregorianUnixOffset
	clockSeq, node := randomClockSeqAndNode()
	return buildV6(ts, clockSeq, node)
}

func buildV6(ts int64, clockSeq uint16, node [6]byte) UUID {
	var u UUID
	// time_high: bits 28–59 (bytes 0–3)
	u.value[0] = byte(ts >> 52)
	u.value[1] = byte(ts >> 44)
	u.value[2] = byte(ts >> 36)
	u.value[3] = byte(ts >> 28)
	// time_mid: bits 12–27 (bytes 4–5)
	u.value[4] = byte(ts >> 20)
	u.value[5] = byte(ts >> 12)
	// version(6)<<12 | time_low: bits 0–11 (bytes 6–7)
	u.value[6] = 0x60 | byte((ts>>8)&0x0F)
	u.value[7] = byte(ts)
	// clock_seq_hi_and_variant: variant(10)<<6 | upper 6 bits of clock_seq
	u.value[8] = 0x80 | byte(clockSeq>>8)&0x3F
	// clock_seq_low: lower 8 bits
	u.value[9] = byte(clockSeq)
	// node: bytes 10–15
	copy(u.value[10:], node[:])
	return u
}

// timeMu guards lastV7time.
var timeMu sync.Mutex

// lastV7time is the last time we returned stored as:
//
//	52 bits of time in milliseconds since epoch
//	12 bits of (fractional nanoseconds) >> 8
var lastV7time int64

const nanoPerMilli = 1000000

// getV7Time returns the time in milliseconds and nanoseconds / 256.
// The returned (milli << 12 + seq) is guaranteed to be greater than
// (milli << 12 + seq) returned by any previous call to getV7Time.
func getV7Time() (milli, seq int64) {
	timeMu.Lock()
	defer timeMu.Unlock()

	nano := time.Now().UnixNano()
	milli = nano / nanoPerMilli
	// Sequence number is between 0 and 3906 (nanoPerMilli>>8)
	seq = (nano - milli*nanoPerMilli) >> 8
	now := milli<<12 + seq
	if now <= lastV7time {
		now = lastV7time + 1
		milli = now >> 12
		seq = now & 0xfff
	}
	lastV7time = now
	return milli, seq
}

// NewUUIDv7 generates a new v7 (timestamp + random) UUID per [rfc9562].
// The first 48 bits encode Unix milliseconds; bytes 6–7 carry the 12-bit
// sub-millisecond sequence for monotonicity. The remaining bits are random.
//
// [rfc9562]: https://datatracker.ietf.org/doc/html/rfc9562#name-uuid-version-7
func NewUUIDv7() UUID {
	/* UUID v7 layout (see [rfc9562]):
			0                   1                   2                   3
	 		0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                           unix_ts_ms                          |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|          unix_ts_ms           |  ver  |       rand_a (seq)    |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|var|                        rand_b                             |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                            rand_b                             |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/
	var u UUID
	t, seq := getV7Time()

	// 48-bit big-endian Unix millis in bytes 0–5
	u.value[0] = byte(t >> 40)
	u.value[1] = byte(t >> 32)
	u.value[2] = byte(t >> 24)
	u.value[3] = byte(t >> 16)
	u.value[4] = byte(t >> 8)
	u.value[5] = byte(t)

	// Fill remaining bytes with random data
	if _, err := rand.Read(u.value[8:]); err != nil {
		// This should be impossible. Previously, some of the syscalls were not 100%
		// reliable. Now, they are: https://github.com/golang/go/issues/66821. And
		// the crypto package itself should panic and never return errors. But
		// since this is relatively recent, let's pretend errors can happen.
		panic(fmt.Sprintf("datatypes: crypto/rand failed: %v", err))
	}
	u.value[6] = 0x70 | (0x0F & byte(seq>>8)) // version 7 + sequence high bits
	u.value[7] = byte(seq)
	u.value[8] = (u.value[8] & 0x3f) | 0x80 // variant 10
	return u
}

// NewUUIDv7At generates a v7 UUID encoding the given timestamp.
// The sub-millisecond sequence is derived from the nanosecond component of t.
// This does not participate in monotonic ordering with NewUUIDv7 calls.
func NewUUIDv7At(t time.Time) UUID {
	nano := t.UnixNano()
	milli := nano / nanoPerMilli
	seq := (nano - milli*nanoPerMilli) >> 8

	var u UUID
	// 48-bit big-endian Unix millis in bytes 0–5
	u.value[0] = byte(milli >> 40)
	u.value[1] = byte(milli >> 32)
	u.value[2] = byte(milli >> 24)
	u.value[3] = byte(milli >> 16)
	u.value[4] = byte(milli >> 8)
	u.value[5] = byte(milli)

	if _, err := rand.Read(u.value[8:]); err != nil {
		panic(fmt.Sprintf("datatypes: crypto/rand failed: %v", err))
	}
	u.value[6] = 0x70 | (0x0F & byte(seq>>8))
	u.value[7] = byte(seq)
	u.value[8] = (u.value[8] & 0x3f) | 0x80
	return u
}

// ParseUUID parses a UUID from its string representation.
// It accepts the canonical form (with dashes) and the 32-hex-digit form (without dashes).
func ParseUUID(s string) (UUID, error) {
	// Strip dashes
	clean := make([]byte, 0, 32)
	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			clean = append(clean, s[i])
		}
	}
	if len(clean) != 32 {
		return UUID{}, fmt.Errorf("datatypes: invalid UUID string: %q", s)
	}
	var u UUID
	if _, err := hex.Decode(u.value[:], clean); err != nil {
		return UUID{}, fmt.Errorf("datatypes: invalid UUID string: %q: %w", s, err)
	}
	return u, nil
}

// String returns the canonical dash-separated UUID string.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u.value[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u.value[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u.value[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u.value[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u.value[10:16])
	return string(buf[:])
}

// Version returns the UUID version number (e.g., 4 for v4, 7 for v7).
func (u UUID) Version() int {
	return int(u.value[6] >> 4)
}

// Timestamp extracts the embedded timestamp from a v1, v6, or v7 UUID.
// For unsupported versions, it returns the zero time and false.
func (u UUID) Timestamp() (time.Time, bool) {
	switch u.Version() {
	case 1:
		// v1: scattered layout — reassemble 60-bit Gregorian timestamp
		// time_low(32) from bytes 0–3, time_mid(16) from bytes 4–5, time_hi(12) from bytes 6–7
		timeLow := uint64(u.value[0])<<24 | uint64(u.value[1])<<16 | uint64(u.value[2])<<8 | uint64(u.value[3])
		timeMid := uint64(u.value[4])<<8 | uint64(u.value[5])
		timeHi := uint64(u.value[6]&0x0F)<<8 | uint64(u.value[7])
		ticks := timeHi<<48 | timeMid<<32 | timeLow
		return gregorianToTime(ticks), true

	case 6:
		// v6: big-endian ordered — reassemble 60-bit Gregorian timestamp
		// time_high(32) from bytes 0–3, time_mid(16) from bytes 4–5, time_low(12) from bytes 6–7
		timeHigh := uint64(u.value[0])<<24 | uint64(u.value[1])<<16 | uint64(u.value[2])<<8 | uint64(u.value[3])
		timeMid := uint64(u.value[4])<<8 | uint64(u.value[5])
		timeLow := uint64(u.value[6]&0x0F)<<8 | uint64(u.value[7])
		ticks := timeHigh<<28 | timeMid<<12 | timeLow
		return gregorianToTime(ticks), true

	case 7:
		// v7: 48-bit Unix millis in bytes 0–5
		ms := uint64(u.value[0])<<40 |
			uint64(u.value[1])<<32 |
			uint64(u.value[2])<<24 |
			uint64(u.value[3])<<16 |
			uint64(u.value[4])<<8 |
			uint64(u.value[5])
		return time.UnixMilli(int64(ms)), true

	default:
		return time.Time{}, false
	}
}

// IsZero returns true if the UUID is the zero value (all bytes are 0).
func (u UUID) IsZero() bool {
	return u.value == [16]byte{}
}

// Equals returns true if the two UUIDs have identical bytes.
func (u UUID) Equals(other UUID) bool {
	return u.value == other.value
}

// Bytes returns the raw 16-byte UUID value.
func (u UUID) Bytes() [16]byte {
	return u.value
}

// MarshalJSON produces the Data API extended JSON format: {"$uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}.
func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"$uuid": u.String()})
}

// UnmarshalJSON parses the Data API extended JSON format: {"$uuid": "..."}.
func (u *UUID) UnmarshalJSON(data []byte) error {
	var wrapper map[string]string
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("datatypes: invalid UUID JSON: %w", err)
	}
	s, ok := wrapper["$uuid"]
	if !ok {
		return fmt.Errorf("datatypes: missing $uuid key in JSON")
	}
	parsed, err := ParseUUID(s)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler. It returns the canonical
// dash-separated UUID string. This is used by table serialization (as opposed
// to MarshalJSON which produces the collection {"$uuid": "..."} format).
func (u UUID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. It parses a canonical
// UUID string (with or without dashes).
func (u *UUID) UnmarshalText(data []byte) error {
	parsed, err := ParseUUID(string(data))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}
