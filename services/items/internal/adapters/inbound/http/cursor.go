package http

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/YanMak/ecommerce/v2/services/items/internal/app/usecase/paging"
)

func encodeCursor(c paging.Cursor) string {
	s := fmt.Sprintf("%d|%d", c.CreatedAt.UnixNano(), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
func decodeCursor(s string) (*paging.Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var ts, id int64
	_, err = fmt.Sscanf(string(b), "%d|%d", &ts, &id)
	if err != nil {
		return nil, err
	}
	return &paging.Cursor{CreatedAt: time.Unix(0, ts), ID: id}, nil
}
