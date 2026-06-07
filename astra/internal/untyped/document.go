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

package untyped

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/datastax/astra-db-go/astra/serdes"
)

// Document represents an untyped document used for a collection operation (as opposed to using a specific struct).
type Document interface {
	isDocument()

	// Get looks up a value in the document using a sequence of keys to navigate
	// through nested maps.
	Get(path ...string) (any, bool)

	// MustGet is like [Document.Get] but panics if the path doesn't exist.
	MustGet(path ...string) any

	// Decode tries to extract the value at the path and store it in dest.
	Decode(dest any, path ...string) error

	// ToMap converts the entire document into a standard map[string]any.
	ToMap() map[string]any
}

// NewDocument is a map-based implementation of [Document], primarily used for insertion.
type NewDocument map[string]any

func (NewDocument) isDocument() {}

// ToMap returns the document as a standard map[string]any.
func (d NewDocument) ToMap() map[string]any {
	return d
}

// Get looks up a value in the document using a sequence of keys to navigate
// through nested maps.
func (d NewDocument) Get(path ...string) (any, bool) {
	return getDeepFromMap(d, path...)
}

// MustGet is like [NewDocument.Get] but panics if the path doesn't exist.
func (d NewDocument) MustGet(path ...string) any {
	return mustGet(d.Get, path, "Document")
}

// Decode tries to extract the value at the path and store it in dest.
func (d NewDocument) Decode(dest any, path ...string) error {
	return decodeFromMap(d, path, dest, serdes.TargetCollection)
}

func (d NewDocument) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetCollection {
		return nil, fmt.Errorf("`NewDocument` can only be serialized for collections, got %s", ctx.Target)
	}
	return serdes.SerializeInto(map[string]any(d), ctx.Target, dst, ctx.Flags)
}

func (d NewDocument) UnmarshalAstraRaw(_ serdes.DecodeCtx, _ []byte) error {
	return fmt.Errorf("cannot deserialize into NewDocument; use the astra.Document interface for results")
}

type ServerDocument struct {
	Data  map[string]json.RawMessage
	Flags serdes.DesFlags
}

func (d *ServerDocument) isDocument() {}

func (d *ServerDocument) ToMap() map[string]any {
	result := make(map[string]any, len(d.Data))

	for name, rawValue := range d.Data {
		if string(rawValue) == "null" {
			result[name] = nil
			continue
		}

		var val any
		_ = serdes.Deserialize(rawValue, &val, nil, serdes.TargetCollection, d.Flags)
		result[name] = val
	}
	return result
}

func (d *ServerDocument) Get(path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	currentRaw, ok := d.Data[path[0]]
	if !ok {
		return nil, false
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetCollection, d.Flags); err != nil {
			return nil, false
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return nil, false
		}
	}

	var generic any
	if err := serdes.Deserialize(currentRaw, &generic, nil, serdes.TargetCollection, d.Flags); err != nil {
		return nil, false
	}
	return generic, true
}

func (d *ServerDocument) MustGet(path ...string) any {
	return mustGet(d.Get, path, "Document")
}

func (d *ServerDocument) Decode(dest any, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("astra: empty path for Decode")
	}

	currentRaw, ok := d.Data[path[0]]
	if !ok {
		return fmt.Errorf("astra: path %q not found", path[0])
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetCollection, d.Flags); err != nil {
			return fmt.Errorf("astra: failed to decode intermediate path %q: %w", path[i-1], err)
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return fmt.Errorf("astra: path %q not found", path[i])
		}
	}

	return serdes.Deserialize(currentRaw, dest, nil, serdes.TargetCollection, d.Flags)
}

func (d *ServerDocument) UnmarshalAstraRaw(ctx serdes.DecodeCtx, value []byte) error {
	d.Data = make(map[string]json.RawMessage)
	d.Flags = ctx.Flags
	return serdes.Deserialize(value, &d.Data, nil, serdes.TargetCollection, d.Flags)
}

func (d *ServerDocument) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetCollection {
		return nil, fmt.Errorf("`Document` can only be serialized for collections, got %s", ctx.Target)
	}
	return serdes.SerializeInto(d.Data, ctx.Target, dst, ctx.Flags)
}

var documentInterfaceType = reflect.TypeFor[Document]()

type DocumentTargetCtx struct{}

func (DocumentTargetCtx) UntypedTargetInterface() reflect.Type {
	return documentInterfaceType
}

func (DocumentTargetCtx) NewUntypedTarget(ctx serdes.DecodeCtx, p unsafe.Pointer) serdes.AstraRawUnmarshaler {
	doc := &ServerDocument{Flags: ctx.Flags}
	*(*Document)(p) = doc
	return doc
}

var GlobalDocumentCtx = DocumentTargetCtx{}

func NewDocumentTargetCtx() serdes.TargetDecodeCtx {
	return GlobalDocumentCtx
}
