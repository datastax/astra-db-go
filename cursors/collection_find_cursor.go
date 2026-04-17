package cursors

type CollectionFindCursor struct {
	FindCursor[CollectionFindCursor]
}

//var _ FindCursor[*CollectionFindCursor] = (*CollectionFindCursor)(nil)

func (c *CollectionFindCursor) Clone() *CollectionFindCursor {
	panic("not implemented")
}
