package app

import (
	"context"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

// campaignRequest captures the browser campaign scope after transport-level
// path validation.
type campaignRequest struct {
	CampaignID string
}

// playRequest captures the authenticated browser request context shared by API,
// interaction, and realtime refresh paths.
type playRequest struct {
	campaignRequest
	UserID string
}

// authContext attaches the resolved play user to downstream gRPC calls without
// forcing each transport path to repeat that mapping detail.
func (r playRequest) authContext(ctx context.Context) context.Context {
	return grpcauthctx.WithUserID(ctx, r.UserID)
}

// requireCampaignRequest keeps campaign path validation explicit at the
// transport edge instead of letting handlers open-code 404 behavior.
func requireCampaignRequest(w http.ResponseWriter, r *http.Request) (campaignRequest, bool) {
	if r == nil {
		if w != nil {
			w.WriteHeader(http.StatusNotFound)
		}
		return campaignRequest{}, false
	}
	campaignID := strings.TrimSpace(r.PathValue("campaignID"))
	if campaignID == "" {
		http.NotFound(w, r)
		return campaignRequest{}, false
	}
	return campaignRequest{CampaignID: campaignID}, true
}

// requirePlayRequest resolves the authenticated browser request context once
// for bootstrap, history, and mutation handlers.
func (s *Server) requirePlayRequest(w http.ResponseWriter, r *http.Request) (playRequest, bool) {
	campaign, ok := requireCampaignRequest(w, r)
	if !ok {
		return playRequest{}, false
	}
	userID, err := s.resolvePlayUserID(r.Context(), r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return playRequest{}, false
	}
	return playRequest{
		campaignRequest: campaign,
		UserID:          userID,
	}, true
}
