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

var (
	typeCodecs atomic.Pointer[map[unsafe.Pointer]codec] // TODO may be able to just reuse the same cache for all targets and let the resolution be at execution time?
	kindCodecs [reflect.String + 1]codec
)

//go:generate go run -modfile=../tools/gen-serdes/go.mod ../tools/gen-serdes/main.go

func init() {
	kindCodecs[reflect.Bool] = codec{boolEncoder, boolDecoder}
	kindCodecs[reflect.String] = codec{stringEncoder, stringDecoder}
}

var nilCodec = codec{
	nullEncoder,
	nullDecoder,
}

type codecCtx struct {
	fieldHint fieldHint
}

type seenStructs = map[reflect.Type]*structInfo

func resolveCodecCaching(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	tid := typePtr(t)
	cache := cacheLoad()

	if c, ok := cache[tid]; ok {
		return c
	}

	codec := resolveCodec(ctx, t, seen, t.Kind() == reflect.Ptr)

	return cacheSet(cache, t, 0, codec)
}

func resolveCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (c codec) {
	if t == nil || t == nilType {
		return nilCodec
	}

	switch t {
	case rawMessageType:
		return codec{rawMessageEncoder, rawMessageDecoder}
	case vectorType:
		return codec{vectorEncoder, vectorDecoder}
	case bigIntType:
		return codec{bigIntEncoder, bigIntDecoder}
	case bigFloatType:
		return codec{bigFloatEncoder, bigFloatDecoder}
	case byteSliceType:
		return codec{binaryEncoder, binaryDecoder}
	case uuidType:
		return codec{uuidEncoder, uuidDecoder}
	case oidType:
		return codec{objectIdEncoder, objectIdDecoder}
	case dApiTimeType:
		return codec{timestampEncoder, timestampDecoder}
	}

	if c.encode != nil {
		return
	}

	k := t.Kind()

	if int(k) < len(kindCodecs) && kindCodecs[k].encode != nil {
		c = kindCodecs[k]
	}

	switch k {
	case reflect.Ptr:
		c = mkPointerCodec(ctx, t, seen)
	case reflect.Struct:
		c = mkStructCodec(ctx, t, seen, canAddr)
	case reflect.Slice:
		c = mkSliceCodec(ctx, t, seen)
	case reflect.Array:
		c = mkArrayCodec(ctx, t, seen, canAddr)
	case reflect.Map:
		c = mkMapCodec(ctx, t, seen)
	case reflect.Interface:
		c = codec{interfaceEncoder, interfaceDecoder}
	default:
		if c.encode == nil {
			c = mkErroredCodec(fmt.Errorf("unsupported type %s", t.String()))
		}
	}

	ptr := reflect.PointerTo(t)

	switch {
	case t.Implements(astraMarshalerType):
		c.encode = mkAstraMarshalerEncoder(t, false)
	case t.Implements(astraRawMarshalerType):
		c.encode = mkAstraRawMarshalerEncoder(t, false)
	case canAddr && ptr.Implements(astraMarshalerType):
		c.encode = mkAstraMarshalerEncoder(t, true)
	case canAddr && ptr.Implements(astraRawMarshalerType):
		c.encode = mkAstraRawMarshalerEncoder(t, true)
	}

	switch {
	case ptr.Implements(astraUnmarshalerType):
		c.decode = mkAstraUnmarshalerDecoder(t)
	case ptr.Implements(astraRawUnmarshalerType):
		c.decode = mkAstraRawUnmarshalerDecoder(t)
	}

	return
}

func cacheLoad() map[unsafe.Pointer]codec {
	p := typeCodecs.Load()
	if p == nil {
		return map[unsafe.Pointer]codec{}
	}
	return *p
}

func cacheSet(oldCache map[unsafe.Pointer]codec, t reflect.Type, i int, c codec) codec {
	if inlined(t) {
		c.encode = mkInlineEncoder(c.encode)
	}

	newCache := make(map[unsafe.Pointer]codec, len(oldCache)+1)
	newCache[typePtr(t)] = c
	maps.Copy(newCache, oldCache)
	typeCodecs.Store(&newCache)

	return c
}

func mkErroredCodec(err error) codec {
	return codec{
		mkErrorEncoder(err),
		mkErrorDecoder(err),
	}
}

func mkPointerCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	el := t.Elem()
	c := resolveCodec(ctx, el, seen, true)

	return codec{
		mkPointerEncoder(c.encode),
		mkPointerDecoder(c.decode, el),
	}
}

func mkStructCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) codec {
	info, err := compileStructInfo(ctx, t, seen, canAddr)

	if err != nil {
		return mkErroredCodec(fmt.Errorf("failed to compile struct %s: %w", t.String(), err))
	}

	return codec{
		mkStructEncoder(info),
		mkStructDecoder(info),
	}
}

func mkEmbeddedStructPointerCodec(t reflect.Type, unexported bool, offset uintptr, field codec) codec {
	return codec{
		mkEmbeddedStructPointerEncoder(field.encode),
		mkEmbeddedStructPointerDecoder(t, unexported, offset, field.decode),
	}
}

func mkSliceCodec(ctx codecCtx, t reflect.Type, seen seenStructs) codec {
	elem := t.Elem()
	c := resolveCodec(ctx, elem, seen, true)
	size := alignedSize(elem)

	return codec{
		mkSliceEncoder(size, c.encode),
		mkSliceDecoder(size, t, c.decode),
	}
}

func mkArrayCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) codec {
	elem := t.Elem()
	size := alignedSize(elem)
	c := resolveCodec(ctx, elem, seen, canAddr)
	n := t.Len()

	return codec{
		mkArrayEncoder(n, size, c.encode),
		mkArrayDecoder(n, size, c.decode),
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

func compileStructInfo(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) (*structInfo, error) {
	if info, ok := seen[t]; ok {
		return info, nil
	}

	info := &structInfo{
		typ:     t,
		offsets: make(map[string]int, t.NumField()),
	}
	seen[t] = info

	fields, err := compileStructFields(ctx, t, seen, canAddr)
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

func compileStructFields(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) ([]fieldInfo, error) {
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

		c := resolveCodec(ctx, f.Type, seen, canAddr)

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
			return nil, fmt.Errorf("unresolvable ambiguity for fieldHint %q in struct %s", subfield.meta.name, t.String())
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
