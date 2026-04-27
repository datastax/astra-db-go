package serdes

import (
	"strconv"
)

//func dynamicEncoder(v reflect.Value, dst []byte) []byte {
//	if v.IsNil() {
//		return append(dst, "null"...)
//	}
//	e := v.Elem()
//	return getEncoder(e.Type())(e, dst)
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
