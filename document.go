package astradb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/datastax/astra-db-go/serdes"
)

type Document interface {
	Get(path ...string) (any, bool)
	Decode(dest any, path ...string) error
	ToMap() map[string]any
}

type NewDocument map[string]any

func (d NewDocument) ToMap() map[string]any {
	return d
}

func (d NewDocument) Get(path ...string) (any, bool) {
	current := d
	for i, p := range path {
		val, ok := current[p]
		if !ok {
			return nil, false
		}

		if i == len(path)-1 {
			return val, true
		}

		nextMap, ok := val.(map[string]any)
		if !ok {
			return nil, false
		}
		current = nextMap
	}
	return nil, false
}

func (d NewDocument) Decode(dest any, path ...string) error {
	val, ok := d.Get(path...)
	if !ok {
		return fmt.Errorf("path not found")
	}

	b, err := serdes.Serialize(val, serdes.TargetCollection)
	if err != nil {
		return err
	}

	return serdes.Deserialize(b, dest, nil, serdes.TargetCollection)
}

func (d NewDocument) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target != serdes.TargetCollection {
		return nil, fmt.Errorf("`NewDocument` can only be serialized for collections, got %s", ctx.Target)
	}
	return serdes.SerializeInto(map[string]any(d), ctx.Target, dst)
}

func (d *NewDocument) UnmarshalAstraRaw(_ serdes.DecodeCtx, _ []byte) error {
	return fmt.Errorf("cannot deserialize into NewDocument; use the astradb.Document interface for results")
}

type serverDocument struct {
	data map[string]json.RawMessage
}

func (d *serverDocument) ToMap() map[string]any {
	result := make(map[string]any, len(d.data))

	for name, rawValue := range d.data {
		if string(rawValue) == "null" {
			result[name] = nil
			continue
		}

		var val any
		_ = serdes.Deserialize(rawValue, &val, nil, serdes.TargetCollection)
		result[name] = val
	}
	return result
}

func (d *serverDocument) Get(path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	currentRaw, ok := d.data[path[0]]
	if !ok {
		return nil, false
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetCollection); err != nil {
			return nil, false
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return nil, false
		}
	}

	if string(currentRaw) == "null" {
		return nil, true
	}

	var generic any
	if err := serdes.Deserialize(currentRaw, &generic, nil, serdes.TargetCollection); err != nil {
		return nil, false
	}
	return generic, true
}

func (d *serverDocument) Decode(dest any, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	currentRaw, ok := d.data[path[0]]
	if !ok {
		return fmt.Errorf("path %s not found", path[0])
	}

	for i := 1; i < len(path); i++ {
		var nextLevel map[string]json.RawMessage
		if err := serdes.Deserialize(currentRaw, &nextLevel, nil, serdes.TargetCollection); err != nil {
			return err
		}
		currentRaw, ok = nextLevel[path[i]]
		if !ok {
			return fmt.Errorf("path %s not found", path[i])
		}
	}

	return serdes.Deserialize(currentRaw, dest, nil, serdes.TargetCollection)
}

func (d *serverDocument) UnmarshalAstraRaw(_ serdes.DecodeCtx, value []byte) error {
	d.data = make(map[string]json.RawMessage)
	return serdes.Deserialize(value, &d.data, nil, serdes.TargetCollection)
}

var documentInterfaceType = reflect.TypeFor[Document]()

type collectionTargetCtx struct{}

func (c collectionTargetCtx) UntypedTargetInterface() reflect.Type {
	return documentInterfaceType
}

func (c collectionTargetCtx) NewUntypedTarget(p unsafe.Pointer) serdes.AstraRawUnmarshaler {
	doc := &serverDocument{}
	*(*Document)(p) = doc
	return doc
}

var collectionCtx = collectionTargetCtx{}
