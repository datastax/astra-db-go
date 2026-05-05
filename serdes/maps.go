package serdes

import (
	"fmt"
	"reflect"
	"sort"
	"unsafe"
)

// Map serdes is complex enough to warrant its own file, as we're allowing for a Cartesian product of features:
// - Serdes w/ a table vs non-table target
// - Serdes w/ a native Go map vs an OrderedMap[K, V]
// - Sorted vs unsorted map encoding (for native Go maps)
//
// The below is not the cleanest code, but it aims to be fairly performant while still being maintainable
//
// The main idea is that we have a generic map encoder/decoder that takes in the logic for iterating over the map and
// for making/setting the map, and then we have thin wrappers around it for each of the 4 combinations of features

// Serdes

func mkMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	mkIter := newMapIterMakerFromMap(t, true) // TODO make trySort an optional flag

	kt := t.Key()
	vt := t.Elem()

	return mkGenericMapCodec(ctx, t, kt, vt, seen, mkIter, mkNativeMapMaker(t))
}

func mkOrderedMapCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	mkIter := newMapIterFromOrderedMap

	kt, _ := t.FieldByName("kType")
	vt, _ := t.FieldByName("vType")

	return mkGenericMapCodec(ctx, t, kt.Type.Elem(), vt.Type.Elem(), seen, mkIter, mkOrderedMapMaker(t))
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
	stringKeyEncoder := func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
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

	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		dst, err := encodeObjectMap(ctx, dst, p)
		if _, ok := err.(rollback); !ok {
			return dst, err
		}

		if ctx.target.kind == tableKind {
			return encodeArrayMap(ctx, dst, p)
		}

		return dst, fmt.Errorf("cannot have a map with non-string keys in tables")
	}
}

func mkMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, maker mapMaker) decoder {
	decodeObjectMap := mkNormalMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, maker)
	decodeArrayMap := mkGenericMapDecoder(t, kt, vt, kz, vz, decodeKey, decodeValue, '[', ']', ',', maker)

	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		// we'll just delegate the `null` case to the normal decoder
		// since it can handle it, and we don't want to duplicate that logic
		if len(src) > 0 && (src[0] == '{' || src[0] == 'n') {
			return decodeObjectMap(ctx, src, p)
		}

		if ctx.target.kind == tableKind {
			return decodeArrayMap(ctx, src, p)
		}

		return src, fmt.Errorf("expected a json object or null when parsing map")
	}
}

func mkGenericMapEncoder(t, kt reflect.Type, encodeKey, encodeValue encoder, open, close, sep byte, mkIter mkMapIter) encoder {
	return func(ctx encodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		m := reflect.NewAt(t, p).Elem()

		if mapIsNil(m) {
			return append(dst, "null"...), nil
		}

		iter := mkIter(m)

		if iter.IsEmpty() {
			return append(dst, "{}"...), nil
		}

		start := len(dst)
		toArray := open == '[' // && close == ']'

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

			if dst, err = encodeKey(ctx, dst, valuePtr(iter.Key())); err != nil {
				return dst[:start], err
			}

			dst = append(dst, sep)

			if kt.Kind() == reflect.String {
				ctx.fieldHint = extractFieldHint(iter.Key().String())
			}

			if dst, err = encodeValue(ctx, dst, valuePtr(iter.Value())); err != nil {
				return dst[:start], err
			}

			if toArray {
				dst = append(dst, close)
			}
		}

		return append(dst, close), nil
	}
}

func mkGenericMapDecoder(t, kt, vt reflect.Type, kz, vz reflect.Value, decodeKey, decodeValue decoder, open, close, sep byte, maker mapMaker) decoder {
	return func(ctx decodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		if b, ok := consumeNull(src); ok {
			*(*unsafe.Pointer)(p) = nil
			return b, nil
		}

		if len(src) < 2 || src[0] != open {
			return src, fmt.Errorf("expected '%c'", open)
		}

		m := reflect.NewAt(t, p).Elem()
		if mapIsNil(m) {
			m = maker.makeMap()
		}

		k := reflect.New(kt).Elem()
		v := reflect.New(vt).Elem()
		kptr, vptr := valuePtr(k), valuePtr(v)

		fromArray := open == '[' // && close == ']'

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
					return src, fmt.Errorf("unexpected end of JSON")
				}
				if src[0] != ',' {
					return src, fmt.Errorf("expected ',' but found '%c'", src[0])
				}
				src = skipWS(src[1:])
			}

			k.Set(kz)
			v.Set(vz)

			if fromArray {
				if len(src) == 0 || src[0] != '[' {
					return src, fmt.Errorf("expected '[' for table entry")
				}
				src = skipWS(src[1:])
			}

			var err error
			if src, err = decodeKey(ctx, src, kptr); err != nil {
				return src, err
			}
			src = skipWS(src)

			if len(src) == 0 || src[0] != sep {
				return src, fmt.Errorf("expected '%c' after key", sep)
			}
			src = skipWS(src[1:])

			if kt.Kind() == reflect.String {
				ctx.fieldHint = extractFieldHint(k.String())
			}

			if src, err = decodeValue(ctx, src, vptr); err != nil {
				return src, err
			}
			src = skipWS(src)

			if fromArray {
				if len(src) == 0 || src[0] != ']' {
					return src, fmt.Errorf("expected ']' after table entry")
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

type mkMapIter = func(m reflect.Value) mapIter

type mapIterType int

const (
	mapUnsortedIter mapIterType = iota
	mapSortedIter
	orderedMapIter
)

type mapIter struct {
	m           reflect.Value
	iter        *reflect.MapIter
	currentNode reflect.Value
	keys        []reflect.Value
	index       int
	iterType    mapIterType
}

type comparator = func(i, j reflect.Value) bool

func newMapIterMakerFromMap(t reflect.Type, trySort bool) func(m reflect.Value) mapIter {
	var cmp comparator
	if trySort {
		cmp = mkComparator(t.Key().Kind())
	}

	return func(m reflect.Value) mapIter {
		wrapper := mapIter{m: m, index: -1, iterType: mapUnsortedIter}

		if trySort {
			wrapper.keys = m.MapKeys()

			if len(wrapper.keys) > 1 {
				sort.Slice(wrapper.keys, func(i, j int) bool {
					return cmp(wrapper.keys[i], wrapper.keys[j])
				})
			}

			wrapper.iterType = mapSortedIter
			return wrapper
		}

		wrapper.iter = m.MapRange()
		return wrapper
	}
}

func newMapIterFromOrderedMap(m reflect.Value) mapIter {
	return mapIter{
		m:           m,
		index:       -1,
		iterType:    orderedMapIter,
		currentNode: m.FieldByIndex([]int{0, 3}),
	}
}

// mkComparator returns the logic once so the loop doesn't have to switch
// TODO any more comparators???
func mkComparator(k reflect.Kind) comparator {
	switch {
	case k == reflect.String:
		return func(i, j reflect.Value) bool { return i.String() < j.String() }
	case k >= reflect.Int && k <= reflect.Int64:
		return func(i, j reflect.Value) bool { return i.Int() < j.Int() }
	case k >= reflect.Uint && k <= reflect.Uintptr:
		return func(i, j reflect.Value) bool { return i.Uint() < j.Uint() }
	case k == reflect.Float32 || k == reflect.Float64:
		return func(i, j reflect.Value) bool { return i.Float() < j.Float() }
	default:
		return func(i, j reflect.Value) bool { return false }
	}
}

func (u *mapIter) Next() bool {
	switch u.iterType {
	case mapSortedIter:
		u.index++
		return u.index < len(u.keys)
	case orderedMapIter:
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
	default:
		return u.iter.Next()
	}
}

func (u *mapIter) Key() reflect.Value {
	switch u.iterType {
	case mapSortedIter:
		return u.keys[u.index]
	case orderedMapIter:
		//return u.currentNode.Elem().FieldByName("key")
		return u.currentNode.Elem().Field(0)
	default:
		return u.iter.Key()
	}
}

func (u *mapIter) Value() reflect.Value {
	switch u.iterType {
	case mapSortedIter:
		return u.m.MapIndex(u.keys[u.index])
	case orderedMapIter:
		//return u.currentNode.Elem().FieldByName("value")
		return u.currentNode.Elem().Field(1)
	default:
		return u.iter.Value()
	}
}

func (u *mapIter) IsEmpty() bool {
	switch u.iterType {
	case mapSortedIter:
		return len(u.keys) == 0
	case orderedMapIter:
		return u.currentNode.IsNil()
	default:
		return !u.iter.Next()
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

type orderedMapFastSetter interface {
	SetAny(k, v any) bool
}

func mkOrderedMapMaker(t reflect.Type) mapMaker {
	implType := t.Field(0).Type.Elem()
	dataType := implType.Field(2).Type

	return mapMaker{
		makeMap: func() reflect.Value {
			// allocates OrderedMap
			m := reflect.New(t).Elem()

			// allocates orderedMap
			implPtr := reflect.New(implType)

			// allocates the backing map
			dataMap := reflect.MakeMap(dataType)

			// sets the data map in orderedMap
			*(*unsafe.Pointer)(implPtr.UnsafePointer()) = dataMap.UnsafePointer()

			// sets *orderedMap in OrderedMap
			*(*unsafe.Pointer)(valuePtr(m)) = implPtr.UnsafePointer()

			return m
		},
		setMap: func(m, k, v reflect.Value) {
			setter := m.Interface().(orderedMapFastSetter)
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
