// Package ptr provides a generic helper for obtaining a pointer to a value.
//
// This is useful when constructing struct literals or calling APIs that accept
// pointer fields, avoiding the need for a temporary variable:
//
//	req := &pb.Request{PageSize: ptr.Of(int32(20))}
package ptr

// To dereferences p, returning the zero value for T when p is nil.
func To[T any](p *T) T {
	if p == nil {
		var v T
		return v
	}
	return *p
}

// Of returns a pointer to a shallow copy of v.
func Of[T any](v T) *T {
	return &v
}
