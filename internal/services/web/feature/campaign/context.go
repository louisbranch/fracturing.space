package campaign

import (
	"context"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

// ReadCampaignContext executes the shared campaign auth/session read path.
func ReadCampaignContext(
	w http.ResponseWriter,
	r *http.Request,
	unavailableTitle string,
	ensureAuthenticated func(http.ResponseWriter, *http.Request) bool,
	resolveSessionUserID func(context.Context, *http.Request) (string, error),
	renderError func(http.ResponseWriter, *http.Request, int, string, string),
) (context.Context, string, bool) {
	if r == nil {
		return nil, "", false
	}
	if ensureAuthenticated != nil && !ensureAuthenticated(w, r) {
		return nil, "", false
	}

	userID, err := resolveSessionUserID(r.Context(), r)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, strings.TrimSpace(unavailableTitle), "failed to resolve current user")
		return nil, "", false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		renderError(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return nil, "", false
	}

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return grpcauthctx.WithUserID(ctx, userID), userID, true
}

// RequireCampaignActor enforces campaign participant presence before actor actions.
func RequireCampaignActor(
	w http.ResponseWriter,
	r *http.Request,
	campaignID string,
	ensureAuthenticated func(http.ResponseWriter, *http.Request) bool,
	resolveCampaignParticipant func(context.Context, string) (*statev1.Participant, error),
	renderError func(http.ResponseWriter, *http.Request, int, string, string),
) (*statev1.Participant, bool) {
	if r == nil {
		return nil, false
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		renderError(w, r, http.StatusBadRequest, "Campaign unavailable", "campaign id is required")
		return nil, false
	}
	if ensureAuthenticated != nil && !ensureAuthenticated(w, r) {
		return nil, false
	}
	participant, err := resolveCampaignParticipant(r.Context(), campaignID)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Campaign unavailable", "failed to verify campaign access")
		return nil, false
	}
	if participant == nil || strings.TrimSpace(participant.GetId()) == "" {
		renderError(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return nil, false
	}
	return participant, true
}

// CanManageCampaignAccess reports whether a campaign access level can perform manager actions.
func CanManageCampaignAccess(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}
