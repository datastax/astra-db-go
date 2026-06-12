package reflectutil

import "reflect"

// UnwindPointerType recursively unwraps pointer types, returning the underlying type.
func UnwindPointerType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
