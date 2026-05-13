package serdes

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
	"strconv"
)

func consumeNull(src []byte) ([]byte, bool) {
	src = skipWS(src)
	if len(src) >= 4 && src[0] == 'n' && src[1] == 'u' && src[2] == 'l' && src[3] == 'l' {
		return src[4:], true
	}
	return src, false
}

func parseInt(src []byte) ([]byte, int64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] == '-' || (src[end] >= '0' && src[end] <= '9')) {
		end++
	}

	if end == 0 {
		return src, 0, fmt.Errorf("expected integer")
	}

	num, err := strconv.ParseInt(unsafeString(src[:end]), 10, 64)
	return src[end:], num, err
}

func parseUint(src []byte) ([]byte, uint64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] >= '0' && src[end] <= '9') {
		end++
	}

	if end == 0 {
		return src, 0, fmt.Errorf("expected unsigned integer")
	}

	num, err := strconv.ParseUint(unsafeString(src[:end]), 10, 64)
	return src[end:], num, err
}

var floatChars = [256]uint8{
	'0': 1, '1': 1, '2': 1, '3': 1, '4': 1,
	'5': 1, '6': 1, '7': 1, '8': 1, '9': 1,
	'.': 1, '-': 1, '+': 1, 'e': 1, 'E': 1,
}

func parseFloat(src []byte) ([]byte, float64, error) {
	src, numStr, err := parseNumber(src)
	if err != nil {
		return src, 0, fmt.Errorf("invalid float: %w", err)
	}

	f, err := strconv.ParseFloat(unsafeString(numStr), 64)
	return src, f, err
}

// parseNumber extracts a number string without parsing it, for use with big.Int and big.Float
func parseNumber(src []byte) ([]byte, []byte, error) {
	src = skipWS(src)
	end := 0

	for end < len(src) && floatChars[src[end]] != 0 {
		end++
	}

	if end == 0 {
		return src, nil, fmt.Errorf("expected number")
	}

	return src[end:], src[:end], nil
}

// parseString is vendored and modified from:
// https://github.com/segmentio/encoding/blob/fd406855de30c54110d23eace25478ab9c6fa2cc/json/parse.go#L405
func parseString(src []byte) ([]byte, []byte, bool, error) {
	src = skipWS(src)

	if len(src) < 2 {
		return src[len(src):], nil, false, fmt.Errorf("expected string")
	}
	if src[0] != '"' {
		return src, nil, false, fmt.Errorf("expected '\"' at the beginning of a string value")
	}

	var n int
	if len(src) >= 9 {
		const mask1 = 0x2222222222222222
		const mask2 = 0x0101010101010101
		const mask3 = 0x8080808080808080
		u := binary.LittleEndian.Uint64(src[1:]) ^ mask1
		if mask := (u - mask2) & ^u & mask3; mask != 0 {
			n = bits.TrailingZeros64(mask)/8 + 2
			goto found
		}
		if len(src) >= 17 {
			u = binary.LittleEndian.Uint64(src[9:]) ^ mask1
			if mask := (u - mask2) & ^u & mask3; mask != 0 {
				n = bits.TrailingZeros64(mask)/8 + 10
				goto found
			}
		}
	}
	n = bytes.IndexByte(src[1:], '"') + 2
	if n <= 1 {
		return src, nil, false, fmt.Errorf("missing '\"' at the end of a string value")
	}

found:
	if bytes.IndexByte(src[1:n], '\\') < 0 {
		return src[n:], src[:n], true, nil
	}
	return src[n:], src[:n], false, nil
}

func parseStringUnquote(src []byte) ([]byte, []byte, bool, error) {
	src, s, escaped, err := parseString(src)
	if err != nil {
		return src, s, false, err
	}

	if escaped {
		return src, s[1 : len(s)-1], false, nil
	}

	res, err := strconv.Unquote(unsafeString(s))
	if err != nil {
		return src, nil, true, err
	}

	return src, []byte(res), true, nil
}

func skipWS(src []byte) []byte {
	for i := range src {
		if src[i] > ' ' {
			return src[i:]
		}
	}
	return nil
}

func skipWSRev(src []byte) []byte {
	for i := len(src) - 1; i >= 0; i-- {
		if src[i] > ' ' {
			return src[:i+1]
		}
	}
	return nil
}

func skipValue(src []byte) ([]byte, error) {
	src = skipWS(src)
	if len(src) == 0 {
		return src, fmt.Errorf("unexpected end of input")
	}

	switch src[0] {
	case '"':
		src, _, _, err := parseString(src)
		if err != nil {
			return src, err
		}
		return src, nil

	case '{', '[':
		depth, open, cls := 0, src[0], byte('}')
		if open == '[' {
			cls = ']'
		}

		for i := 0; i < len(src); i++ {
			if src[i] == open {
				depth++
			} else if src[i] == cls {
				depth--
				if depth == 0 {
					return src[i+1:], nil
				}
			}
		}
		return src, fmt.Errorf("unexpected end of input while skipping value")

	default:
		i := 0
		for i < len(src) && src[i] != ',' && src[i] != '}' && src[i] != ']' {
			i++
		}
		return src[i:], nil
	}
}

// small lut for hex conversion
const hexTable = "0123456789abcdef"

// appendString efficiently encodes a string to JSON, but does not handle 100% of edge cases
// It's designed to be fast and simple for all common and even most uncommon cases, but it'll
// rely on the server to cover the last 0.01% of edge cases that are extremely rare in practice
func appendString(dst []byte, s string) []byte {
	dst = append(dst, '"')

	start := 0
	for i := 0; i < len(s); i++ {
		b := s[i]

		if b >= 0x20 && b != '\\' && b != '"' {
			continue
		}

		dst = append(dst, s[start:i]...)

		switch b {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			// 3. Simplified hex escape for control characters (0x00 - 0x1F)
			// JSON spec requires \u00xx for these.
			// Replacing strconv.QuoteRune with a manual hex append is much faster.
			dst = append(dst, '\\', 'u', '0', '0', hexTable[b>>4], hexTable[b&0x0f])
		}
		start = i + 1
	}

	dst = append(dst, s[start:]...)
	dst = append(dst, '"')
	return dst
}
