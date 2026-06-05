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
	"net"
	"time"
)

// ---------------------------------------------------------
// Document API Objects
// ---------------------------------------------------------

type SimpleObjectWithVector struct {
	ID               *int      `json:"_id,omitempty"`
	Name             string    `json:"name"`
	VectorEmbeddings []float32 `json:"$vector,omitempty"` // Vectors are usually float32
}

type SimpleObjectWithVectorize struct {
	ID   *int   `json:"_id,omitempty"`
	Name string `json:"name"`
	// Go doesn't support computed properties in structs for JSON.
	// You must populate this field manually before marshaling.
	StringToVectorize string `json:"$vectorize,omitempty"`
}

type SimpleObjectWithVectorizeResult struct {
	SimpleObjectWithVectorize          // Embedding (Inheritance)
	Similarity                *float64 `json:"$similarity,omitempty"`
}

type SimpleObjectWithObjectId struct {
	// standard "any" allows string (hex) or specific ObjectId types
	ID   any    `json:"_id,omitempty"`
	Name string `json:"name"`
}

type SimpleObjectWithGuidId struct {
	ID   *string `json:"_id,omitempty"` // Go uses strings for UUIDs in JSON
	Name string  `json:"name"`
}

type SimpleObject struct {
	ID         any        `json:"_id,omitempty"`
	Name       string     `json:"name"`
	Properties Properties `json:"properties"`
}

type SerializationTest struct {
	TestID           int        `json:"_id"`
	NestedProperties Properties `json:"nestedProperties"`
}

type Properties struct {
	PropertyOne         string    `json:"propertyOne"`
	PropertyTwo         string    `json:"propertyTwo"`
	IntProperty         int       `json:"intProperty"`
	StringArrayProperty []string  `json:"stringArrayProperty"`
	BoolProperty        bool      `json:"boolProperty"`
	TimeProperty        time.Time `json:"timeProperty"`
	UTCTime             time.Time `json:"utcTime"`
	SkipWhenNull        *string   `json:"skipWhenNull,omitempty"`
}

type SimpleObjectSkipNulls struct {
	ID          int     `json:"_id"`
	Name        string  `json:"name"`
	PropertyOne *string `json:"propertyOne,omitempty"`
	PropertyTwo *string `json:"propertyTwo,omitempty"`
}

type DifferentIdsObject struct {
	TheID any    `json:"_id"` // Matches 'object' in C#
	Name  string `json:"name"`
}

// ---------------------------------------------------------
// Nested Objects (Restaurant)
// ---------------------------------------------------------

type Restaurant struct {
	ID           string       `json:"id"` // Guid
	Name         string       `json:"name"`
	RestaurantID string       `json:"restaurantId"`
	Cuisine      string       `json:"cuisine"`
	Address      Address      `json:"address"`
	Borough      string       `json:"borough"`
	Grades       []GradeEntry `json:"grades"`
}

type Address struct {
	Building    string    `json:"building"`
	Coordinates []float64 `json:"coordinates"`
	Street      string    `json:"street"`
	ZipCode     string    `json:"zipCode"`
}

type GradeEntry struct {
	Date  time.Time `json:"date"`
	Grade string    `json:"grade"`
	Score *float32  `json:"score,omitempty"`
}

// ---------------------------------------------------------
// Table API Objects (Row Objects)
// ---------------------------------------------------------

type SimpleRowObject struct {
	Name string `json:"name" astra:"pk"`
}

type RowBook struct {
	Title         string     `json:"title" astra:"pk,1"`
	Author        any        `json:"author" astra:"vectorize,provider=nvidia,model=NV-Embed-QA"`
	NumberOfPages int        `json:"numberOfPages" astra:"pk,2"`
	DueDate       *time.Time `json:"dueDate"`
	Genres        []string   `json:"genres"` // HashSet mapped to Slice for JSON compatibility
	Rating        float32    `json:"rating"`
}

// TableName implements the interface to define the custom table name
func (RowBook) TableName() string {
	return "bookTestTable"
}

type RowBookSinglePrimaryKey struct {
	Title         string    `json:"title" astra:"pk,1"`
	Author        string    `json:"author"`
	NumberOfPages int       `json:"numberOfPages"`
	DueDate       time.Time `json:"dueDate"`
	Genres        []string  `json:"genres"`
	Rating        float32   `json:"rating"`
}

func (RowBookSinglePrimaryKey) TableName() string {
	return "bookTestTableSinglePrimaryKey"
}

type RowEventByDay struct {
	EventDate time.Time `json:"event_date" astra:"pk,1"`
	ID        string    `json:"id" astra:"pk,2"` // Guid
	Title     string    `json:"title"`
	Location  string    `json:"location"`
	Category  string    `json:"category"`
}

type RowBookWithSimilarity struct {
	RowBook            // Embedding
	Similarity float64 `json:"$similarity"`
}

type RowTestObject struct {
	Name              string             `json:"renamed" astra:"pk,1"`
	Vector            []float32          `json:"vector" astra:"pk,2,vector,dim=4"`
	StringToVectorize any                `json:"stringToVectorize" astra:"pk,3,vectorize,provider=nvidia,model=NV-Embed-QA"`
	Text              string             `json:"text" astra:"pk,4"`
	Inet              net.IP             `json:"inet"`
	Int               int                `json:"int" astra:"pk,5"`
	TinyInt           uint8              `json:"tinyInt" astra:"pk,6"`
	SmallInt          int16              `json:"smallInt" astra:"pk,7"`
	BigInt            int64              `json:"bigInt" astra:"pk,8"`
	Decimal           float64            `json:"decimal" astra:"pk,9"` // Go uses float64 for generic decimal, or math/big
	Double            float64            `json:"double" astra:"pk,10"`
	Float             float32            `json:"float" astra:"pk,11"`
	IntDictionary     map[string]int     `json:"intDictionary"`
	DecimalDictionary map[string]float64 `json:"decimalDictionary"`
	StringSet         []string           `json:"stringSet"` // Set -> Slice for JSON
	IntSet            []int              `json:"intSet"`    // Set -> Slice for JSON
	StringList        []string           `json:"stringList"`
	ObjectList        []Properties       `json:"objectList" astra:"jsonString"`
	Boolean           bool               `json:"boolean" astra:"pk,12"`
	Date              time.Time          `json:"date" astra:"pk,13"`
	UUID              string             `json:"uuid" astra:"pk,14"`
	Blob              []byte             `json:"blob"`
	Duration          time.Duration      `json:"duration"`
}

func (RowTestObject) TableName() string {
	return "testTable"
}

// ---------------------------------------------------------
// Primary Key Tests
// ---------------------------------------------------------

type CompositePrimaryKey struct {
	KeyTwo string `json:"keyTwo" astra:"pk,2"`
	KeyOne string `json:"keyOne" astra:"pk,1"`
}

type CompoundPrimaryKey struct {
	KeyTwo            string `json:"keyTwo" astra:"pk,2"`
	KeyOne            string `json:"keyOne" astra:"pk,1"`
	SortTwoDescending string `json:"sortTwoDescending" astra:"ck,2,desc"`
	SortOneAscending  string `json:"sortOneAscending" astra:"ck,1,asc"`
}

type BrokenCompositePrimaryKey struct {
	KeyTwo string `json:"keyTwo" astra:"pk,3"`
	KeyOne string `json:"keyOne" astra:"pk,1"`
}

type BrokenCompoundPrimaryKey struct {
	KeyTwo            string `json:"keyTwo" astra:"pk,2"`
	KeyOne            string `json:"keyOne" astra:"pk,1"`
	SortTwoDescending string `json:"sortTwoDescending" astra:"ck,2,desc"`
	SortOneAscending  string `json:"sortOneAscending" astra:"ck,0,asc"`
}

// ---------------------------------------------------------
// Nullable Book
// ---------------------------------------------------------

type Book struct {
	ID                *string `json:"_id,omitempty"`
	Title             *string `json:"title,omitempty"`
	Author            *string `json:"author,omitempty"`
	NumberOfPages     *int    `json:"number_of_pages,omitempty"`
	IsCheckedOut      *bool   `json:"isCheckedOut,omitempty"`
	StringToVectorize *string `json:"$vectorize,omitempty"`
}
