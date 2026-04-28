package serdes

import (
	"reflect"
	"unsafe"
)

type typedCodec struct {
	codec
	typ reflect.Type
}

type Target struct {
	kind            targetKind
	typeOverrides   map[unsafe.Pointer]codec
	kindOverrides   map[reflect.Kind]func(codecCtx, reflect.Type, seenStructs, bool) codec
	dollarDatatypes map[string]typedCodec
}

type targetKind int

const (
	collectionKind targetKind = 1
	tableKind                 = 2
)

type purity int

const (
	pure purity = iota
	impure
)

func (p Target) String() string {
	switch p.kind {
	case collectionKind:
		return "collection"
	case tableKind:
		return "table"
	default:
		panic("unknown target")
	}
}
