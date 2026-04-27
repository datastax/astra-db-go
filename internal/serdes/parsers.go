package serdes

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
	"strconv"
)

func decodeInt(src []byte) ([]byte, int64, error) {
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

func decodeUint(src []byte) ([]byte, uint64, error) {
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

func decodeFloat(src []byte) ([]byte, float64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] == '-' || src[end] == '.' || (src[end] >= '0' && src[end] <= '9')) {
		end++
	}

	if end == 0 {
		return src, 0, fmt.Errorf("expected float")
	}

	f, err := strconv.ParseFloat(unsafeString(src[:end]), 64)
	return src[end:], f, err
}

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
