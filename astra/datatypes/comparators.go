package datatypes

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/v2/internal/typecache"
)

// Comparator compares two values via pointers.
// Return <0 for a < b, 0 for a == b, and >0 for a > b.
type Comparator func(a, b unsafe.Pointer) int

// Comparable is for types that can compare themselves to another value of the same type.
// The other arg will be the same type as the receiver.
type Comparable interface {
	CompareTo(other any) int
}

var (
	comparatorRegistry typecache.Map[Comparator]
)

// RegisterComparator registers a Comparator for type t, making it available via ComparatorFor.
// This is required for any type not natively supported by ComparatorFor (e.g. interface{}/any).
// It is safe to call concurrently; reads in ComparatorFor are lock-free.
func RegisterComparator(t reflect.Type, cmp Comparator) {
	comparatorRegistry.Store(t, cmp)
}

// ComparatorFor returns a Comparator for the given type. It checks the registry first, then
// falls back to built-in support for primitives, time.Time, *big.Int, *big.Float, and Comparable.
//
// reflect.Type is only used once at construction to pick the logic; the returned closure
// uses unsafe.Pointer casts and is allocation-free.
//
// The Comparable branch is the exception — it uses reflect.NewAt and .Interface() which allocates
// on each call, but Comparable keys are niche enough that it shouldn't matter.
func ComparatorFor(t reflect.Type) Comparator {
	if c, ok := comparatorRegistry.Load(t); ok {
		return c
	}

	if t.Implements(comparableType) {
		return func(a, b unsafe.Pointer) int {
			av := reflect.NewAt(t, a).Elem().Interface().(Comparable) // maybe replace w/ i-face trick if we need more speed
			bv := reflect.NewAt(t, b).Elem().Interface()
			return av.CompareTo(bv)
		}
	}

	switch t {
	case timeType:
		return func(a, b unsafe.Pointer) int {
			return (*time.Time)(a).Compare(*(*time.Time)(b))
		}
	case bigIntPtrType:
		return func(a, b unsafe.Pointer) int {
			return (*(**big.Int)(a)).Cmp(*(**big.Int)(b))
		}
	case bigFloatPtrType:
		return func(a, b unsafe.Pointer) int {
			return (*(**big.Float)(a)).Cmp(*(**big.Float)(b))
		}
	case jsonMsgType:
		return func(a, b unsafe.Pointer) int { return strings.Compare(*(*string)(a), *(*string)(b)) }
	}

	switch t.Kind() {
	case reflect.Int:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int)(a), *(*int)(b)) }
	case reflect.Int8:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int8)(a), *(*int8)(b)) }
	case reflect.Int16:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int16)(a), *(*int16)(b)) }
	case reflect.Int32:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int32)(a), *(*int32)(b)) }
	case reflect.Int64:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*int64)(a), *(*int64)(b)) }
	case reflect.Uint, reflect.Uintptr:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint)(a), *(*uint)(b)) }
	case reflect.Uint8:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint8)(a), *(*uint8)(b)) }
	case reflect.Uint16:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint16)(a), *(*uint16)(b)) }
	case reflect.Uint32:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint32)(a), *(*uint32)(b)) }
	case reflect.Uint64:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*uint64)(a), *(*uint64)(b)) }
	case reflect.Float32:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*float32)(a), *(*float32)(b)) }
	case reflect.Float64:
		return func(a, b unsafe.Pointer) int { return cmp.Compare(*(*float64)(a), *(*float64)(b)) }
	case reflect.String:
		return func(a, b unsafe.Pointer) int { return strings.Compare(*(*string)(a), *(*string)(b)) }
	}

	panic(fmt.Sprintf("ComparatorFor: no comparator available for type %s. Either have it implement datatypes.Comparable, or use datatypes.RegisterComparator()", t))
}

var (
	timeType        = reflect.TypeOf(time.Time{})
	bigIntPtrType   = reflect.TypeFor[*big.Int]()
	bigFloatPtrType = reflect.TypeFor[*big.Float]()
	jsonMsgType     = reflect.TypeFor[json.RawMessage]()
	comparableType  = reflect.TypeOf((*interface{ CompareTo(any) int })(nil)).Elem()
)
