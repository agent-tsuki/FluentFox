package humautil

// Input wraps a typed request body so huma handlers don't need per-endpoint
// input structs when the only field is Body.
//
//	func (h *Handler) CreateFoo(ctx context.Context, in *humautil.Input[CreateFooRequest]) ...
type Input[T any] struct {
	Body T
}

// Output wraps a typed response body so huma handlers don't need per-endpoint
// output structs when the only field is Body.
//
//	func (h *Handler) CreateFoo(...) (*humautil.Output[FooResponse], error)
type Output[T any] struct {
	Body T
}

// MessageBody is the standard envelope for endpoints that return only a
// human-readable confirmation string (register, verify, logout, etc.).
type MessageBody struct {
	Message string `json:"message" doc:"Human-readable response message"`
}
