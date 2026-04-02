// Package ptr provides a generic helper for obtaining a pointer to a value.
//
// This is useful when constructing struct literals or calling APIs that accept
// pointer fields, avoiding the need for a temporary variable:
//
//	req := &pb.Request{PageSize: ptr.To(int32(20))}
package ptr

// To returns a pointer to a shallow copy of v.
func To[T any](v T) *T {
	return &v
}
