package serdes

import (
	"testing"
)

func TestMapEncodingDebug(t *testing.T) {
	m := map[string]int{
		"math":    95,
		"english": 88,
	}

	data, err := Serialize(m)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	t.Logf("Map serialized: %s", string(data))
	
	// Should produce something like: {"math":95,"english":88}
	if len(data) < 10 {
		t.Errorf("Serialized map is too short: %s", string(data))
	}
}
