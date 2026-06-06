package harness

import (
	"github.com/datastax/astra-db-go/astra"
	"github.com/datastax/astra-db-go/astra/filter"
	"github.com/datastax/astra-db-go/internal/testlib"
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
