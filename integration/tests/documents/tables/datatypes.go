package tables

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"net"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

type columnAsserter struct {
	t       *harness.T
	key     string
	col     string
	counter int32
	table   *astra.Table
	eqOn    func(any) any
}

func mkColumnAsserter(t *harness.T, col string, table *astra.Table, eqOn func(any) any) *columnAsserter {
	if table == nil {
		table = t.Table
	}
	if eqOn == nil {
		eqOn = func(x any) any { return x }
	}
	return &columnAsserter{
		t:     t,
		key:   t.Key(0),
		col:   col,
		table: table,
		eqOn:  eqOn,
	}
}

func (a *columnAsserter) ok(value any) {
	a.t.Helper()
	a.okExp(value, value)
}

func (a *columnAsserter) okExp(value any, expected any) {
	a.t.Helper()
	obj := astra.NewRow{"text": a.key, "int": a.counter, a.col: value}
	a.counter++

	_, err := a.table.InsertOne(a.t.Ctx, obj)
	testlib.FailIfErr(a.t, err, "expected insert to succeed for %v", value)

	res := a.table.FindOne(a.t.Ctx, filter.And(filter.Eq("text", obj["text"]), filter.Eq("int", obj["int"])))
	testlib.FailIfErr(a.t, res.Err(), "expected find to succeed")

	var found astra.Row
	testlib.FailIfErr(a.t, res.Decode(&found), "expected decode to succeed")

	actual, _ := found.Get(a.col)
	testlib.NoDiff(a.t, a.eqOn(expected), a.eqOn(actual))
}

func (a *columnAsserter) notOk(value any) {
	a.t.Helper()
	obj := astra.NewRow{"text": a.key, "int": a.counter, a.col: value}
	a.counter++

	_, err := a.table.InsertOne(a.t.Ctx, obj)
	testlib.FailIf(a.t, err == nil, "expected insert to fail for %v", value)
}

func bigFloat(str string) *big.Float {
	f, _ := new(big.Float).SetString(str)
	return f
}

func cmpAny(a, b unsafe.Pointer) int {
	return strings.Compare(fmt.Sprintf("%v", *(*any)(a)), fmt.Sprintf("%v", *(*any)(b)))
}

func init() {
	s := harness.ParallelSuite("datatypes")
	s.Truncate(harness.SelectTables, harness.SelectBefore)

	s.Run("should handle different text insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "text", nil, nil)

		colAsserter.notOk("")
		colAsserter.notOk(strings.Repeat("a", 65536))

		colAsserter.ok(strings.Repeat("a", 65535))
		colAsserter.ok("A!@#$%^&*()")
		colAsserter.ok("⨳⨓⨋")
	})

	for _, col := range []string{"int", "tinyint", "smallint"} {
		s.Run("should handle different "+col+" insertion cases", func(t *harness.T) {
			colAsserter := mkColumnAsserter(t, col, nil, func(x any) any {
				return fmt.Sprint(x)
			})

			colAsserter.notOk(1.1)
			colAsserter.notOk(1e50)
			colAsserter.notOk(math.Inf(1))
			colAsserter.notOk(math.NaN())
			colAsserter.notOk("Infinity")
			colAsserter.notOk("123")

			colAsserter.ok(1)
			colAsserter.ok(-1)
		})
	}

	s.Run("should handle different ascii insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "ascii", nil, nil)

		colAsserter.notOk("⨳⨓⨋")
		colAsserter.notOk("é")
		colAsserter.notOk("\x80")

		colAsserter.ok("")
		colAsserter.ok(strings.Repeat("a", 65535))
		colAsserter.ok("A!@#$%^&*()")
		colAsserter.ok("\u0000")
		colAsserter.ok("\x7F")
	})

	s.Run("should handle different blob insertion cases", func(t *harness.T) {
		buffer := []byte{0x0, 0x1}
		base64Str := base64.StdEncoding.EncodeToString(buffer)

		colAsserter := mkColumnAsserter(t, "blob", nil, func(v any) any {
			if b, ok := v.([]byte); ok {
				return base64.StdEncoding.EncodeToString(b)
			}
			return v
		})

		colAsserter.notOk(base64Str)

		colAsserter.okExp(map[string]any{"$binary": base64Str}, buffer)
		colAsserter.ok(buffer)
		colAsserter.ok([]byte{})
	})

	s.Run("should handle the numerous different boolean insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "boolean", nil, nil)

		colAsserter.notOk("true")
		colAsserter.notOk(1)

		colAsserter.ok(true)
		colAsserter.ok(false)
	})

	s.Run("should handle different decimal insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "decimal", nil, nil)

		colAsserter.notOk("123123.12312312312")
		colAsserter.notOk(math.NaN())
		colAsserter.notOk(math.Inf(1))
		colAsserter.notOk(math.Inf(-1))
		colAsserter.notOk("Infinity")

		colAsserter.okExp(123123.123, bigFloat("123123.123"))
		colAsserter.okExp(big.NewInt(123123), big.NewFloat(123123))

		colAsserter.ok(bigFloat("1.1212121131231231231231231231231231231231233122"))
		colAsserter.ok(bigFloat("-1e50"))
		colAsserter.ok(bigFloat("-1e-50"))
	})

	for _, col := range []string{"float", "double"} {
		s.Run("should handle different "+col+" insertion cases", func(t *harness.T) {
			colAsserter := mkColumnAsserter(t, col, nil, func(x any) any {
				return fmt.Sprint(x)
			})

			colAsserter.notOk("123")
			colAsserter.notOk("nan")
			colAsserter.notOk("infinity")

			colAsserter.ok(123123.125)
			colAsserter.okExp(big.NewInt(123123), 123123.0)
			colAsserter.okExp(bigFloat("123.12523"), 123.12523)
			colAsserter.okExp("NaN", math.NaN())
			colAsserter.okExp("Infinity", math.Inf(1))
			colAsserter.okExp("-Infinity", math.Inf(-1))
			colAsserter.ok(math.NaN())
			colAsserter.ok(math.Inf(1))
			colAsserter.ok(math.Inf(-1))
		})
	}

	for _, col := range []string{"bigint", "varint"} {
		s.Run("should handle different "+col+" insertion cases", func(t *harness.T) {
			colAsserter := mkColumnAsserter(t, col, nil, func(v any) any {
				if bi, ok := v.(*big.Int); ok {
					return bi.String()
				}
				if i, ok := v.(int64); ok {
					return big.NewInt(i).String()
				}
				return v
			})

			colAsserter.notOk("123")
			colAsserter.notOk(math.NaN())
			colAsserter.notOk(math.Inf(1))
			colAsserter.notOk(math.Inf(-1))

			if col == "bigint" {
				colAsserter.okExp(123123, int64(123123))
				colAsserter.okExp(int64(math.MaxInt64), int64(math.MaxInt64))
			} else {
				colAsserter.okExp(123123, big.NewInt(123123))
				colAsserter.okExp(int64(math.MaxInt64), big.NewInt(9223372036854775807).String())
			}
			colAsserter.okExp(big.NewInt(23423432049238904), big.NewInt(23423432049238904))
		})
	}

	s.Run("should handle different inet insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "inet", nil, func(v any) any {
			if ip, ok := v.(net.IP); ok {
				return ip.String()
			}
			return v
		})

		colAsserter.notOk("127.0.0.1/16")
		colAsserter.notOk("127.0.0.1:80")
		colAsserter.notOk("6f4e:1900:4545:3:200:f6ff:fe21:645cf")
		colAsserter.notOk("10.10.10.1000")

		colAsserter.okExp("::ffff:192.168.0.1", "192.168.0.1")
		colAsserter.okExp("127.1", "127.0.0.1")
		colAsserter.okExp("127.0.1", "127.0.0.1")
		colAsserter.okExp("localhost", "127.0.0.1")
		colAsserter.okExp("192.168.36095", "192.168.140.255")
		colAsserter.okExp("192.11046143", "192.168.140.255")

		colAsserter.ok(net.ParseIP("127.0.0.1"))
		colAsserter.okExp(net.ParseIP("::1"), "::1")
		colAsserter.okExp(net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), "2001:db8:85a3::8a2e:370:7334")
		colAsserter.okExp(net.ParseIP("2001:db8:85a3::8a2e:370:7334"), "2001:db8:85a3::8a2e:370:7334")
		colAsserter.ok(net.ParseIP("168.201.203.205"))
	})

	s.Run("should handle different vector insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "vector", nil, func(v any) any {
			if vec, ok := v.(datatypes.Vector); ok {
				return vec
			}
			return v
		})

		colAsserter.notOk("hey, wait, this ain't vectorize...")
		colAsserter.notOk(datatypes.NewVector([]float32{.5, .5, .5, .5, .5, .5}))

		colAsserter.ok(datatypes.NewVector([]float32{.5, .5, .5, .5, .5}))
		colAsserter.okExp([]float32{.5, .5, .5, .5, .5}, datatypes.NewVector([]float32{.5, .5, .5, .5, .5}))
	})

	s.Run("should handle different vectorize insertion cases", func(t *harness.T) {
		dummyVec := make([]float32, 1024)
		for i := range dummyVec {
			dummyVec[i] = .5
		}

		colAsserter := mkColumnAsserter(t, "vector1", t.Table_, func(v any) any {
			if vec, ok := v.(datatypes.Vector); ok {
				return vec.Dimension()
			}
			return v
		})

		colAsserter.notOk(datatypes.NewVector([]float32{.5, .5, .5, .5, .5}))

		colAsserter.okExp("toto, I've a feeling we're in vectorize again", 1024)
		colAsserter.okExp(datatypes.NewVector(dummyVec), 1024)
	})

	s.Run("should handle different map insertion cases", func(t *harness.T) {
		uuid1 := datatypes.NewUUIDv1()
		uuid4 := datatypes.NewUUIDv4()

		colAsserter := mkColumnAsserter(t, "map", nil, nil)

		colAsserter.notOk([]any{})
		colAsserter.notOk(map[int64]any{1: map[string]any{"car": "bus"}})
		colAsserter.notOk(map[string]any{"1n": map[string]any{}})
		colAsserter.notOk([][]any{{int64(5), map[string]any{"id": uuid1.String(), "name": "Charlie", "extra": "i should not be here"}}})

		colAsserter.okExp(nil, datatypes.NewSortedMapWithComparator[any, any](cmpAny))
		colAsserter.okExp(map[string]any{}, datatypes.NewSortedMapWithComparator[any, any](cmpAny))
		colAsserter.okExp(datatypes.NewSortedMap[int64, map[string]any](), datatypes.NewSortedMapWithComparator[any, any](cmpAny))

		expectedMap1 := datatypes.NewSortedMapWithComparator[any, any](cmpAny)
		expectedMap1.Set(int64(1), map[string]any{"id": uuid1, "name": nil, "age": nil})
		expectedMap1.Set(int64(2), map[string]any{"id": nil, "name": "John", "age": nil})

		colAsserter.okExp(
			[][]any{{int64(1), map[string]any{"id": uuid1.String()}}, {int64(2), map[string]any{"name": "John"}}},
			expectedMap1,
		)

		expectedMap2 := datatypes.NewSortedMapWithComparator[any, any](cmpAny)
		expectedMap2.Set(int64(3), map[string]any{"id": uuid1, "name": "Alice", "age": big.NewInt(25)})
		expectedMap2.Set(int64(4), map[string]any{"id": uuid4, "name": "Bob", "age": big.NewInt(30)})

		inputMap2 := datatypes.NewSortedMapWithComparator[int64, map[string]any](datatypes.ComparatorFor(reflect.TypeFor[int64]()))
		inputMap2.Set(int64(3), map[string]any{"id": uuid1.String(), "name": "Alice", "age": big.NewInt(25)})
		inputMap2.Set(int64(4), map[string]any{"id": uuid4, "name": "Bob", "age": big.NewInt(30)})

		colAsserter.okExp(inputMap2, expectedMap2)

		largeKey, _ := new(big.Int).SetString("99999999999999999999999999999999999999999999999999999999999999999999", 10)
		expectedMap3 := datatypes.NewSortedMapWithComparator[any, any](cmpAny)
		expectedMap3.Set(largeKey, map[string]any{"id": uuid1, "name": nil, "age": nil})

		inputMap3 := datatypes.NewSortedMapWithComparator[any, map[string]any](cmpAny)
		inputMap3.Set(largeKey, map[string]any{"id": uuid1})

		colAsserter.okExp(inputMap3, expectedMap3)
	})

	s.Run("should handle different set insertion cases", func(t *harness.T) {
		uuid1 := datatypes.NewUUIDv1()
		uuid4 := datatypes.NewUUIDv4()

		colAsserter := mkColumnAsserter(t, "set", nil, func(v any) any {
			if slice, ok := v.([]any); ok {
				return datatypes.NewSet(slice...)
			}
			return v
		})

		colAsserter.notOk(map[string]any{})
		colAsserter.notOk(datatypes.NewSetWithComparator[any](cmpAny, uuid1, "uuid4")) // Mixed types or strings

		colAsserter.okExp(nil, datatypes.NewSetWithComparator[any](cmpAny))
		colAsserter.okExp([]any{}, datatypes.NewSetWithComparator[any](cmpAny))
		colAsserter.okExp(datatypes.NewSetWithComparator[any](cmpAny), datatypes.NewSetWithComparator[any](cmpAny))

		colAsserter.okExp([]any{uuid1.String(), uuid4}, datatypes.NewSetWithComparator[any](cmpAny, uuid1, uuid4))
		colAsserter.okExp(datatypes.NewSetWithComparator[any](cmpAny, uuid1.String(), uuid4), datatypes.NewSetWithComparator[any](cmpAny, uuid1, uuid4))
		colAsserter.okExp([]any{uuid1, uuid1, uuid4}, datatypes.NewSetWithComparator[any](cmpAny, uuid1, uuid4))

		largeSet := make([]any, 100)
		for i := range largeSet {
			largeSet[i] = datatypes.NewUUIDv7()
		}
		colAsserter.ok(datatypes.NewSet(largeSet...))
	})

	s.Run("should handle different list insertion cases", func(t *harness.T) {
		uuid1 := datatypes.NewUUIDv1()
		uuid4 := datatypes.NewUUIDv4()

		colAsserter := mkColumnAsserter(t, "list", nil, func(v any) any {
			if v == nil {
				return []any{}
			}
			return v
		})

		colAsserter.notOk([]any{uuid1})
		colAsserter.notOk([]any{"uuid4"})
		colAsserter.notOk([]any{map[string]any{"car": "bus"}})
		colAsserter.notOk([]any{map[string]any{"id": uuid1, "name": "x", "extra": "bad"}})

		colAsserter.okExp(nil, []any{})
		colAsserter.okExp([]any{}, []any{})

		colAsserter.okExp(
			[]any{
				map[string]any{"id": uuid1.String()},
				map[string]any{"name": "John"},
				map[string]any{"age": big.NewInt(33)},
			},
			[]map[string]any{
				{"id": uuid1, "name": nil, "age": nil},
				{"id": nil, "name": "John", "age": nil},
				{"id": nil, "name": nil, "age": big.NewInt(33)},
			},
		)

		colAsserter.okExp(
			[]any{
				map[string]any{"id": uuid1.String(), "name": "Alice", "age": big.NewInt(25)},
				map[string]any{"id": uuid4, "name": "Bob", "age": big.NewInt(30)},
			},
			[]map[string]any{
				{"id": uuid1, "name": "Alice", "age": big.NewInt(25)},
				{"id": uuid4, "name": "Bob", "age": big.NewInt(30)},
			},
		)

		colAsserter.okExp(
			[]any{
				map[string]any{"id": uuid1.String(), "name": "Charlie", "age": big.NewInt(28)},
				map[string]any{"id": uuid4.String(), "name": "Dave", "age": big.NewInt(31)},
			},
			[]map[string]any{
				{"id": uuid1, "name": "Charlie", "age": big.NewInt(28)},
				{"id": uuid4, "name": "Dave", "age": big.NewInt(31)},
			},
		)
	})

	s.Run("should handle different time insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "time", nil, nil)

		colAsserter.notOk("24:00:00")
		colAsserter.notOk("00:60:00")
		colAsserter.notOk("00:00:00.0000000000")
		colAsserter.notOk("1:00:00")
		colAsserter.notOk("-01:00:00")
		colAsserter.notOk("23-59-00")
		colAsserter.notOk("12:34:56Z+05:30")
		colAsserter.notOk(3123123)

		colAsserter.okExp("23:59:00.", datatypes.MustParseTimeOnly("23:59:00"))
		colAsserter.okExp("23:59", datatypes.MustParseTimeOnly("23:59:00"))
		colAsserter.okExp("00:00:00.000000000", datatypes.MustParseTimeOnly("00:00:00"))
		colAsserter.ok(datatypes.MustParseTimeOnly("23:59:59.999999999"))

		d := time.Date(1970, 1, 1, 23, 59, 59, 999000000, time.UTC)
		colAsserter.okExp(datatypes.TimeOnlyFromTime(d), datatypes.MustParseTimeOnly("23:59:59.999"))
	})

	s.Run("should handle different date insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "date", nil, nil)

		colAsserter.notOk("+2000-01-01")

		colAsserter.okExp("0000-01-01", datatypes.MustParseDateOnly("0000-01-01"))
		colAsserter.okExp("1970-01-01", datatypes.MustParseDateOnly("1970-01-01"))
		colAsserter.okExp("-0001-01-01", datatypes.MustParseDateOnly("-0001-01-01"))
		colAsserter.ok(datatypes.MustParseDateOnly("9999-12-31"))
		colAsserter.ok(datatypes.MustParseDateOnly("+500000-12-31"))
		colAsserter.ok(datatypes.MustParseDateOnly("-500000-12-31"))

		d := time.Date(1970, 1, 1, 23, 59, 59, 999000000, time.UTC)
		colAsserter.okExp(datatypes.DateOnlyFromTime(d), datatypes.MustParseDateOnly("1970-01-01"))
		colAsserter.ok(datatypes.NewDateOnly(1970, 1, 1))
	})

	s.Run("should handle different timestamp insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "timestamp", nil, func(v any) any {
			if tim, ok := v.(time.Time); ok {
				return tim.UTC()
			}
			return v
		})

		colAsserter.notOk(3123123)

		now := time.Now().UTC().Truncate(time.Millisecond)
		colAsserter.ok(now)

		d1, _ := time.Parse(time.RFC3339Nano, "1970-01-01T00:00:00.000Z")
		colAsserter.ok(d1)

		d2, _ := time.Parse(time.RFC3339Nano, "1970-01-01T00:00:00.000+07:00")
		colAsserter.okExp("1970-01-01T00:00:00.000+07:00", d2)
	})

	s.Run("should handle different duration insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "duration", nil, nil)

		colAsserter.notOk("1 hour")

		colAsserter.okExp("1h", datatypes.MustParseDuration("1h"))
	})

	s.Run("should handle different UDT insertion cases", func(t *harness.T) {
		colAsserter := mkColumnAsserter(t, "udt", nil, nil)

		colAsserter.notOk(map[string]any{"invalid": "structure"})
		colAsserter.notOk(map[string]any{"description": "test", "tags": "invalid", "metadata": map[string]any{}})

		colAsserter.ok(map[string]any{
			"name": "test description",
			"age":  big.NewInt(3),
			"id":   datatypes.NewUUIDv4(),
		})

		colAsserter.okExp(map[string]any{}, nil)
	})
}
