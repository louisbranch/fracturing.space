package invites

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// invitesView builds the invite management page view from page and request state.
func invitesView(
	page *campaigndetail.PageContext,
	campaignID string,
	participants []campaignapp.CampaignParticipant,
	invites []campaignapp.CampaignInvite,
	r *http.Request,
) campaignrender.InvitesPageView {
	view := campaignrender.InvitesPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	if view.CanManageInvites {
		view.InviteSeatOptions = mapInviteSeatOptions(participants, invites)
	}
	view.Invites = mapInvitesView(invites, r)
	return view
}

// invitesBreadcrumbs returns the root breadcrumb trail for the invites surface.
func invitesBreadcrumbs(page *campaigndetail.PageContext) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.campaign_invites.title")},
	}
}

// mapInviteSeatOptions filters eligible invite seats into stable form options.
func mapInviteSeatOptions(participants []campaignapp.CampaignParticipant, invites []campaignapp.CampaignInvite) []campaignrender.InviteSeatOptionView {
	pendingByParticipantID := make(map[string]struct{}, len(invites))
	for _, invite := range invites {
		participantID := strings.TrimSpace(invite.ParticipantID)
		if participantID == "" || !campaignInviteIsPending(invite.Status) {
			continue
		}
		pendingByParticipantID[participantID] = struct{}{}
	}

	result := make([]campaignrender.InviteSeatOptionView, 0, len(participants))
	for _, participant := range participants {
		participantID := strings.TrimSpace(participant.ID)
		if participantID == "" {
			continue
		}
		if campaignInviteSeatController(participant.Controller) != "human" {
			continue
		}
		if strings.TrimSpace(participant.UserID) != "" {
			continue
		}
		if _, exists := pendingByParticipantID[participantID]; exists {
			continue
		}

		label := strings.TrimSpace(participant.Name)
		if label == "" {
			label = participantID
		}
		result = append(result, campaignrender.InviteSeatOptionView{
			ParticipantID: participantID,
			Label:         label,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		leftLabel := strings.ToLower(strings.TrimSpace(result[i].Label))
		rightLabel := strings.ToLower(strings.TrimSpace(result[j].Label))
		if leftLabel == rightLabel {
			return strings.TrimSpace(result[i].ParticipantID) < strings.TrimSpace(result[j].ParticipantID)
		}
		return leftLabel < rightLabel
	})

	return result
}

// mapInvitesView projects invite rows into render state and public URLs.
func mapInvitesView(items []campaignapp.CampaignInvite, r *http.Request) []campaignrender.InviteView {
	result := make([]campaignrender.InviteView, 0, len(items))
	for _, invite := range items {
		publicURL := ""
		if campaignInviteIsPending(invite.Status) {
			publicURL = absolutePublicInviteURL(r, invite.ID)
		}
		result = append(result, campaignrender.InviteView{
			ID:                invite.ID,
			ParticipantID:     invite.ParticipantID,
			ParticipantName:   invite.ParticipantName,
			RecipientUsername: invite.RecipientUsername,
			HasRecipient:      invite.HasRecipient,
			PublicURL:         publicURL,
			Status:            invite.Status,
		})
	}
	return result
}

// absolutePublicInviteURL builds an absolute invite URL when the request host is known.
func absolutePublicInviteURL(r *http.Request, inviteID string) string {
	path := routepath.PublicInvite(inviteID)
	if r == nil {
		return path
	}
	scheme := "http"
	if requestmeta.IsHTTPS(r) {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Host)
	if host == "" && r.URL != nil {
		host = strings.TrimSpace(r.URL.Host)
	}
	if host == "" {
		return path
	}
	return (&url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}).String()
}

// campaignInviteIsPending reports whether an invite still exposes a public link.
func campaignInviteIsPending(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "invite_status_pending":
		return true
	default:
		return false
	}
}

// campaignInviteSeatController normalizes participant controller values for invite eligibility checks.
func campaignInviteSeatController(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human", "controller_human":
		return "human"
	case "ai", "controller_ai":
		return "ai"
	default:
		return ""
	}
}
