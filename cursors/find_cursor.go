package cursors

import (
	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/sort"
)

type FindCursor[TSelf any] interface {
	AbstractCursor[TSelf]
	GetSortVector() *datatypes.DataAPIVector
	Filter(filter filter.Filter) TSelf
	Sort(sort sort.Sort) TSelf
	Project(projection map[string]any) TSelf
	Limit(limit int) TSelf
	Skip(skip int) TSelf
	IncludeSortVector(include bool) TSelf
	IncludeSimilarity(include bool) TSelf
	Warnings() results.Warnings
}
