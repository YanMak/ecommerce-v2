package paging

import "time"

type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

type KeysetResult[T any] struct {
	Items   []T
	Cursor  *Cursor // nil если больше нет
	HasNext bool
}
