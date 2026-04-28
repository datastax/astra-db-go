package serdes

import (
	"reflect"
	"unsafe"
)

type Target struct {
	kind          targetKind
	typeOverrides map[unsafe.Pointer]codec
	kindOverrides map[reflect.Kind]func(codecCtx, reflect.Type, seenStructs, bool) codec
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
