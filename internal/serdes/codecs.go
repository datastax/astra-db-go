package serdes

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"unsafe"
)

type codec struct {
	encode encoder
	decode decoder
}

type AstraCodec interface {
	FromAstraValue(v any) error
	ToAstraValue() any
}

var (
	typeCodecs atomic.Pointer[map[unsafe.Pointer]codec]
	kindCodecs [reflect.String + 1]codec
)

//go:generate go run -modfile=../../tools/gen-serdes/go.mod ../../tools/gen-serdes/main.go

func init() {
	kindCodecs[reflect.Bool] = codec{encodeBoolKind, decodeBoolKind}
	kindCodecs[reflect.String] = codec{encodeStringKind, decodeStringKind}
}

var nilCodec = codec{
	encodeNull,
	decodeNull,
}

type seenStructs = map[reflect.Type]*structInfo

func resolveCodecCaching(t reflect.Type, seen seenStructs) codec {
	cache := cacheLoad()
	if c, ok := cache[typeid(t)]; ok {
		return c
	}

	codec, shouldCache := resolveCodec(t, seen, t.Kind() == reflect.Ptr)
	if shouldCache {
		return cacheSet(cacheLoad(), t, codec)
	}
	return codec
}

func resolveCodec(t reflect.Type, seen seenStructs, canAddr bool) (codec, bool) {
	if t == nil || t == nilType {
		return nilCodec, false
	}

	if t.Implements(astraCodecType) || (t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(astraCodecType)) {
		return mkCustomCodec(t), true
	}

	k := t.Kind()
	if int(k) < len(kindCodecs) && kindCodecs[k].encode != nil {
		return kindCodecs[k], false
	}

	switch t {
	case uuidType:
		panic("uuid codec")
	case vectorType:
		panic("vector codec")
	case timeType:
		panic("time codec")
	case ipType:
		panic("ip codec")
	}

	switch k {
	case reflect.Ptr:
		return mkPointerCodec(t, seen), true
	case reflect.Struct:
		return mkStructCodec(t, seen, canAddr), true
	case reflect.Map:
		return mkMapCodec(t, seen), true
	case reflect.Slice:
		return mkSliceCodec(t, seen), true
	case reflect.Array:
		return mkArrayCodec(t, seen, canAddr), true
	case reflect.Interface:
		return mkInterfaceCodec(), true
	default:
		panic("unsupported type: " + t.String())
	}
}

func cacheLoad() map[unsafe.Pointer]codec {
	p := typeCodecs.Load()
	if p == nil {
		return nil
	}
	return *p
}

func cacheSet(oldCache map[unsafe.Pointer]codec, t reflect.Type, c codec) codec {
	if inlined(t) {
		c.encode = encodeInlined(c.encode)
	}

	newCache := make(map[unsafe.Pointer]codec, len(oldCache)+1)
	maps.Copy(newCache, oldCache)
	newCache[typeid(t)] = c
	typeCodecs.Store(&newCache)
	return c
}

func mkErroredCodec(err error) codec {
	return codec{
		encodeError(err),
		decodeError(err),
	}
}

func mkCustomCodec(t reflect.Type) codec {
	return codec{encodeCustom(t), decodeCustom(t)}
}

func mkPointerCodec(t reflect.Type, seen seenStructs) codec {
	el := t.Elem()
	c, _ := resolveCodec(el, seen, true)

	return codec{
		encodePointer(c.encode),
		decodePointer(c.decode, el),
	}
}

func mkStructCodec(t reflect.Type, seen seenStructs, canAddr bool) codec {
	info, err := compileStructInfo(t, seen, canAddr)

	if err != nil {
		return mkErroredCodec(fmt.Errorf("failed to compile struct %s: %w", t.String(), err))
	}

	return codec{
		encodeStruct(info),
		decodeStruct(info),
	}
}

func mkEmbeddedStructPointerCodec(t reflect.Type, unexported bool, offset uintptr, field codec) codec {
	return codec{
		encodeEmbeddedStructPointer(field.encode),
		decodeEmbeddedStructPointer(t, unexported, offset, field.decode),
	}
}

func mkMapCodec(t reflect.Type, seen seenStructs) codec {
	kt := t.Key()
	vt := t.Elem()

	//kc := resolveCodecCaching(kt, seen, false)
	kc := kindCodecs[reflect.String]
	vc, _ := resolveCodec(vt, seen, false)

	kz := reflect.Zero(kt)
	vz := reflect.Zero(vt)

	if inlined(vt) {
		vc.encode = encodeInlined(vc.encode)
	}

	return codec{
		encodeMap(t, kc.encode, vc.encode),
		decodeMap(kt, vt, kz, vz, kc.decode, vc.decode),
	}
}

func mkSliceCodec(t reflect.Type, seen seenStructs) codec {
	elem := t.Elem()
	c, _ := resolveCodec(elem, seen, true)
	size := elem.Size()

	return codec{
		encodeSlice(size, c.encode),
		decodeSlice(size, t, c.decode),
	}
}

func mkArrayCodec(t reflect.Type, seen seenStructs, canAddr bool) codec {
	elem := t.Elem()
	n := t.Len()
	size := elem.Size()

	c, _ := resolveCodec(elem, seen, canAddr)

	return codec{
		encodeArray(n, size, c.encode),
		decodeArray(n, size, t, c.decode),
	}
}

func mkInterfaceCodec() codec {
	return codec{
		encodeInterface,
		decodeInterface,
	}
}

type structInfo struct {
	fields  []fieldInfo
	offsets map[string]int
	typ     reflect.Type
}

type fieldInfo struct {
	prefix []byte
	typ    reflect.Type
	codec  codec
	offset uintptr
	ord    int
	meta   jsonMeta
}

type jsonMeta struct {
	name      string
	omitempty bool
	ignored   bool
	tagged    bool
}

func compileStructInfo(t reflect.Type, seen seenStructs, canAddr bool) (*structInfo, error) {
	if info, ok := seen[t]; ok {
		return info, nil
	}

	info := &structInfo{
		typ:     t,
		offsets: make(map[string]int, t.NumField()),
	}
	seen[t] = info

	fields, err := compileStructFields(t, seen, canAddr)
	info.fields = fields

	if err != nil {
		delete(seen, t)
		return nil, err
	}

	for i := range fields {
		f := &fields[i]
		info.offsets[f.meta.name] = i
	}

	return info, nil
}

func compileStructFields(t reflect.Type, seen seenStructs, canAddr bool) ([]fieldInfo, error) {
	type embeddedField struct {
		ord        int
		offset     uintptr
		pointer    bool
		unexported bool
		subtype    *structInfo
		subfield   *fieldInfo
	}

	topLevelNames := make(map[string]struct{})
	ambiguousNames := make(map[string]int)
	ambiguousTags := make(map[string]int)

	fields := make([]fieldInfo, 0, t.NumField())
	embeddedFields := make([]embeddedField, 0, 5)

	for i := range t.NumField() {
		f := t.Field(i)

		var (
			embedded   = f.Anonymous
			unexported = len(f.PkgPath) != 0
		)

		if unexported && !embedded { // unexported
			continue
		}

		meta := parseJsonMeta(f)

		if meta.ignored {
			continue
		}

		if embedded && !meta.tagged { // embeddedFields
			typ := f.Type
			ptr := f.Type.Kind() == reflect.Ptr

			if ptr {
				typ = typ.Elem()
			}

			if typ.Kind() == reflect.Struct {
				subtype, err := compileStructInfo(typ, seen, canAddr)
				if err != nil {
					return nil, err
				}

				for j := range subtype.fields {
					embeddedFields = append(embeddedFields, embeddedField{
						ord:        i<<32 | j,
						offset:     f.Offset,
						pointer:    ptr,
						unexported: unexported,
						subtype:    subtype,
						subfield:   &subtype.fields[j],
					})
				}

				continue
			}

			if unexported { // ignore unexported non-struct types
				continue
			}
		}

		c, _ := resolveCodec(f.Type, seen, canAddr)

		fields = append(fields, fieldInfo{
			codec:  c,
			offset: f.Offset,
			meta:   meta,
			ord:    i << 32,
			typ:    f.Type,
		})

		// Seed the counters so embedded fields know they are secondary
		topLevelNames[meta.name] = struct{}{}
		ambiguousNames[meta.name]++
		ambiguousTags[meta.name]++
	}

	for _, embfield := range embeddedFields {
		ambiguousNames[embfield.subfield.meta.name]++
		if embfield.subfield.meta.tagged {
			ambiguousTags[embfield.subfield.meta.name]++
		}
	}

	for _, embfield := range embeddedFields {
		subfield := *embfield.subfield

		switch resolveEmbeddedAmbiguity(subfield.meta, topLevelNames, ambiguousNames, ambiguousTags) {
		case shadowed:
			continue
		case ambiguous:
			return nil, fmt.Errorf("unresolvable ambiguity for field %q in struct %s", subfield.meta.name, t.String())
		case unambiguous:
			// all good
		}

		if embfield.pointer {
			subfield.codec = mkEmbeddedStructPointerCodec(embfield.subtype.typ, embfield.unexported, subfield.offset, subfield.codec)
			subfield.offset = embfield.offset
		} else {
			subfield.offset += embfield.offset
		}

		// To prevent dominant flags more than one level below the embeddedFields one.
		subfield.meta.tagged = false

		// To ensure the order of the fields in the output is the same is in the struct type.
		subfield.ord = embfield.ord

		fields = append(fields, subfield)
	}

	for i := range fields {
		fields[i].prefix = []byte(`,"` + fields[i].meta.name + `":`)
	}

	sort.Slice(fields, func(i, j int) bool { return fields[i].ord < fields[j].ord })
	return fields, nil
}

func parseJsonMeta(f reflect.StructField) jsonMeta {
	var info jsonMeta
	info.name = f.Name

	if parts := strings.Split(f.Tag.Get("json"), ","); len(parts) != 0 {
		if len(parts[0]) != 0 {
			info.name = parts[0]
			info.tagged = true
		}

		if info.name == "-" && len(parts) == 1 {
			info.ignored = true
			return info
		}

		for _, opt := range parts[1:] { // TODO do we want to somehow warn if 'string' is used as an option since I don't want to support it?
			switch opt {
			case "omitempty":
				info.omitempty = true
			}
		}
	}

	return info
}

type embeddedAmbiguity int

const (
	unambiguous embeddedAmbiguity = iota
	shadowed
	ambiguous
)

func resolveEmbeddedAmbiguity(meta jsonMeta, topLevelNames map[string]struct{}, nameCounts, tagCounts map[string]int) embeddedAmbiguity {
	// 1. Shadowing: If a top-level field exists, the embedded one NEVER wins.
	if _, exists := topLevelNames[meta.name]; exists {
		return shadowed
	}

	// 2. No Collision: If this name only appears once in all embedded structs, it's the winner.
	if nameCounts[meta.name] == 1 {
		return unambiguous
	}

	// 3. Tag Dominance: If there are multiple fields with this name,
	// a field wins ONLY if it is the ONLY one with an explicit tag.
	if tagCounts[meta.name] == 1 && meta.tagged {
		return unambiguous
	}

	// 4. Ambiguity: Multiple tags or zero tags with the same name.
	if tagCounts[meta.name] != 1 {
		return ambiguous
	}

	return shadowed
}
