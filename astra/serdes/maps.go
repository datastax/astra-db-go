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
	"fmt"
	"reflect"
	"sort"
	"unsafe"

	"github.com/datastax/astra-db-go/astra/datatypes"
)

// Map serdes is complex enough to warrant its own file, as we're allowing for a Cartesian product of features:
// - Serdes w/ a table vs non-table target
// - Serdes w/ a native Go map vs an LinkedMap[K, V]
// - Sorted vs unsorted map encoding (for native Go maps)
//
// The below is not the cleanest code, but it aims to be fairly performant while still being maintainable
//
// The main idea is that we have a generic map encoder/decoder that takes in the logic for iterating over the map and
// for making/setting the map, and then we have thin wrappers around it for each of the 4 combinations of features

// Serdes

func mkSetCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt, _ := t.FieldByName("kType")
	et := kt.Type.Elem()

	c := resolveCodec(ctx, et, seen, false)

	return codec{
		mkGenericMapEncoder(t, et, c.encode, nil, '[', ']', 0, newMapIterFromSortedMap),
		mkGenericMapDecoder(t, et, emptyType, reflect.Zero(et), emptyEmpty, c.decode, nil, '[', ']', 0, mkSortedMapMaker(t)),
	}
}

func mkMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt := t.Key()
	vt := t.Elem()
	return mkGenericMapCodec(ctx, t, kt, vt, seen, newMapIterMakerFromMap(t), mkNativeMapMaker(t))
}

func mkLinkedMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt, _ := t.FieldByName("kType")
	vt, _ := t.FieldByName("vType")
	return mkGenericMapCodec(ctx, t, kt.Type.Elem(), vt.Type.Elem(), seen, newMapIterFromLinkedMap, mkLinkedMapMaker(t))
}

func mkSortedMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	kt, _ := t.FieldByName("kType")
	vt, _ := t.FieldByName("vType")
	return mkGenericMapCodec(ctx, t, kt.Type.Elem(), vt.Type.Elem(), seen, newMapIterFromSortedMap, mkSortedMapMaker(t))
}

func mkGenericMapCodec(ctx codecCtx, t, kt, vt reflect.Type, seen seenStructs, mkIter mkMapIter, maker mapMaker) codec {
	kz := reflect.Zero(kt)
	vz := reflect.Zero(vt)

	kc := resolveCodec(ctx, kt, seen, false)
	vc := resolveCodec(ctx, vt, seen, false)

	if inlined(vt) {
		vc.encode = mkInlineEncoder(vc.encode)
	}

	return codec{
		mkMapEncoder(t, kt, kc.encode, vc.encode, mkIter),
		mkMapDecoder(t, kt, vt, kz, vz, kc.decode, vc.decode, maker),
	}
}

func mkMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder, mkIter mkMapIter) encoder {
	// We'll do our best to encode maps as if the keys will always be encoded as strings, but as we really don't have a
	// way of knowing ahead of time if the codecs will produce string keys or not, we check every time and cancel the
	// encoding if we find a non-string key
	//
	// Once canceled, then proper action can be taken depending on the target (i.e. either erroring, or encoding
	// as an associative array if the target is a table)
	//
	// It's not ideal, but it is what it is. In the future, we may consider some sort of solution like this:
	//
	// we encode as a normal map, but we leave in some extra spaces to hot swap the `{}:`s for `[],`s:
	// '{ "foo": 1 , "bar": 2 , 3'   <- non string key detected, hot swap time:
	// '[["foo", 1],["bar", 2],[3'   <- we only need to keep track of the comma indexes to swap the correct characters
	stringKeyEncoder := func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		dst, err := encodeKey(ctx, dst, p)
		if err != nil {
			return dst, err
		}

		if dst[len(dst)-1] != '"' {
			if dst = skipWSRev(dst); dst[len(dst)-1] != '"' { // probably near-zero performance gain but eh /shrug
				return dst, rollback{}
			}
		}

		return dst, nil
	}

	encodeObjectMap := mkNormalMapEncoder(t, kt, stringKeyEncoder, encodeValue, mkIter)
	encodeArrayMap := mkGenericMapEncoder(t, kt, encodeKey, encodeValue, '[', ']', ',', mkIter)

	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		dst, err := encodeObjectMap(ctx, dst, p)
		if _, ok := err.(rollback); !ok {
			return dst, err
		}

		if ctx.Target == TargetTable {
			return encodeArrayMap(ctx, dst, p)
		}

		return dst, &UnsupportedValueError{Msg: "maps with non-string keys are only supported for tables"}
	}
}

func mkMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, maker mapMaker) decoder {
	decodeObjectMap := mkNormalMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, maker)
	decodeArrayMap := mkGenericMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, '[', ']', ',', maker)

	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		// we'll just delegate the `null` case to the normal decoder
		// since it can handle it, and we don't want to duplicate that logic
		if len(src) > 0 && (src[0] == '{' || src[0] == 'n') {
			return decodeObjectMap(ctx, src, p)
		}

		if ctx.Target == TargetTable {
			return decodeArrayMap(ctx, src, p)
		}

		return src, ctx.unmarshalTypeError(src, t)
	}
}

func mkGenericMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder, open, close, sep byte, mkIter mkMapIter) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		m := reflect.NewAt(t, p).Elem()

		if mapIsNil(m) {
			return append(dst, "null"...), nil
		}

		iter := mkIter(ctx, m)

		if iter.IsEmpty() {
			if open == '[' && encodeValue == nil {
				return append(dst, "[]"...), nil
			}
			return append(dst, "{}"...), nil // empty object on purpose, even for assoc arrays b/c of a data api bug
		}

		start := len(dst)
		toArray := open == '[' && encodeValue != nil

		first := true
		var err error

		dst = append(dst, open)

		for iter.Next() {
			if !first {
				dst = append(dst, ',')
			}
			first = false

			if toArray {
				dst = append(dst, open)
			}

			var next []byte
			if next, err = encodeKey(ctx, dst, valuePtr(iter.Key())); err != nil {
				return dst[:start], wrapPath(err, "key")
			}
			dst = next

			if encodeValue != nil {
				dst = append(dst, sep)

				if kt.Kind() == reflect.String {
					ctx.fieldHint = extractFieldHint(iter.Key().String())
				}

				if next, err = encodeValue(ctx, dst, valuePtr(iter.Value())); err != nil {
					return dst[:start], wrapPath(err, fmt.Sprintf("[%v]", iter.Key().Interface()))
				}
				dst = next
			}

			if toArray {
				dst = append(dst, close)
			}
		}

		return append(dst, close), nil
	}
}

func mkGenericMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, open, close, sep byte, maker mapMaker) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if b, ok := consumeNull(src); ok {
			*(*unsafe.Pointer)(p) = nil
			return b, nil
		}

		if len(src) < 2 || src[0] != open {
			msg := "expected '{' at the start of an object"
			if open == '[' {
				msg = "expected '[' at the start of an array"
			}
			return src, ctx.unmarshalTypeErrorWrap(src, t, ctx.syntaxError(src, msg))
		}

		m := reflect.NewAt(t, p).Elem()
		if mapIsNil(m) {
			m = maker.makeMap()
		}

		k := reflect.New(kt).Elem()
		v := reflect.New(vt).Elem()
		kptr, vptr := valuePtr(k), valuePtr(v)

		fromArray := open == '[' && decodeValue != nil

		src = src[1:]
		for i := 0; ; i++ {
			src = skipWS(src)

			if len(src) != 0 && src[0] == close {
				if m.Kind() == reflect.Map {
					*(*unsafe.Pointer)(p) = m.UnsafePointer()
				} else {
					*(*unsafe.Pointer)(p) = *(*unsafe.Pointer)(valuePtr(m))
				}
				return src[1:], nil
			}

			if i != 0 {
				if len(src) == 0 {
					return src, ctx.syntaxError(src, "unexpected end of JSON")
				}
				if src[0] != ',' {
					return src, ctx.syntaxError(src, fmt.Sprintf("expected ',' but found '%c'", src[0]))
				}
				src = skipWS(src[1:])
			}

			k.Set(kz)
			v.Set(vz)

			if fromArray {
				if len(src) == 0 || src[0] != '[' {
					return src, ctx.syntaxError(src, "expected '[' for table entry")
				}
				src = skipWS(src[1:])
			}

			srcAfter, err := decodeKey(ctx, src, kptr)
			if err != nil {
				return srcAfter, wrapPath(err, "key")
			}
			src = skipWS(srcAfter)

			if decodeValue != nil {
				if len(src) == 0 || src[0] != sep {
					return src, ctx.syntaxError(src, fmt.Sprintf("expected '%c' after key", sep))
				}
				src = skipWS(src[1:])

				if kt.Kind() == reflect.String {
					ctx.fieldHint = extractFieldHint(k.String())
				}

				srcAfter, err := decodeValue(ctx, src, vptr)
				if err != nil {
					return srcAfter, wrapPath(err, fmt.Sprintf("[%v]", k.Interface()))
				}

				src = skipWS(srcAfter)
			}

			if fromArray {
				if len(src) == 0 || src[0] != ']' {
					return src, ctx.syntaxError(src, "expected ']' after table entry")
				}
				src = skipWS(src[1:])
			}

			maker.setMap(m, k, v)
		}
	}
}

func mkNormalMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder, mkIter mkMapIter) encoder {
	return mkGenericMapEncoder(t, kt, encodeKey, encodeValue, '{', '}', ':', mkIter)
}

func mkNormalMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, maker mapMaker) decoder {
	return mkGenericMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, '{', '}', ':', maker)
}

// Iterator

type mkMapIter = func(ctx EncodeCtx, m reflect.Value) mapIter

type mapIterType int

const (
	mapUnsortedIter mapIterType = iota
	mapSortedIter
	linkedMapIter
	sortedMapIter
)

type mapIter struct {
	m           reflect.Value
	iter        *reflect.MapIter
	currentNode reflect.Value
	keys        []reflect.Value
	index       int
	iterType    mapIterType
}

func newMapIterMakerFromMap(t reflect.Type) mkMapIter {
	return func(ctx EncodeCtx, m reflect.Value) mapIter {
		wrapper := mapIter{m: m, index: -1, iterType: mapUnsortedIter}

		if ctx.Flags&SortMapKeys != 0 {
			wrapper.keys = m.MapKeys()

			if len(wrapper.keys) > 1 {
				cmp := datatypes.ComparatorFor(t.Key())

				sort.Slice(wrapper.keys, func(i, j int) bool {
					return cmp(valuePtr(wrapper.keys[i]), valuePtr(wrapper.keys[j])) < 0
				})
			}

			wrapper.iterType = mapSortedIter
			return wrapper
		}

		wrapper.iter = m.MapRange()
		return wrapper
	}
}

func newMapIterFromLinkedMap(_ EncodeCtx, m reflect.Value) mapIter {
	return mapIter{
		m:           m,
		index:       -1,
		iterType:    linkedMapIter,
		currentNode: m.FieldByIndex([]int{0, 3}),
	}
}

func newMapIterFromSortedMap(_ EncodeCtx, m reflect.Value) mapIter {
	// SortedMap.Field(0) is *sortedMap[K,V]
	// sortedMap fields: 0=kType, 1=vType, 2=cmp, 3=head, 4=len
	// head is the sentinel; first real node is head.next[0] (node.Field(2).Index(0))
	firstNode := m.FieldByIndex([]int{0, 3}).Elem().Field(2).Index(0)
	return mapIter{
		m:           m,
		index:       -1,
		iterType:    sortedMapIter,
		currentNode: firstNode,
	}
}

func (u *mapIter) Next() bool {
	switch u.iterType {
	case mapSortedIter:
		u.index++
		return u.index < len(u.keys)
	case linkedMapIter:
		if u.index == -1 {
			u.index = 0
			return !u.currentNode.IsNil()
		}

		if u.currentNode.IsNil() {
			return false
		}

		//u.currentNode = u.currentNode.Elem().FieldByName("next")
		u.currentNode = u.currentNode.Elem().Field(3)

		return !u.currentNode.IsNil()
	case sortedMapIter:
		if u.index == -1 {
			u.index = 0
			return !u.currentNode.IsNil()
		}

		if u.currentNode.IsNil() {
			return false
		}

		u.currentNode = u.currentNode.Elem().Field(2).Index(0) // next[0]

		return !u.currentNode.IsNil()
	default:
		return u.iter.Next()
	}
}

func (u *mapIter) Key() reflect.Value {
	switch u.iterType {
	case mapSortedIter:
		return u.keys[u.index]
	case linkedMapIter:
		//return u.currentNode.Elem().FieldByName("key")
		return u.currentNode.Elem().Field(0)
	case sortedMapIter:
		return u.currentNode.Elem().Field(0) // key
	default:
		return u.iter.Key()
	}
}

func (u *mapIter) Value() reflect.Value {
	switch u.iterType {
	case mapSortedIter:
		return u.m.MapIndex(u.keys[u.index])
	case linkedMapIter:
		//return u.currentNode.Elem().FieldByName("value")
		return u.currentNode.Elem().Field(1)
	case sortedMapIter:
		return u.currentNode.Elem().Field(1) // value
	default:
		return u.iter.Value()
	}
}

func (u *mapIter) IsEmpty() bool {
	switch u.iterType {
	case mapSortedIter:
		return len(u.keys) == 0
	case linkedMapIter:
		return u.currentNode.IsNil()
	case sortedMapIter:
		return u.currentNode.IsNil()
	default:
		return u.m.Len() == 0
	}
}

// Setter

type mapMaker struct {
	makeMap func() reflect.Value
	mapLen  func(m reflect.Value) int
	setMap  func(m, k, v reflect.Value)
}

func mkNativeMapMaker(t reflect.Type) mapMaker {
	return mapMaker{
		makeMap: func() reflect.Value {
			return reflect.MakeMap(t)
		},
		setMap: func(m, k, v reflect.Value) {
			m.SetMapIndex(k, v)
		},
	}
}

type linkedMapFastSetter interface {
	SetAny(k, v any) bool
}

func mkLinkedMapMaker(t reflect.Type) mapMaker {
	implType := t.Field(0).Type.Elem() // linkedMap[K,V]
	dataType := implType.Field(2).Type // map[K]*LinkedMapNode[K,V]
	dataOff := implType.Field(2).Offset

	return mapMaker{
		makeMap: func() reflect.Value {
			m := reflect.New(t).Elem()       // LinkedMap[K,V]
			implPtr := reflect.New(implType) // *linkedMap[K,V]
			implAddr := implPtr.UnsafePointer()

			// set data field (field 2)
			dataAddr := unsafe.Pointer(uintptr(implAddr) + dataOff)
			*(*unsafe.Pointer)(dataAddr) = reflect.MakeMap(dataType).UnsafePointer()

			// wire *linkedMap into LinkedMap
			*(*unsafe.Pointer)(valuePtr(m)) = implAddr

			return m
		},
		setMap: func(m, k, v reflect.Value) {
			setter := m.Interface().(linkedMapFastSetter)
			setter.SetAny(k.Interface(), v.Interface())
		},
	}
}

func mkSortedMapMaker(t reflect.Type) mapMaker {
	implType := t.Field(0).Type.Elem() // sortedMap[K,V]
	kt := implType.Field(0).Type       // [0]K
	cmpOff := implType.Field(2).Offset
	headOff := implType.Field(3).Offset
	headType := implType.Field(3).Type.Elem()

	cmp := datatypes.ComparatorFor(kt.Elem())

	return mapMaker{
		makeMap: func() reflect.Value {
			m := reflect.New(t).Elem()       // SortedMap[K,V]
			implPtr := reflect.New(implType) // *sortedMap[K,V]
			implAddr := implPtr.UnsafePointer()

			// set comparator (field 2)
			cmpAddr := unsafe.Pointer(uintptr(implAddr) + cmpOff)
			*(*datatypes.Comparator)(cmpAddr) = cmp

			// set sentinel head (field 3)
			headAddr := unsafe.Pointer(uintptr(implAddr) + headOff)
			*(*unsafe.Pointer)(headAddr) = reflect.New(headType).UnsafePointer()

			// wire *sortedMap into SortedMap
			*(*unsafe.Pointer)(valuePtr(m)) = implAddr

			return m
		},
		setMap: func(m, k, v reflect.Value) {
			setter := m.Interface().(linkedMapFastSetter)
			setter.SetAny(k.Interface(), v.Interface())
		},
	}
}

// Utils

func mapIsNil(m reflect.Value) bool {
	if m.Kind() == reflect.Map {
		return m.IsNil()
	}
	return *(*unsafe.Pointer)(valuePtr(m)) == nil
}
