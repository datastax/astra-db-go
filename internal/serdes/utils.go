package serdes

import "reflect"

func deref(v reflect.Value) (reflect.Value, int) {
	levels := 0

	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
		levels++
	}

	return v, levels
}
