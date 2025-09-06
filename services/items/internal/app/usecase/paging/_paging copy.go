package paging

import "time"

type OffsetPage struct {
	Page    int32 // 1..N
	PerPage int32
}

type OffsetResult[T any] struct {
	Items   []T
	Total   int64
	Page    int32
	PerPage int32
	HasNext bool
}

type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

type KeysetResult[T any] struct {
	Items   []T
	Cursor  *Cursor // nil если больше нет
	HasNext bool
}
