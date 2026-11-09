package audio

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

const adminRouteTag = "Audio admin"

// AdminServices are the super-admin audio pages' use cases.
type AdminServices struct {
	Voices  *VoiceListService
	Set     *VoiceSetService
	Engine  *EngineStatusService
	Preview *PreviewService
	Jobs    *JobService
}

type AdminHandler struct {
	svc    *AdminServices
	logger *zap.Logger
}

func NewAdminHandler(svc *AdminServices, log *zap.Logger) *AdminHandler {
	return &AdminHandler{svc: svc, logger: log}
}

// Every route needs a signed-in user; each service checks super-admin in the database.
func RegisterAdminRoutes(api huma.API, h *AdminHandler) {
	register(api, "list-audio-voices", http.MethodGet, "/admin/audio/voices",
		"List each voice key, its engine style and how many of its clips exist", h.ListVoices)
	register(api, "set-audio-voice", http.MethodPut, "/admin/audio/voices/{key}",
		"Give a voice key another engine style (its clips are made afresh)", h.SetVoice)
	register(api, "get-audio-engine", http.MethodGet, "/admin/audio/engine",
		"Whether the voice engine answers, and the voices and styles it has", h.EngineStatus)
	register(api, "preview-audio", http.MethodPost, "/admin/audio/preview",
		"Speak a text in any engine style to hear it; nothing is stored", h.Preview)
	register(api, "list-audio-jobs", http.MethodGet, "/admin/audio/jobs",
		"List the audio worker's recent jobs", h.ListJobs)
	register(api, "create-audio-job", http.MethodPost, "/admin/audio/jobs",
		"Queue a job for the audio worker: generate missing clips, or install a voice model", h.CreateJob)
	register(api, "cancel-audio-job", http.MethodPost, "/admin/audio/jobs/{id}/cancel",
		"Stop a queued or running audio job", h.CancelJob)
}

func register[I, O any](api huma.API, id, method, path, summary string, handler func(context.Context, *I) (*O, error)) {
	huma.Register(api, huma.Operation{
		OperationID: id,
		Method:      method,
		Path:        path,
		Summary:     summary,
		Tags:        []string{adminRouteTag},
		Security:    humautil.BearerSecurity(),
	}, handler)
}

func (h *AdminHandler) ListVoices(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[AdminVoicesResponse]], error) {
	resp, err := h.svc.Voices.List(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *AdminHandler) SetVoice(ctx context.Context, input *SetVoiceInput) (*humautil.Output[humautil.APIResponse[AdminVoice]], error) {
	v, err := h.svc.Set.Set(ctx, middleware.ContextUserID(ctx), input.Key, input.Body.StyleID)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(AdminVoice{
		Key: v.Key, Profile: v.Profile(), Engine: v.Engine, StyleID: v.StyleID, SpeakerName: v.SpeakerName,
		StyleName: v.StyleName, ModelUUID: v.ModelUUID, Credit: v.Credit, UpdatedAt: &v.UpdatedAt,
	}), nil
}

func (h *AdminHandler) EngineStatus(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[EngineStatus]], error) {
	resp, err := h.svc.Engine.Status(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *AdminHandler) Preview(ctx context.Context, input *humautil.Input[PreviewRequest]) (*humautil.Output[humautil.APIResponse[PreviewResponse]], error) {
	resp, err := h.svc.Preview.Preview(ctx, middleware.ContextUserID(ctx), input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *AdminHandler) ListJobs(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[JobsResponse]], error) {
	resp, err := h.svc.Jobs.List(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *AdminHandler) CreateJob(ctx context.Context, input *humautil.Input[CreateJobRequest]) (*humautil.Output[humautil.APIResponse[JobDTO]], error) {
	resp, err := h.svc.Jobs.Create(ctx, middleware.ContextUserID(ctx), input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *AdminHandler) CancelJob(ctx context.Context, input *JobIDPath) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	if err := h.svc.Jobs.Cancel(ctx, middleware.ContextUserID(ctx), input.ID); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "job cancelled"}), nil
}
