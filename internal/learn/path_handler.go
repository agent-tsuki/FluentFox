package learn

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

// PathHandler serves what the learning path pages store besides answers:
// unit steps, kanji notes and test history. It is apart from Handler, which
// already has many services.
type PathHandler struct {
	units  *UnitProgressService
	notes  *KanjiNoteService
	tests  *TestHistoryService
	logger *zap.Logger
}

func NewPathHandler(units *UnitProgressService, notes *KanjiNoteService, tests *TestHistoryService, log *zap.Logger) *PathHandler {
	return &PathHandler{units: units, notes: notes, tests: tests, logger: log}
}

// Like RegisterRoutes, every route needs a token and a verified email.
func RegisterPathRoutes(api huma.API, h *PathHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "list-unit-progress",
		Method:      http.MethodGet,
		Path:        "/learn/units",
		Summary:     "List every unit of a track with the user's progress",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.ListUnitProgress)

	huma.Register(api, huma.Operation{
		OperationID: "mark-unit-step",
		Method:      http.MethodPut,
		Path:        "/learn/units/{type}/{id}/progress",
		Summary:     "Record a unit step (watched, traced, written, practised); the first time is kept",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.MarkUnitStep)

	huma.Register(api, huma.Operation{
		OperationID: "get-kanji-note",
		Method:      http.MethodGet,
		Path:        "/learn/kanji/{char}/note",
		Summary:     "Get the user's note on a kanji; empty text when there is none",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.GetKanjiNote)

	huma.Register(api, huma.Operation{
		OperationID: "update-kanji-note",
		Method:      http.MethodPut,
		Path:        "/learn/kanji/{char}/note",
		Summary:     "Save the user's note on a kanji; empty text deletes it",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.UpdateKanjiNote)

	huma.Register(api, huma.Operation{
		OperationID: "list-test-history",
		Method:      http.MethodGet,
		Path:        "/learn/tests/history",
		Summary:     "List the user's best result per test (kind and scope) and whether it was passed",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.ListTestHistory)
}

// ListUnitProgress handles GET /learn/units.
func (h *PathHandler) ListUnitProgress(
	ctx context.Context, input *UnitProgressListInput,
) (*humautil.Output[humautil.APIResponse[UnitProgressList]], error) {
	resp, err := h.units.List(ctx, middleware.ContextUserID(ctx), input.Track)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// MarkUnitStep handles PUT /learn/units/{type}/{id}/progress.
func (h *PathHandler) MarkUnitStep(ctx context.Context, input *UnitProgressInput) (*humautil.Output[humautil.APIResponse[UnitProgressData]], error) {
	resp, err := h.units.MarkStep(ctx, middleware.ContextUserID(ctx), input)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetKanjiNote handles GET /learn/kanji/{char}/note.
func (h *PathHandler) GetKanjiNote(ctx context.Context, input *KanjiCharInput) (*humautil.Output[humautil.APIResponse[KanjiNoteResponse]], error) {
	resp, err := h.notes.Get(ctx, middleware.ContextUserID(ctx), input.Char)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// UpdateKanjiNote handles PUT /learn/kanji/{char}/note.
func (h *PathHandler) UpdateKanjiNote(ctx context.Context, input *KanjiNoteInput) (*humautil.Output[humautil.APIResponse[KanjiNoteResponse]], error) {
	resp, err := h.notes.Put(ctx, middleware.ContextUserID(ctx), input.Char, &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// ListTestHistory handles GET /learn/tests/history.
func (h *PathHandler) ListTestHistory(ctx context.Context, input *TestHistoryInput) (*humautil.Output[humautil.APIResponse[TestHistory]], error) {
	resp, err := h.tests.List(ctx, middleware.ContextUserID(ctx), input.Track)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
