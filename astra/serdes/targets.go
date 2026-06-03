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
	"reflect"
	"unsafe"
)

type TargetDecodeCtx interface {
	UntypedTargetInterface() reflect.Type
	NewUntypedTarget(ctx DecodeCtx, p unsafe.Pointer) AstraRawUnmarshaler
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
