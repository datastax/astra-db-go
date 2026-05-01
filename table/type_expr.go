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

import "fmt"

// typeExpr is the parsed form of the value of a `type=<T>` modifier on an
// astra struct tag. It supports bracket-parameterized containers
// (set[T], list[T], map[K]V), the udt[<name>] form, and the `infer` leaf
// keyword which defers to the Go field type at that position.
type typeExpr struct {
	// name is the unparameterized keyword: a scalar ("text", "ascii", ...,
	// "vector", "duration"), a container ("set", "list", "map"), "udt", or
	// "infer".
	name string
	// elem is the element type for "set" and "list", or the value type for
	// "map". nil for scalars, udt, and infer.
	elem *typeExpr
	// key is the key type for "map". nil elsewhere.
	key *typeExpr
	// udtName is the name inside udt[<name>]. Empty unless name == "udt".
	udtName string
}

// inferExpr returns a fresh infer-leaf node. Used by the parser to auto-fill
// children of a bare container (set / list / map without brackets).
func inferExpr() *typeExpr { return &typeExpr{name: "infer"} }

// parseTypeExpr parses a full type expression, erroring on empty input,
// malformed syntax, or trailing characters.
func parseTypeExpr(s string) (typeExpr, error) {
	if s == "" {
		return typeExpr{}, fmt.Errorf("empty type= value")
	}
	p := typeExprParser{src: s}
	expr, err := p.parseExpr()
	if err != nil {
		return typeExpr{}, err
	}
	if p.pos < len(p.src) {
		return typeExpr{}, fmt.Errorf("unexpected trailing characters in type= value: %q", p.src[p.pos:])
	}
	return expr, nil
}

type typeExprParser struct {
	src string
	pos int
}

func (p *typeExprParser) parseExpr() (typeExpr, error) {
	ident, err := p.readIdent()
	if err != nil {
		return typeExpr{}, err
	}

	switch ident {
	case "set", "list":
		if !p.hasByte('[') {
			return typeExpr{name: ident, elem: inferExpr()}, nil
		}
		p.pos++
		child, err := p.parseExpr()
		if err != nil {
			return typeExpr{}, fmt.Errorf("%s[...]: %w", ident, err)
		}
		if err := p.consume(']'); err != nil {
			return typeExpr{}, fmt.Errorf("%s[...]: %w", ident, err)
		}
		return typeExpr{name: ident, elem: &child}, nil

	case "map":
		if !p.hasByte('[') {
			return typeExpr{name: "map", key: inferExpr(), elem: inferExpr()}, nil
		}
		p.pos++
		keyExpr, err := p.parseExpr()
		if err != nil {
			return typeExpr{}, fmt.Errorf("map[K]V: %w", err)
		}
		if err := p.consume(']'); err != nil {
			return typeExpr{}, fmt.Errorf("map[K]V: %w", err)
		}
		if p.pos >= len(p.src) {
			return typeExpr{}, fmt.Errorf("map[K]V requires both key and value types")
		}
		valExpr, err := p.parseExpr()
		if err != nil {
			return typeExpr{}, fmt.Errorf("map[K]V: %w", err)
		}
		return typeExpr{name: "map", key: &keyExpr, elem: &valExpr}, nil

	case "udt":
		if !p.hasByte('[') {
			return typeExpr{}, fmt.Errorf("udt requires a name (use udt[<name>])")
		}
		p.pos++
		name, err := p.readIdent()
		if err != nil {
			return typeExpr{}, fmt.Errorf("udt[<name>] requires a name")
		}
		if err := p.consume(']'); err != nil {
			return typeExpr{}, fmt.Errorf("udt[%s]: %w", name, err)
		}
		return typeExpr{name: "udt", udtName: name}, nil

	case "infer":
		if p.hasByte('[') {
			return typeExpr{}, fmt.Errorf("infer is a leaf; brackets not allowed")
		}
		return typeExpr{name: "infer"}, nil

	default:
		if p.hasByte('[') {
			return typeExpr{}, fmt.Errorf("scalar type %q cannot take bracket parameters", ident)
		}
		if !knownScalars[ident] {
			return typeExpr{}, fmt.Errorf("unknown type %q", ident)
		}
		return typeExpr{name: ident}, nil
	}
}

func (p *typeExprParser) readIdent() (string, error) {
	if p.pos >= len(p.src) {
		return "", fmt.Errorf("expected identifier, got end of input")
	}
	c := p.src[p.pos]
	if !isIdentStart(c) {
		return "", fmt.Errorf("expected identifier at position %d, got %q", p.pos, c)
	}
	start := p.pos
	p.pos++
	for p.pos < len(p.src) && isIdentCont(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos], nil
}

func (p *typeExprParser) hasByte(b byte) bool {
	return p.pos < len(p.src) && p.src[p.pos] == b
}

func (p *typeExprParser) consume(b byte) error {
	if p.pos >= len(p.src) {
		return fmt.Errorf("expected %q, got end of input", b)
	}
	if p.src[p.pos] != b {
		return fmt.Errorf("expected %q at position %d, got %q", b, p.pos, p.src[p.pos])
	}
	p.pos++
	return nil
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// knownScalars is the set of leaf type names valid as a type= scalar. Containers
// (set, list, map), udt, and infer are recognized separately in parseExpr.
var knownScalars = map[string]bool{
	TypeText:      true,
	TypeAscii:     true,
	TypeInt:       true,
	TypeBigInt:    true,
	TypeSmallInt:  true,
	TypeTinyInt:   true,
	TypeFloat:     true,
	TypeDouble:    true,
	TypeDecimal:   true,
	TypeBoolean:   true,
	TypeDate:      true,
	TypeTime:      true,
	TypeTimestamp: true,
	TypeUUID:      true,
	TypeTimeUUID:  true,
	TypeBlob:      true,
	TypeVarint:    true,
	TypeInet:      true,
	TypeVector:    true,
	TypeDuration:  true,
}
