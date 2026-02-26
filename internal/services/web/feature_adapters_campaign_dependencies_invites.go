package web

import (
	"context"
	"net/http"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func buildCampaignFeatureInviteDependencies(h *handler, campaignCache *campaignfeature.CampaignCache, d *campaignfeature.AppCampaignDependencies) {
	d.ListInviteContactOptions = func(ctx context.Context, campaignID string, ownerUserID string, invites []*statev1.Invite) []webtemplates.CampaignInviteContactOption {
		var listContacts func(context.Context, *connectionsv1.ListContactsRequest) (*connectionsv1.ListContactsResponse, error)
		if h != nil && h.connectionsClient != nil {
			listContacts = func(ctx context.Context, req *connectionsv1.ListContactsRequest) (*connectionsv1.ListContactsResponse, error) {
				return h.connectionsClient.ListContacts(ctx, req)
			}
		}
		cachedCampaignParticipants := campaignCache.CachedCampaignParticipants
		var listParticipants func(context.Context, *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error)
		if h != nil && h.participantClient != nil {
			listParticipants = func(ctx context.Context, req *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error) {
				return h.participantClient.ListParticipants(ctx, req)
			}
		}
		setCampaignParticipantsCache := campaignCache.SetCampaignParticipantsCache
		return campaignfeature.ListInviteContactOptions(
			ctx,
			campaignID,
			ownerUserID,
			invites,
			listContacts,
			cachedCampaignParticipants,
			listParticipants,
			setCampaignParticipantsCache,
		)
	}

	d.LookupInviteRecipientVerification = func(ctx context.Context, recipientUserID string) (webtemplates.CampaignInviteVerification, error) {
		var lookupUser func(context.Context, *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error)
		if h != nil && h.connectionsClient != nil {
			lookupUser = func(ctx context.Context, req *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error) {
				return h.connectionsClient.LookupUserProfile(ctx, req)
			}
		}
		return campaignfeature.LookupInviteRecipientVerification(ctx, recipientUserID, lookupUser)
	}

	d.ResolveInviteRecipientUserID = func(ctx context.Context, recipientUserID string) (string, error) {
		var lookupUser func(context.Context, *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error)
		if h != nil && h.connectionsClient != nil {
			lookupUser = func(ctx context.Context, req *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error) {
				return h.connectionsClient.LookupUserProfile(ctx, req)
			}
		}
		return campaignfeature.ResolveInviteRecipientUserID(ctx, recipientUserID, lookupUser)
	}

	d.RenderInviteRecipientLookupError = func(w http.ResponseWriter, r *http.Request, err error) {
		campaignfeature.RenderInviteRecipientLookupError(w, r, err, func(response http.ResponseWriter, req *http.Request, statusCode int, title, message string) {
			h.renderErrorPage(response, req, statusCode, title, message)
		})
	}
}
