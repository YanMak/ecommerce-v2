package paging

type OffsetParams struct {
	Page    int32
	PerPage int32
}

type OffsetOpts struct {
	DefaultPerPage int32 // напр. 20
	MaxPerPage     int32 // напр. 200
}

type OffsetResult struct {
	Limit   int32
	Offset  int32
	Page    int32
	PerPage int32
}

func NormalizeOffset(p OffsetParams, o OffsetOpts) OffsetResult {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = o.DefaultPerPage
	}
	if o.MaxPerPage > 0 && p.PerPage > o.MaxPerPage {
		p.PerPage = o.MaxPerPage
	}
	return OffsetResult{
		Limit:   p.PerPage,
		Offset:  (p.Page - 1) * p.PerPage,
		Page:    p.Page,
		PerPage: p.PerPage,
	}
}
