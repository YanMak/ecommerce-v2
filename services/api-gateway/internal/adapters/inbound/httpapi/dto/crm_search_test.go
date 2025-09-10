package dto

import (
	"testing"
	"time"
)

// маленькие хелперы
func ptr[T any](v T) *T { return &v }
func tAt(y int, m time.Month, d, hh, mm, ss int) time.Time {
	return time.Date(y, m, d, hh, mm, ss, 0, time.UTC)
}

func TestCRMSearchDTO_Validate(t *testing.T) {
	t1 := tAt(2025, time.September, 8, 12, 0, 0)
	t2 := tAt(2025, time.September, 9, 12, 0, 0)

	tests := []struct {
		name   string
		dto    CRMSearchDTO
		wantN  int             // ожидаемое кол-во нарушений
		fields map[string]bool // какие поля должны присутствовать среди нарушений
	}{
		{
			name:  "ok_minimal",
			dto:   CRMSearchDTO{Page: 1, PerPage: 10},
			wantN: 0,
		},
		{
			name:   "invalid_page_zero",
			dto:    CRMSearchDTO{Page: 0, PerPage: 10},
			wantN:  1,
			fields: map[string]bool{"page": true},
		},
		{
			name:   "invalid_per_page_zero",
			dto:    CRMSearchDTO{Page: 1, PerPage: 0},
			wantN:  1,
			fields: map[string]bool{"per_page": true},
		},
		{
			name:   "invalid_per_page_too_big",
			dto:    CRMSearchDTO{Page: 1, PerPage: maxPerPage + 1},
			wantN:  1,
			fields: map[string]bool{"per_page": true},
		},
		{
			name:   "invalid_created_range_reversed",
			dto:    CRMSearchDTO{Page: 1, PerPage: 10, CreatedFrom: &t2, CreatedTo: &t1},
			wantN:  1,
			fields: map[string]bool{"created_from/created_to": true},
		},
		{
			name:   "invalid_updated_range_reversed",
			dto:    CRMSearchDTO{Page: 1, PerPage: 10, UpdatedFrom: &t2, UpdatedTo: &t1},
			wantN:  1,
			fields: map[string]bool{"updated_from/updated_to": true},
		},
		{
			name:  "multiple_violations",
			dto:   CRMSearchDTO{Page: 0, PerPage: maxPerPage + 1, CreatedFrom: &t2, CreatedTo: &t1},
			wantN: 3,
			fields: map[string]bool{
				"page":                    true,
				"per_page":                true,
				"created_from/created_to": true,
			},
		},
		{
			name:  "boundary_per_page_ok",
			dto:   CRMSearchDTO{Page: 1, PerPage: maxPerPage}, // ровно граница
			wantN: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viol := tc.dto.Validate()
			if len(viol) != tc.wantN {
				t.Fatalf("got %d violations, want %d; viol=%v", len(viol), tc.wantN, viol)
			}
			for field := range tc.fields {
				found := false
				for _, v := range viol {
					if v.Field == field {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected violation for field %q not found; viol=%v", field, viol)
				}
			}
		})
	}
}

func TestCRMSearchDTO_ToParams(t *testing.T) {
	q := "foo"
	inn := "123"
	cid := int64(42)
	open := true
	cf := tAt(2025, time.September, 8, 12, 0, 0)
	ct := tAt(2025, time.September, 9, 12, 0, 0)
	uf := tAt(2025, time.September, 10, 12, 0, 0)
	ut := tAt(2025, time.September, 11, 12, 0, 0)

	dto := CRMSearchDTO{
		Q: &q, Inn: &inn,
		CreatedFrom: &cf, CreatedTo: &ct,
		UpdatedFrom: &uf, UpdatedTo: &ut,
		CategoryID: &cid, Opened: &open,
		Page: 3, PerPage: 50,
	}
	p := dto.ToParams()

	if p.Q == nil || *p.Q != q {
		t.Fatalf("Q mismatch: got=%v", p.Q)
	}
	if p.Inn == nil || *p.Inn != inn {
		t.Fatalf("Inn mismatch: got=%v", p.Inn)
	}
	if p.CategoryID == nil || *p.CategoryID != cid {
		t.Fatalf("CategoryID mismatch: got=%v", p.CategoryID)
	}
	if p.Opened == nil || *p.Opened != open {
		t.Fatalf("Opened mismatch: got=%v", p.Opened)
	}
	if p.CreatedFrom == nil || !p.CreatedFrom.Equal(cf) {
		t.Fatalf("CreatedFrom mismatch: got=%v", p.CreatedFrom)
	}
	if p.CreatedTo == nil || !p.CreatedTo.Equal(ct) {
		t.Fatalf("CreatedTo mismatch: got=%v", p.CreatedTo)
	}
	if p.UpdatedFrom == nil || !p.UpdatedFrom.Equal(uf) {
		t.Fatalf("UpdatedFrom mismatch: got=%v", p.UpdatedFrom)
	}
	if p.UpdatedTo == nil || !p.UpdatedTo.Equal(ut) {
		t.Fatalf("UpdatedTo mismatch: got=%v", p.UpdatedTo)
	}
	if p.Page != dto.Page || p.PerPage != dto.PerPage {
		t.Fatalf("paging mismatch: got=(%d,%d) want=(%d,%d)", p.Page, p.PerPage, dto.Page, dto.PerPage)
	}
}
