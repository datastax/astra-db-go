// Copyright DataStax, Inc.
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

// Package options provides options, types, and utilities for Astra DB operations.
package options

import "reflect"

type Validator interface {
	Validate() error
}

// Defaulter is an optional interface that options types can implement
// to populate default values. If a type implements Defaulter, MergeOptions
// calls SetDefaults before applying user-provided setters, so user values
// always override defaults.
type Defaulter interface {
	SetDefaults()
}

// Builder is an interface that wraps a List method to return a
// slice of option setters. This follows the MongoDB Go driver pattern
// for composable options.
type Builder[T Validator] interface {
	List() []func(*T)
}

// NoopBuilder returns a [Builder] implementation that just copies
// from the source to the target.
func NoopBuilder[T any](src *T) []func(*T) {
	return []func(*T){
		func(target *T) {
			copyNonNilFields(src, target)
		},
	}
}

// MergeOptions merges multiple Builder options into a single options struct.
// It applies each option's setters sequentially, with later options overriding
// earlier ones for the same fields. Calls `Validate` on the result and returns
// errors (if any).
// Note: result will never be nil.
func MergeOptions[T Validator](opts ...Builder[T]) (*T, error) {
	result := new(T)
	if d, ok := any(result).(Defaulter); ok {
		d.SetDefaults()
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		for _, setter := range opt.List() {
			setter(result)
		}
	}
	err := (*result).Validate()
	return result, err
}

// copyNonNilFields copies all non-nil pointer fields from src to dst.
// Used by options structs to implement Builder without manual field enumeration.
func copyNonNilFields[T any](src, dst *T) {
	srcVal := reflect.ValueOf(src).Elem()
	dstVal := reflect.ValueOf(dst).Elem()

	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		switch srcField.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
			if !srcField.IsNil() {
				dstVal.Field(i).Set(srcField)
			}
		}
	}
}
