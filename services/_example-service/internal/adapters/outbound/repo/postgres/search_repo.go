package postgresrepo

import (
	"context"
	"strconv"
	"strings"

	"lesson2/adapters/outbound/postgres"
	"lesson2/app/search"
)

type Repo struct {
	DB *postgres.Pool
}

func New(db *postgres.Pool) *Repo { return &Repo{DB: db} }

func (r *Repo) Search(ctx context.Context, tenantID string, q search.Query) ([]search.Item, error) {
	sb := strings.Builder{}
	args := []any{tenantID}
	i := 2

	sb.WriteString(`
		select id::text, name, price::float8, archived, extract(epoch from created_at)::bigint
		from products
		where tenant_id = $1
	`)

	if !q.IncludeArchived {
		sb.WriteString(" and archived = false")
	}
	if q.Body.Q != "" {
		sb.WriteString(" and name ilike $" + itoa(i))
		args = append(args, "%"+q.Body.Q+"%")
		i++
	}
	if q.CreatedFrom != nil {
		sb.WriteString(" and created_at >= $" + itoa(i))
		args = append(args, *q.CreatedFrom)
		i++
	}
	if q.CreatedTo != nil {
		sb.WriteString(" and created_at <= $" + itoa(i))
		args = append(args, *q.CreatedTo)
		i++
	}

	// простая сортировка по первому ключу из q.Sort
	order := "created_at desc"
	if len(q.Sort) > 0 {
		k := q.Sort[0]
		dir := "asc"
		if strings.HasPrefix(k, "-") {
			dir, k = "desc", k[1:]
		}
		switch k {
		case "created_at", "name", "price":
			order = k + " " + dir
		}
	}
	sb.WriteString(" order by " + order)

	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	sb.WriteString(" limit $" + itoa(i))
	args = append(args, limit)
	i++
	sb.WriteString(" offset $" + itoa(i))
	args = append(args, offset)
	i++

	rows, err := r.DB.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []search.Item
	for rows.Next() {
		var it search.Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Price, &it.Archived, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func itoa(i int) string { return strconv.Itoa(i) }
