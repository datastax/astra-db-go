package serdes

import (
	"errors"
)

var errInvalid = errors.New("invalid json")

//func makeMapDecoder(t reflect.Type) decoder {
//	elemType := t.Elem()
//	return func(src []byte, v reflect.Value) (int, error) {
//		off := skipWS(src, 0)
//		if off < len(src) && src[off] == 'n' {
//			return off + 4, nil
//		}
//		if off >= len(src) || src[off] != '{' {
//			return off, errInvalid
//		}
//		off++
//		if v.IsNil() {
//			v.Set(reflect.MakeMap(t))
//		}
//		for {
//			off = skipWS(src, off)
//			if off >= len(src) {
//				return off, errInvalid
//			}
//			if src[off] == '}' {
//				return off + 1, nil
//			}
//			key, n, err := parseStringRaw(src[off:])
//			if err != nil {
//				return off, err
//			}
//			off += n
//			off = skipWS(src, off)
//			if off >= len(src) || src[off] != ':' {
//				return off, errInvalid
//			}
//			off++
//			newElem := reflect.New(elemType).Elem()
//			n, err = getDecoder(elemType)(src[off:], newElem)
//			if err != nil {
//				return off, err
//			}
//			off += n
//			v.SetMapIndex(reflect.ValueOf(key), newElem)
//			off = skipWS(src, off)
//			if off < len(src) && src[off] == ',' {
//				off++
//				continue
//			}
//		}
//	}
//}
//
//func makeSliceDecoder(t reflect.Type) decoder {
//	elemType := t.Elem()
//	return func(src []byte, v reflect.Value) (int, error) {
//		off := skipWS(src, 0)
//		if off < len(src) && src[off] == 'n' {
//			v.Set(reflect.Zero(t))
//			return off + 4, nil
//		}
//		if off >= len(src) || src[off] != '[' {
//			return off, errInvalid
//		}
//		off++
//		slice := reflect.MakeSlice(t, 0, 4)
//		for {
//			off = skipWS(src, off)
//			if off >= len(src) {
//				return off, errInvalid
//			}
//			if src[off] == ']' {
//				v.Set(slice)
//				return off + 1, nil
//			}
//			newElem := reflect.New(elemType).Elem()
//			n, err := getDecoder(elemType)(src[off:], newElem)
//			if err != nil {
//				return off, err
//			}
//			off += n
//			slice = reflect.Append(slice, newElem)
//			off = skipWS(src, off)
//			if off < len(src) && src[off] == ',' {
//				off++
//				continue
//			}
//		}
//	}
//}
//
//func makePtrDecoder(t reflect.Type) decoder {
//	elemDec := getDecoder(t.Elem())
//	return func(src []byte, v reflect.Value) (int, error) {
//		off := skipWS(src, 0)
//		if off < len(src) && src[off] == 'n' {
//			v.Set(reflect.Zero(t))
//			return off + 4, nil
//		}
//		if v.IsNil() {
//			v.Set(reflect.New(t.Elem()))
//		}
//		return elemDec(src, v.Elem())
//	}
//}
//
//// --- Primitive & Interface Decoders ---
//
//func decodeInterface(src []byte, v reflect.Value) (int, error) {
//	off := skipWS(src, 0)
//	if off >= len(src) {
//		return off, errInvalid
//	}
//
//	var n int
//	var err error
//	switch src[off] {
//	case '{':
//		m := reflect.MakeMap(reflect.TypeOf(map[string]any(nil)))
//		n, err = makeMapDecoder(m.Type())(src[off:], m)
//		v.Set(m)
//	case '[':
//		var s []any
//		rv := reflect.ValueOf(&s).Elem()
//		n, err = makeSliceDecoder(rv.Type())(src[off:], rv)
//		v.Set(rv)
//	case '"':
//		var s string
//		rv := reflect.ValueOf(&s).Elem()
//		n, err = decodeString(src[off:], rv)
//		v.Set(rv)
//	case 't', 'f':
//		var b bool
//		rv := reflect.ValueOf(&b).Elem()
//		n, err = decodeBool(src[off:], rv)
//		v.Set(rv)
//	case 'n':
//		v.Set(reflect.Zero(v.Type()))
//		return off + 4, nil
//	default:
//		var f float64
//		rv := reflect.ValueOf(&f).Elem()
//		n, err = decodeFloat(src[off:], rv)
//		v.Set(rv)
//	}
//	return off + n, err
//}

// --- Helpers ---

func parseStringRaw(src []byte) (string, int, error) {
	off := skipWSOld(src, 0)
	if off >= len(src) || src[off] != '"' {
		return "", off, errInvalid
	}
	off++
	start := off
	for off < len(src) {
		if src[off] == '\\' {
			off += 2
			continue
		}
		if src[off] == '"' {
			return string(src[start:off]), off + 1, nil
		}
		off++
	}
	return "", off, errInvalid
}

func skipWSOld(src []byte, off int) int {
	for off < len(src) && (src[off] <= ' ') {
		off++
	}
	return off
}
