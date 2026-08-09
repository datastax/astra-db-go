package typeutil

import (
	"bytes"
	"reflect"
	"strings"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
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

type CustomGenericTypeID int

const (
	None CustomGenericTypeID = iota
	LinkedMapType
	SortedMapType
	SetType
)

func GetCustomGenericTypeID(t reflect.Type) CustomGenericTypeID {
	if t.PkgPath() != datatypesPkgPath {
		return None
	}

	tName := t.Name()

	switch {
	case strings.HasPrefix(tName, someLinkedMapTypeName):
		return LinkedMapType
	case strings.HasPrefix(tName, someSortedMapTypeName):
		return SortedMapType
	case strings.HasPrefix(tName, someSetTypeName):
		return SetType
	}

	return None
}

func GetCustomGenericTypeKey(t reflect.Type) reflect.Type {
	switch GetCustomGenericTypeID(t) {
	case LinkedMapType, SortedMapType:
		kt, _ := t.FieldByName("kType")
		return kt.Type.Elem()
	default:
		panic("GetCustomGenericTypeKey called on invalid type '" + t.Name() + "'. This is an astra-db-go bug and should be reported to the developers.")
	}
}

func GetCustomGenericTypeValue(t reflect.Type) reflect.Type {
	switch GetCustomGenericTypeID(t) {
	case SetType:
		vt, _ := t.FieldByName("kType") // reminder that datatypes.Set[T] is just a datatypes.SortedMap[T, struct{}]
		return vt.Type.Elem()
	case LinkedMapType, SortedMapType:
		kt, _ := t.FieldByName("vType")
		return kt.Type.Elem()
	default:
		panic("GetCustomGenericTypeValue called on invalid type '" + t.Name() + "'. This is an astra-db-go bug and should be reported to the developers.")
	}
}

func typeNameUntilBracket(t reflect.Type) string {
	name := t.Name()
	if idx := bytes.IndexByte([]byte(name), '['); idx != -1 {
		return name[:idx+1]
	}
	return name
}
