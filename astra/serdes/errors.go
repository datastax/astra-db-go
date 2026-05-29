package serdes

import (
	"errors"
	"fmt"
	"reflect"
)

type pathContext struct {
	Struct string
	Field  string
}

func (p *pathContext) setField(part string) { p.Field = joinPath(part, p.Field) }
func (p *pathContext) setStruct(name string) {
	if p.Struct == "" {
		p.Struct = name
	}
}

func (p *pathContext) fullPath() string {
	return joinPath(p.Struct, p.Field)
}

func (p *pathContext) formatPath() string {
	if path := p.fullPath(); path != "" {
		return " of field " + path
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

	return fmt.Errorf("field %s: %w", part, err)
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
	msg    string // description of error
	Offset int64  // error occurred after reading Offset bytes
	Err    error  // underlying error
}

func (e *SyntaxError) Error() string {
	msg := "serdes: syntax error"
	if e.Offset >= 0 && !hasSerdesOffset(e.Err) {
		msg += fmt.Sprintf(" at offset %d", e.Offset)
	}
	msg += e.formatPath() + ": " + e.msg
	if e.Err != nil {
		msg += ": " + displayError(e.Err)
	}
	return msg
}

func (e *SyntaxError) Unwrap() error { return e.Err }

// An UnmarshalTypeError describes a JSON/BSON value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	pathContext
	Value  string       // description of JSON/BSON value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Offset int64        // error occurred after reading Offset bytes
	Err    error        // underlying error
}

func (e *UnmarshalTypeError) Error() string {
	msg := "serdes: cannot unmarshal " + e.Value + " into Go value" + e.formatPath() + " of type " + e.Type.String()
	if e.Err != nil {
		msg += ": " + displayError(e.Err)
	}
	if e.Offset >= 0 && !hasSerdesOffset(e.Err) {
		msg += fmt.Sprintf(" at offset %d", e.Offset)
	}
	return msg
}

func (e *UnmarshalTypeError) Unwrap() error { return e.Err }

// A MarshalerError represents an error from calling a MarshalAstra method.
type MarshalerError struct {
	pathContext
	Type reflect.Type
	Err  error
}

func (e *MarshalerError) Error() string {
	return "serdes: error calling MarshalAstra" + e.formatPath() + " of type " + e.Type.String() + ": " + displayError(e.Err)
}

func (e *MarshalerError) Unwrap() error { return e.Err }

// An UnmarshalerError represents an error from calling an UnmarshalAstra method.
type UnmarshalerError struct {
	pathContext
	Type reflect.Type
	Err  error
}

func (e *UnmarshalerError) Error() string {
	return "serdes: error calling UnmarshalAstra" + e.formatPath() + " of type " + e.Type.String() + ": " + displayError(e.Err)
}

func (e *UnmarshalerError) Unwrap() error { return e.Err }

// An UnsupportedValueError is returned when a value is not supported by Astra,
// such as a cycle, a missing tag, or a target mismatch.
type UnsupportedValueError struct {
	pathContext
	Value any
	Msg   string
}

func (e *UnsupportedValueError) Error() string {
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

func (e *UnsupportedTypeError) Error() string {
	return "serdes: unsupported type: " + e.Type.String() + e.formatPath()
}

func displayError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) >= 8 && s[:8] == "serdes: " {
		return s[8:]
	}
	return s
}

func hasSerdesOffset(err error) bool {
	if err == nil {
		return false
	}
	var se *SyntaxError
	if errors.As(err, &se) {
		return se.Offset >= 0
	}
	var te *UnmarshalTypeError
	if errors.As(err, &te) {
		return te.Offset >= 0
	}
	return false
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

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "serdes: Deserialize(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "serdes: Deserialize(non-pointer " + e.Type.String() + ")"
	}
	return "serdes: Deserialize(nil " + e.Type.String() + ")"
}

func getValueType(src []byte) string {
	if len(src) == 0 {
		return "eof"
	}
	src = skipWS(src)
	if len(src) == 0 {
		return "eof"
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
