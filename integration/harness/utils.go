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

package harness

import (
	"github.com/datastax/astra-db-go/v2/astra"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func (t *T) NoDiff(want, got any) {
	t.Helper()
	if diff := testlib.Diff(t, want, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}

type tOrCSelect int
type bOrASelect int

const (
	SelectCollections tOrCSelect = 1 << iota
	SelectTables
)

const (
	SelectBefore bOrASelect = 1 << iota
	SelectAfter
)

func (s *S) Truncate(tOrC tOrCSelect, bOrA bOrASelect) {
	bOrA.addToSuite(s, func(t *T) {
		if tOrC&SelectCollections != 0 {
			testlib.AwaitAll(t, []*astra.Collection{t.Collection, t.Collection_}, func(c *astra.Collection) (any, error) {
				return c.DeleteMany(t.Ctx, filter.F{})
			})
		}
		if tOrC&SelectTables != 0 {
			testlib.AwaitAll(t, []*astra.Table{t.Table, t.Table_}, func(tb *astra.Table) (any, error) {
				return nil, tb.DeleteMany(t.Ctx, filter.F{})
			})
		}
	})
}

func (s *S) Drop(tOrC tOrCSelect, bOrA bOrASelect) {
	bOrA.addToSuite(s, func(t *T) {
		if tOrC&SelectCollections != 0 {
			testlib.AwaitAll(t, []*astra.Collection{t.Collection, t.Collection_}, func(c *astra.Collection) (any, error) {
				return nil, c.Drop(t.Ctx)
			})
		}
		if tOrC&SelectTables != 0 {
			testlib.AwaitAll(t, []*astra.Table{t.Table, t.Table_}, func(tb *astra.Table) (any, error) {
				return nil, tb.Drop(t.Ctx)
			})
		}
	})
}

func (bOrA bOrASelect) addToSuite(s *S, block func(t *T)) {
	if bOrA&SelectBefore != 0 {
		s.Before(block)
	}
	if bOrA&SelectAfter != 0 {
		s.After(block)
	}
}
