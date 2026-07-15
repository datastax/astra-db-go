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

package testlib

import (
	"math/big"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var cmpOpts = []cmp.Option{
	cmpopts.EquateEmpty(),
	cmp.AllowUnexported(
		big.Int{},
		big.Float{},
		datatypes.UUID{},
		datatypes.ObjectId{},
		datatypes.Duration{},
		time.Time{},
	),
	cmp.Comparer(func(x, y big.Int) bool { return x.Cmp(&y) == 0 }),
	cmp.Comparer(func(x, y big.Float) bool { return x.Cmp(&y) == 0 }),
	cmp.Comparer(func(x, y datatypes.Duration) bool { return x.Equals(y) }),
	cmp.Comparer(func(x, y datatypes.Vector) bool { return x.AsBase64() == y.AsBase64() }),
	cmp.Comparer(func(x, y datatypes.SortedMap[any, any]) bool { return x.Len() == y.Len() }), // fine for now
	cmp.Comparer(func(x, y time.Time) bool { return x.Equal(y) }),
}

func Diff(t HasFatal, a, b any, opts ...cmp.Option) string {
	t.Helper()
	return cmp.Diff(a, b, append(cmpOpts, opts...)...)
}

func NoDiff(t HasFatal, want, got any) {
	t.Helper()
	if diff := Diff(t, want, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}
