// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/datastax/astra-db-go/astra/cursors"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/astra/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/astra/options"

	"github.com/google/go-cmp/cmp"
)

func init() {
	harness.Register(
		harness.IntegrationTest{Name: "CollectionNestedCreate", Run: CollectionNestedCreate},
		harness.IntegrationTest{Name: "CollectionNestedInsertOne", Run: CollectionNestedInsertOne},
		harness.IntegrationTest{Name: "CollectionNestedInsertMany", Run: CollectionNestedInsertMany},
		harness.IntegrationTest{Name: "CollectionNestedFindOne", Run: CollectionNestedFindOne},
		harness.IntegrationTest{Name: "CollectionNestedFindByNestedField", Run: CollectionNestedFindByNestedField},
		harness.IntegrationTest{Name: "CollectionNestedDrop", Run: CollectionNestedDrop},
	)
}

const nestedCollectionName = "GoTestNested"

func getTestRestaurant() Restaurant {
	score1 := float32(11)
	score2 := float32(17)
	return Restaurant{
		ID:           "rest-001",
		Name:         "Vella",
		RestaurantID: "40356018",
		Cuisine:      "Italian",
		Borough:      "Manhattan",
		Address: Address{
			Building:    "1480",
			Coordinates: []float64{-73.9557413, 40.7720266},
			Street:      "2 Avenue",
			ZipCode:     "10075",
		},
		Grades: []GradeEntry{
			{Date: time.Date(2014, 10, 1, 0, 0, 0, 0, time.UTC), Grade: "A", Score: &score1},
			{Date: time.Date(2014, 1, 16, 0, 0, 0, 0, time.UTC), Grade: "B", Score: &score2},
		},
	}
}

func getTestRestaurants() []Restaurant {
	score1 := float32(11)
	score2 := float32(9)
	score3 := float32(22)
	return []Restaurant{
		{
			ID:           "rest-002",
			Name:         "Riviera",
			RestaurantID: "40356068",
			Cuisine:      "Italian",
			Borough:      "Brooklyn",
			Address: Address{
				Building:    "2780",
				Coordinates: []float64{-73.98241999999999, 40.579505},
				Street:      "Stillwell Avenue",
				ZipCode:     "11224",
			},
			Grades: []GradeEntry{
				{Date: time.Date(2014, 6, 10, 0, 0, 0, 0, time.UTC), Grade: "A", Score: &score1},
			},
		},
		{
			ID:           "rest-003",
			Name:         "Chez Marie",
			RestaurantID: "40356078",
			Cuisine:      "French",
			Borough:      "Manhattan",
			Address: Address{
				Building:    "120",
				Coordinates: []float64{-73.992615, 40.727055},
				Street:      "E 7 Street",
				ZipCode:     "10009",
			},
			Grades: []GradeEntry{
				{Date: time.Date(2015, 3, 20, 0, 0, 0, 0, time.UTC), Grade: "A", Score: &score2},
				{Date: time.Date(2014, 9, 12, 0, 0, 0, 0, time.UTC), Grade: "C", Score: &score3},
			},
		},
	}
}

// CollectionNestedCreate creates a collection for nested document tests.
func CollectionNestedCreate(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	_, err := db.CreateCollection(ctx, nestedCollectionName, options.CreateCollection().SetIndexingAllow("*"))
	return err
}

// CollectionNestedInsertOne inserts a single Restaurant and verifies the insert succeeded.
func CollectionNestedInsertOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(nestedCollectionName)

	restaurant := getTestRestaurant()
	resp, err := c.InsertOne(ctx, restaurant)
	if err != nil {
		return fmt.Errorf("insertOne failed: %w", err)
	}
	var insertedID string
	if err := resp.DecodeID(&insertedID); err != nil {
		return err
	}
	if insertedID == "" {
		return fmt.Errorf("expected inserted ID, got empty string")
	}

	slog.Info("Inserted nested restaurant document", "id", insertedID)
	return nil
}

// CollectionNestedInsertMany inserts multiple Restaurants with nested data.
func CollectionNestedInsertMany(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(nestedCollectionName)

	restaurants := getTestRestaurants()
	resp, err := c.InsertMany(ctx, restaurants)
	if err != nil {
		return fmt.Errorf("insertMany failed: %w", err)
	}
	if resp.InsertedCount() != len(restaurants) {
		return fmt.Errorf("expected %d inserted IDs, got %d", len(restaurants), resp.InsertedCount())
	}

	slog.Info("Inserted nested restaurant documents", "count", resp.InsertedCount())
	return nil
}

// CollectionNestedFindOne retrieves a Restaurant by ID and verifies all nested data round-trips correctly.
func CollectionNestedFindOne(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(nestedCollectionName)

	original := getTestRestaurant()

	var found Restaurant
	err := c.FindOne(ctx, filter.Eq("id", original.ID)).Decode(&found)
	if err != nil {
		return fmt.Errorf("findOne failed: %w", err)
	}

	if !cmp.Equal(original, found) {
		return fmt.Errorf("round-trip mismatch: %s", cmp.Diff(original, found))
	}

	slog.Info("Nested findOne round-trip verified", "name", found.Name, "address", found.Address.Street)
	return nil
}

// CollectionNestedFindByNestedField queries restaurants using a dot-notation filter on a nested field.
func CollectionNestedFindByNestedField(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	c := db.Collection(nestedCollectionName)

	// Find all restaurants in Manhattan
	cursor := c.Find(filter.Eq("borough", "Manhattan"))
	defer cursor.Close()

	var results []Restaurant
	if err := cursor.DecodeAll(ctx, &results); err != nil {
		return fmt.Errorf("find by nested field failed: %w", err)
	}

	if len(results) == 0 {
		return errors.New("expected at least 1 Manhattan restaurant")
	}

	// Verify all results are in Manhattan
	for _, r := range results {
		if r.Borough != "Manhattan" {
			return fmt.Errorf("expected borough Manhattan, got %q", r.Borough)
		}
	}

	// Also test dot-notation on a nested field
	cursor2 := c.Find(filter.Eq("address.zipCode", "10075"))
	defer cursor2.Close()

	zipResults, err := cursors.DecodeAll[Restaurant](ctx, cursor2)
	if err != nil {
		return fmt.Errorf("find by dot-notation zipCode failed: %w", err)
	}

	if len(zipResults) != 1 {
		return fmt.Errorf("expected 1 restaurant with zipCode 10075, got %d", len(zipResults))
	}
	if zipResults[0].Name != "Vella" {
		return fmt.Errorf("expected Vella, got %q", zipResults[0].Name)
	}

	slog.Info("Nested field queries verified", "manhattanCount", len(results), "zipCodeMatch", zipResults[0].Name)
	return nil
}

// CollectionNestedDrop cleans up the nested test collection.
func CollectionNestedDrop(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()
	return db.DropCollection(ctx, nestedCollectionName)
}
