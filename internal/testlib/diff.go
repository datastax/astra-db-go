package testlib

import (
	"math/big"
	"time"

	"github.com/datastax/astra-db-go/astra/datatypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var cmpOpts = []cmp.Option{
	cmpopts.EquateEmpty(),
	cmp.AllowUnexported(big.Int{}, big.Float{}, datatypes.UUID{}, datatypes.ObjectId{}, datatypes.Duration{}, time.Time{}),
	cmp.Comparer(func(x, y big.Int) bool { return x.Cmp(&y) == 0 }),
	cmp.Comparer(func(x, y big.Float) bool { return x.Cmp(&y) == 0 }),
	cmp.Comparer(func(x, y datatypes.Duration) bool { return x.Equals(y) }),
	cmp.Comparer(func(x, y time.Time) bool { return x.Equal(y) }),
}

func Diff(a, b any, opts ...cmp.Option) string {
	return cmp.Diff(a, b, append(cmpOpts, opts...)...)
}
