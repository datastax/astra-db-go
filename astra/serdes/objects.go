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
	"strings"
	"unsafe"

	"github.com/datastax/astra-db-go/v2/internal/structutil"
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

			var next []byte
			var err error
			if next, err = f.codec.encode(ctx, dst, v); err != nil {
				//goland:noinspection GoTypeAssertionOnErrors
				if _, ok := err.(rollback); ok {
					dst = dst[:lengthBeforeKey]
					continue
				}

				return dst[:start], wrapField(err, info.typ.Name(), f.meta.name)
			}
			dst = next
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
			return src, ctx.unmarshalTypeErrorWrap(src, info.typ, ctx.syntaxError(src, "expected '{' at the start of an object"))
		}
		src = src[1:] // skip '{'

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

			srcAfter, key, _, err := parseStringUnquote(ctx, src)
			if err != nil {
				return srcAfter, err
			}
			src = skipWS(srcAfter)

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
	meta   jsonMeta
	empty  func(unsafe.Pointer) bool
}

type jsonMeta struct {
	name            string
	omitempty       bool
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
	metas, err := structutil.GetFields(t)
	if err != nil {
		return nil, err
	}

	fields := make([]fieldInfo, 0, len(metas))

	for _, meta := range metas {
		c := resolveCodec(ctx, meta.Field.Type, seen, canAddr)
		c, offset := resolveCodecInEmbeddedFields(t, meta.Index, c, meta)

		jm := jsonMeta{
			name:            meta.Name,
			omitempty:       meta.OmitEmpty,
			allowUnexported: meta.AllowUnexported,
		}

		fields = append(fields, fieldInfo{
			codec:  c,
			offset: offset,
			meta:   jm,
			typ:    meta.Field.Type,
			empty:  emptyFuncFor(meta.OmitEmpty, meta.Field.Type),
		})
	}

	for i := range fields {
		fields[i].prefix = []byte(`,"` + fields[i].meta.name + `":`)
	}

	return fields, nil
}

func resolveCodecInEmbeddedFields(parentType reflect.Type, indexPath []int, leafCodec codec, leafMeta structutil.FieldMeta) (codec, uintptr) {
	f := parentType.Field(indexPath[0])

	if len(indexPath) == 1 {
		return leafCodec, f.Offset
	}

	embeddedType := f.Type
	isEmbeddedStructPtr := embeddedType.Kind() == reflect.Pointer
	if isEmbeddedStructPtr {
		embeddedType = embeddedType.Elem()
	}

	innerCodec, innerOffset := resolveCodecInEmbeddedFields(embeddedType, indexPath[1:], leafCodec, leafMeta)

	if isEmbeddedStructPtr {
		unexported := len(f.PkgPath) != 0
		wrappedCodec := mkEmbeddedStructPointerCodec(f.Type.Elem(), unexported, leafMeta.AllowUnexported, innerOffset, innerCodec)
		return wrappedCodec, f.Offset
	}

	return innerCodec, innerOffset + f.Offset
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
