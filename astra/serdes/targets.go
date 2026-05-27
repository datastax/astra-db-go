package serdes

import (
	"reflect"
	"unsafe"
)

type TargetDecodeCtx interface {
	UntypedTargetInterface() reflect.Type
	NewUntypedTarget(p unsafe.Pointer) AstraRawUnmarshaler
}

type typedCodec struct {
	codec
	typ reflect.Type
}

type Target int

const (
	TargetNone Target = iota
	TargetCollection
	TargetTable
)

var collectionDollarDatatypes = map[string]typedCodec{
	"$uuid":     {codec{uuidEncoder, uuidDecoder}, uuidType},
	"$objectId": {codec{objectIdEncoder, objectIdDecoder}, oidType},
	"$date":     {codec{timeEncoder, timeDecoder}, timeType},
	"$binary":   {codec{binaryEncoder, binaryDecoder}, byteSliceType},
}

var tableDollarDatatypes = map[string]typedCodec{
	"$binary": {codec{binaryEncoder, binaryDecoder}, byteSliceType},
}

var noneDollarDatatypes = map[string]typedCodec{}

func (t Target) dollarDatatypes() map[string]typedCodec {
	switch t {
	case TargetCollection:
		return collectionDollarDatatypes
	case TargetTable:
		return tableDollarDatatypes
	default:
		return noneDollarDatatypes
	}
}

func (t Target) String() string {
	switch t {
	case TargetCollection:
		return "collection"
	case TargetTable:
		return "table"
	default:
		return "none"
	}
}
