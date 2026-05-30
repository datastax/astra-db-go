package serdes

import (
	"errors"
	"reflect"
	"testing"
)

func TestErrorMessages(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		Zip    int    `json:"zip"`
	}
	type User struct {
		Name      string    `json:"name"`
		Addresses []Address `json:"addresses"`
	}

	t.Run("UnmarshalTypeError with path", func(t *testing.T) {
		data := []byte(`{"name": "John", "addresses": [{"street": "Main St", "zip": "not-a-number"}]}`)
		var user User
		err := Deserialize(data, &user, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := `serdes: cannot unmarshal string into 'User.addresses[0].zip' (type int) near: '"not-a-number"}]...'`
		if err.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, err.Error())
		}

		var typeErr *UnmarshalTypeError
		if !errors.As(err, &typeErr) {
			t.Error("expected error to be *UnmarshalTypeError")
		} else {
			if typeErr.Value != "string" {
				t.Errorf("expected Value 'string', got %q", typeErr.Value)
			}
			if typeErr.Type != reflect.TypeFor[int]() {
				t.Errorf("expected Type 'int', got %v", typeErr.Type)
			}
			if typeErr.Struct != "User" {
				t.Errorf("expected Struct 'User', got %q", typeErr.Struct)
			}
			if typeErr.Field != "addresses[0].zip" {
				t.Errorf("expected Field 'addresses[0].zip', got %q", typeErr.Field)
			}
		}
	})

	t.Run("SyntaxError with offset", func(t *testing.T) {
		data := []byte(`{"name": "John", "addresses": [{"street": "Main St", "zip": 123]}`) // missing '}' before ']'
		var user User
		err := Deserialize(data, &user, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := `serdes: syntax error in 'User.addresses[0]': expected ',' after field value near: ']}'`
		if err.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, err.Error())
		}
	})

	t.Run("Custom Unmarshaler Error", func(t *testing.T) {
		_ = AstraUnmarshaler(nil) // check interface
	})
	t.Run("EOF SyntaxError", func(t *testing.T) {
		data := []byte(`{"name": "John", "addresses": [`)
		var user User
		err := Deserialize(data, &user, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := `serdes: cannot unmarshal EOF into 'User.addresses[0]' (type serdes.Address): expected '{' at the start of an object`
		if err.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, err.Error())
		}
	})

	t.Run("Nested serdes errors - prefix suppression", func(t *testing.T) {
		m := &mockUnmarshaler{err: &SyntaxError{msg: "inner problem"}}
		data := []byte(`"some-value"`)
		err := Deserialize(data, m, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Should not have "serdes: serdes:" or redundant "syntax error:"
		expected := "serdes: error calling UnmarshalAstra for type serdes.mockUnmarshaler: inner problem near: '\"some-value\"'"
		if err.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, err.Error())
		}
	})

	t.Run("Path Precision - root level", func(t *testing.T) {
		data := []byte(`"not-an-object"`)
		var user User
		err := Deserialize(data, &user, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expected := `serdes: cannot unmarshal string into type serdes.User: expected '{' at the start of an object near: '"not-an-object"'`
		if err.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, err.Error())
		}
	})
}

type mockUnmarshaler struct {
	err error
}

func (m *mockUnmarshaler) UnmarshalAstra(ctx DecodeCtx, value any) error {
	return m.err
}

func TestCustomUnmarshalerError(t *testing.T) {
	t.Run("wraps custom error", func(t *testing.T) {
		m := &mockUnmarshaler{err: errors.New("custom failure")}
		data := []byte(`"some-value"`)
		err := Deserialize(data, m, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var unmarshalErr *UnmarshalerError
		if !errors.As(err, &unmarshalErr) {
			t.Fatalf("expected *UnmarshalerError, got %T: %v", err, err)
		}

		if unmarshalErr.Err.Error() != "custom failure" {
			t.Errorf("expected wrapped error 'custom failure', got %v", unmarshalErr.Err)
		}

		expected := "serdes: error calling UnmarshalAstra for type serdes.mockUnmarshaler: custom failure near: '\"some-value\"'"
		if unmarshalErr.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, unmarshalErr.Error())
		}
	})
}
