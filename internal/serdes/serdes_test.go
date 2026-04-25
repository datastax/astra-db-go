package serdes

import "testing"

type data struct {
	Num int
}

func TestSer(t *testing.T) {
	b, _ := Serialize(`"foo"`)
	t.Logf("Serialized: %s", b)
}

func TestDes(t *testing.T) {
	var d *string
	err := Deserialize([]byte(`"foo"`), &d)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}
	t.Logf("Deserialized: %#v", *d)
}
