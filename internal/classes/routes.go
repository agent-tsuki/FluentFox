package classes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/humautil"
)

const (
	routeTag      = "Classes"
	adminRouteTag = "Classes admin"
	channelsPath  = "/admin/classes/channels"
	channelPath   = "/admin/classes/channels/{id}"
	coursesPath   = "/admin/classes/courses"
	coursePath    = "/admin/classes/courses/{id}"
)

// The catalog is Public(); every admin route needs a token and is checked
// against the user's grants inside the services.
func RegisterRoutes(api huma.API, catalog *CatalogHandler, channels *ChannelAdminHandler, courses *CourseAdminHandler) {
	registerCatalogRoutes(api, catalog)
	registerChannelRoutes(api, channels)
	registerChannelMediaRoutes(api, channels)
	registerCourseRoutes(api, courses)
}

func registerCatalogRoutes(api huma.API, h *CatalogHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "list-classes",
		Method:      http.MethodGet,
		Path:        "/classes",
		Summary:     "List published courses, with per-level counts and locked levels",
		Tags:        []string{routeTag},
		Metadata:    humautil.Public(),
	}, h.ListCourses)

	huma.Register(api, huma.Operation{
		OperationID: "get-class",
		Method:      http.MethodGet,
		Path:        "/classes/{slug}",
		Summary:     "Get a published course with its videos and channel introduction",
		Tags:        []string{routeTag},
		Metadata:    humautil.Public(),
	}, h.GetCourse)
}

func adminOp(id, method, path, summary string) huma.Operation {
	return huma.Operation{
		OperationID: id,
		Method:      method,
		Path:        path,
		Summary:     summary,
		Tags:        []string{adminRouteTag},
		Security:    humautil.BearerSecurity(),
	}
}

func registerChannelRoutes(api huma.API, h *ChannelAdminHandler) {
	huma.Register(api, adminOp("list-admin-class-channels", http.MethodGet, channelsPath,
		"List the channels you may edit"), h.ListChannels)
	huma.Register(api, adminOp("get-admin-class-channel", http.MethodGet, channelPath,
		"Get a channel with the YouTube, custom and effective value of each field"), h.GetChannel)
	huma.Register(api, adminOp("create-class-channel", http.MethodPost, channelsPath,
		"Create a channel; it starts hidden"), h.CreateChannel)
	huma.Register(api, adminOp("update-class-channel", http.MethodPatch, channelPath,
		"Change channel text fields; name and description set the custom value"), h.UpdateChannel)
	huma.Register(api, adminOp("set-class-channel-visibility", http.MethodPatch, channelPath+"/visibility",
		"Publish or hide a channel and all its courses"), h.SetVisibility)
	huma.Register(api, adminOp("sync-class-channel", http.MethodPost, channelPath+"/sync",
		"Refresh the channel's YouTube values; custom values are kept"), h.SyncChannel)
	huma.Register(api, adminOp("list-class-channel-history", http.MethodGet, channelPath+"/history",
		"List recent changes to the channel and its courses"), h.History)
}

func registerChannelMediaRoutes(api huma.API, h *ChannelAdminHandler) {
	avatar := adminOp("upload-class-channel-avatar", http.MethodPost, channelPath+"/avatar",
		"Upload a channel picture (JPEG or PNG); it is cropped square and resized")
	avatar.MaxBodyBytes = common.AVATAR_MAX_BYTES
	huma.Register(api, avatar, h.UploadAvatar)

	banner := adminOp("upload-class-channel-banner", http.MethodPost, channelPath+"/banner",
		"Upload a channel banner (JPEG or PNG); it keeps its shape and is at most 2048px wide")
	banner.MaxBodyBytes = common.CLASS_BANNER_MAX_BYTES
	huma.Register(api, banner, h.UploadBanner)

	huma.Register(api, adminOp("set-class-channel-intro-video", http.MethodPatch, channelPath+"/intro-video",
		"Set the intro video from a video id or any YouTube URL"), h.SetIntroVideo)
	huma.Register(api, adminOp("reset-class-channel-field", http.MethodDelete, channelPath+"/custom/{field}",
		"Clear a custom value so the YouTube value shows again"), h.ResetField)
}

func registerCourseRoutes(api huma.API, h *CourseAdminHandler) {
	huma.Register(api, adminOp("list-admin-class-courses", http.MethodGet, coursesPath,
		"List courses, hidden ones included"), h.ListCourses)
	huma.Register(api, adminOp("get-admin-class-course", http.MethodGet, coursePath,
		"Get a course, hidden or not"), h.GetCourse)
	huma.Register(api, adminOp("create-class-course", http.MethodPost, coursesPath,
		"Create a course; it starts hidden"), h.CreateCourse)
	huma.Register(api, adminOp("update-class-course", http.MethodPatch, coursePath,
		"Change some course fields"), h.UpdateCourse)
	huma.Register(api, adminOp("delete-class-course", http.MethodDelete, coursePath,
		"Delete a course and its videos"), h.DeleteCourse)
	huma.Register(api, adminOp("set-class-course-visibility", http.MethodPatch, coursePath+"/visibility",
		"Publish, hide or feature a course"), h.SetVisibility)
	huma.Register(api, adminOp("sync-class-course-playlist", http.MethodPost, coursePath+"/sync-playlist",
		"Replace the course's videos with its YouTube playlist"), h.SyncPlaylist)
}
