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

		expected := `serdes: cannot unmarshal string into Go value of field Address.addresses[0].zip of type int: syntax error at offset 60: expected integer`
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
			if typeErr.Offset != 60 {
				t.Errorf("expected Offset 60, got %d", typeErr.Offset)
			}
			if typeErr.Struct != "Address" {
				t.Errorf("expected Struct 'Address', got %q", typeErr.Struct)
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

		expected := `serdes: syntax error at offset 63 of field User.addresses[0]: expected ',' after field value`
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

		expected := `serdes: cannot unmarshal eof into Go value of field User.addresses[0] of type serdes.Address at offset 31`
		if err.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, err.Error())
		}
	})

	t.Run("Nested serdes errors - prefix suppression", func(t *testing.T) {
		m := &mockUnmarshaler{err: &SyntaxError{msg: "inner problem", Offset: 5}}
		data := []byte(`"some-value"`)
		err := Deserialize(data, m, nil, TargetCollection)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Should not have "serdes: serdes:"
		expected := "serdes: error calling UnmarshalAstra of field mockUnmarshaler of type serdes.mockUnmarshaler: syntax error at offset 5: inner problem"
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

		expected := `serdes: cannot unmarshal string into Go value of field User of type serdes.User at offset 0`
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

		expected := "serdes: error calling UnmarshalAstra of field mockUnmarshaler of type serdes.mockUnmarshaler: custom failure"
		if unmarshalErr.Error() != expected {
			t.Errorf("\nexpected: %s\ngot:      %s", expected, unmarshalErr.Error())
		}
	})
}
