package serdes

import (
	"errors"
	"reflect"
	"strconv"
)

var errInvalid = errors.New("invalid json")

//var (
//	decodeCache  sync.Map // map[reflect.Type]decoder
//	kindDecoders [reflect.UnsafePointer + 1]decoder
//)
//
//func getDecoder(t reflect.Type) decoder {
//	kind := t.Kind()
//
//	// 1. FAST PATH: Primitives (Kind-based)
//	// We only bypass the map if the type isn't a custom unmarshaler.
//	if int(kind) < len(kindDecoders) && kindDecoders[kind] != nil {
//		if !t.Implements(unmarshalerType) && !reflect.PointerTo(t).Implements(unmarshalerType) {
//			return kindDecoders[kind]
//		}
//	}
//
//	// 2. CACHE PATH: Complex Types (Type-based)
//	if d, ok := decodeCache.Load(t); ok {
//		return d.(decoder)
//	}
//
//	// 3. SLOW PATH: Reflection (Run once per type)
//	var d decoder
//
//	// Check custom unmarshaler first
//	if t.Implements(unmarshalerType) || reflect.PointerTo(t).Implements(unmarshalerType) {
//		d = makeCustomDecoder(t)
//	} else {
//		switch kind {
//		case reflect.Interface:
//			d = decodeInterface
//		case reflect.Map:
//			d = makeMapDecoder(t)
//		case reflect.Pointer:
//			d = makePtrDecoder(t)
//		case reflect.Struct:
//			d = makeStructDecoder(t)
//		case reflect.Slice, reflect.Array:
//			d = makeSliceDecoder(t)
//		default:
//			d = func(src []byte, v reflect.Value) (int, error) { return skipValue(src, 0) }
//		}
//	}
//
//	decodeCache.Store(t, d)
//	return d
//}

// --- Structural Decoders ---
//
//func makeStructDecoder(t reflect.Type) decoder {
//	fields := make(map[string]struct {
//		index int
//		dec   decoder
//	})
//	for i := 0; i < t.NumField(); i++ {
//		f := t.Field(i)
//		if f.PkgPath != "" {
//			continue // skip unexported
//		}
//		fields[f.Name] = struct {
//			index int
//			dec   decoder
//		}{i, getDecoder(f.Type)}
//	}
//
//	return func(src []byte, v reflect.Value) (int, error) {
//		off := skipWS(src, 0)
//		if off >= len(src) || src[off] != '{' {
//			return off, errInvalid
//		}
//		off++
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
//			if info, ok := fields[key]; ok {
//				n, err := info.dec(src[off:], v.Field(info.index))
//				if err != nil {
//					return off, err
//				}
//				off += n
//			} else {
//				n, err := skipValue(src, off)
//				if err != nil {
//					return off, err
//				}
//				off += n
//			}
//			off = skipWS(src, off)
//			if off < len(src) && src[off] == ',' {
//				off++
//				continue
//			}
//		}
//	}
//}
//
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

func decodeInt(src []byte, v reflect.Value) (int, error) {
	off := skipWSOld(src, 0)
	start := off
	for off < len(src) && (src[off] == '-' || (src[off] >= '0' && src[off] <= '9')) {
		off++
	}
	i, err := strconv.ParseInt(string(src[start:off]), 10, 64)
	v.SetInt(i)
	return off, err
}

func decodeUint(src []byte, v reflect.Value) (int, error) {
	off := skipWSOld(src, 0)
	start := off
	for off < len(src) && (src[off] >= '0' && src[off] <= '9') {
		off++
	}
	i, err := strconv.ParseUint(string(src[start:off]), 10, 64)
	v.SetUint(i)
	return off, err
}

func decodeFloat(src []byte, v reflect.Value) (int, error) {
	off := skipWSOld(src, 0)
	start := off
	for off < len(src) && (src[off] == '-' || src[off] == '.' || (src[off] >= '0' && src[off] <= '9')) {
		off++
	}
	f, err := strconv.ParseFloat(string(src[start:off]), 64)
	v.SetFloat(f)
	return off, err
}

func decodeBool(src []byte, v reflect.Value) (int, error) {
	off := skipWSOld(src, 0)
	if off+4 <= len(src) && string(src[off:off+4]) == "true" {
		v.SetBool(true)
		return off + 4, nil
	} else if off+5 <= len(src) && string(src[off:off+5]) == "false" {
		v.SetBool(false)
		return off + 5, nil
	}
	return off, errInvalid
}

func decodeString(src []byte, v reflect.Value) (int, error) {
	s, n, err := parseStringRaw(src)
	v.SetString(s)
	return n, err
}

//
//func makeCustomDecoder(t reflect.Type) decoder {
//	return func(src []byte, v reflect.Value) (int, error) {
//		var intermediate any
//		n, err := decodeInterface(src, reflect.ValueOf(&intermediate).Elem())
//		if err != nil {
//			return n, err
//		}
//		if v.CanAddr() && reflect.PointerTo(v.Type()).Implements(unmarshalerType) {
//			err = v.Addr().Interface().(AstraUnmarshaler).FromAstraValue(intermediate)
//		} else {
//			err = v.Interface().(AstraUnmarshaler).FromAstraValue(intermediate)
//		}
//		return n, err
//	}
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

func skipValue(src []byte, off int) (int, error) {
	off = skipWSOld(src, off)
	if off >= len(src) {
		return off, errInvalid
	}
	switch src[off] {
	case '"':
		_, n, err := parseStringRaw(src[off:])
		return off + n, err
	case '{', '[':
		depth, open, close := 0, src[off], src[off]+2
		if open == '{' {
			close = '}'
		}
		for off < len(src) {
			if src[off] == open {
				depth++
			}
			if src[off] == close {
				depth--
			}
			off++
			if depth == 0 {
				return off, nil
			}
		}
	default:
		for off < len(src) && src[off] != ',' && src[off] != '}' && src[off] != ']' {
			off++
		}
	}
	return off, nil
}
