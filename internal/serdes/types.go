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
