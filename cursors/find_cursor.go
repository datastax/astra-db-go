package cursors

import (
	"context"
	"encoding/json"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/results"
)

type FindCursor interface {
	AbstractCursor
	GetSortVector() *datatypes.DataAPIVector
	Warnings() results.Warnings
}

type findPage struct {
	NextPageState *string                  `json:"nextPageState"`
	Result        []json.RawMessage        `json:"results"`
	SortVector    *datatypes.DataAPIVector `json:"sortVector"`
}

var _ FindCursor = (*findCursorImpl)(nil)
var _ abstractCursorInternal[json.RawMessage] = (*findCursorImpl)(nil)

type findCursorImpl struct {
	abstractCursorImpl[json.RawMessage]
	findPage *findPage
}

func (c *findCursorImpl) buffer() *[]json.RawMessage {
	return &c.findPage.Result
}

func (c *findCursorImpl) fetchNextPage(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (c *findCursorImpl) decode(raw json.RawMessage, result any) error {
	//TODO implement me
	panic("implement me")
}

func (c *findCursorImpl) rewind() {
	c.findPage = nil
}

func (c *findCursorImpl) close() {
	c.findPage = nil
}

func newFindCursorImpl() *findCursorImpl {
	cursor := &findCursorImpl{}
	cursor.impl = cursor
	return cursor
}

func (c *findCursorImpl) GetSortVector() *datatypes.DataAPIVector {
	//TODO implement me
	panic("implement me")
}

func (c *findCursorImpl) Warnings() results.Warnings {
	//TODO implement me
	panic("implement me")
}
