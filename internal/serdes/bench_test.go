package serdes

import (
	"encoding/json"
	"testing"
)

type UserID int64
type Email string

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zip_code,omitempty"`
}

type ContactInfo struct {
	Email Email  `json:"email"`
	Phone string `json:"phone,omitempty"`
}

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

// just to try to prevent compiler optimizations
var (
	result     []byte
	userResult User
)

func BenchmarkSerDesComparison(b *testing.B) {
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
		Score:    &scorePtr,
		Rating:   &ratingPtrPtr,
		IsActive: true,
		Address: Address{
			Street: "123 Main St", City: "New York", ZipCode: "10001",
		},
		BestFriend: &User{
			ID: 456, Name: "Jane Doe", IsActive: true,
			Address: Address{Street: "456 Oak Ave", City: "Boston"},
		},
	}

	jsonData, _ := json.Marshal(user)

	// Serialization benchmarks
	b.Run("Serialize/Custom-Collection", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = Serialize(user, CollectionTarget)
		}
	})

	b.Run("Serialize/Custom-Table", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = Serialize(user, TableTarget)
		}
	})

	b.Run("Serialize/StdJSON", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = json.Marshal(user)
		}
	})

	// Deserialization benchmarks
	b.Run("Deserialize/Custom-Collection", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var u User
			_ = Deserialize(jsonData, &u, CollectionTarget)
			userResult = u
		}
	})

	b.Run("Deserialize/Custom-Table", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var u User
			_ = Deserialize(jsonData, &u, TableTarget)
			userResult = u
		}
	})

	b.Run("Deserialize/StdJSON", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var u User
			_ = json.Unmarshal(jsonData, &u)
			userResult = u
		}
	})
}
