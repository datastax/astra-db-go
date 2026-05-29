package serdes

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

// Serdes

func mkStructCodec(ctx codecCtx, t reflect.Type, seen seenStructs, canAddr bool) codec {
	info, err := compileStructInfo(ctx, t, seen, canAddr)

	if err != nil {
		return mkErroredCodec(&UnsupportedValueError{Msg: fmt.Sprintf("failed to compile struct %s: %v", t.String(), err)})
	}

	return codec{
		mkStructEncoder(info),
		mkStructDecoder(info),
	}
}

func mkEmbeddedStructPointerCodec(t reflect.Type, unexported bool, allowed bool, offset uintptr, field codec) codec {
	return codec{
		mkEmbeddedStructPointerEncoder(field.encode),
		mkEmbeddedStructPointerDecoder(t, unexported, allowed, offset, field.decode),
	}
}

func mkStructEncoder(info *structInfo) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		start := len(dst)
		dst = append(dst, '{')
		firstField := true

		for i := range info.fields {
			f := &info.fields[i]
			v := unsafe.Add(p, f.offset)

			if f.meta.omitempty && f.empty(v) {
				continue
			}

			lengthBeforeKey := len(dst)

			if firstField {
				dst = append(dst, f.prefix[1:]...) // skip leading comma for the first field
				firstField = false
			} else {
				dst = append(dst, f.prefix...)
			}


			ctx.fieldHint = extractFieldHint(f.meta.name)

			var err error
			if dst, err = f.codec.encode(ctx, dst, v); err != nil {
				//goland:noinspection GoTypeAssertionOnErrors
				if _, ok := err.(rollback); ok {
					dst = dst[:lengthBeforeKey]
					continue
				}

				return dst[:start], wrapField(err, info.typ.Name(), f.meta.name)
			}
		}

		return append(dst, '}'), nil
	}
}

func mkEmbeddedStructPointerEncoder(encode encoder) encoder {
	return func(ctx EncodeCtx, dst []byte, p unsafe.Pointer) ([]byte, error) {
		p = *(*unsafe.Pointer)(p)
		if p == nil {
			return dst, rollback{}
		}
		return encode(ctx, dst, p)
	}
}

func mkStructDecoder(info *structInfo) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		src = skipWS(src)

		if src, ok := consumeNull(src); ok {
			return src, nil
		}

		if len(src) == 0 || src[0] != '{' {
			return src, ctx.unmarshalTypeError(src, info.typ)
		}
		src = src[1:] // skip '{'

		var key []byte
		var err error

		for i := 0; ; i++ {
			src = skipWS(src)

			if len(src) == 0 {
				return src, ctx.syntaxError(src, "unexpected end of JSON")
			}

			if src[0] == '}' {
				return src[1:], nil
			}

			if i > 0 {
				if src[0] != ',' {
					return src, ctx.syntaxError(src, "expected ',' after field value")
				}
				src = skipWS(src[1:])
			}

			src, key, _, err = parseStringUnquote(ctx, src)
			if err != nil {
				return src, err
			}
			src = skipWS(src)

			if len(src) == 0 || src[0] != ':' {
				return src, ctx.syntaxError(src, "expected ':' after key")
			}
			src = skipWS(src[1:])

			if fieldIdx, ok := info.offsets[unsafeString(key)]; ok {
				f := &info.fields[fieldIdx]

				ptr := unsafe.Pointer(uintptr(p) + f.offset)
				ctx.fieldHint = extractFieldHint(f.meta.name)

				src, err = f.codec.decode(ctx, src, ptr)
				if err != nil {
					return src, wrapField(err, info.typ.Name(), f.meta.name)
				}
			} else {
				src, err = skipValue(ctx, src)
				if err != nil {
					return src, err
				}
			}
		}
	}
}

func mkEmbeddedStructPointerDecoder(t reflect.Type, unexported bool, allowed bool, offset uintptr, decode decoder) decoder {
	return func(ctx DecodeCtx, src []byte, p unsafe.Pointer) ([]byte, error) {
		v := *(*unsafe.Pointer)(p)

		if v == nil {
			if unexported && !allowed {
				return nil, &UnsupportedValueError{Msg: fmt.Sprintf("cannot set embedded pointer to unexported struct without the \"allowunexported\" struct tag: %s", t)}
			}
			v = unsafe.Pointer(reflect.New(t).Pointer())
			*(*unsafe.Pointer)(p) = v
		}

		return decode(ctx, src, unsafe.Pointer(uintptr(v)+offset))
	}
}

// Struct parsing

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
	empty  func(unsafe.Pointer) bool
}

type jsonMeta struct {
	name            string
	omitempty       bool
	ignored         bool
	tagged          bool
	allowUnexported bool
}

func (i structInfo) String() string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "struct %s {", i.typ.String())
	for _, f := range i.fields {
		_, _ = fmt.Fprintf(&sb, "\n  %s (offset: %d, type: %s, codec: %p)", f.meta.name, f.offset, f.typ.String(), f.codec.encode)
	}
	sb.WriteString("\n}")
	return sb.String()
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

		meta := parseJsonMeta(f)

		if meta.ignored {
			continue
		}

		if unexported && !embedded && !meta.allowUnexported {
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

			if unexported && !meta.allowUnexported { // ignore unallowed unexported non-struct types
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
			empty:  emptyFuncFor(meta.omitempty, f.Type),
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
			return nil, &UnsupportedValueError{Msg: fmt.Sprintf("unresolvable ambiguity for field %q in struct %s", subfield.meta.name, t.String())}
		case unambiguous:
			// all good
		}

		if embfield.pointer {
			subfield.codec = mkEmbeddedStructPointerCodec(embfield.subtype.typ, embfield.unexported, subfield.meta.allowUnexported, subfield.offset, subfield.codec)
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

	sort.Slice(fields, func(i, j int) bool { return fields[i].ord < fields[j].ord })

	return fields, nil
}

func parseJsonMeta(f reflect.StructField) jsonMeta {
	var info jsonMeta
	info.name = f.Name

	if parts := strings.Split(f.Tag.Get("json"), ","); len(parts) != 0 {
		if len(parts[0]) > 0 {
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
			case "allowunexported":
				info.allowUnexported = true
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
		return shadowed // top level field with the same name exists so ignore this embedded field
	}

	if nameCounts[meta.name] == 1 {
		return unambiguous // no collisions so all good to go
	}

	if tagCounts[meta.name] == 1 && meta.tagged {
		return unambiguous // multiple fields with the same name, so the field with the tag wins
	}

	if tagCounts[meta.name] != 1 {
		return ambiguous // zero or multiple tags w/ the same name so we can't resolve anything
	}

	return shadowed // field collided and lost to a tagged field.
}

// vendored from segmentio/encode/json
func emptyFuncFor(omitempty bool, t reflect.Type) func(unsafe.Pointer) bool {
	if !omitempty {
		return nil
	}

	switch t {
	case byteSliceType, rawMessageType:
		return func(p unsafe.Pointer) bool { return (*slice)(p).len == 0 }
	}

	switch t.Kind() {
	case reflect.Array:
		if t.Len() == 0 {
			return func(unsafe.Pointer) bool { return true }
		}

	case reflect.Map:
		return func(p unsafe.Pointer) bool { return reflect.NewAt(t, p).Elem().Len() == 0 }

	case reflect.Slice:
		return func(p unsafe.Pointer) bool { return (*slice)(p).len == 0 }

	case reflect.String:
		return func(p unsafe.Pointer) bool { return len(*(*string)(p)) == 0 }

	case reflect.Bool:
		return func(p unsafe.Pointer) bool { return !*(*bool)(p) }

	case reflect.Int, reflect.Uint:
		return func(p unsafe.Pointer) bool { return *(*uint)(p) == 0 }

	case reflect.Uintptr:
		return func(p unsafe.Pointer) bool { return *(*uintptr)(p) == 0 }

	case reflect.Int8, reflect.Uint8:
		return func(p unsafe.Pointer) bool { return *(*uint8)(p) == 0 }

	case reflect.Int16, reflect.Uint16:
		return func(p unsafe.Pointer) bool { return *(*uint16)(p) == 0 }

	case reflect.Int32, reflect.Uint32:
		return func(p unsafe.Pointer) bool { return *(*uint32)(p) == 0 }

	case reflect.Int64, reflect.Uint64:
		return func(p unsafe.Pointer) bool { return *(*uint64)(p) == 0 }

	case reflect.Float32:
		return func(p unsafe.Pointer) bool { return *(*float32)(p) == 0 }

	case reflect.Float64:
		return func(p unsafe.Pointer) bool { return *(*float64)(p) == 0 }

	case reflect.Pointer:
		return func(p unsafe.Pointer) bool { return *(*unsafe.Pointer)(p) == nil }

	case reflect.Interface:
		return func(p unsafe.Pointer) bool { return (*iface)(p).ptr == nil }

	default:
		return func(unsafe.Pointer) bool { return false }
	}

	return func(unsafe.Pointer) bool { return false }
}
