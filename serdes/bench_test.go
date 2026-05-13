package serdes

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/datatypes"
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
			result, _ = Serialize(user, TargetCollection)
		}
	})

	b.Run("Serialize/Custom-Table", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = Serialize(user, TargetTable)
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
			_ = Deserialize(jsonData, &u, nil, TargetCollection)
			userResult = u
		}
	})

	b.Run("Deserialize/Custom-Table", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var u User
			_ = Deserialize(jsonData, &u, nil, TargetTable)
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

type UUIDM struct {
	value [16]byte
}

func (u UUIDM) MarshalAstraRaw(ctx EncodeCtx, dst []byte) ([]byte, error) {
	if ctx.Target == TargetCollection {
		return encodeDollarDatatype(dst, []byte("uuid"), func(dst []byte) ([]byte, error) {
			return encodeUUIDM(dst, u)
		})
	}
	return encodeUUIDM(dst, u)
}

func encodeUUIDM(dst []byte, p UUIDM) ([]byte, error) {
	dst = append(dst, '"')
	dst = append(dst, p.String()...)
	dst = append(dst, '"')
	return dst, nil
}

func (u UUIDM) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u.value[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u.value[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u.value[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u.value[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u.value[10:16])
	return string(buf[:])
}

func (u *UUIDM) UnmarshalAstraRaw(target Target, value []byte) error {
	var uuid datatypes.UUID
	var err error

	if target == TargetCollection {
		_, uuid, err = parseDollarDatatype(value, []byte("uuid"), decodeUUID)
	} else {
		_, uuid, err = decodeUUID(value)
	}

	if err == nil {
		u.value = uuid.Bytes()
	}
	return nil
}

func (u *UUIDM) UnmarshalJSON(data []byte) error {
	var uuid datatypes.UUID
	var err error

	_, uuid, err = decodeUUID(data)

	if err == nil {
		u.value = uuid.Bytes()
	}
	return nil
}

type Data1 struct {
	Name string
	Uuid datatypes.UUID
	Age  int
}

type Data2 struct {
	Name string
	Uuid UUIDM
	Age  int
}

var (
	data1Result Data1
	data2Result Data2
)

func BenchmarkDirectVsMarshal(b *testing.B) {
	uuid, _ := datatypes.ParseUUID("59e1cc0d-a48f-432b-ad49-04486d7dd2b5")
	data1 := Data1{"Firstname M. Lastname", uuid, 37}
	data2 := Data2{"Firstname M. Lastname", UUIDM{uuid.Bytes()}, 37}

	jsonData, _ := Serialize(data1, TargetTable)

	b.Logf("uuid: %v", uuid)
	b.Logf("json: %v", string(jsonData))

	b.Run("Serialize/Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = Serialize(data1, TargetTable)
		}
	})

	b.Run("Serialize/Marshal-Astra", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = Serialize(data2, TargetTable)
		}
	})

	b.Run("Serialize/Marshal-JSON", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _ = json.Marshal(data2)
		}
	})

	b.Run("Deserialize/Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var d Data1
			_ = Deserialize(jsonData, &d, nil, TargetTable)
			data1Result = d
		}
	})

	b.Run("Deserialize/Unmarshal-Astra", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var d Data2
			_ = Deserialize(jsonData, &d, nil, TargetTable)
			data2Result = d
		}
	})

	b.Run("Deserialize/Unmarshal-JSON", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var d Data2
			_ = json.Unmarshal(jsonData, &d)
			data2Result = d
		}
	})
}
