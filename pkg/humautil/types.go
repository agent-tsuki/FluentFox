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

// APIResponse is the standard success envelope for every API endpoint.
//
//	{ "status": "success", "data": { ... } }
type APIResponse[T any] struct {
	Status string `json:"status" doc:"Always 'success' for 2xx responses"`
	Data   T      `json:"data"   doc:"Response payload"`
}

// PagedResponse is the standard envelope for paginated list endpoints.
//
//	{ "status": "success", "data": [...], "current_page": 1, "total_pages": 5, ... }
type PagedResponse[T any] struct {
	Status      string `json:"status"`
	Data        []T    `json:"data"`
	CurrentPage int    `json:"current_page" doc:"Current page number (1-based)"`
	TotalPages  int    `json:"total_pages"  doc:"Total number of pages"`
	TotalItems  int64  `json:"total_items"  doc:"Total number of items across all pages"`
	PerPage     int    `json:"per_page"     doc:"Items per page"`
}

// OK wraps data in a success APIResponse ready to return from a handler.
func OK[T any](data T) *Output[APIResponse[T]] {
	return &Output[APIResponse[T]]{Body: APIResponse[T]{Status: "success", Data: data}}
}

// OKPaged wraps a slice in a success PagedResponse ready to return from a handler.
func OKPaged[T any](data []T, currentPage, totalPages int, totalItems int64, perPage int) *Output[PagedResponse[T]] {
	return &Output[PagedResponse[T]]{Body: PagedResponse[T]{
		Status:      "success",
		Data:        data,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
		PerPage:     perPage,
	}}
}
