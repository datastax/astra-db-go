package datatypes

import (
	"fmt"
	"reflect"
	"testing"
)

func TestInlined(t *testing.T) {
	s := NewSet[string]()

	typ := reflect.TypeOf(s)
	implTyp := reflect.TypeOf(s.m().sortedMap)

	fmt.Printf("Struct: %s\n", typ)
	fmt.Printf("  Total Size:      %d bytes\n", typ.Size())
	fmt.Printf("  Field 0 (*impl): %d bytes, Offset: %d\n", typ.Field(0).Type.Size(), typ.Field(0).Offset)

	fmt.Printf("\nIs Struct Size == Pointer Size? %v\n", typ.Size() == implTyp.Size())
}

func TestInlined2(t *testing.T) {
	typ := reflect.TypeOf(linkedMap[string, int]{})

	fmt.Printf("Struct: %s\n", typ)
	fmt.Printf("Total Size: %d bytes\n\n", typ.Size())

	fmt.Println("Field Offsets:")
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fmt.Printf("  %-6s | Offset: %2d | Size: %2d | Type: %s\n",
			f.Name, f.Offset, f.Type.Size(), f.Type)
	}
}
