package access

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

const (
	routeTag        = "Access"
	permissionsPath = "/admin/permissions"
	invitesPath     = "/admin/invites"
	invitePath      = "/admin/invites/{id}"
	publicInvite    = "/invites/{token}"
)

// Admin routes need a signed-in, verified user; each service then checks
// permissions.manage against the database.
func RegisterRoutes(api huma.API, permissions *PermissionHandler, invites *InviteHandler) {
	registerPermissionRoutes(api, permissions)
	registerInviteRoutes(api, invites)
}

func registerPermissionRoutes(api huma.API, h *PermissionHandler) {
	register(api, "get-my-permissions", http.MethodGet, "/admin/me/permissions",
		"List the signed-in user's own permissions", h.GetMine)
	register(api, "list-permissions", http.MethodGet, permissionsPath,
		"List permission grants, optionally for one user", h.List)
	register(api, "grant-permission", http.MethodPost, permissionsPath,
		"Grant a permission to a user by email", h.Grant)
	register(api, "revoke-permission", http.MethodDelete, permissionsPath+"/{id}",
		"Remove one permission grant", h.Revoke)
}

func registerInviteRoutes(api huma.API, h *InviteHandler) {
	register(api, "create-invite", http.MethodPost, invitesPath,
		"Email a staff invite that carries permission grants", h.Create)
	register(api, "list-invites", http.MethodGet, invitesPath,
		"List staff invites", h.List)
	register(api, "resend-invite", http.MethodPost, invitePath+"/resend",
		"Send the invite again with a new link and expiry", h.Resend)
	register(api, "revoke-invite", http.MethodDelete, invitePath,
		"Withdraw an invite that has not been accepted", h.Revoke)

	huma.Register(api, huma.Operation{
		OperationID: "preview-invite",
		Method:      http.MethodGet,
		Path:        publicInvite,
		Summary:     "Show what an invite link grants, before signing in",
		Tags:        []string{routeTag},
		Metadata:    humautil.Public(),
	}, h.Preview)

	register(api, "accept-invite", http.MethodPost, publicInvite+"/accept",
		"Accept an invite sent to your verified email", h.Accept)
}

func register[I, O any](api huma.API, id, method, path, summary string, handler func(context.Context, *I) (*O, error)) {
	huma.Register(api, huma.Operation{
		OperationID: id,
		Method:      method,
		Path:        path,
		Summary:     summary,
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, handler)
}
