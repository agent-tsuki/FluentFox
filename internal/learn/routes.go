package learn

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

// No route here is Public(), so every one needs a token and a verified email.
func RegisterRoutes(api huma.API, h *Handler) {
	registerOverviewRoutes(api, h)
	registerSessionRoutes(api, h)
}

func registerOverviewRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-learning-settings",
		Method:      http.MethodGet,
		Path:        "/learn/settings",
		Summary:     "Get the learning settings, created with defaults on first read",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.GetSettings)

	huma.Register(api, huma.Operation{
		OperationID: "update-learning-settings",
		Method:      http.MethodPut,
		Path:        "/learn/settings",
		Summary:     "Change some learning settings",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.UpdateSettings)

	huma.Register(api, huma.Operation{
		OperationID: "get-learn-summary",
		Method:      http.MethodGet,
		Path:        "/learn/summary",
		Summary:     "Get due counts, new items left today, next unit and goal progress",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.GetSummary)
}

func registerSessionRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID:   "start-lesson",
		Method:        http.MethodPost,
		Path:          "/learn/lessons",
		Summary:       "Start a lesson: new items in path order and their check quiz",
		Tags:          []string{"Learn"},
		DefaultStatus: http.StatusCreated,
		Security:      humautil.BearerSecurity(),
	}, h.StartLesson)

	huma.Register(api, huma.Operation{
		OperationID:   "start-review",
		Method:        http.MethodPost,
		Path:          "/learn/reviews",
		Summary:       "Start a review session with the cards due now",
		Tags:          []string{"Learn"},
		DefaultStatus: http.StatusCreated,
		Security:      humautil.BearerSecurity(),
	}, h.StartReview)

	huma.Register(api, huma.Operation{
		OperationID:   "submit-session-answers",
		Method:        http.MethodPost,
		Path:          "/learn/sessions/{id}/answers",
		Summary:       "Submit 1 to 50 answers; a resent answer returns its stored result",
		Tags:          []string{"Learn"},
		DefaultStatus: http.StatusOK,
		Security:      humautil.BearerSecurity(),
	}, h.SubmitAnswers)

	huma.Register(api, huma.Operation{
		OperationID:   "complete-session",
		Method:        http.MethodPost,
		Path:          "/learn/sessions/{id}/complete",
		Summary:       "Complete a session and get its result screen",
		Tags:          []string{"Learn"},
		DefaultStatus: http.StatusOK,
		Security:      humautil.BearerSecurity(),
	}, h.CompleteSession)
}
