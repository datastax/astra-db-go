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
	fluentExample := update.Coll().Set("color", "blue").
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

func TestUpdateFieldOperators(t *testing.T) {
	// This test takes many examples from the docs and verifies that fluent/map both produce the same JSON:
	// https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operator-collections.html
	// Note that the examples in the docs aren't always in the same order as the JSON output from our code,
	// since json.Marshal orders keys alphabetically.
	tests := []struct {
		expectedJSON string
		fluent       *update.CollectionUpdateBuilder
		raw          update.U
	}{
		{
			`{"$set":{"number_of_pages": 423, "rating": 4.5}}`,
			update.Coll().Set("number_of_pages", 423).Set("rating", 4.5),
			update.U{"$set": update.U{"number_of_pages": 423, "rating": 4.5}},
		},
		{
			`{"$setOnInsert":{"is_checked_out":false,"rating":5}}`,
			update.Coll().SetOnInsert("rating", 5).SetOnInsert("is_checked_out", false),
			update.U{"$setOnInsert": update.U{"is_checked_out": false, "rating": 5}},
		},
		{
			`{"$unset":{"borrower": "","due_date": ""}}`,
			update.Coll().Unset("borrower", "due_date"),
			update.U{"$unset": update.U{"borrower": "", "due_date": ""}},
		},
		{
			`{"$currentDate":{"due_date":true}}`,
			update.Coll().CurrentDate("due_date"),
			update.U{"$currentDate": update.U{"due_date": true}},
		},
		{
			`{"$inc":{"number_of_pages":25}}`,
			update.Coll().Inc("number_of_pages", 25),
			update.U{"$inc": update.U{"number_of_pages": 25}},
		},
		{
			`{"$min":{"rating":3.9}}`,
			update.Coll().Min("rating", 3.9),
			update.U{"$min": update.U{"rating": 3.9}},
		},
		{
			`{"$max":{"rating":3.9}}`,
			update.Coll().Max("rating", 3.9),
			update.U{"$max": update.U{"rating": 3.9}},
		},
		{
			`{"$mul":{"rating":1.2}}`,
			update.Coll().Mul("rating", 1.2),
			update.U{"$mul": update.U{"rating": 1.2}},
		},
		{
			`{"$rename":{"old_field":"new_field","other_old_field":"other_new_field"}}`,
			update.Coll().Rename("old_field", "new_field").Rename("other_old_field", "other_new_field"),
			update.U{"$rename": update.U{"old_field": "new_field", "other_old_field": "other_new_field"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.expectedJSON, func(t *testing.T) {
			assertJSONEqual(t, tt.expectedJSON, tt.fluent, tt.raw)
		})
	}
}

func TestUpdateArrayOperators(t *testing.T) {
	// Mix of docs examples and some additional chaining examples.
	// https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operator-collections.html#add-to-set
	tests := []struct {
		expectedJSON string
		fluent       *update.CollectionUpdateBuilder
		raw          update.U
	}{
		{
			`{"$addToSet":{"genres":"SciFi"}}`, // Docs example
			update.Coll().AddToSet("genres", "SciFi"),
			update.U{"$addToSet": update.U{"genres": "SciFi"}},
		},
		{
			`{"$pop":{"genres":-1}}`, // Docs example
			update.Coll().Pop("genres", -1),
			update.U{"$pop": update.U{"genres": -1}},
		},
		{
			`{"$push":{"genres":"SciFi"}}`, // Docs example
			update.Coll().Push("genres", "SciFi"),
			update.U{"$push": update.U{"genres": "SciFi"}},
		},
		{
			`{"$push":{"genres":{"$each":["Mystery","Fiction"]}}}`, // Docs example
			update.Coll().PushEach("genres", "Mystery", "Fiction"),
			update.U{"$push": update.U{"genres": map[string]any{"$each": []string{"Mystery", "Fiction"}}}},
		},
		{
			`{"$push":{"genres":{"$each":["Mystery","Fiction"],"$position": 3}}}`, // Docs example
			update.Coll().PushEachPosition("genres", 3, "Mystery", "Fiction"),
			update.U{"$push": update.U{"genres": map[string]any{"$each": []string{"Mystery", "Fiction"}, "$position": 3}}},
		},
		{
			`{"$pop":{"genres":1}}`, // Test pop with value 1 to remove last item in array
			update.Coll().Pop("genres", 1),
			update.U{"$pop": update.U{"genres": 1}},
		},
		{
			`{"$addToSet":{"tags":{"$each":["a","b"]}}}`, // Test AddToSetEach
			update.Coll().AddToSetEach("tags", "a", "b"),
			update.U{"$addToSet": update.U{"tags": map[string]any{"$each": []string{"a", "b"}}}},
		},
		{
			// Chaining array + field operators together
			`{"$addToSet":{"genres":{"$each":["sci-fi","fantasy"]}},"$set":{"name":"Bob"}}`,
			update.Coll().Set("name", "Bob").AddToSetEach("genres", "sci-fi", "fantasy"),
			update.U{"$set": update.U{"name": "Bob"}, "$addToSet": update.U{"genres": map[string]any{"$each": []string{"sci-fi", "fantasy"}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.expectedJSON, func(t *testing.T) {
			assertJSONEqual(t, tt.expectedJSON, tt.fluent, tt.raw)
		})
	}
}

func TestCollectionUpdateBuilderZeroValuePanic(t *testing.T) {
	// This test just verifies that a zero-value CollectionUpdateBuilder doesn't panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()
	u := update.CollectionUpdateBuilder{}
	u.Set("BestDB", "Astra")
}

func TestAdvancedChaining(t *testing.T) {
	u := update.Coll().Unset("borrower", "due_date")
	u.Unset("phone")
	u = u.Unset("email")
	expected := `{"$unset":{"borrower":"","due_date":"","email":"","phone":""}}`
	j, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}
	if string(j) != cleanString(expected) {
		t.Errorf("\nGOT:\n%s\n\nWANT:\n%s", j, cleanString(expected))
	}
}

func TestTableUpdateBuilder(t *testing.T) {
	// Some of the tests were taken directly from the docs:
	// https://docs.datastax.com/en/astra-db-serverless/api-reference/update-operators-tables.html
	tests := []struct {
		name         string
		expectedJSON string
		fluent       *update.TableUpdateBuilder
		raw          update.U
	}{
		{
			"set single simple field",
			`{"$set":{"name":"Bob"}}`,
			update.Table().Set("name", "Bob"),
			update.U{"$set": update.U{"name": "Bob"}},
		},
		{
			"Docs example: unset",
			`{"$unset":{"genres":""}}`,
			update.Table().Unset("genres"),
			update.U{"$unset": update.U{"genres": ""}},
		},
		{
			"unset multiple fields",
			`{"$unset":{"email":"","phone":""}}`,
			update.Table().Unset("phone", "email"),
			update.U{"$unset": update.U{"email": "", "phone": ""}},
		},
		{
			"set and unset combined",
			`{"$set":{"name":"Bob"},"$unset":{"phone":""}}`,
			update.Table().Set("name", "Bob").Unset("phone"),
			update.U{"$set": update.U{"name": "Bob"}, "$unset": update.U{"phone": ""}},
		},
		{
			"Docs example: set non-map columns",
			`{"$set":{"genres":["Fiction","Drama"],"rating":4.5}}`,
			update.Table().Set("rating", 4.5).Set("genres", []string{"Fiction", "Drama"}),
			update.U{"$set": update.U{"rating": 4.5, "genres": []string{"Fiction", "Drama"}}},
		},
		{
			"Docs example: set map columns",
			`{"$set":{"map_column_int_str":[[1,"value1"],[2,"value2"]],"map_column_str_str":{"key1":"value1","key2":"value2"}}}`,
			update.Table().
				Set("map_column_int_str", [][]any{{1, "value1"}, {2, "value2"}}).
				Set("map_column_str_str", update.U{"key1": "value1", "key2": "value2"}),
			update.U{"$set": update.U{
				"map_column_int_str": [][]any{{1, "value1"}, {2, "value2"}},
				"map_column_str_str": update.U{"key1": "value1", "key2": "value2"},
			}},
		},
		{
			"push single value",
			`{"$push":{"genres":"SciFi"}}`,
			update.Table().Push("genres", "SciFi"),
			update.U{"$push": update.U{"genres": "SciFi"}},
		},
		{
			"Docs example: push each multiple values",
			`{"$push":{"genres":"SciFi","topics":{"$each":["robots","AI"]}}}`,
			update.Table().Push("genres", "SciFi").PushEach("topics", "robots", "AI"),
			update.U{"$push": update.U{"genres": "SciFi", "topics": update.U{"$each": []string{"robots", "AI"}}}},
		},
		{
			"Docs example: pullAll multiple values",
			`{"$pullAll":{"genres":["SciFi","Romance"]}}`,
			update.Table().PullAll("genres", "SciFi", "Romance"),
			update.U{"$pullAll": update.U{"genres": []string{"SciFi", "Romance"}}},
		},
		{
			"Docs example: pullAll multiple values",
			`{"$pullAll":{"metadata":["language","edition"]}}`,
			update.Table().PullAll("metadata", "language", "edition"),
			update.U{"$pullAll": update.U{"metadata": []string{"language", "edition"}}},
		},
		{
			// Docs example: $push with map columns (single and $each, both key formats)
			"Docs example: push map columns",
			`{"$push":{"map_column_int_str":[1,"value1"],"map_column_int_str_2":{"$each":[[1,"value1"],[2,"value2"]]},"map_column_str_str":{"key1":"value1"},"map_column_str_str_2":{"$each":[{"key1":"value1"},["key2","value2"]]}}}`,
			update.Table().
				Push("map_column_int_str", []any{1, "value1"}).
				Push("map_column_str_str", update.U{"key1": "value1"}).
				PushEach("map_column_int_str_2", []any{1, "value1"}, []any{2, "value2"}).
				PushEach("map_column_str_str_2", update.U{"key1": "value1"}, []any{"key2", "value2"}),
			update.U{"$push": update.U{
				"map_column_int_str":   []any{1, "value1"},
				"map_column_str_str":   update.U{"key1": "value1"},
				"map_column_int_str_2": update.U{"$each": []any{[]any{1, "value1"}, []any{2, "value2"}}},
				"map_column_str_str_2": update.U{"$each": []any{update.U{"key1": "value1"}, []any{"key2", "value2"}}},
			}},
		},
		{
			"set push and pullAll combined",
			`{"$pullAll":{"genres":["Romance"]},"$push":{"topics":"AI"},"$set":{"name":"Bob"}}`,
			update.Table().Set("name", "Bob").Push("topics", "AI").PullAll("genres", "Romance"),
			update.U{"$set": update.U{"name": "Bob"}, "$push": update.U{"topics": "AI"}, "$pullAll": update.U{"genres": []string{"Romance"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertJSONEqual(t, tt.expectedJSON, tt.fluent, tt.raw)
		})
	}
}

func TestTableUpdateBuilderZeroValuePanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()
	u := update.TableUpdateBuilder{}
	u.Set("BestDB", "Astra")
}

// TestCollectionUpdateInterface verifies that CollectionUpdateBuilder and U satisfy CollectionUpdate.
func TestCollectionUpdateInterface(t *testing.T) {
	var _ update.CollectionUpdate = update.Coll().Set("x", 1)
	var _ update.CollectionUpdate = update.U{"$set": update.U{"x": 1}}
}

// TestTableUpdateInterface verifies that TableUpdateBuilder and U satisfy TableUpdate.
func TestTableUpdateInterface(t *testing.T) {
	var _ update.TableUpdate = update.Table().Set("x", 1)
	var _ update.TableUpdate = update.U{"$set": update.U{"x": 1}}
}
