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

// Joined is a type-safe wrapper for a slice of builders.
// It is returned by [Join] and should be used in structs that store
// accumulated options to ensure [Join] is used for combination.
type Joined[T any] []Builder[T]

// Setters implements [Builder] for [Joined].
func (j Joined[T]) Setters() []func(*T) {
	var res []func(*T)
	for _, b := range j {
		if b != nil && !reflect.ValueOf(b).IsNil() {
			res = append(res, b.Setters()...)
		}
	}
	return res
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

// Join combines a base slice of builders with additional builders into a new
// Joined slice. It always performs a copy to ensure that the original slice's
// underlying array is never modified, making handle creation thread-safe.
func Join[T any](base []Builder[T], additional ...Builder[T]) Joined[T] {
	if len(additional) == 0 {
		return base
	}
	res := make(Joined[T], 0, len(base)+len(additional))
	res = append(res, base...)
	res = append(res, additional...)
	return res
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
// It starts with a default-initialized struct and then applies each
// option's setters sequentially, with later options overriding earlier ones.
// Note: result will never be nil and will always have defaults applied.
func Merge[T any](opts ...Builder[T]) *T {
	result := new(T)
	if d, ok := any(result).(Defaulter); ok {
		d.SetDefaults()
	}

	for _, opt := range opts {
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			continue
		}
		// If an option is wrapped in Replace, we reset the struct to defaults
		// before applying that option's setters.
		if isReplace(opt) {
			*result = *new(T)
			if d, ok := any(result).(Defaulter); ok {
				d.SetDefaults()
			}
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
	if v, ok := any(result).(Validator); ok {
		if err := validateRecursive(v); err != nil {
			return result, err
		}
	}
	return result, nil
}

// MergeInto merges multiple Builder options into an existing options struct.
// It initializes the target if it is nil and applies each option's setters
// sequentially. This function does NOT apply defaults unless it is a
// Replace operation, ensuring hierarchical merging remains sparse.
func MergeInto[T any](target **T, opts ...Builder[T]) {
	for _, opt := range opts {
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			continue
		}
		if isReplace(opt) {
			*target = new(T)
			if d, ok := any(*target).(Defaulter); ok {
				d.SetDefaults()
			}
		}
		if *target == nil {
			*target = new(T)
		}
		for _, setter := range opt.Setters() {
			setter(*target)
		}
	}
}

// Replace wraps a Builder to signal that it should overwrite the existing
// struct rather than merging into it when used with MergeInto.
func Replace[T any](b Builder[T]) Builder[T] {
	return &replaceBuilder[T]{Builder: b}
}

type replaceBuilder[T any] struct {
	Builder[T]
}

func (b *replaceBuilder[T]) isReplace() {}

func isReplace(b any) bool {
	_, ok := b.(interface{ isReplace() })
	return ok
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
				dstVal.Field(i).Set(srcField)
			}
		}
	}
}
