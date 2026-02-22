package web

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func renderAppCampaignInvitesPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	renderAppCampaignInvitesPageWithContextAndContacts(w, r, page, campaignID, invites, nil, canManageInvites)
}

func renderAppCampaignInvitesPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	renderAppCampaignInvitesPageWithContextAndContacts(w, r, page, campaignID, invites, nil, canManageInvites)
}

func renderAppCampaignInvitesPageWithContextAndContacts(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManageInvites bool) {
	renderAppCampaignInvitesPageWithContextAndContactsAndVerification(
		w,
		r,
		page,
		campaignID,
		invites,
		contacts,
		canManageInvites,
		webtemplates.CampaignInviteVerification{},
	)
}

func renderAppCampaignInvitesPageWithContextAndContactsAndVerification(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManageInvites bool, verification webtemplates.CampaignInviteVerification) {
	// renderAppCampaignInvitesPage exposes write controls only to managed roles.
	campaignID = strings.TrimSpace(campaignID)
	inviteItems := make([]webtemplates.CampaignInviteItem, 0, len(invites))
	unknownInviteID := "unknown-invite"
	unknownRecipient := "unknown-recipient"
	if page.Loc != nil {
		unknownInviteID = webtemplates.T(page.Loc, "game.campaign_invite.unknown_id")
		unknownRecipient = webtemplates.T(page.Loc, "game.campaign_invite.unknown_recipient")
	}
	for _, invite := range invites {
		if invite == nil {
			continue
		}
		inviteID := strings.TrimSpace(invite.GetId())
		displayInviteID := inviteID
		if displayInviteID == "" {
			displayInviteID = unknownInviteID
		}
		recipient := strings.TrimSpace(invite.GetRecipientUserId())
		if recipient == "" {
			recipient = unknownRecipient
		}
		inviteItems = append(inviteItems, webtemplates.CampaignInviteItem{
			ID:    inviteID,
			Label: displayInviteID + " - " + recipient,
		})
	}
	if err := writePage(w, r, webtemplates.CampaignInvitesPage(page, campaignID, canManageInvites, inviteItems, contacts, verification), composeHTMXTitleForPage(page, "game.campaign_invites.title")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_campaign_invites_page")
	}
}

func (h *handler) renderCampaignInvitesVerificationPage(w http.ResponseWriter, r *http.Request, campaignID string, userID string, canManageInvites bool, verification webtemplates.CampaignInviteVerification) {
	if h == nil || h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}
	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	var invites []*statev1.Invite
	if cachedInvites, ok := h.cachedCampaignInvites(ctx, campaignID, userID); ok {
		invites = cachedInvites
	} else {
		resp, err := h.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
			return
		}
		invites = resp.GetInvites()
		h.setCampaignInvitesCache(ctx, campaignID, userID, invites)
	}
	contactOptions := h.listInviteContactOptions(ctx, campaignID, userID, invites)
	renderReq := r.WithContext(ctx)
	renderAppCampaignInvitesPageWithContextAndContactsAndVerification(
		w,
		renderReq,
		h.pageContextForCampaign(w, renderReq, campaignID),
		campaignID,
		invites,
		contactOptions,
		canManageInvites,
		verification,
	)
}
