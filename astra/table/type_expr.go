package table

import (
	"fmt"
	"math/big"
	"net"
	"reflect"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/internal/typeutil"
	"github.com/datastax/astra-db-go/v2/internal/reflectutil"
)

type genTypeExprHint int

const (
	genTypeExprHintNone genTypeExprHint = iota
	genTypeExprHintSet
)

func shouldGenTypeExpr(info fieldInfo) (bool, genTypeExprHint) {
	if info.typeExpr == "" {
		return true, genTypeExprHintNone
	}
	if info.typeExpr == "set" {
		return true, genTypeExprHintSet
	}
	return false, genTypeExprHintNone
}

// Well-known reflect types for comparison in type mapping.
var (
	reflectUUID      = reflect.TypeFor[datatypes.UUID]()
	reflectVector    = reflect.TypeFor[datatypes.Vector]()
	reflectTime      = reflect.TypeFor[time.Time]()
	reflectDuration  = reflect.TypeFor[datatypes.Duration]()
	reflectIP        = reflect.TypeFor[net.IP]()
	reflectByteSlice = reflect.TypeFor[[]byte]()
	reflectDateOnly  = reflect.TypeFor[datatypes.DateOnly]()
	reflectTimeOnly  = reflect.TypeFor[datatypes.TimeOnly]()
	reflectBigInt    = reflect.TypeFor[big.Int]()
	reflectBigFloat  = reflect.TypeFor[big.Float]()
)

func genTypeExpr(t reflect.Type, hint genTypeExprHint) (string, error) {
	t = reflectutil.UnwindPointerType(t)

	switch t {
	case reflectUUID:
		return "uuid", nil
	case reflectVector:
		return "vector", nil
	case reflectTime:
		return "timestamp", nil
	case reflectDuration:
		return "duration", nil
	case reflectIP:
		return "inet", nil
	case reflectByteSlice:
		return "blob", nil
	case reflectDateOnly:
		return "date", nil
	case reflectTimeOnly:
		return "time", nil
	case reflectBigInt:
		return "varint", nil
	case reflectBigFloat:
		return "decimal", nil
	}

	switch t.Kind() {
	case reflect.String:
		return "text", nil
	case reflect.Int, reflect.Int32:
		return "int", nil
	case reflect.Int64:
		return "bigint", nil
	case reflect.Int16:
		return "smallint", nil
	case reflect.Int8, reflect.Uint8:
		return "tinyint", nil
	case reflect.Float32:
		return "float", nil
	case reflect.Float64:
		return "double", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Slice:
		if hint == genTypeExprHintSet {
			return genListLikeTypeExpr("set", t.Elem())
		}
		return genListLikeTypeExpr("list", t.Elem())
	case reflect.Map:
		return genMapTypeExpr(t.Key(), t.Elem())
	case reflect.Struct:
		switch typeutil.GetCustomGenericTypeID(t) {
		case typeutil.SetType:
			return genListLikeTypeExpr("set", typeutil.GetCustomGenericTypeKey(t))
		case typeutil.LinkedMapType, typeutil.SortedMapType:
			return genMapTypeExpr(typeutil.GetCustomGenericTypeKey(t), typeutil.GetCustomGenericTypeValue(t))
		default:
			return "", fmt.Errorf("struct type %q requires explicit type= modifier", t.Name())
		}
	default:
		return "", fmt.Errorf("unsupported Go type: %q", t)
	}
}

func genListLikeTypeExpr(container string, vt reflect.Type) (string, error) {
	vc, err := genTypeExpr(vt, genTypeExprHintNone)
	if err != nil {
		return "", fmt.Errorf("slice element: %w", err)
	}

	return fmt.Sprintf("%s[%s]", container, vc), nil
}

func genMapTypeExpr(kt, vt reflect.Type) (string, error) {
	kc, err := genTypeExpr(kt, genTypeExprHintNone)
	if err != nil {
		return "", fmt.Errorf("map key: %w", err)
	}

	vc, err := genTypeExpr(vt, genTypeExprHintNone)
	if err != nil {
		return "", fmt.Errorf("map element: %w", err)
	}

	return fmt.Sprintf("map[%s]%s", kc, vc), nil
}

func resolveTypeExpr(expr string, modifier fieldModifier) (Column, error) {
	parsed, err := parseTypeExpr(expr)
	if err != nil {
		return Column{}, err
	}
	return resolveTypeExprRecursive(parsed, modifier)
}

func resolveTypeExprRecursive(expr typeExpr, modifier fieldModifier) (Column, error) {
	switch expr.name {
	case "udt":
		if expr.udtName == "" {
			return Column{}, fmt.Errorf("udt requires a name")
		}
		return UDT(expr.udtName), nil

	case TypeSet:
		elem, err := resolveTypeExprRecursive(*expr.elem, nil)
		if err != nil {
			return Column{}, fmt.Errorf("set element: %w", err)
		}
		return Set(elem), nil

	case TypeList:
		elem, err := resolveTypeExprRecursive(*expr.elem, nil)
		if err != nil {
			return Column{}, fmt.Errorf("list element: %w", err)
		}
		return List(elem), nil

	case TypeMap:
		keyCol, err := resolveTypeExprRecursive(*expr.key, nil)
		if err != nil {
			return Column{}, fmt.Errorf("map key: %w", err)
		}
		valCol, err := resolveTypeExprRecursive(*expr.elem, nil)
		if err != nil {
			return Column{}, fmt.Errorf("map value: %w", err)
		}
		return Map(keyCol.Type, valCol), nil

	case TypeVector:
		if dimMod, ok := modifier.(dimFieldMod); ok {
			return Vector(dimMod.dim), nil
		}
		return Column{}, fmt.Errorf("type=vector requires dim=N")

	default:
		if factory, ok := scalarFactories[expr.name]; ok {
			return factory(), nil
		}
		return Column{}, fmt.Errorf("unknown type override %q", expr.name)
	}
}

// scalarFactories maps a scalar type name to the Column factory that produces
// the corresponding column. Vector and the container types are handled
// directly in resolveTypeExprOld because they need extra inputs.
var scalarFactories = map[string]func() Column{
	TypeText:      Text,
	TypeAscii:     Ascii,
	TypeInt:       Int,
	TypeBigInt:    BigInt,
	TypeSmallInt:  SmallInt,
	TypeTinyInt:   TinyInt,
	TypeFloat:     Float,
	TypeDouble:    Double,
	TypeDecimal:   Decimal,
	TypeBoolean:   Boolean,
	TypeDate:      Date,
	TypeTime:      Time,
	TypeTimestamp: Timestamp,
	TypeUUID:      UUID,
	TypeTimeUUID:  TimeUUID,
	TypeBlob:      Blob,
	TypeVarint:    Varint,
	TypeInet:      Inet,
	TypeDuration:  Duration,
}

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
		if !p.hasNext('[') {
			return typeExpr{}, fmt.Errorf("missing brackets in type=%s[V]", ident)
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
		if !p.hasNext('[') {
			return typeExpr{}, fmt.Errorf("missing brackets in type=map[K]V")
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
		if !p.hasNext('[') {
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

	default:
		if p.hasNext('[') {
			return typeExpr{}, fmt.Errorf("scalar type %q cannot take bracket parameters", ident)
		}
		if !isKnownScalar(ident) {
			return typeExpr{}, fmt.Errorf("unknown type %q", ident)
		}
		return typeExpr{name: ident}, nil
	}
}

func isKnownScalar(s string) bool {
	return scalarFactories[s] != nil || s == "vector"
}

func (p *typeExprParser) readIdent() (string, error) {
	if p.pos >= len(p.src) {
		return "", fmt.Errorf("expected type identifier, got end of input")
	}
	c := p.src[p.pos]
	if !isIdentStart(c) {
		return "", fmt.Errorf("expected type identifier at position %d, got %q", p.pos, c)
	}
	start := p.pos
	p.pos++
	for p.pos < len(p.src) && isIdentCont(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos], nil // should already be lowercase
}

func (p *typeExprParser) consume(b byte) error {
	if p.pos >= len(p.src) {
		return fmt.Errorf("expected %q, got end of input", b)
	}
	if !p.hasNext(b) {
		return fmt.Errorf("expected %q at position %d, got %q", b, p.pos, p.src[p.pos])
	}
	p.pos++
	return nil
}

func (p *typeExprParser) hasNext(b byte) bool {
	return p.pos < len(p.src) && p.src[p.pos] == b
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
