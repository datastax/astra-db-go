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

package serdes

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type pathContext struct {
	Struct string
	Field  string
}

func (p *pathContext) setField(part string) { p.Field = joinPath(part, p.Field) }
func (p *pathContext) setStruct(name string) {
	p.Struct = name
}

func (p *pathContext) fullPath() string {
	return joinPath(p.Struct, p.Field)
}

func (p *pathContext) formatPath() string {
	if path := p.fullPath(); path != "" {
		return " in '" + path + "'"
	}
	return ""
}

type pathError interface {
	setField(string)
	setStruct(string)
}

func wrapPath(err error, part string) error {
	if err == nil {
		return nil
	}

	// Internal rollback errors should never be wrapped with path context
	if _, ok := err.(rollback); ok {
		return err
	}

	var pe pathError
	if errors.As(err, &pe) {
		pe.setField(part)
		return err
	}

	return fmt.Errorf("field '%s': %w", part, err)
}

func wrapStruct(err error, name string) error {
	if err == nil || name == "" {
		return err
	}

	var pe pathError
	if errors.As(err, &pe) {
		pe.setStruct(name)
	}

	return err
}

func wrapField(err error, structName, fieldName string) error {
	if err == nil {
		return nil
	}
	err = wrapStruct(err, structName)
	return wrapPath(err, fieldName)
}

// A SyntaxError is a description of a JSON/BSON syntax error.
type SyntaxError struct {
	pathContext
	msg     string // description of error
	Snippet string // snippet of JSON near the error
	Err     error  // underlying error
}

func (e *SyntaxError) diagnostic() string {
	msg := e.msg
	if e.Err != nil {
		msg += ": " + displayError(e.Err)
	}
	return msg
}

func (e SyntaxError) Error() string {
	msg := "serdes: syntax error"
	if path := e.formatPath(); path != "" {
		msg += path
	}
	msg += ": " + e.diagnostic()
	return withSnippet(e, msg)
}

func (e *SyntaxError) Unwrap() error { return e.Err }

// An UnmarshalTypeError describes a JSON/BSON value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	pathContext
	Value   string       // description of JSON/BSON value - "bool", "array", "number -5"
	Type    reflect.Type // type of Go value it could not be assigned to
	Snippet string       // snippet of JSON near the error
	Err     error        // underlying error
}

func (e *UnmarshalTypeError) diagnostic() string {
	if e.Err != nil {
		return displayError(e.Err)
	}
	return ""
}

func (e UnmarshalTypeError) Error() string {
	msg := "serdes: cannot unmarshal " + e.Value
	path := e.fullPath()
	if path != "" && path != e.Type.String() && path != e.Type.Name() {
		msg += " into '" + path + "' (type " + e.Type.String() + ")"
	} else {
		msg += " into type " + e.Type.String()
	}

	if diag := e.diagnostic(); diag != "" {
		// Suppress redundant "expected [type]" if it matches the target type
		typeName := getValueName(e.Type)
		if !strings.Contains(diag, "expected "+e.Type.String()) && !strings.Contains(diag, "expected "+typeName) {
			msg += ": " + diag
		}
	}

	return withSnippet(e, msg)
}

func getValueName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return "unsigned integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	default:
		return t.String()
	}
}

func (e *UnmarshalTypeError) Unwrap() error { return e.Err }

func errorSnippet(b []byte, flags DesFlags) string {
	if len(b) == 0 {
		return ""
	}
	ctx := 16
	if flags&ExtendedErrorContext != 0 {
		ctx = 64
	}
	if len(b) > ctx {
		return string(b[:ctx]) + "..."
	}
	return string(b)
}

func innermostSnippet(err error) string {
	var snippet string
	for err != nil {
		if se, ok := err.(*SyntaxError); ok && se.Snippet != "" {
			snippet = se.Snippet
		} else if te, ok := err.(*UnmarshalTypeError); ok && te.Snippet != "" {
			snippet = te.Snippet
		} else if ue, ok := err.(*UnmarshalerError); ok && ue.Snippet != "" {
			snippet = ue.Snippet
		}
		err = errors.Unwrap(err)
	}
	return snippet
}

func withSnippet(err error, msg string) string {
	if s := innermostSnippet(err); s != "" && !strings.Contains(msg, "near: '"+s+"'") {
		return msg + " near: '" + s + "'"
	}
	return msg
}

// A MarshalerError represents an error from calling a MarshalAstra method.
type MarshalerError struct {
	pathContext
	Type reflect.Type
	Err  error
}

func (e *MarshalerError) diagnostic() string {
	return displayError(e.Err)
}

func (e MarshalerError) Error() string {
	msg := "serdes: error calling MarshalAstra"
	path := e.fullPath()
	if path != "" && path != e.Type.String() && path != e.Type.Name() {
		msg += " in '" + path + "'"
	} else {
		msg += " for type " + e.Type.String()
	}
	msg += ": " + e.diagnostic()
	return msg
}

func (e *MarshalerError) Unwrap() error { return e.Err }

// An UnmarshalerError represents an error from calling an UnmarshalAstra method.
type UnmarshalerError struct {
	pathContext
	Type    reflect.Type
	Snippet string
	Err     error
}

func (e *UnmarshalerError) diagnostic() string {
	return displayError(e.Err)
}

func (e UnmarshalerError) Error() string {
	msg := "serdes: error calling UnmarshalAstra"
	path := e.fullPath()
	if path != "" && path != e.Type.String() && path != e.Type.Name() {
		msg += " in '" + path + "'"
	} else {
		msg += " for type " + e.Type.String()
	}
	msg += ": " + e.diagnostic()
	return withSnippet(e, msg)
}

func (e *UnmarshalerError) Unwrap() error { return e.Err }

// An UnsupportedValueError is returned when a value is not supported by Astra,
// such as a cycle, a missing tag, or a target mismatch.
type UnsupportedValueError struct {
	pathContext
	Value any
	Msg   string
}

func (e UnsupportedValueError) Error() string {
	msg := "serdes: unsupported value"
	if e.Msg != "" {
		msg += ": " + e.Msg
	}
	return msg + e.formatPath()
}

// An UnsupportedTypeError is returned when attempting to marshal
// an unsupported type.
type UnsupportedTypeError struct {
	pathContext
	Type reflect.Type
}

func (e UnsupportedTypeError) Error() string {
	return "serdes: unsupported type: " + e.Type.String() + e.formatPath()
}

func displayError(err error) string {
	if err == nil {
		return ""
	}
	if de, ok := err.(interface{ diagnostic() string }); ok {
		return de.diagnostic()
	}
	s := err.Error()
	if len(s) >= 8 && s[:8] == "serdes: " {
		return s[8:]
	}
	return s
}

func joinPath(part, current string) string {
	if part == "" {
		return current
	}
	if current == "" {
		return part
	}
	if current[0] == '[' {
		return part + current
	}
	return part + "." + current
}

// An InvalidUnmarshalError describes an invalid argument passed to Deserialize.
// (The argument to Deserialize must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "serdes: Deserialize(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "serdes: Deserialize(non-pointer " + e.Type.String() + ")"
	}
	return "serdes: Deserialize(nil " + e.Type.String() + ")"
}

func nextType(src []byte) string {
	src = skipWS(src)
	if len(src) == 0 {
		return "EOF"
	}
	switch src[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "bool"
	case 'n':
		return "null"
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return "number"
	default:
		return "unknown"
	}
}
