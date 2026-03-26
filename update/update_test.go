package update_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/datastax/astra-db-go/update"
)

// cleanString removes all whitespace characters from a string.
func cleanString(s string) string {
	// Use a regular expression to replace all whitespace characters (including spaces, tabs, newlines)
	// with an empty string.
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}

// assertJSONEqual marshals the given arguments to JSON and compares
// them to the expected string. Whitespace isn't taken into account.
func assertJSONEqual(t *testing.T, expected string, args ...any) {
	for _, arg := range args {
		argJSON, err := json.Marshal(arg)
		if err != nil {
			t.Fatalf("failed to marshal argument: %v", err)
		}
		if cleanString(string(argJSON)) != cleanString(expected) {
			t.Errorf("\nGOT:\n%s\n\nWANT:\n%s", argJSON, expected)
			return
		}
	}
}

// Example taken from:
// https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/update-many.html#update-multiple-properties
// But - changed order since json.Marshal will order keys alphabetically.
const exampleFromDocs = `{
	"$inc": {
		"age": 1
	},
	"$set": {
		"classes": ["biology", "algebra", "swimming"],
		"color": "blue"
	},
	"$unset": {
		"phone": ""
	}
}`

// This test verifies both direct map / fluent produce the same JSON output as the example from the docs.
func TestUpdateManyExample(t *testing.T) {
	u := update.U{
		"$set": map[string]any{
			"color":   "blue",
			"classes": []string{"biology", "algebra", "swimming"},
		},
		"$unset": map[string]any{
			"phone": "",
		},
		"$inc": map[string]any{
			"age": 1,
		},
	}
	j, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}
	if string(j) != cleanString(exampleFromDocs) {
		t.Errorf("\nGOT:\n%s\n\nWANT:\n%s", j, cleanString(exampleFromDocs))
	}
	// Now let's do the same thing for the fluent builder API, which should produce the same result.
	fluentExample := update.Set("color", "blue").
		Set("classes", []string{"biology", "algebra", "swimming"}).
		Unset("phone").
		Inc("age", 1)
	j, err = json.Marshal(fluentExample)
	if err != nil {
		t.Fatalf("failed to marshal fluent update: %v", err)
	}
	if string(j) != cleanString(exampleFromDocs) {
		t.Errorf("\nGOT:\n%s\n\nWANT:\n%s", j, cleanString(exampleFromDocs))
	}
}

func TestUpdateJSONMarshal(t *testing.T) {
	tests := []struct {
		expectedJSON string
		fluent       *update.Updater
		raw          update.U
	}{
		{
			`{"$set":{"age":30,"name":"Bob"}}`,
			update.Set("name", "Bob").Set("age", 30),
			update.U{"$set": update.U{"name": "Bob", "age": 30}},
		},
		{
			`{"$unset":{"age":"","name":""}}`,
			update.Unset("name", "age"),
			update.U{"$unset": update.U{"name": "", "age": ""}},
		},
		{
			`{"$inc":{"counter":5}}`,
			update.Inc("counter", 5),
			update.U{"$inc": update.U{"counter": 5}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.expectedJSON, func(t *testing.T) {
			assertJSONEqual(t, tt.expectedJSON, tt.fluent, tt.raw)
		})
	}
}

func TestUpdaterZeroValuePanic(t *testing.T) {
	// This test just verifies that a zero-value Update doesn't panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()
	u := update.Updater{}
	u.Set("BestDB", "Astra")
}
