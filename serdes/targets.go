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
	unknownKind targetKind = iota
	collectionKind
	tableKind
)

type Target struct {
	kind            targetKind
	dollarDatatypes map[string]typedCodec
}

var TargetUnknown = Target{
	kind: unknownKind,
}

var TargetCollection = Target{
	kind: collectionKind,
	dollarDatatypes: map[string]typedCodec{
		"$uuid":     {codec{uuidEncoder, uuidDecoder}, uuidType},
		"$objectId": {codec{objectIdEncoder, objectIdDecoder}, oidType},
		"$date":     {codec{timestampEncoder, timestampDecoder}, dApiTimeType},
		"$binary":   {codec{binaryEncoder, binaryDecoder}, byteSliceType},
	},
}

var TargetTable = Target{
	kind: tableKind,
	dollarDatatypes: map[string]typedCodec{
		"$binary": {codec{binaryEncoder, binaryDecoder}, byteSliceType},
	},
}

func (p Target) String() string {
	switch p.kind {
	case collectionKind:
		return "collection"
	case tableKind:
		return "table"
	default:
		return "unknown"
	}
}
