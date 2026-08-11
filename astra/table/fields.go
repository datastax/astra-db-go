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
	"reflect"
	"strconv"
	"strings"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/internal/reflectutil"
)

type fieldInfo struct {
	modifier fieldModifier
	typeExpr string
}

func compileFields(t reflect.Type) (datatypes.LinkedMap[string, fieldInfo], error) {
	res := datatypes.NewLinkedMap[string, fieldInfo]()

	t = reflectutil.UnwindPointerType(t)
	if t.Kind() != reflect.Struct {
		return res, fmt.Errorf("expected struct, got %q", t.Kind())
	}

	fields, err := reflectutil.GetFlattenedFields(t)
	if err != nil {
		return res, err
	}

	for _, f := range fields {
		if strings.HasPrefix(f.Name, "$") {
			continue
		}

		if res.Has(f.Name) {
			return res, fmt.Errorf("duplicate field name %q", f.Name)
		}

		resolved, err, skip := resolveField(f.Field)
		if err != nil {
			return res, fmt.Errorf("field %q: %w", f.Name, err)
		}

		if !skip {
			res.Set(f.Name, resolved)
		}
	}

	if res.Len() == 0 {
		return res, fmt.Errorf("no columns found")
	}

	return res, nil
}

func resolveField(typ reflect.StructField) (fieldInfo, error, bool) {
	info, err, skip := parseAstraTag(typ.Tag.Get("astra"))

	if err != nil || skip {
		return info, err, skip
	}

	if yes, hint := shouldGenTypeExpr(info); yes {
		expr, err := genTypeExpr(typ.Type, hint)
		if err != nil {
			return info, fmt.Errorf("error generating Data API type for %q: %w", typ.Name, err), false
		}
		info.typeExpr = expr
	}

	if info.modifier != nil {
		if err := info.modifier.Validate(info); err != nil {
			return info, err, false
		}
	}

	return info, nil, false
}

func parseAstraTag(tag string) (fieldInfo, error, bool) {
	if tag == "" {
		return fieldInfo{}, nil, false
	}

	sections := splitTag(tag)

	fieldModifiers := []fieldModifier{
		pkFieldMod{},
		ckFieldMod{},
		dimFieldMod{},
	}

	var info fieldInfo

outer:
	for _, section := range sections {
		if section == "-" {
			if len(sections) > 1 {
				return info, fmt.Errorf("skip '-' cannot be combined with other modifiers"), false
			}
			return info, nil, true
		}

		if strings.HasPrefix(section, "type=") {
			if info.typeExpr != "" {
				return info, fmt.Errorf("duplicate type= expression"), false
			}
			info.typeExpr = strings.ToLower(strings.TrimPrefix(section, "type="))
			continue outer
		}

		for _, modifier := range fieldModifiers {
			result, err, ok := modifier.Parse(section)

			if info.modifier != nil && ok {
				return info, fmt.Errorf("conflicting modifiers %q and %q present", info.modifier.Name(), modifier.Name()), false
			}

			if err != nil {
				return info, fmt.Errorf("error parsing %q modifier: %w", modifier.Name(), err), false
			}

			if ok {
				info.modifier = result
				continue outer
			}
		}

		return info, fmt.Errorf("unknown modifier %q", section), false
	}
	return info, nil, false
}

func splitTag(tag string) []string {
	var result []string
	var depth, start = 0, 0

	for i, c := range tag {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(tag[start:i]))
				start = i + 1
			}
		}
	}
	result = append(result, strings.TrimSpace(tag[start:]))
	return result
}

type fieldModifier interface {
	Name() string
	Parse(section string) (fieldModifier, error, bool)
	Validate(info fieldInfo) error
}

type dimFieldMod struct {
	dim int
}

func (d dimFieldMod) Name() string {
	return "dim=N"
}

func (d dimFieldMod) Parse(section string) (fieldModifier, error, bool) {
	if !strings.HasPrefix(section, "dim=") {
		return nil, nil, false
	}

	dim, err := strconv.Atoi(strings.TrimPrefix(section, "dim="))
	if err != nil {
		return nil, fmt.Errorf("invalid dim value: %w", err), true
	}
	d.dim = dim
	return d, nil, true
}

func (d dimFieldMod) Validate(info fieldInfo) error {
	if info.typeExpr != "vector" {
		return fmt.Errorf("dim=N is only compatible with the vector type")
	}
	return nil
}

type pkFieldMod struct {
	ord int
}

func (p pkFieldMod) Name() string {
	return "pk[...]"
}

func (p pkFieldMod) Parse(section string) (fieldModifier, error, bool) {
	ord, _, err, matched := parseTableKey(section, "pk", 0)
	if err != nil || !matched {
		return nil, err, matched
	}

	p.ord = ord
	return p, nil, true
}

func (p pkFieldMod) Validate(_ fieldInfo) error {
	return nil
}

type ckFieldMod struct {
	ord  int
	desc bool
}

func (c ckFieldMod) Name() string {
	return "ck[...]"
}

func (c ckFieldMod) Parse(section string) (fieldModifier, error, bool) {
	ord, args, err, matched := parseTableKey(section, "ck", 1)
	if err != nil || !matched {
		return nil, err, matched
	}

	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "asc":
			c.desc = false
		case "desc":
			c.desc = true
		default:
			return nil, fmt.Errorf("invalid column order %q (expected either 'asc' or 'desc')", args[0]), false
		}
	}

	c.ord = ord
	return c, nil, true
}

func (c ckFieldMod) Validate(_ fieldInfo) error {
	return nil
}

func parseTableKey(section string, prefix string, extraArgs int) (int, []string, error, bool) {
	if !strings.HasPrefix(section, prefix) {
		return 0, nil, nil, false
	}
	section = strings.TrimPrefix(section, prefix)

	if section == "" {
		return 0, nil, nil, true
	}

	if !strings.HasPrefix(section, "[") || !strings.HasSuffix(section, "]") {
		return 0, nil, fmt.Errorf("expected brackets in after %q in %q", prefix, section), true
	}

	args := strings.Split(section[1:len(section)-1], ",")
	if len(args) == 1 && args[0] == "" {
		return 0, nil, fmt.Errorf("expected at least the ordinal if brackets are present in %q", section), true
	}

	if len(args) > extraArgs+1 {
		return 0, nil, fmt.Errorf("too many arguments in %q (expected at most %d; got %d)", section, extraArgs+1, len(args)), true
	}

	ordStr := args[0]
	ord, err := strconv.Atoi(ordStr)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid ordinal %q: %w", ordStr, err), false
	}
	return ord, args[1:], nil, true
}
