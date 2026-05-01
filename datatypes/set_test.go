package datatypes_test

import (
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
)

type setTestCase struct {
	Name string
	Age  int
}

func TestSet_Add(t *testing.T) {
	s := datatypes.Set[setTestCase]()
	// Double-add to make sure comparable is working.
	s.Add(setTestCase{"Alice", 23})
	s.Add(setTestCase{"Alice", 23})
	if !s.Has(setTestCase{"Alice", 23}) {
		t.Errorf("Set should have Alice, but does not")
	}
	if s.Len() != 1 {
		t.Errorf("Set length should be 1, got %d", s.Len())
	}
	// Add another value and make sure len grows
	s.Add(setTestCase{"Bob", 19})
	if s.Len() != 2 {
		t.Errorf("Set length should be 1, got %d", s.Len())
	}
}
