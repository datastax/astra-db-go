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

package table

import (
	"fmt"
	"strconv"
	"strings"
)

// tagInfo holds parsed astra struct tag data.
type tagInfo struct {
	isPK         bool
	pkOrdinal    int // 1-based; 0 = unset
	isCK         bool
	ckOrdinal    int // 1-based; 0 = unset
	ckDescending bool
	typeOverride string
	dimension    int
	skip         bool
}

// parseAstraTag parses the value of an "astra" struct tag.
//
// Grammar: <role>[<brackets>][,<modifier>...]
//
// Roles: pk, ck, -
// Brackets:
//
//	pk[N]
//	ck[N], ck[ORD], ck[N,ORD] (where ORD is asc or desc)
//
// Modifiers: type=<T>, dim=<N>
func parseAstraTag(raw string) (tagInfo, error) {
	var info tagInfo
	if raw == "" {
		return info, nil
	}

	// 1. Extract role and optional bracket
	remaining := raw
	var roleToken string
	var bracketPayload string

	if strings.HasPrefix(raw, "-") && (len(raw) == 1 || raw[1] == ',') {
		return tagInfo{skip: true}, nil
	}

	isRole := false
	if strings.HasPrefix(raw, "pk") && (len(raw) == 2 || raw[2] == '[' || raw[2] == ',') {
		roleToken = "pk"
		isRole = true
	} else if strings.HasPrefix(raw, "ck") && (len(raw) == 2 || raw[2] == '[' || raw[2] == ',') {
		roleToken = "ck"
		isRole = true
	}

	if isRole {
		remaining = remaining[2:]
		if len(remaining) > 0 && remaining[0] == '[' {
			endIdx := strings.IndexByte(remaining, ']')
			if endIdx == -1 {
				return info, fmt.Errorf("missing closing bracket in tag %q", raw)
			}
			bracketPayload = remaining[1:endIdx]
			remaining = remaining[endIdx+1:]
		}
		if len(remaining) > 0 {
			if remaining[0] != ',' {
				return info, fmt.Errorf("expected comma after role token in tag %q", raw)
			}
			remaining = remaining[1:]
		}
	} else {
		roleToken = ""
	}

	switch roleToken {
	case "pk":
		info.isPK = true
		if bracketPayload != "" {
			n, err := strconv.Atoi(bracketPayload)
			if err != nil {
				return info, fmt.Errorf("pk ordinal must be a number, got %q", bracketPayload)
			}
			if n < 1 {
				return info, fmt.Errorf("pk ordinal must be >= 1, got %d", n)
			}
			info.pkOrdinal = n
		}

	case "ck":
		info.isCK = true
		if bracketPayload != "" {
			parts := strings.Split(bracketPayload, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "asc" {
					info.ckDescending = false
				} else if part == "desc" {
					info.ckDescending = true
				} else {
					n, err := strconv.Atoi(part)
					if err == nil {
						if n < 1 {
							return info, fmt.Errorf("ck ordinal must be >= 1, got %d", n)
						}
						info.ckOrdinal = n
					} else {
						return info, fmt.Errorf("ck bracket token must be a number, 'asc', or 'desc', got %q", part)
					}
				}
			}
		}
	}

	// 2. Parse remaining tokens as modifiers
	if remaining == "" {
		return info, nil
	}

	tokens := strings.Split(remaining, ",")
	for _, tok := range tokens {
		switch {
		case strings.HasPrefix(tok, "type="):
			info.typeOverride = tok[len("type="):]
		case strings.HasPrefix(tok, "dim="):
			d, err := strconv.Atoi(tok[len("dim="):])
			if err != nil {
				return info, fmt.Errorf("dim= value must be a number, got %q", tok[len("dim="):])
			}
			info.dimension = d
		default:
			return info, fmt.Errorf("unknown astra tag token %q", tok)
		}
	}

	return info, nil
}
