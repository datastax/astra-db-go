package serdes

import (
	"testing"
)

// Type aliases
type UserID int64
type Email string

// Embedded structs
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zip_code,omitempty"`
}

type ContactInfo struct {
	Email Email  `json:"email"`
	Phone string `json:"phone,omitempty"`
}

// Complex nested struct with embedded fields and deep pointers
type User struct {
	ID         UserID     `json:"id"`
	Name       string     `json:"name"`
	Age        *int       `json:"age,omitempty"`
	Score      **int      `json:"score,omitempty"`  // Double pointer
	Rating     ***float64 `json:"rating,omitempty"` // Triple pointer
	IsActive   bool       `json:"is_active"`
	Address               // Embedded value struct
	BestFriend *User      `json:"best_friend,omitempty"` // Nested pointer
	Ignored    string     `json:"-"`
}

func TestComplexStructSerialization(t *testing.T) {
	age := 30
	score := 95
	scorePtr := &score
	rating := 4.5
	ratingPtr := &rating
	ratingPtrPtr := &ratingPtr

	user := User{
		ID:       123,
		Name:     "John Doe",
		Age:      &age,
		Score:    &scorePtr,     // **int
		Rating:   &ratingPtrPtr, // ***float64
		IsActive: true,
		Address: Address{
			Street:  "123 Main St",
			City:    "New York",
			ZipCode: "10001",
		},
		BestFriend: &User{
			ID:       456,
			Name:     "Jane Doe",
			IsActive: true,
			Address: Address{
				Street: "456 Oak Ave",
				City:   "Boston",
			},
		},
		Ignored: "should not appear",
	}

	encoded, err := Serialize(user)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	t.Logf("Serialized complex struct:\n%s", string(encoded))
}

func TestComplexStructDeserialization(t *testing.T) {
	jsonData := []byte(`{
		"id": 789,
		"name": "Bob Smith",
		"age": 25,
		"score": 88,
		"rating": 3.7,
		"is_active": false,
		"street": "789 Pine St",
		"city": "Seattle",
		"zip_code": "98101",
		"email": "bob@example.com",
		"phone": "555-9999",
		"best_friend": {
			"id": 999,
			"name": "Alice",
			"is_active": true,
			"street": "111 Elm St",
			"city": "Portland"
		}
	}`)

	var user User
	err := Deserialize(jsonData, &user)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	t.Logf("Deserialized complex struct:")
	t.Logf("  ID: %d", user.ID)
	t.Logf("  Name: %s", user.Name)
	if user.Age != nil {
		t.Logf("  Age: %d", *user.Age)
	}
	if user.Score != nil && *user.Score != nil {
		t.Logf("  Score: %d", **user.Score)
	}
	if user.Rating != nil && *user.Rating != nil && **user.Rating != nil {
		t.Logf("  Rating: %.1f", ***user.Rating)
	}
	t.Logf("  IsActive: %v", user.IsActive)
	t.Logf("  Address: %s, %s %s", user.Address.Street, user.Address.City, user.Address.ZipCode)
	if user.BestFriend != nil {
		t.Logf("  BestFriend: %s (ID: %d)", user.BestFriend.Name, user.BestFriend.ID)
	}
}
