package serdes

import (
	"encoding/json"
	"net"
	"reflect"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

type fieldHint int

const (
	unknownField fieldHint = iota
	vectorField
	vectorizeField
)

var (
	astraCodecType = reflect.TypeFor[AstraCodec]()
)

var (
	nilType          = reflect.TypeOf(nil)
	anyType          = reflect.TypeFor[any]()
	stringType       = reflect.TypeFor[string]()
	float32SliceType = reflect.TypeFor[[]float32]()
	uuidType         = reflect.TypeFor[datatypes.UUID]()
	oidType          = reflect.TypeFor[datatypes.ObjectId]()
	vectorType       = reflect.TypeFor[datatypes.DataAPIVector]()
	timeType         = reflect.TypeFor[time.Time]()
	ipType           = reflect.TypeFor[net.IP]()
	rawMessageType   = reflect.TypeFor[json.RawMessage]()
)

var (
	uuidTag = []byte("uuid")
	oidTag  = []byte("objectid")
)

var (
	stringEmpty = reflect.Zero(stringType)
	anyEmpty    = reflect.Zero(anyType)
)

type iface struct {
	typ unsafe.Pointer
	ptr unsafe.Pointer
}

type slice struct {
	data unsafe.Pointer
	len  int
	cap  int
}

func typePtr(t reflect.Type) unsafe.Pointer {
	return (*iface)(unsafe.Pointer(&t)).ptr
}

func valuePtr(v reflect.Value) unsafe.Pointer {
	return (*iface)(unsafe.Pointer(&v)).ptr
}

func inlined(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr:
		return true
	case reflect.Map:
		return true
	case reflect.Struct:
		return t.NumField() == 1 && inlined(t.Field(0).Type)
	default:
		return false
	}
}

//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func alignedSize(t reflect.Type) uintptr {
	a := t.Align()
	s := t.Size()
	return align(uintptr(a), s)
}

func align(align, size uintptr) uintptr {
	if align != 0 && (size%align) != 0 {
		size = ((size / align) + 1) * align
	}
	return size
}

func extractFieldHint(field string) fieldHint {
	if len(field) > 0 && field[0] == '$' {
		switch {
		case field[1:] == "vector":
			return vectorField
		case field[1:] == "vectorize":
			return vectorizeField
		}
	}
	return unknownField
}
