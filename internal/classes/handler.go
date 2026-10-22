package classes

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type CatalogHandler struct {
	list   *ListCoursesService
	detail *CourseDetailService
	logger *zap.Logger
}

func NewCatalogHandler(list *ListCoursesService, detail *CourseDetailService, log *zap.Logger) *CatalogHandler {
	return &CatalogHandler{list: list, detail: detail, logger: log}
}

// ChannelAdminServices groups the channel use cases so the handler
// constructor stays readable.
type ChannelAdminServices struct {
	List       *ChannelListService
	Read       *ChannelReadService
	Create     *ChannelCreateService
	Update     *ChannelUpdateService
	Media      *ChannelMediaService
	IntroVideo *IntroVideoService
	Reset      *ResetFieldService
	Visibility *ChannelVisibilityService
	Sync       *ChannelSyncService
	History    *ChannelHistoryService
}

type ChannelAdminHandler struct {
	svc    ChannelAdminServices
	logger *zap.Logger
}

func NewChannelAdminHandler(svc *ChannelAdminServices, log *zap.Logger) *ChannelAdminHandler {
	return &ChannelAdminHandler{svc: *svc, logger: log}
}

type CourseAdminServices struct {
	List       *CourseAdminListService
	Get        *CourseAdminGetService
	Create     *CourseCreateService
	Update     *CourseUpdateService
	Delete     *CourseDeleteService
	Visibility *CourseVisibilityService
	Sync       *PlaylistSyncService
}

type CourseAdminHandler struct {
	svc    CourseAdminServices
	logger *zap.Logger
}

func NewCourseAdminHandler(svc *CourseAdminServices, log *zap.Logger) *CourseAdminHandler {
	return &CourseAdminHandler{svc: *svc, logger: log}
}

// Response shapes, named so handler signatures stay readable.
type (
	courseListOutput    = humautil.Output[humautil.APIResponse[CourseListResponse]]
	courseDetailOutput  = humautil.Output[humautil.APIResponse[CourseDetail]]
	adminChannelOutput  = humautil.Output[humautil.APIResponse[AdminChannel]]
	adminChannelsOutput = humautil.Output[humautil.APIResponse[[]AdminChannel]]
	historyOutput       = humautil.Output[humautil.APIResponse[[]ChangeLogResponse]]
	adminCourseOutput   = humautil.Output[humautil.APIResponse[AdminCourse]]
	adminCoursesOutput  = humautil.Output[humautil.APIResponse[[]AdminCourse]]
	messageOutput       = humautil.Output[humautil.APIResponse[humautil.MessageBody]]
)

type EmptyInput struct{}

func (h *CatalogHandler) ListCourses(ctx context.Context, input *ListCoursesInput) (*courseListOutput, error) {
	filter := CatalogFilter{Level: common.JLPTLevel(input.Level), Platform: common.ClassPlatform(input.Platform)}
	if input.Free != "" {
		free, err := strconv.ParseBool(input.Free)
		if err != nil {
			return nil, humautil.MapErr(exceptions.ErrClassInputInvalid, middleware.LoggerFromContext(ctx, h.logger))
		}
		filter.Free = &free
	}
	resp, err := h.list.List(ctx, filter)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *CatalogHandler) GetCourse(ctx context.Context, input *CourseSlugInput) (*courseDetailOutput, error) {
	resp, err := h.detail.Get(ctx, input.Slug)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *ChannelAdminHandler) channel(ctx context.Context, resp *AdminChannel, err error) (*adminChannelOutput, error) {
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(*resp), nil
}

func (h *ChannelAdminHandler) ListChannels(ctx context.Context, _ *EmptyInput) (*adminChannelsOutput, error) {
	resp, err := h.svc.List.List(ctx)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *ChannelAdminHandler) GetChannel(ctx context.Context, input *IDInput) (*adminChannelOutput, error) {
	resp, err := h.svc.Read.Get(ctx, input.ID)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) CreateChannel(ctx context.Context, input *humautil.Input[CreateChannelRequest]) (*adminChannelOutput, error) {
	resp, err := h.svc.Create.Create(ctx, &input.Body)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) UpdateChannel(ctx context.Context, input *IDBodyInput[UpdateChannelRequest]) (*adminChannelOutput, error) {
	resp, err := h.svc.Update.Update(ctx, input.ID, &input.Body)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) UploadAvatar(ctx context.Context, input *ImageUploadInput) (*adminChannelOutput, error) {
	resp, err := h.svc.Media.UploadAvatar(ctx, input.ID, input.RawBody)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) UploadBanner(ctx context.Context, input *ImageUploadInput) (*adminChannelOutput, error) {
	resp, err := h.svc.Media.UploadBanner(ctx, input.ID, input.RawBody)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) SetIntroVideo(ctx context.Context, input *IDBodyInput[IntroVideoRequest]) (*adminChannelOutput, error) {
	resp, err := h.svc.IntroVideo.Set(ctx, input.ID, input.Body.VideoID)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) ResetField(ctx context.Context, input *ResetFieldInput) (*adminChannelOutput, error) {
	resp, err := h.svc.Reset.Reset(ctx, input.ID, common.ChannelField(input.Field))
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) SetVisibility(ctx context.Context, input *IDBodyInput[ChannelVisibilityRequest]) (*adminChannelOutput, error) {
	resp, err := h.svc.Visibility.Set(ctx, input.ID, input.Body.IsPublished)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) SyncChannel(ctx context.Context, input *IDInput) (*adminChannelOutput, error) {
	resp, err := h.svc.Sync.Sync(ctx, input.ID)
	return h.channel(ctx, &resp, err)
}

func (h *ChannelAdminHandler) History(ctx context.Context, input *IDInput) (*historyOutput, error) {
	resp, err := h.svc.History.List(ctx, input.ID)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *CourseAdminHandler) course(ctx context.Context, resp *AdminCourse, err error) (*adminCourseOutput, error) {
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(*resp), nil
}

func (h *CourseAdminHandler) ListCourses(ctx context.Context, input *AdminCoursesInput) (*adminCoursesOutput, error) {
	log := middleware.LoggerFromContext(ctx, h.logger)
	var channelID *uuid.UUID
	if input.ChannelID != "" {
		id, err := uuid.Parse(input.ChannelID)
		if err != nil {
			return nil, humautil.MapErr(exceptions.ErrClassInputInvalid, log)
		}
		channelID = &id
	}
	resp, err := h.svc.List.List(ctx, channelID)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}
	return humautil.OK(resp), nil
}

func (h *CourseAdminHandler) GetCourse(ctx context.Context, input *IDInput) (*adminCourseOutput, error) {
	resp, err := h.svc.Get.Get(ctx, input.ID)
	return h.course(ctx, &resp, err)
}

func (h *CourseAdminHandler) CreateCourse(ctx context.Context, input *humautil.Input[CreateCourseRequest]) (*adminCourseOutput, error) {
	resp, err := h.svc.Create.Create(ctx, &input.Body)
	return h.course(ctx, &resp, err)
}

func (h *CourseAdminHandler) UpdateCourse(ctx context.Context, input *IDBodyInput[UpdateCourseRequest]) (*adminCourseOutput, error) {
	resp, err := h.svc.Update.Update(ctx, input.ID, &input.Body)
	return h.course(ctx, &resp, err)
}

func (h *CourseAdminHandler) DeleteCourse(ctx context.Context, input *IDInput) (*messageOutput, error) {
	if err := h.svc.Delete.Delete(ctx, input.ID); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "course deleted"}), nil
}

func (h *CourseAdminHandler) SetVisibility(ctx context.Context, input *IDBodyInput[CourseVisibilityRequest]) (*adminCourseOutput, error) {
	resp, err := h.svc.Visibility.Set(ctx, input.ID, &input.Body)
	return h.course(ctx, &resp, err)
}

func (h *CourseAdminHandler) SyncPlaylist(ctx context.Context, input *IDInput) (*adminCourseOutput, error) {
	resp, err := h.svc.Sync.Sync(ctx, input.ID)
	return h.course(ctx, &resp, err)
}
