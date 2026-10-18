package profile

import (
	"strings"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
)

// AvatarResolver decides which avatar a profile shows and builds its URL.
type AvatarResolver struct {
	uploadBaseURL string
	presets       map[string]common.AvatarPreset
}

// uploadBaseURL is the public R2 origin (R2_PUBLIC_URL), e.g.
// https://cdn.foxsensei.com.
func NewAvatarResolver(uploadBaseURL string) *AvatarResolver {
	presets := make(map[string]common.AvatarPreset, len(common.AVATAR_PRESETS))
	for _, p := range common.AVATAR_PRESETS {
		presets[p.ID] = p
	}
	return &AvatarResolver{uploadBaseURL: strings.TrimRight(uploadBaseURL, "/"), presets: presets}
}

// A choice the user made (upload, then preset) always beats the Google/X
// picture, which in turn beats the default.
func (a *AvatarResolver) For(p *users.UserProfile) AvatarResponse {
	if p.ProfileImage != nil && *p.ProfileImage != "" && a.uploadBaseURL != "" {
		return AvatarResponse{Kind: AvatarKindUpload, URL: a.uploadBaseURL + "/" + *p.ProfileImage}
	}
	if p.AvatarPreset != nil {
		if preset, ok := a.presets[*p.AvatarPreset]; ok {
			return presetAvatar(AvatarKindPreset, preset)
		}
	}
	if p.OAuthImageURL != nil && *p.OAuthImageURL != "" {
		return AvatarResponse{Kind: AvatarKindOAuth, URL: *p.OAuthImageURL}
	}
	return presetAvatar(AvatarKindDefault, a.presets[common.DEFAULT_AVATAR_PRESET_ID])
}

func (a *AvatarResolver) IsPreset(id string) bool {
	_, ok := a.presets[id]
	return ok
}

func (a *AvatarResolver) Presets() []AvatarPresetResponse {
	out := make([]AvatarPresetResponse, 0, len(common.AVATAR_PRESETS))
	for _, p := range common.AVATAR_PRESETS {
		out = append(out, AvatarPresetResponse{ID: p.ID, Group: string(p.Group), URL: p.Path})
	}
	return out
}

func presetAvatar(kind string, p common.AvatarPreset) AvatarResponse {
	id := p.ID
	return AvatarResponse{Kind: kind, PresetID: &id, URL: p.Path}
}
