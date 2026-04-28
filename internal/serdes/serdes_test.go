package serdes

import "testing"

func TestSerdesAny_Collection(t *testing.T) {
	str := `{"$vector":[0.1, 0.2, 0.3],"nested":{"$uuid":"123e4567-e89b-12d3-a456-426614174000"}}}`
	//str := `{"$vector":[0.1, 0.2, 0.3]}`
	var dst any

	if err := Deserialize([]byte(str), &dst, CollectionTarget); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	t.Logf("deserialized: %#v", dst)
}
