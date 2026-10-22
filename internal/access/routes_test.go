package access

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"go.uber.org/zap"
)

// Registration panics on a bad tag or a clashing path, so this catches both
// without a database.
func TestRegisterRoutes_RegistersEveryEndpoint(t *testing.T) {
	_, api := humatest.New(t)
	RegisterRoutes(api, NewPermissionHandler(nil, nil, nil, nil, zap.NewNop()), NewInviteHandler(InviteServices{}, zap.NewNop()))

	paths := api.OpenAPI().Paths
	tests := []struct {
		path, method string
	}{
		{"/admin/me/permissions", http.MethodGet},
		{"/admin/permissions", http.MethodGet},
		{"/admin/permissions", http.MethodPost},
		{"/admin/permissions/{id}", http.MethodDelete},
		{"/admin/invites", http.MethodPost},
		{"/admin/invites", http.MethodGet},
		{"/admin/invites/{id}/resend", http.MethodPost},
		{"/admin/invites/{id}", http.MethodDelete},
		{"/invites/{token}", http.MethodGet},
		{"/invites/{token}/accept", http.MethodPost},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			item := paths[tt.path]
			if item == nil {
				t.Fatalf("path %s not registered", tt.path)
			}
			ops := map[string]bool{
				http.MethodGet: item.Get != nil, http.MethodPost: item.Post != nil, http.MethodDelete: item.Delete != nil,
			}
			if !ops[tt.method] {
				t.Errorf("%s %s not registered", tt.method, tt.path)
			}
		})
	}
}
