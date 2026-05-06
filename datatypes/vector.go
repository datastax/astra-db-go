package datatypes

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// DataAPIVector represents a vector that marshals to the {$binary: "..."} format.
// Internally it stores either a []float32 or a base64 string.
type DataAPIVector struct {
	floats []float32
	b64    string
	isB64  bool
}

// NewVector creates a DataAPIVector from a []float32 or a base64-encoded string.
// If v is not a supported type, returns zero value vector.
//
// Example usage:
//
//	v1 := datatypes.NewVector([]float32{0.1, 0.2, 0.3})
//	v2 := datatypes.NewVector("PaPXCr8euFI+x64U")
func NewVector(v any) DataAPIVector {
	switch val := v.(type) {
	case []float32:
		return DataAPIVector{floats: val}
	case string:
		return DataAPIVector{b64: val, isB64: true}
	case DataAPIVector:
		return val
	default:
		return DataAPIVector{}
	}
}

// Dimension returns the number of float32 elements in the vector.
func (v DataAPIVector) Dimension() int {
	if !v.isB64 {
		return len(v.floats)
	}
	trimmed := strings.TrimRight(v.b64, "=")
	return len(trimmed) * 3 / 16
}

// AsFloatArray returns the vector as a []float32, decoding from base64 if necessary.
func (v DataAPIVector) AsFloatArray() ([]float32, error) {
	if !v.isB64 {
		return v.floats, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(v.b64)
	if err != nil {
		return nil, fmt.Errorf("datatypes: base64 decode: %w", err)
	}
	if len(decoded)%4 != 0 {
		return nil, fmt.Errorf("datatypes: invalid binary length: %d, must be a multiple of 4", len(decoded))
	}
	floats := make([]float32, len(decoded)/4)
	for i := 0; i < len(decoded); i += 4 {
		bits := binary.BigEndian.Uint32(decoded[i : i+4])
		floats[i/4] = math.Float32frombits(bits)
	}
	return floats, nil
}

// AsBase64 returns the vector as a base64-encoded binary string, encoding from
// floats if necessary.
func (v DataAPIVector) AsBase64() string {
	if v.isB64 {
		return v.b64
	}
	return string(floatsToBase64(nil, v.floats))
}

func (v DataAPIVector) AppendBase64(dst []byte) []byte {
	if v.isB64 {
		return append(dst, v.b64...)
	}
	return floatsToBase64(dst, v.floats)
}

// floatsToBase64 encodes a []float32 as a big-endian base64 string.
func floatsToBase64(dst []byte, floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.BigEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return base64.StdEncoding.AppendEncode(dst, buf)
}

// MarshalJSON produces the {$binary: "..."} format, using the stored base64
// directly if available.
func (v DataAPIVector) MarshalJSON() ([]byte, error) {
	encoded := v.AsBase64()
	return []byte(fmt.Sprintf(`{"$binary":"%s"}`, encoded)), nil
}

// UnmarshalJSON parses either {$binary: "..."} (stored as base64) or a raw
// float array (stored as floats), preserving the original representation.
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
		v.floats = floats
		// Probably redundant since these are the zer values, but in case something was
		// previously set, we want to clear it out.
		v.b64 = ""
		v.isB64 = false
	case '{':
		// {$binary: "..."} format — store the base64 string directly
		var wrapper struct {
			Binary string `json:"$binary"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			return fmt.Errorf("unmarshal DataAPIVector: %w", err)
		}
		v.b64 = wrapper.Binary
		v.isB64 = true
		v.floats = nil
	default:
		return fmt.Errorf("unmarshal DataAPIVector: unexpected format (first byte: %q)", trimmed[0])
	}
	return nil // All good
}
