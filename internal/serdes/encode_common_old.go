package serdes

import (
	"strconv"
)

//func getEncoder(t reflect.Type) encoder {
//	// 1. DYNAMIC FALLBACK: For interface{} types
//	if t.Kind() == reflect.Interface {
//		return dynamicEncoder
//	}
//
//	// 2. CUSTOM MARSHALER
//	if t.Implements(marshalerType) || (t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(marshalerType)) {
//		return makeCustomMarshalerEncoder(t)
//	}
//
//	// 3. FLATTEN POINTERS
//	if t.Kind() == reflect.Pointer {
//		curr := t
//		depth := 0
//		for curr.Kind() == reflect.Pointer {
//			curr = curr.Elem()
//			depth++
//		}
//		baseEnc := getEncoder(curr)
//		return func(v reflect.Value, dst []byte) []byte {
//			for i := 0; i < depth; i++ {
//				if v.IsNil() {
//					return append(dst, "null"...)
//				}
//				v = v.Elem()
//			}
//			return baseEnc(v, dst)
//		}
//	}
//
//	// 4. FAST PATH: Primitives
//	kind := t.Kind()
//	if int(kind) < len(kindEncoders) && kindEncoders[kind] != nil {
//		return kindEncoders[kind]
//	}
//
//	// 5. CACHE PATH
//	if c, ok := typeEncoders.Load(t); ok {
//		return c.(encoder)
//	}
//
//	var c encoder
//	switch kind {
//	case reflect.Struct:
//		c = makeStructEncoder(t)
//	case reflect.Slice, reflect.Array:
//		c = makeSliceEncoder(t)
//	case reflect.Map:
//		c = makeMapEncoder(t)
//	default:
//		c = func(v reflect.Value, dst []byte) []byte { return append(dst, "null"...) }
//	}
//
//	typeEncoders.Store(t, c)
//	return c
//}
//
//func dynamicEncoder(v reflect.Value, dst []byte) []byte {
//	if v.IsNil() {
//		return append(dst, "null"...)
//	}
//	e := v.Elem()
//	return getEncoder(e.Type())(e, dst)
//}
//
//func makeCustomMarshalerEncoder(t reflect.Type) encoder {
//	return func(v reflect.Value, dst []byte) []byte {
//		if !v.Type().Implements(marshalerType) && v.CanAddr() {
//			v = v.Addr()
//		}
//		if v.Kind() == reflect.Pointer && v.IsNil() {
//			return append(dst, "null"...)
//		}
//
//		m := v.Interface().(AstraMarshaler)
//		res := m.ToAstraValue()
//		if res == nil {
//			return append(dst, "null"...)
//		}
//
//		rv := reflect.ValueOf(res)
//		return getEncoder(rv.Type())(rv, dst)
//	}
//}
//
//func makeStructEncoder(t reflect.Type) encoder {
//	type fieldMeta struct {
//		prefix []byte
//		ord  int
//		enc    encoder
//	}
//	var fields []fieldMeta
//	for i := 0; i < t.NumField(); i++ {
//		f := t.Field(i)
//		if f.PkgPath != "" {
//			continue
//		}
//
//		// JSON keys must be quoted: "FieldName":
//		key := `"` + f.Name + `":`
//		prefix := key
//		if len(fields) > 0 {
//			prefix = "," + key
//		}
//
//		fields = append(fields, fieldMeta{
//			prefix: []byte(prefix),
//			ord:  i,
//			enc:    getEncoder(f.Type),
//		})
//	}
//	return func(v reflect.Value, dst []byte) []byte {
//		dst = append(dst, '{')
//		for i := range fields {
//			f := &fields[i]
//			dst = append(dst, f.prefix...)
//			dst = f.enc(v.Field(f.ord), dst)
//		}
//		return append(dst, '}')
//	}
//}
//
//func makeSliceEncoder(t reflect.Type) encoder {
//	elemEnc := getEncoder(t.Elem())
//	return func(v reflect.Value, dst []byte) []byte {
//		if v.IsNil() {
//			return append(dst, "null"...)
//		}
//		dst = append(dst, '[')
//		n := v.Len()
//		for i := 0; i < n; i++ {
//			if i > 0 {
//				dst = append(dst, ',')
//			}
//			dst = elemEnc(v.Index(i), dst)
//		}
//		return append(dst, ']')
//	}
//}
//
//func makeMapEncoder(t reflect.Type) encoder {
//	valEnc := getEncoder(t.Elem())
//	return func(v reflect.Value, dst []byte) []byte {
//		if v.IsNil() {
//			return append(dst, "null"...)
//		}
//		dst = append(dst, '{')
//		iter := v.MapRange()
//		first := true
//		for iter.Next() {
//			if !first {
//				dst = append(dst, ',')
//			}
//			// JSON map keys must be strings
//			dst = appendString(dst, iter.Key().String())
//			dst = append(dst, ':')
//			dst = valEnc(iter.Value(), dst)
//			first = false
//		}
//		return append(dst, '}')
//	}
//}

func appendString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0x20 && b != '\\' && b != '"' {
			continue
		}
		dst = append(dst, s[start:i]...)
		switch b {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			dst = append(dst, []byte(strconv.QuoteRune(rune(b)))...)
		}
		start = i + 1
	}
	return append(append(dst, s[start:]...), '"')
}
