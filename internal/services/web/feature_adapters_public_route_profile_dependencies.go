package web

import (
	"context"
	"net/http"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	publicprofilefeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/publicprofile"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) publicProfileRouteDependencies(w http.ResponseWriter, r *http.Request) publicprofilefeature.PublicProfileHandlers {
	return publicprofilefeature.PublicProfileHandlers{
		LookupProfile: func(ctx context.Context, req *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error) {
			if h.connectionsClient == nil {
				return nil, nil
			}
			return h.connectionsClient.LookupUserProfile(ctx, req)
		},
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
		RenderErrorPage: h.renderErrorPage,
	}
}
