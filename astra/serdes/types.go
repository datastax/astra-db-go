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
	"encoding/json"
	"math/big"
	"net"
	"reflect"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/astra/datatypes"
)

type fieldHint int

const (
	unknownField fieldHint = iota
	vectorField
	vectorizeField
)

type rollback struct{}

func (rollback) Error() string {
	return "rollback"
}

var (
	astraMarshalerType      = reflect.TypeFor[AstraMarshaler]()
	astraRawMarshalerType   = reflect.TypeFor[AstraRawMarshaler]()
	astraUnmarshalerType    = reflect.TypeFor[AstraUnmarshaler]()
	astraRawUnmarshalerType = reflect.TypeFor[AstraRawUnmarshaler]()
	jsonMarshalerType       = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshalerType     = reflect.TypeFor[json.Unmarshaler]()
)

var (
	nilType          = reflect.TypeOf(nil)
	anyType          = reflect.TypeFor[any]()
	emptyType        = reflect.TypeFor[struct{}]()
	stringType       = reflect.TypeFor[string]()
	float32SliceType = reflect.TypeFor[[]float32]()
	byteSliceType    = reflect.TypeFor[[]byte]()
	uuidType         = reflect.TypeFor[datatypes.UUID]()
	oidType          = reflect.TypeFor[datatypes.ObjectId]()
	dateOnlyType     = reflect.TypeFor[datatypes.DateOnly]()
	timeOnlyType     = reflect.TypeFor[datatypes.TimeOnly]()
	durationType     = reflect.TypeFor[datatypes.DataAPIDuration]()
	vectorType       = reflect.TypeFor[datatypes.Vector]()
	timeType         = reflect.TypeFor[time.Time]()
	ipType           = reflect.TypeFor[net.IP]()
	rawMessageType   = reflect.TypeFor[json.RawMessage]()
	bigIntType       = reflect.TypeFor[big.Int]()
	bigFloatType     = reflect.TypeFor[big.Float]()
)

var (
	datatypesPkgPath = someLinkedMapType.PkgPath() // this will be the same for all special datatypes for now

	someLinkedMapType     = reflect.TypeFor[datatypes.LinkedMap[any, any]]()
	someLinkedMapTypeName = typeNameUntilBracket(someLinkedMapType)

	someSortedMapType     = reflect.TypeFor[datatypes.SortedMap[any, any]]()
	someSortedMapTypeName = typeNameUntilBracket(someSortedMapType)

	someSetType     = reflect.TypeFor[datatypes.Set[any]]()
	someSetTypeName = typeNameUntilBracket(someSetType)
)

var (
	emptyEmpty  = reflect.Zero(emptyType)
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
	align := uintptr(t.Align())
	size := t.Size()
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

func typeNameUntilBracket(t reflect.Type) string {
	name := t.Name()
	if idx := bytes.IndexByte([]byte(name), '['); idx != -1 {
		return name[:idx+1]
	}
	return name
}
