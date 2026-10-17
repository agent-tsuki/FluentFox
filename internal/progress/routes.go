package progress

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

// No route here is Public(), so every one needs a token and a verified email.
func RegisterRoutes(api huma.API, h *Handler) {
	registerSummaryRoutes(api, h)
	registerDetailRoutes(api, h)
}

func registerSummaryRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-progress-overview",
		Method:      http.MethodGet,
		Path:        "/progress/overview",
		Summary:     "Items per mastery level, retention and answer time per track; today; streak",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetOverview)

	huma.Register(api, huma.Operation{
		OperationID: "get-progress-forecast",
		Method:      http.MethodGet,
		Path:        "/progress/forecast",
		Summary:     "Reviews due per local day and track; overdue reviews count on today",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetForecast)

	huma.Register(api, huma.Operation{
		OperationID: "get-progress-calendar",
		Method:      http.MethodGet,
		Path:        "/progress/calendar",
		Summary:     "Answers, minutes, accuracy and goal per local day, with streaks",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetCalendar)
}

func registerDetailRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-weak-items",
		Method:      http.MethodGet,
		Path:        "/progress/weak",
		Summary:     "Weakest cards with reasons and a practice suggestion",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetWeakItems)

	huma.Register(api, huma.Operation{
		OperationID: "get-weak-groups",
		Method:      http.MethodGet,
		Path:        "/progress/weak-groups",
		Summary:     "Units and kana groups missed more often, or answered slower, than the user's average",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetWeakGroups)

	huma.Register(api, huma.Operation{
		OperationID: "get-confusions",
		Method:      http.MethodGet,
		Path:        "/progress/confusions",
		Summary:     "The user's most frequent mix-ups",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetConfusions)

	huma.Register(api, huma.Operation{
		OperationID: "get-item-history",
		Method:      http.MethodGet,
		Path:        "/progress/items/{type}/{id}",
		Summary:     "Per-card level, recall chance, next due, stats and the last 20 answers of an item",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetItemHistory)

	huma.Register(api, huma.Operation{
		OperationID: "get-progress-dashboard",
		Method:      http.MethodGet,
		Path:        "/progress/dashboard",
		Summary:     "The dashboard in the frontend DashboardData shape",
		Tags:        []string{"Progress"},
		Security:    humautil.BearerSecurity(),
	}, h.GetDashboard)
}
