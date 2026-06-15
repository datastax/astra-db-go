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

package serdes

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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

func parseInt(ctx DecodeCtx, src []byte) ([]byte, int64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] == '-' || (src[end] >= '0' && src[end] <= '9')) {
		end++
	}

	if end == 0 {
		return src, 0, ctx.syntaxError(src, "expected int64 but found "+nextJsonType(src))
	}

	num, err := strconv.ParseInt(unsafeString(src[:end]), 10, 64)
	if err != nil {
		return src[end:], 0, ctx.syntaxErrorWrap(src, "invalid int64", err)
	}
	return src[end:], num, nil
}

func parseUint(ctx DecodeCtx, src []byte) ([]byte, uint64, error) {
	src = skipWS(src)
	end := 0
	for end < len(src) && (src[end] >= '0' && src[end] <= '9') {
		end++
	}

	if end == 0 {
		return src, 0, ctx.syntaxError(src, "expected uint64 but found "+nextJsonType(src))
	}

	num, err := strconv.ParseUint(unsafeString(src[:end]), 10, 64)
	if err != nil {
		return src[end:], 0, ctx.syntaxErrorWrap(src, "invalid uint64", err)
	}
	return src[end:], num, nil
}

var floatChars = [256]uint8{
	'0': 1, '1': 1, '2': 1, '3': 1, '4': 1,
	'5': 1, '6': 1, '7': 1, '8': 1, '9': 1,
	'.': 1, '-': 1, '+': 1, 'e': 1, 'E': 1,
}

func parseFloat(ctx DecodeCtx, src []byte) ([]byte, float64, error) {
	src, numStr, err := parseNumber(ctx, src)
	if err != nil {
		return src, 0, ctx.syntaxError(src, "expected float64 but found "+nextJsonType(src))
	}

	f, err := strconv.ParseFloat(unsafeString(numStr), 64)
	if err != nil {
		return src, 0, ctx.syntaxErrorWrap(src, "invalid float", err)
	}
	return src, f, nil
}

// parseNumber extracts a number string without parsing it, for use with big.Int and big.Float
func parseNumber(ctx DecodeCtx, src []byte) ([]byte, []byte, error) {
	src = skipWS(src)
	end := 0

	for end < len(src) && floatChars[src[end]] != 0 {
		end++
	}

	if end == 0 {
		return src, nil, ctx.syntaxError(src, "expected number")
	}

	num := src[:end]
	if num[0] == '+' {
		return src, nil, ctx.syntaxError(src, "invalid leading '+' in number")
	}

	start := 0
	if num[0] == '-' {
		start = 1
	}

	if start >= len(num) {
		return src, nil, ctx.syntaxError(src, "expected digit after '-'")
	}

	// No leading zero unless followed by decimal or exponent
	if len(num)-start > 1 && num[start] == '0' {
		next := num[start+1]
		if next != '.' && next != 'e' && next != 'E' {
			return src, nil, ctx.syntaxError(src, "invalid leading zero in number")
		}
	}

	// Validate decimal placement
	if dot := bytes.IndexByte(num, '.'); dot >= 0 {
		// Cannot be first char, last char, and MUST be followed by a digit
		if dot == start || dot == len(num)-1 || num[dot+1] < '0' || num[dot+1] > '9' {
			return src, nil, ctx.syntaxError(src, "invalid decimal point in number")
		}
	}

	return src[end:], num, nil
}

type stringKind int

const (
	erroredString stringKind = iota
	simpleString
	escapedString
	unicodeString
)

// parseString is vendored and modified from:
// https://github.com/segmentio/encoding/blob/fd406855de30c54110d23eace25478ab9c6fa2cc/json/parse.go#L405
func parseString(ctx DecodeCtx, src []byte) ([]byte, []byte, stringKind, error) {
	src = skipWS(src)

	if len(src) < 2 {
		return src[len(src):], nil, erroredString, ctx.syntaxError(src, "expected string")
	}
	if src[0] != '"' {
		return src, nil, erroredString, ctx.syntaxError(src, "expected '\"' at the beginning of a string value")
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
		return src, nil, erroredString, ctx.syntaxError(src, "missing '\"' at the end of a string value")
	}

found:
	if bytes.IndexByte(src[1:n], '\\') < 0 {
		return src[n:], src[:n], simpleString, nil
	}

	kind := escapedString
	for i := 1; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
			if i < len(src) { // TODO any more I should just delegate to json.Unmarshal for?
				c := src[i]
				if c == 'u' || c == '/' {
					kind = unicodeString
				}
			}
		case '"':
			return src[i+1:], src[:i+1], kind, nil
		}
	}

	return src, nil, erroredString, ctx.syntaxError(src, "missing '\"' at the end of a string value")
}

func parseStringUnquote(ctx DecodeCtx, src []byte) ([]byte, []byte, bool, error) {
	src, s, kind, err := parseString(ctx, src)
	if err != nil {
		return src, s, false, err
	}

	switch kind {
	case simpleString:
		return src, s[1 : len(s)-1], false, nil
	case escapedString:
		res, err := strconv.Unquote(unsafeString(s))
		if err != nil {
			return src, nil, true, ctx.syntaxErrorWrap(src, "invalid string escape", err)
		}
		return src, []byte(res), true, nil
	case unicodeString:
		var res string
		if err := json.Unmarshal(s, &res); err != nil {
			return src, nil, true, ctx.syntaxErrorWrap(src, "invalid string escape", err)
		}
		return src, []byte(res), true, nil
	}

	panic(fmt.Sprintf("unexpected string kind: %d", kind))
}

func skipWS(src []byte) []byte {
	for i, c := range src {
		if c != ' ' && c != '\n' && c != '\t' && c != '\r' {
			return src[i:]
		}
	}
	return nil
}

func skipWSRev(src []byte) []byte {
	for i := len(src) - 1; i >= 0; i-- {
		c := src[i]
		if c != ' ' && c != '\n' && c != '\t' && c != '\r' {
			return src[:i+1]
		}
	}
	return nil
}

func skipValue(ctx DecodeCtx, src []byte) ([]byte, error) {
	src = skipWS(src)
	if len(src) == 0 {
		return src, ctx.syntaxError(src, "unexpected end of input")
	}

	switch src[0] {
	case '"':
		src, _, _, err := parseString(ctx, src)
		if err != nil {
			return src, err
		}
		return src, nil

	case '{', '[':
		depth, open, cls := 0, src[0], byte('}')
		if open == '[' {
			cls = ']'
		}

		for i := 0; i < len(src); {
			switch src[i] {
			case '"':
				rest, _, _, err := parseString(ctx, src[i:])
				if err != nil {
					return src, err
				}
				i = len(src) - len(rest)
			case open:
				depth++
				i++
			case cls:
				depth--
				if depth == 0 {
					return src[i+1:], nil
				}
				i++
			default:
				i++
			}
		}
		return src, ctx.syntaxError(src, "unexpected end of input while skipping value")

	default:
		i := 0
		for i < len(src) && src[i] != ',' && src[i] != '}' && src[i] != ']' {
			i++
		}
		if i == 0 {
			return src, ctx.syntaxError(src, "expected a value")
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
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
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
