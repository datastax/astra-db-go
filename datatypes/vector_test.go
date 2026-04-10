package datatypes_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/internal/testutils"
)

func dataAPIVectorTests() []testutils.JSONTestCase {
	return []testutils.JSONTestCase{{
		// https://docs.datastax.com/en/astra-db-serverless/api-reference/sort-rows.html#example-sorting-against-a-search-vector
		Name:     "example from docs",
		Expected: `{"$binary":"PaPXCr8euFI+x64U"}`,
		Args: []any{
			datatypes.NewVector([]float32{0.08, -0.62, 0.39}),
		},
	}, {
		// https://github.com/datastax/astra-db-csharp/issues/67
		Name:     "example from .NET client issue",
		Expected: `{"$binary":"PczMzb5MzM0+mZma"}`,
		Args: []any{
			datatypes.NewVector([]float32{0.10000000149011612, -0.20000000298023224, 0.30000001192092896}),
		},
	}, {
		Name:     "empty vector",
		Expected: `{"$binary":""}`,
		Args: []any{
			datatypes.NewVector([]float32{}),
		},
	}}
}

func TestDataAPIVectorMarshalJSON(t *testing.T) {
	tests := dataAPIVectorTests()
	testutils.RunJSONTestCases(t, tests)
}

func TestDataAPIVectorUnmarshalJSON(t *testing.T) {
	// Re-using this test case struct even though the names are SLIGHTLY off in this context.
	tests := dataAPIVectorTests()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var v datatypes.DataAPIVector
			err := json.Unmarshal([]byte(tt.Expected), &v)
			if err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}
			got, err := v.AsFloatArray()
			if err != nil {
				t.Fatalf("AsFloatArray error: %v", err)
			}
			expected, _ := tt.Args[0].(datatypes.DataAPIVector).AsFloatArray()
			if !slices.Equal(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	}
	testutils.RunJSONTestCases(t, tests)
}

type testDocument struct {
	Title               string                  `json:"title"`
	Author              string                  `json:"author"`
	SummaryGenresVector datatypes.DataAPIVector `json:"summary_genres_vector"`
}

// Example:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/row-methods/insert-one.html#example-vector
//
//	{
//	  "insertOne": {
//	    "document": {
//	      "title": "Computed Wilderness",
//	      "author" :"Ryan Eau",
//	      "summary_genres_vector": {"$binary": "PaPXCr8euFI+x64U"}
//	    }
//	  }
//	}
func TestDataAPIVectorTableDocsExample(t *testing.T) {
	// insertOnePayload is the payload for insertOne commands.
	type insertOnePayload struct {
		Document any `json:"document"`
	}
	cmd := testutils.NewTestCmd("insertOne", insertOnePayload{
		Document: testDocument{
			Title:               "Computed Wilderness",
			Author:              "Ryan Eau",
			SummaryGenresVector: datatypes.NewVector([]float32{0.08, -0.62, 0.39}),
		},
	})
	expected := `{"insertOne":{"document":{"title":"Computed Wilderness","author":"Ryan Eau","summary_genres_vector":{"$binary":"PaPXCr8euFI+x64U"}}}}`
	testutils.AssertJSONEqual(t, expected, cmd)
}

// TestDataAPIVectorWhitespace ensures that our defensive whitespace logic works.
func TestDataAPIVectorWhitespace(t *testing.T) {
	var v datatypes.DataAPIVector
	if err := v.UnmarshalJSON([]byte("  [0.08, -0.62, 0.39]")); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	expectedFloats := []float32{0.08, -0.62, 0.39}
	got, err := v.AsFloatArray()
	if err != nil {
		t.Fatalf("AsFloatArray error: %v", err)
	}
	if !slices.Equal(got, expectedFloats) {
		t.Errorf("got %v, want %v", got, expectedFloats)
	}
}

func TestDataAPIVectorNonBinary(t *testing.T) {
	// We are expecting both test cases to produce this
	expectedFloats := []float32{0.08, -0.62, 0.39}
	testCases := []string{
		// Binary
		`{"title":"Computed Wilderness","author":"Ryan Eau","summary_genres_vector": { "$binary": "PaPXCr8euFI+x64U"}}`,
		// Non-binary
		`{"title":"Computed Wilderness","author":"Ryan Eau","summary_genres_vector":   [ 0.08,-0.62,0.39]}`,
	}
	for _, tc := range testCases {
		v := testDocument{}
		json.Unmarshal([]byte(tc), &v)
		got, err := v.SummaryGenresVector.AsFloatArray()
		if err != nil {
			t.Fatalf("AsFloatArray error: %v", err)
		}
		if !slices.Equal(expectedFloats, got) {
			t.Errorf("\ngot: %v\nwant: %v", got, expectedFloats)
		}
	}
}

// TestDimension tests the Dimension method for both float and base64 representations.
func TestDimension(t *testing.T) {
	tests := []struct {
		name string
		vec  datatypes.DataAPIVector
		want int
	}{
		{"from floats - 3 elements", datatypes.NewVector([]float32{0.08, -0.62, 0.39}), 3},
		{"from floats - empty", datatypes.NewVector([]float32{}), 0},
		{"from floats - 1 element", datatypes.NewVector([]float32{1.0}), 1},
		{"from base64 - 3 elements", datatypes.NewVector("PaPXCr8euFI+x64U"), 3},
		{"from base64 - empty", datatypes.NewVector(""), 0},
		{"zero value", datatypes.DataAPIVector{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vec.Dimension(); got != tt.want {
				t.Errorf("Dimension() = %d, want %d", got, tt.want)
			}
		})
	}
}
