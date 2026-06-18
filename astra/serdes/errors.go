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

func (p pathContext) fullPath() string {
	return joinPath(p.Struct, p.Field)
}

func (p pathContext) formatPath() string {
	if path := p.fullPath(); path != "" {
		return " parsing '" + path + "'"
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

type DecodeAction int

const (
	DecodeActionSyntax DecodeAction = iota
	DecodeActionTypeMismatch
	DecodeActionCustomUnmarshaler
	DecodeActionUnsupported
	DecodeActionInvalid
)

type DecodeError struct {
	pathContext
	Action  DecodeAction
	Msg     string
	Value   string
	Type    reflect.Type
	Snippet string
	Offset  int64
	Err     error
}

func (e *DecodeError) diagnostic() string {
	var diag string
	if e.Action == DecodeActionSyntax && e.Msg != "" {
		diag = e.Msg
	}

	if e.Err != nil {
		if innerDiag := displayError(e.Err); innerDiag != "" {
			if diag != "" {
				diag += ": " + innerDiag
			} else {
				diag = innerDiag
			}
		}
	}

	if e.Action == DecodeActionTypeMismatch {
		typeName := getValueName(e.Type)
		if strings.Contains(diag, "expected "+e.Type.String()) || strings.Contains(diag, "expected "+typeName) {
			return ""
		}
	}
	return diag
}

func (e *DecodeError) Error() string {
	msg := "serdes: "

	switch e.Action {
	case DecodeActionInvalid:
		if e.Type == nil {
			return msg + "Deserialize(nil)"
		}
		if e.Type.Kind() != reflect.Ptr {
			return msg + "Deserialize(non-pointer " + e.Type.String() + ")"
		}
		return msg + "Deserialize(nil " + e.Type.String() + ")"

	case DecodeActionUnsupported:
		msg += "unsupported value"
		if e.Msg != "" {
			msg += ": " + e.Msg
		}
		return msg + e.formatPath()

	case DecodeActionTypeMismatch:
		msg += "cannot unmarshal " + e.Value
		path := e.fullPath()
		if path != "" && path != e.Type.String() && path != e.Type.Name() {
			msg += " into '" + path + "' (type " + e.Type.String() + ")"
		} else {
			msg += " into type " + e.Type.String()
		}

	case DecodeActionCustomUnmarshaler:
		msg += "error calling UnmarshalAstra"
		path := e.fullPath()
		if path != "" && path != e.Type.String() && path != e.Type.Name() {
			msg += " in '" + path + "'"
		} else {
			msg += " for type " + e.Type.String()
		}

	case DecodeActionSyntax:
		msg += "syntax error"
		if path := e.fullPath(); path != "" {
			msg += " parsing '" + path + "'"
		}
	}

	if diag := e.diagnostic(); diag != "" {
		msg += ": " + diag
	}

	return withSnippet(e, msg)
}

func (e *DecodeError) Unwrap() error { return e.Err }

type EncodeAction int

const (
	EncodeActionCustomMarshaler EncodeAction = iota
	EncodeActionUnsupportedValue
	EncodeActionUnsupportedType
)

type EncodeError struct {
	pathContext
	Action EncodeAction
	Msg    string
	Type   reflect.Type
	Err    error
}

func (e *EncodeError) diagnostic() string {
	return displayError(e.Err)
}

func (e *EncodeError) Error() string {
	msg := "serdes: "

	switch e.Action {
	case EncodeActionUnsupportedType:
		return msg + "unsupported type: " + e.Type.String() + e.formatPath()

	case EncodeActionUnsupportedValue:
		msg += "unsupported value"
		if e.Msg != "" {
			msg += ": " + e.Msg
		}
		return msg + e.formatPath()

	case EncodeActionCustomMarshaler:
		msg += "error calling MarshalAstra"
		path := e.fullPath()
		if path != "" && path != e.Type.String() && path != e.Type.Name() {
			msg += " in '" + path + "'"
		} else {
			msg += " for type " + e.Type.String()
		}
		if e.Err != nil {
			msg += ": " + e.diagnostic()
		}
		return msg
	}

	return msg
}

func escapeControlChars(s string) string {
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\b", `\b`)
	s = strings.ReplaceAll(s, "\f", `\f`)
	return s
}

func (e *EncodeError) Unwrap() error { return e.Err }

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

func errorSnippet(c DecodeCtx, src []byte) string {
	if c.payload == nil || len(*c.payload) == 0 {
		return ""
	}

	payload := *c.payload
	offset := len(payload) - len(src)

	ctxLen := 10
	if c.Flags&ExtendedErrorSnippet != 0 {
		ctxLen = 32
	}

	start := offset - ctxLen
	if start < 0 {
		start = 0
	}
	end := offset + ctxLen
	if end > len(payload) {
		end = len(payload)
	}

	before := string(payload[start:offset])
	after := string(payload[offset:end])

	before = escapeControlChars(before)
	after = escapeControlChars(after)

	res := ""
	if start > 0 {
		res += "..."
	}
	if offset == len(payload) {
		res += before + "<EOF>"
	} else {
		res += before + after
	}
	if end < len(payload) {
		res += "..."
	}

	return res
}

func innermostOffset(err error) int64 {
	var offset int64 = -1
	for err != nil {
		if de, ok := err.(*DecodeError); ok {
			if de.Offset >= 0 {
				offset = de.Offset
			}
		}
		err = errors.Unwrap(err)
	}
	return offset
}

func innermostSnippet(err error) string {
	var snippet string
	for err != nil {
		if de, ok := err.(*DecodeError); ok {
			if s := de.Snippet; s != "" {
				snippet = s
			}
		}
		err = errors.Unwrap(err)
	}
	return snippet
}

func withSnippet(err error, msg string) string {
	if s := innermostSnippet(err); s != "" && !strings.Contains(msg, " near '"+s+"'") {
		msg += " near '" + s + "'"
	}
	if o := innermostOffset(err); o >= 0 && !strings.Contains(msg, " (offset") {
		msg += fmt.Sprintf(" (offset %d)", o)
	}
	return msg
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

func nextJsonType(src []byte) string {
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

func errorOffset(c DecodeCtx, src []byte) int64 {
	if c.payload == nil {
		return -1
	}
	return int64(len(*c.payload) - len(src))
}

func (c DecodeCtx) syntaxError(src []byte, msg string) error {
	return &DecodeError{Action: DecodeActionSyntax, Msg: msg, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src)}
}

func (c DecodeCtx) syntaxErrorWrap(src []byte, msg string, err error) error {
	return &DecodeError{Action: DecodeActionSyntax, Msg: msg, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src), Err: err}
}

func (c DecodeCtx) unmarshalTypeError(src []byte, t reflect.Type) error {
	return &DecodeError{Action: DecodeActionTypeMismatch, Value: nextJsonType(src), Type: t, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src)}
}

func (c DecodeCtx) unmarshalValueTypeError(src []byte, t reflect.Type, value string) error {
	return &DecodeError{Action: DecodeActionTypeMismatch, Value: value, Type: t, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src)}
}

func (c DecodeCtx) unmarshalTypeErrorWrap(src []byte, t reflect.Type, err error) error {
	return &DecodeError{Action: DecodeActionTypeMismatch, Value: nextJsonType(src), Type: t, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src), Err: err}
}

func (c DecodeCtx) unmarshalValueTypeErrorWrap(src []byte, t reflect.Type, value string, err error) error {
	return &DecodeError{Action: DecodeActionTypeMismatch, Value: value, Type: t, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src), Err: err}
}

func (c DecodeCtx) unsupportedValueError(src []byte, msg string) error {
	return &DecodeError{Action: DecodeActionUnsupported, Msg: msg, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src)}
}

func (c DecodeCtx) customUnmarshalerError(src []byte, t reflect.Type, err error) error {
	return &DecodeError{Action: DecodeActionCustomUnmarshaler, Type: t, Err: err, Snippet: errorSnippet(c, src), Offset: errorOffset(c, src)}
}

func (c EncodeCtx) unsupportedTypeError(t reflect.Type) error {
	return &EncodeError{Action: EncodeActionUnsupportedType, Type: t}
}

func (c EncodeCtx) unsupportedValueError(msg string) error {
	return &EncodeError{Action: EncodeActionUnsupportedValue, Msg: msg}
}

func (c EncodeCtx) customMarshalerError(t reflect.Type, err error) error {
	return &EncodeError{Action: EncodeActionCustomMarshaler, Type: t, Err: err}
}
