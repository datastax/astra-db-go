package serdes

import (
	"net"
	"reflect"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/datatypes"
)

var (
	astraCodecType = reflect.TypeFor[AstraCodec]()
)

var (
	nilType    = reflect.TypeOf(nil)
	anyType    = reflect.TypeFor[any]()
	uuidType   = reflect.TypeFor[datatypes.UUID]()
	vectorType = reflect.TypeFor[datatypes.DataAPIVector]()
	timeType   = reflect.TypeFor[time.Time]()
	ipType     = reflect.TypeFor[net.IP]()
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

func typeid(t reflect.Type) unsafe.Pointer {
	return (*iface)(unsafe.Pointer(&t)).ptr
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

// noescape hides a pointer from escape analysis.  noescape is
// the identity function but escape analysis doesn't think the
// output depends on the input. noescape is inlined and currently
// compiles down to zero instructions.
// USE CAREFULLY!
// This was copied from the runtime; see issues 23382 and 7921.
//
//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}
