package profile

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/humautil"
)

const (
	routeTag    = "Profile"
	profilePath = "/profile"
	avatarPath  = "/profile/avatar"
)

// No route here is Public(), so every one needs a token and a verified email.
func RegisterRoutes(api huma.API, profiles *ProfileHandler, sessions *SessionHandler) {
	registerProfileRoutes(api, profiles)
	registerSessionRoutes(api, sessions)
}

func registerProfileRoutes(api huma.API, h *ProfileHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-profile",
		Method:      http.MethodGet,
		Path:        profilePath,
		Summary:     "Get the signed-in user's profile, avatar and last login",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.GetProfile)

	huma.Register(api, huma.Operation{
		OperationID: "update-profile",
		Method:      http.MethodPatch,
		Path:        profilePath,
		Summary:     "Change some profile fields; username and email are read-only",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.UpdateProfile)

	huma.Register(api, huma.Operation{
		OperationID: "list-avatar-presets",
		Method:      http.MethodGet,
		Path:        "/profile/avatars",
		Summary:     "List the built-in avatars",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.ListAvatarPresets)

	huma.Register(api, huma.Operation{
		OperationID:  "upload-avatar",
		Method:       http.MethodPut,
		Path:         avatarPath,
		Summary:      "Upload your own avatar (JPEG or PNG, up to 2 MB); it is cropped square and resized",
		Tags:         []string{routeTag},
		Security:     humautil.BearerSecurity(),
		MaxBodyBytes: common.AVATAR_MAX_BYTES,
	}, h.UploadAvatar)

	huma.Register(api, huma.Operation{
		OperationID: "remove-avatar",
		Method:      http.MethodDelete,
		Path:        avatarPath,
		Summary:     "Remove your upload or preset choice; the Google/X picture or default shows again",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.RemoveAvatar)
}

func registerSessionRoutes(api huma.API, h *SessionHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "list-sessions",
		Method:      http.MethodGet,
		Path:        "/profile/sessions",
		Summary:     "List the devices that are signed in",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.ListSessions)

	huma.Register(api, huma.Operation{
		OperationID: "update-current-session-location",
		Method:      http.MethodPut,
		Path:        "/profile/sessions/current/location",
		Summary:     "Save this device's GPS position from the browser",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.UpdateCurrentLocation)

	huma.Register(api, huma.Operation{
		OperationID: "revoke-other-sessions",
		Method:      http.MethodPost,
		Path:        "/profile/sessions/revoke-others",
		Summary:     "Sign out every other device",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.RevokeOtherSessions)

	huma.Register(api, huma.Operation{
		OperationID: "revoke-session",
		Method:      http.MethodDelete,
		Path:        "/profile/sessions/{id}",
		Summary:     "Sign out one device",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.RevokeSession)
}
