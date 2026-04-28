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
	typeCodecs atomic.Pointer[[3]map[unsafe.Pointer]codec]
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

type codecCtx struct {
	target    Target
	fieldHint fieldHint
}

type seenStructs = map[reflect.Type]*structInfo

func resolveCodecCaching(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	tid := typePtr(t)
	cache := cacheLoad()

	if c, ok := cache[0][tid]; ok {
		return c
	}

	if c, ok := cache[ctx.target.kind][tid]; ok {
		return c
	}

	codec, purity := resolveCodec(ctx, t, seen, t.Kind() == reflect.Ptr)

	return cacheSet(cache, t, int(purity)*int(ctx.target.kind), codec)
}

func resolveCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (c codec, p purity) {
	if t == nil || t == nilType {
		return nilCodec, pure
	}

	if t.Implements(astraCodecType) || (t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(astraCodecType)) {
		return mkCustomCodec(t), pure
	}

	k := t.Kind()
	if int(k) < len(kindCodecs) && kindCodecs[k].encode != nil {
		return kindCodecs[k], pure
	}

	if c, ok := ctx.target.typeOverrides[typePtr(t)]; ok {
		return c, impure
	}

	if mkC, ok := ctx.target.kindOverrides[k]; ok {
		return mkC(ctx, t, seen, canAddr), impure
	}

	switch t {
	case rawMessageType:
		return codec{encodeRawMessage, decodeRawMessage}, pure
	case vectorType:
		return codec{encodeVector, decodeVector}, pure
	}

	if c.encode != nil {
		return
	}

	switch k {
	case reflect.Ptr:
		c, p = mkPointerCodec(ctx, t, seen)
	case reflect.Struct:
		c, p = mkStructCodec(ctx, t, seen, canAddr)
	case reflect.Slice:
		c, p = mkSliceCodec(ctx, t, seen)
	case reflect.Array:
		c, p = mkArrayCodec(ctx, t, seen, canAddr)
	case reflect.Interface:
		c, p = mkInterfaceCodec(), pure
	default:
		panic("unsupported type: " + t.String())
	}

	return
}

func cacheLoad() [3]map[unsafe.Pointer]codec {
	p := typeCodecs.Load()
	if p == nil {
		return [3]map[unsafe.Pointer]codec{}
	}
	return *p
}

func cacheSet(oldCache [3]map[unsafe.Pointer]codec, t reflect.Type, i int, c codec) codec {
	if inlined(t) {
		c.encode = encodeInlined(c.encode)
	}

	newCacheArray := oldCache

	oldMap := newCacheArray[i]
	newMap := make(map[unsafe.Pointer]codec, len(oldMap)+1)
	maps.Copy(newMap, oldMap)
	newMap[typePtr(t)] = c

	newCacheArray[i] = newMap
	typeCodecs.Store(&newCacheArray)

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

func mkPointerCodec(ctx codecCtx, t reflect.Type, seen seenStructs) (codec, purity) {
	el := t.Elem()
	c, p := resolveCodec(ctx, el, seen, true)

	return codec{
		encodePointer(c.encode),
		decodePointer(c.decode, el),
	}, p
}

func mkStructCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (codec, purity) {
	info, err := compileStructInfo(ctx, t, seen, canAddr)

	if err != nil {
		return mkErroredCodec(fmt.Errorf("failed to compile struct %s: %w", t.String(), err)), pure
	}

	return codec{
		encodeStruct(info),
		decodeStruct(info),
	}, info.purity
}

func mkEmbeddedStructPointerCodec(t reflect.Type, unexported bool, offset uintptr, field codec) codec {
	return codec{
		encodeEmbeddedStructPointer(field.encode),
		decodeEmbeddedStructPointer(t, unexported, offset, field.decode),
	}
}

func mkSliceCodec(ctx codecCtx, t reflect.Type, seen seenStructs) (codec, purity) {
	elem := t.Elem()
	c, p := resolveCodec(ctx, elem, seen, true)
	size := alignedSize(elem)

	return codec{
		encodeSlice(size, c.encode),
		decodeSlice(size, t, c.decode),
	}, p
}

func mkArrayCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (codec, purity) {
	elem := t.Elem()
	size := alignedSize(elem)
	c, p := resolveCodec(ctx, elem, seen, canAddr)
	n := t.Len()

	return codec{
		encodeArray(n, size, c.encode),
		decodeArray(n, size, c.decode),
	}, p
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
	purity  purity
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

func compileStructInfo(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (*structInfo, error) {
	if info, ok := seen[t]; ok {
		return info, nil
	}

	info := &structInfo{
		typ:     t,
		offsets: make(map[string]int, t.NumField()),
		purity:  pure,
	}
	seen[t] = info

	fields, p, err := compileStructFields(ctx, t, seen, canAddr)
	info.fields = fields
	info.purity = p

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

func compileStructFields(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) ([]fieldInfo, purity, error) {
	overallPurity := pure
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

		if unexported && !embedded {
			continue
		}

		meta := parseJsonMeta(f)

		if meta.ignored {
			continue
		}

		if embedded && !meta.tagged { // embedded w/out a tagged name
			typ := f.Type
			ptr := f.Type.Kind() == reflect.Ptr

			if ptr {
				typ = typ.Elem()
			}

			if typ.Kind() == reflect.Struct {
				subtype, err := compileStructInfo(ctx, typ, seen, canAddr)
				if err != nil {
					return nil, pure, err
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

				overallPurity |= subtype.purity
				continue
			}

			if unexported { // ignore unexported non-struct types
				continue
			}
		}

		c, p := resolveCodec(ctx, f.Type, seen, canAddr)
		overallPurity |= p

		fields = append(fields, fieldInfo{
			codec:  c,
			offset: f.Offset,
			meta:   meta,
			ord:    i << 32,
			typ:    f.Type,
		})

		// seeds the counters so embedded fields know they are secondary
		topLevelNames[meta.name] = struct{}{}
		ambiguousNames[meta.name]++
		ambiguousTags[meta.name]++
	}

	// first pass to count the number of fields w/ each name so we can resolve ambiguities in the next pass
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
			// TODO this is allowed w/ normal json ser/des but I'm tempted to error for correctness
			return nil, pure, fmt.Errorf("unresolvable ambiguity for fieldHint %q in struct %s", subfield.meta.name, t.String())
		case unambiguous:
			// all good
		}

		if embfield.pointer {
			subfield.codec = mkEmbeddedStructPointerCodec(embfield.subtype.typ, embfield.unexported, subfield.offset, subfield.codec)
			subfield.offset = embfield.offset
		} else {
			subfield.offset += embfield.offset
		}

		// prevents dominant flags more than one level below the embedded one
		subfield.meta.tagged = false

		// ensures order of the fields is the same is in the struct type
		subfield.ord = embfield.ord

		fields = append(fields, subfield)
	}

	for i := range fields {
		fields[i].prefix = []byte(`,"` + fields[i].meta.name + `":`)
	}

	// TODO:
	// I'm just sorting because it's cheap and easy (since only called when building the codec)
	// That being said it doesn't really matter for the output so I'm fine to remove it...
	sort.Slice(fields, func(i, j int) bool { return fields[i].ord < fields[j].ord })

	return fields, overallPurity, nil
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
	if _, exists := topLevelNames[meta.name]; exists {
		return shadowed // top level fieldHint with the same name exists so ignore this embedded fieldHint
	}

	if nameCounts[meta.name] == 1 {
		return unambiguous // no collisions so all good to go
	}

	if tagCounts[meta.name] == 1 && meta.tagged {
		return unambiguous // multiple fields with the same name, so the fieldHint with the tag wins
	}

	if tagCounts[meta.name] != 1 {
		return ambiguous // zero or multiple tags w/ the same name so we can't resolve anything
	}

	return shadowed // fieldHint collided and lost to a tagged fieldHint.
}
