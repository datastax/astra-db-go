package serdes

import (
	"reflect"
)

type typedCodec struct {
	codec
	typ reflect.Type
}

type targetKind int

const (
	noTarget targetKind = iota
	collectionTarget
	tableTarget
)

type Target struct {
	kind            targetKind
	dollarDatatypes map[string]typedCodec
}

var TargetNone = Target{
	kind: noTarget,
}

var TargetCollection = Target{
	kind: collectionTarget,
	dollarDatatypes: map[string]typedCodec{
		"$uuid":     {codec{uuidEncoder, uuidDecoder}, uuidType},
		"$objectId": {codec{objectIdEncoder, objectIdDecoder}, oidType},
		"$date":     {codec{timestampEncoder, timestampDecoder}, dApiTimeType},
		"$binary":   {codec{binaryEncoder, binaryDecoder}, byteSliceType},
	},
}

var TargetTable = Target{
	kind: tableTarget,
	dollarDatatypes: map[string]typedCodec{
		"$binary": {codec{binaryEncoder, binaryDecoder}, byteSliceType},
	},
}

func (t Target) Is(other Target) bool {
	return t.kind == other.kind
}

func (t Target) String() string {
	switch t.kind {
	case collectionTarget:
		return "collection"
	case tableTarget:
		return "table"
	default:
		return "none"
	}
}
