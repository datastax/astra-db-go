package serdes

import (
	"encoding/json"
	"testing"
)

// Global variables to prevent compiler optimization
var (
	result     []byte
	userResult User
)

func BenchmarkSerDesComparison(b *testing.B) {
	// --- SETUP DATA ---
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

	// --- SERIALIZATION COMPARISON ---
	b.Run("Serialize/Custom-Unsafe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = Serialize(user)
		}
	})

	b.Run("Serialize/StdJSON", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = json.Marshal(user)
		}
	})

	// --- DESERIALIZATION COMPARISON ---
	b.Run("Deserialize/Custom-Unsafe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var u User
			_ = Deserialize(jsonData, &u)
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
