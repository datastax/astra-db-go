// Copyright IBM Corp.

package ptr_test

import (
	"testing"
	"time"

	"github.com/datastax/astra-db-go/astra/ptr"
)

func TestTo(t *testing.T) {
	// Test with bool
	b := ptr.To(true)
	if *b != true {
		t.Errorf("To(true) = %v; want true", *b)
	}

	// Test with int
	i := ptr.To(100)
	if *i != 100 {
		t.Errorf("To(100) = %v; want 100", *i)
	}
}

func TestFrom(t *testing.T) {
	// Test with existing value
	val := 42
	if got := ptr.From(&val); got != 42 {
		t.Errorf("From(&42) = %v; want 42", got)
	}

	// Test with nil (should return zero value)
	var nilInt *int
	if got := ptr.From(nilInt); got != 0 {
		t.Errorf("From(nil) for int = %v; want 0", got)
	}
}

func TestFromWithDefault(t *testing.T) {
	defaultValue := "default"

	t.Run("returns value when pointer is not nil", func(t *testing.T) {
		val := "hello"
		if got := ptr.FromWithDefault(&val, defaultValue); got != "hello" {
			t.Errorf("got %v; want hello", got)
		}
	})

	t.Run("returns default when pointer is nil", func(t *testing.T) {
		if got := ptr.FromWithDefault(nil, defaultValue); got != "default" {
			t.Errorf("got %v; want default", got)
		}
	})

	t.Run("test with time.Time to mimic options", func(t *testing.T) {
		interval := ptr.FromWithDefault(nil, 5*time.Second)
		if interval != 5*time.Second {
			t.Errorf("got %v; want 5s", interval)
		}
	})
}
