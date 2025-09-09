package search

import (
	"fmt"
	"lesson2/adapters/inbound/httpapi/respond"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SearchQuery struct {
	Limit           int        `query:"limit" json:"limit"`
	Offset          int        `query:"offset" json:"offset"`
	Sort            []string   `query:"sort" json:"sort"`
	CreatedFrom     *time.Time `query:"created_from" json:"created_from"`
	CreatedTo       *time.Time `query:"created_to" json:"created_to"`
	IncludeArchived bool       `query:"include_archived" json:"include_archived"`
}

func BindSearchQuery(r *http.Request) (SearchQuery, []respond.FieldError, error) {
	q := r.URL.Query()
	limit := atoi(q.Get("limit"), 20)
	offset := atoi(q.Get("offset"), 0)
	if limit < 1 || limit > 100 {
		return SearchQuery{}, []respond.FieldError{{Loc: "query.limit", Msg: "must be 1..100"}}, fmt.Errorf("invalid")
	}
	if offset < 0 {
		return SearchQuery{}, []respond.FieldError{{Loc: "query.offset", Msg: "must be >= 0"}}, fmt.Errorf("invalid")
	}

	var fromp, top *time.Time
	if s := q.Get("created_from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		//t, err := time.Parse("2006-01-01", s)
		if err != nil {
			return SearchQuery{}, []respond.FieldError{{Loc: "query.created_from", Msg: "must be RFC3339"}}, fmt.Errorf("invalid")
		}
		fromp = &t
	}
	if s := q.Get("created_to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		//t, err := time.Parse("2006-01-01", s)
		if err != nil {
			return SearchQuery{}, []respond.FieldError{{Loc: "query.created_to", Msg: "must be RFC3339"}}, fmt.Errorf("invalid")
		}
		top = &t
	}
	return SearchQuery{
		Limit: limit, Offset: offset,
		Sort:        csv(q["sort"]),
		CreatedFrom: fromp, CreatedTo: top,
		IncludeArchived: q.Get("include_archived") == "true",
	}, nil, nil
}

func csv(vals []string) []string {
	var out []string
	for _, v := range vals {
		for _, p := range strings.Split(v, ",") {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}
func atoi(s string, def int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}
