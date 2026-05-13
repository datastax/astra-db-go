package datatypes

import (
	"fmt"
	"reflect"
	"testing"
)

func TestInlined(t *testing.T) {
	s := Set[string]{data: make(map[string]struct{})}

	typ := reflect.TypeOf(s)
	mapTyp := reflect.TypeOf(s.data)

	fmt.Printf("Struct: %s\n", typ)
	fmt.Printf("  Total Size:      %d bytes\n", typ.Size())
	fmt.Printf("  Field 0 (map):   %d bytes, Offset: %d\n", typ.Field(0).Type.Size(), typ.Field(0).Offset)
	fmt.Printf("  Field 1 ([0]T):  %d bytes, Offset: %d\n", typ.Field(1).Type.Size(), typ.Field(1).Offset)

	fmt.Printf("\nIs Struct Size == Map Size? %v\n", typ.Size() == mapTyp.Size())
}

func TestInlined2(t *testing.T) {
	typ := reflect.TypeOf(orderedMap[string, int]{})

	fmt.Printf("Struct: %s\n", typ)
	fmt.Printf("Total Size: %d bytes\n\n", typ.Size())

	fmt.Println("Field Offsets:")
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fmt.Printf("  %-6s | Offset: %2d | Size: %2d | Type: %s\n",
			f.Name, f.Offset, f.Type.Size(), f.Type)
	}
}
