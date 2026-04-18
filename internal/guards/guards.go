package guards

import (
	"fmt"
	"reflect"
)

func RequireSlice(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("expected slice, got %s", rv.Kind())
	}

	return rv, nil
}

func RequireSlicePtr(v any) (reflect.Value, reflect.Value, error) {
	ptr := reflect.ValueOf(v)
	if ptr.Kind() != reflect.Ptr {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("need ptr")
	}

	slice := ptr.Elem()
	if slice.Kind() == reflect.Interface {
		slice = slice.Elem()
	}

	if slice.Kind() != reflect.Slice {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf("need slice")
	}

	return ptr, slice, nil
}
