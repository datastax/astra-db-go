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
	Children() []any
}

// Builder is an interface that wraps a Setters method to return a
// slice of option setters. This follows the MongoDB Go driver pattern
// for composable options.
type Builder[T any] interface {
	Setters() []func(*T)
}

// ShouldMerge is implemented by options types whose pointer fields should
// be merged sub-field-by-sub-field rather than replaced wholesale when
// encountered during a copyNonNilFields pass.
// Implementations must handle nil receivers gracefully.
type ShouldMerge interface {
	Merge(other ShouldMerge) ShouldMerge
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
func validateRecursive(v any) error {
	if cv, ok := v.(ChildValidator); ok {
		for _, child := range cv.Children() {
			if err := validateRecursive(child); err != nil {
				return err
			}
		}
	}
	if val, ok := v.(Validator); ok {
		return val.Validate()
	}
	return nil
}

// Merge merges multiple Builder options into a single options struct.
// It starts with a default-initialized struct and then applies each
// option's setters sequentially, with later options overriding earlier ones.
// Note: result will never be nil and will always have defaults applied.
func Merge[T any](opts ...Builder[T]) *T {
	result := new(T)

	for _, opt := range opts {
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			continue
		}

		for _, setter := range opt.Setters() {
			setter(result)
		}
	}
	return result
}

// MergeAndValidate merges multiple Builder options into a single options struct
// and then recursively validates the result.
func MergeAndValidate[T any](opts ...Builder[T]) (*T, error) {
	result := Merge(opts...)
	if err := validateRecursive(result); err != nil {
		return result, err
	}
	return result, nil
}

// MergeInto merges multiple Builder options into an existing options struct.
// It initializes the target if it is nil and applies each option's setters
// sequentially.
func MergeInto[T any](target **T, opts ...Builder[T]) {
	for _, opt := range opts {
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			continue
		}
		if *target == nil {
			*target = new(T)
		}
		for _, setter := range opt.Setters() {
			setter(*target)
		}
	}
}

// Used by options structs to implement Builder without manual field enumeration.
func copyNonNilFields[T any](src, dst *T) {
	srcVal := reflect.ValueOf(src).Elem()
	dstVal := reflect.ValueOf(dst).Elem()

	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		switch srcField.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
			if !srcField.IsNil() {
				if sm, ok := srcField.Interface().(ShouldMerge); ok {
					dstSM := dstVal.Field(i).Interface().(ShouldMerge)
					dstVal.Field(i).Set(reflect.ValueOf(dstSM.Merge(sm)))
					continue
				}
				dstVal.Field(i).Set(srcField)
			}
		}
	}
}
