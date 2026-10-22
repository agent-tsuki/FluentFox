package classes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/youtube"
)

type ChannelListService struct {
	repo   ChannelLister
	grants GrantLister
	logger *zap.Logger
}

func NewChannelListService(repo ChannelLister, grants GrantLister, log *zap.Logger) *ChannelListService {
	return &ChannelListService{repo: repo, grants: grants, logger: log}
}

// List returns only the channels the user may work on; a user with no
// classes grant at all is refused.
func (s *ChannelListService) List(ctx context.Context) ([]AdminChannel, error) {
	all, rawIDs, err := s.grants.EditableChannelIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list editable channels: %w", err)
	}
	var ids []uuid.UUID
	if !all {
		if len(rawIDs) == 0 {
			return nil, exceptions.ErrForbidden
		}
		ids = make([]uuid.UUID, 0, len(rawIDs))
		for _, raw := range rawIDs {
			// A grant can outlive a malformed or deleted id; it simply matches nothing.
			if id, err := uuid.Parse(raw); err == nil {
				ids = append(ids, id)
			}
		}
	}
	channels, err := s.repo.ListChannels(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list editable channels: %w", err)
	}
	out := make([]AdminChannel, 0, len(channels))
	for i := range channels {
		out = append(out, toAdminChannel(&channels[i]))
	}
	return out, nil
}

type ChannelReadService struct {
	repo   ChannelReader
	perms  PermissionChecker
	logger *zap.Logger
}

func NewChannelReadService(repo ChannelReader, perms PermissionChecker, log *zap.Logger) *ChannelReadService {
	return &ChannelReadService{repo: repo, perms: perms, logger: log}
}

func (s *ChannelReadService) Get(ctx context.Context, id uuid.UUID) (AdminChannel, error) {
	if err := requireAny(ctx, s.perms, channelRef(id), common.PERMISSION_CHANNEL_EDIT, common.PERMISSION_CHANNEL_MEDIA); err != nil {
		return AdminChannel{}, err
	}
	ch, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return AdminChannel{}, notFoundAs(err, exceptions.ErrChannelNotFound, "get channel")
	}
	return toAdminChannel(&ch), nil
}

type ChannelCreateService struct {
	repo   ChannelWriter
	perms  PermissionChecker
	logger *zap.Logger
}

func NewChannelCreateService(repo ChannelWriter, perms PermissionChecker, log *zap.Logger) *ChannelCreateService {
	return &ChannelCreateService{repo: repo, perms: perms, logger: log}
}

// New channels start hidden. The name is an editor's value, so it goes in
// name_custom; a later sync fills name_youtube without replacing it.
func (s *ChannelCreateService) Create(ctx context.Context, req *CreateChannelRequest) (AdminChannel, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CHANNEL_EDIT, common.ResourceRef{}); err != nil {
		return AdminChannel{}, fmt.Errorf("create channel: %w", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return AdminChannel{}, exceptions.ErrClassInputInvalid
	}
	ch := Channel{
		Slug: req.Slug, Platform: common.ClassPlatform(req.Platform), URL: req.URL,
		YouTubeChannelID: req.YouTubeChannelID, Handle: req.Handle, NameCustom: &name, AccentColor: req.AccentColor,
		BestFor: database.TextArray{}, NotFor: database.TextArray{}, Pros: database.TextArray{}, Cons: database.TextArray{},
	}
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.CreateChannel(ctx, tx, &ch); err != nil {
			return fmt.Errorf("insert channel: %w", err)
		}
		log := ChangeLog{
			ActorID: middleware.ContextUserID(ctx), EntityType: string(common.RESOURCE_TYPE_CHANNEL),
			EntityID: ch.ID, Field: logFieldCreated, NewValue: &ch.Slug, CreatedAt: time.Now(),
		}
		if err := s.repo.AppendChangeLog(ctx, tx, []ChangeLog{log}); err != nil {
			return fmt.Errorf("log channel creation: %w", err)
		}
		return nil
	})
	if err != nil {
		return AdminChannel{}, fmt.Errorf("create channel: %w", err)
	}
	s.logger.Info("class channel created", zap.String("slug", ch.Slug), zap.Stringer("channel_id", ch.ID))
	return toAdminChannel(&ch), nil
}

type ChannelUpdateService struct {
	repo   ChannelWriter
	perms  PermissionChecker
	logger *zap.Logger
}

func NewChannelUpdateService(repo ChannelWriter, perms PermissionChecker, log *zap.Logger) *ChannelUpdateService {
	return &ChannelUpdateService{repo: repo, perms: perms, logger: log}
}

func (s *ChannelUpdateService) Update(ctx context.Context, id uuid.UUID, req *UpdateChannelRequest) (AdminChannel, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CHANNEL_EDIT, channelRef(id)); err != nil {
		return AdminChannel{}, fmt.Errorf("update channel: %w", err)
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return AdminChannel{}, exceptions.ErrClassInputInvalid
	}
	_, after, err := updateChannel(ctx, s.repo, id, func(_ *gorm.DB, ch *Channel, cs *changeSet) error {
		applyChannelUpdate(ch, req, cs)
		return nil
	})
	if err != nil {
		return AdminChannel{}, err
	}
	return toAdminChannel(&after), nil
}

func applyChannelUpdate(ch *Channel, req *UpdateChannelRequest, cs *changeSet) {
	if req.Name != nil {
		cs.Text(customColumn(common.ChannelFieldName), ch.NameCustom, req.Name)
	}
	if req.Description != nil {
		cs.Text(customColumn(common.ChannelFieldDescription), ch.DescriptionCustom, req.Description)
	}
	setText(cs, "youtube_channel_id", ch.YouTubeChannelID, req.YouTubeChannelID)
	setText(cs, "handle", ch.Handle, req.Handle)
	setText(cs, "accent_color", ch.AccentColor, req.AccentColor)
	if req.URL != nil {
		cs.Str("url", ch.URL, *req.URL)
	}
	if req.TeachingStyle != nil {
		cs.Str("teaching_style", ch.TeachingStyle, *req.TeachingStyle)
	}
	if req.Verdict != nil {
		cs.Str("verdict", ch.Verdict, *req.Verdict)
	}
	setList(cs, "best_for", ch.BestFor, req.BestFor)
	setList(cs, "not_for", ch.NotFor, req.NotFor)
	setList(cs, "pros", ch.Pros, req.Pros)
	setList(cs, "cons", ch.Cons, req.Cons)
}

// An empty string clears an optional column.
func setText(cs *changeSet, column string, current, requested *string) {
	if requested == nil {
		return
	}
	if *requested == "" {
		cs.Text(column, current, nil)
		return
	}
	cs.Text(column, current, requested)
}

func setList(cs *changeSet, column string, current database.TextArray, requested []string) {
	if requested == nil {
		return
	}
	cs.Value(column, listOrEmpty(current), database.TextArray(requested))
}

type ChannelVisibilityService struct {
	repo   ChannelWriter
	perms  PermissionChecker
	logger *zap.Logger
}

func NewChannelVisibilityService(repo ChannelWriter, perms PermissionChecker, log *zap.Logger) *ChannelVisibilityService {
	return &ChannelVisibilityService{repo: repo, perms: perms, logger: log}
}

func (s *ChannelVisibilityService) Set(ctx context.Context, id uuid.UUID, published bool) (AdminChannel, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CLASSES_PUBLISH, channelRef(id)); err != nil {
		return AdminChannel{}, fmt.Errorf("set channel visibility: %w", err)
	}
	_, after, err := updateChannel(ctx, s.repo, id, func(_ *gorm.DB, ch *Channel, cs *changeSet) error {
		cs.Value("is_published", ch.IsPublished, published)
		return nil
	})
	if err != nil {
		return AdminChannel{}, err
	}
	return toAdminChannel(&after), nil
}

type IntroVideoService struct {
	repo   ChannelWriter
	perms  PermissionChecker
	logger *zap.Logger
}

func NewIntroVideoService(repo ChannelWriter, perms PermissionChecker, log *zap.Logger) *IntroVideoService {
	return &IntroVideoService{repo: repo, perms: perms, logger: log}
}

// Set accepts a bare id or any YouTube video URL and stores only the id.
func (s *IntroVideoService) Set(ctx context.Context, id uuid.UUID, input string) (AdminChannel, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CHANNEL_MEDIA, channelRef(id)); err != nil {
		return AdminChannel{}, fmt.Errorf("set intro video: %w", err)
	}
	videoID, err := youtube.ParseVideoID(input)
	if err != nil {
		return AdminChannel{}, exceptions.ErrVideoIDInvalid
	}
	_, after, err := updateChannel(ctx, s.repo, id, func(_ *gorm.DB, ch *Channel, cs *changeSet) error {
		cs.Text(customColumn(common.ChannelFieldIntroVideoID), ch.IntroVideoIDCustom, &videoID)
		return nil
	})
	if err != nil {
		return AdminChannel{}, err
	}
	return toAdminChannel(&after), nil
}

type ResetFieldService struct {
	repo   ChannelWriter
	perms  PermissionChecker
	images *imageCleaner
	logger *zap.Logger
}

func NewResetFieldService(repo ChannelWriter, perms PermissionChecker, store ObjectStore, uploadBaseURL string, log *zap.Logger) *ResetFieldService {
	return &ResetFieldService{repo: repo, perms: perms, images: newImageCleaner(store, uploadBaseURL, log), logger: log}
}

// Reset clears the custom value so the YouTube value shows again. A cleared
// upload is deleted from storage after the change is saved.
func (s *ResetFieldService) Reset(ctx context.Context, id uuid.UUID, field common.ChannelField) (AdminChannel, error) {
	var probe Channel
	if _, ok := probe.fieldPair(field); !ok {
		return AdminChannel{}, exceptions.ErrClassFieldInvalid
	}
	perm := common.PERMISSION_CHANNEL_EDIT
	if common.CHANNEL_MEDIA_FIELDS[field] {
		perm = common.PERMISSION_CHANNEL_MEDIA
	}
	if err := s.perms.Require(ctx, perm, channelRef(id)); err != nil {
		return AdminChannel{}, fmt.Errorf("reset channel field: %w", err)
	}
	before, after, err := updateChannel(ctx, s.repo, id, func(_ *gorm.DB, ch *Channel, cs *changeSet) error {
		p, _ := ch.fieldPair(field)
		cs.Text(customColumn(field), p.Custom, nil)
		return nil
	})
	if err != nil {
		return AdminChannel{}, err
	}
	old, _ := before.fieldPair(field)
	s.images.Delete(ctx, old.Custom)
	return toAdminChannel(&after), nil
}

type ChannelHistoryService struct {
	repo   HistoryReader
	perms  PermissionChecker
	logger *zap.Logger
}

func NewChannelHistoryService(repo HistoryReader, perms PermissionChecker, log *zap.Logger) *ChannelHistoryService {
	return &ChannelHistoryService{repo: repo, perms: perms, logger: log}
}

func (s *ChannelHistoryService) List(ctx context.Context, id uuid.UUID) ([]ChangeLogResponse, error) {
	if err := requireAny(ctx, s.perms, channelRef(id), classesPermissions...); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListChannelHistory(ctx, id, common.CLASS_HISTORY_LIMIT)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("list channel history: %w", err)
	}
	out := make([]ChangeLogResponse, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, ChangeLogResponse{
			ID: r.ID, ActorID: r.ActorID, ActorUsername: r.ActorUsername,
			EntityType: r.EntityType, EntityID: r.EntityID, Field: r.Field,
			OldValue: r.OldValue, NewValue: r.NewValue, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}
