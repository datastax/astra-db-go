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

//go:generate go run -modfile=../../tools/gen-options/go.mod ../../tools/gen-options/main.go -pkg .

// Package options provides options, types, and utilities for Astra DB operations.
package options

import (
	"reflect"
)

type Validator interface {
	Validate() error
}

// ChildValidator is an optional interface that options types can implement
// to return nested validators for hierarchical validation. If a type implements
// ChildValidator, MergeAndValidate calls Validate on all children.
// This will be codegen'd for any struct with nested [Validator] structs.
type ChildValidator interface {
	Children() []Validator
}

// Defaulter is an optional interface that options types can implement
// to populate default values. If a type implements Defaulter, Merge
// calls SetDefaults before applying user-provided setters, so user values
// always override defaults.
type Defaulter interface {
	SetDefaults()
}

// Builder is an interface that wraps a Setters method to return a
// slice of option setters. This follows the MongoDB Go driver pattern
// for composable options.
type Builder[T any] interface {
	Setters() []func(*T)
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

// validateRecursive walks the ChildValidator tree depth-first, validating
// children before the parent. This ensures grandchildren (and deeper) are
// validated even when MergeAndValidate only sees the top-level result.
func validateRecursive(v Validator) error {
	if cv, ok := v.(ChildValidator); ok {
		for _, child := range cv.Children() {
			if err := validateRecursive(child); err != nil {
				return err
			}
		}
	}
	return v.Validate()
}

// Merge merges multiple Builder options into a single options struct.
// It applies each option's setters sequentially, with later options overriding
// earlier ones for the same fields. No validation is performed.
// Note: result will never be nil.
func Merge[T any](opts ...Builder[T]) *T {
	result := new(T)
	if d, ok := any(result).(Defaulter); ok {
		d.SetDefaults()
	}
	for _, opt := range opts {
		// Guard against both plain nil and typed nil (a non-nil interface
		// wrapping a nil concrete pointer), which can happen when a caller
		// declares a typed builder variable but never initializes it.
		// See also: TestMergeAndValidate_TypedNilBuilder - which will panic
		// without this check.
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			continue
		}
		for _, setter := range opt.Setters() {
			setter(result)
		}
	}
	return result
}

// MergeAndValidate merges multiple Builder options into a single options struct,
// then recursively validates the result. It applies each option's setters
// sequentially, with later options overriding earlier ones for the same fields.
// Note: result will never be nil.
func MergeAndValidate[T any](opts ...Builder[T]) (*T, error) {
	result := Merge(opts...)
	if v, ok := any(result).(Validator); ok {
		if err := validateRecursive(v); err != nil {
			return result, err
		}
	}
	return result, nil
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
