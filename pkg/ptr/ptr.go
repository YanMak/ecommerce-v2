package ptr

// To возвращает &v.
func To[T any](v T) *T { return &v }

// Val возвращает значение или zero, если p == nil.
func Val[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// ValOr возвращает *p или def, если p == nil.
func ValOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}
