package datatypes

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// DataAPIVector is a wrapper for []float32 that marshals to the {$binary: "..."} format
type DataAPIVector struct {
	Values []float32
}

// MarshalJSON allows the struct to properly marshal to the {$binary: "..."}
// format when using json.Marshal()
func (v DataAPIVector) MarshalJSON() ([]byte, error) {
	// 4 bytes per float32
	buf := make([]byte, len(v.Values)*4)

	for i, f := range v.Values {
		// Convert float32 to bits and write as Big-Endian
		bits := math.Float32bits(f)
		binary.BigEndian.PutUint32(buf[i*4:], bits)
	}
	// base64 encode and wrap in the expected JSON structure
	encoded := base64.StdEncoding.EncodeToString(buf)
	return []byte(fmt.Sprintf(`{"$binary":"%s"}`, encoded)), nil
}

// UnmarshalJSON allows the struct to parse the {$binary: "..."} format
// back into a proper []float32 slice.
func (v *DataAPIVector) UnmarshalJSON(data []byte) error {
	// Peek at the first non-whitespace byte to determine the format. This is pretty defensive,
	// but, JSON allows leading spaces so we are trimming them.
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		// Nothing to do; will just be zero value.
		return nil
	}

	switch trimmed[0] {
	case '[':
		// Raw array of floats: [0.1, 0.2, 0.3]
		var floats []float32
		if err := json.Unmarshal(trimmed, &floats); err != nil {
			return fmt.Errorf("unmarshal DataAPIVector float array: %w", err)
		}
		v.Values = floats
	case '{':
		// Unmarshal the {$binary: "..."} format to extract the string value
		var wrapper struct {
			Binary string `json:"$binary"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			return fmt.Errorf("unmarshal DataAPIVector: %w", err)
		}
		// Decode the base64 string into a slice of bytes
		decoded, err := base64.StdEncoding.DecodeString(wrapper.Binary)
		if err != nil {
			return fmt.Errorf("base64 decode: %w", err)
		}
		// Each float32 is 4 bytes, so the total length should be a multiple of 4
		if len(decoded)%4 != 0 {
			return fmt.Errorf("invalid binary length: %d. must be a multiple of 4", len(decoded))
		}
		v.Values = make([]float32, len(decoded)/4)
		// Advance through the byte slice 4 bytes at a time, converting each chunk to a float32
		for i := 0; i < len(decoded); i += 4 {
			bits := binary.BigEndian.Uint32(decoded[i : i+4])
			v.Values[i/4] = math.Float32frombits(bits)
		}
	default:
		return fmt.Errorf("unmarshal DataAPIVector: unexpected format (first byte: %q)", trimmed[0])
	}
	return nil // All good
}
