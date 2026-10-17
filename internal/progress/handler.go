package progress

import (
	"context"

	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type Handler struct {
	overview   *OverviewService
	forecast   *ForecastService
	calendar   *CalendarService
	weakItems  *WeakItemsService
	weakGroups *WeakGroupsService
	confusions *ConfusionService
	items      *ItemHistoryService
	dashboard  *DashboardService
	logger     *zap.Logger
}

func NewHandler(
	overview *OverviewService,
	forecast *ForecastService,
	calendar *CalendarService,
	weakItems *WeakItemsService,
	weakGroups *WeakGroupsService,
	confusions *ConfusionService,
	items *ItemHistoryService,
	dashboard *DashboardService,
	log *zap.Logger,
) *Handler {
	return &Handler{
		overview: overview, forecast: forecast, calendar: calendar, weakItems: weakItems, weakGroups: weakGroups,
		confusions: confusions, items: items, dashboard: dashboard, logger: log,
	}
}

type EmptyInput struct{}

// GetOverview handles GET /progress/overview.
func (h *Handler) GetOverview(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[ProgressOverview]], error) {
	resp, err := h.overview.Get(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetForecast handles GET /progress/forecast.
func (h *Handler) GetForecast(ctx context.Context, input *ForecastInput) (*humautil.Output[humautil.APIResponse[ProgressForecast]], error) {
	resp, err := h.forecast.Get(ctx, middleware.ContextUserID(ctx), input.Days)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetCalendar handles GET /progress/calendar.
func (h *Handler) GetCalendar(ctx context.Context, input *CalendarInput) (*humautil.Output[humautil.APIResponse[ProgressCalendar]], error) {
	resp, err := h.calendar.Get(ctx, middleware.ContextUserID(ctx), input)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetWeakItems handles GET /progress/weak.
func (h *Handler) GetWeakItems(ctx context.Context, input *WeakItemsInput) (*humautil.Output[humautil.APIResponse[WeakItemsResponse]], error) {
	resp, err := h.weakItems.Get(ctx, middleware.ContextUserID(ctx), input)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetWeakGroups handles GET /progress/weak-groups.
func (h *Handler) GetWeakGroups(ctx context.Context, input *TrackInput) (*humautil.Output[humautil.APIResponse[WeakGroupsResponse]], error) {
	resp, err := h.weakGroups.Get(ctx, middleware.ContextUserID(ctx), input.Track)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetConfusions handles GET /progress/confusions.
func (h *Handler) GetConfusions(ctx context.Context, input *ConfusionsInput) (*humautil.Output[humautil.APIResponse[ConfusionsResponse]], error) {
	resp, err := h.confusions.Get(ctx, middleware.ContextUserID(ctx), input)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetItemHistory handles GET /progress/items/{type}/{id}.
func (h *Handler) GetItemHistory(ctx context.Context, input *ItemHistoryInput) (*humautil.Output[humautil.APIResponse[ItemHistory]], error) {
	resp, err := h.items.Get(ctx, middleware.ContextUserID(ctx), input)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetDashboard handles GET /progress/dashboard.
func (h *Handler) GetDashboard(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[DashboardData]], error) {
	resp, err := h.dashboard.Get(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
