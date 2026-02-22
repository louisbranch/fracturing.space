package admin

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

func (h *Handler) handleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.ParticipantsListPage(campaignID, campaignName, loc),
		templates.ParticipantsListFullPage(campaignID, campaignName, pageCtx),
		htmxLocalizedPageTitle(loc, "title.participants", templates.AppName()),
	)
}

// handleParticipantsTable renders the participants table.
func (h *Handler) handleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	participantClient := h.participantClient()
	if participantClient == nil {
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participant_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list participants: %v", err)
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participants_unavailable"), loc)
		return
	}

	participants := response.GetParticipants()
	if len(participants) == 0 {
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.no_participants"), loc)
		return
	}

	rows := buildParticipantRows(participants, loc)
	h.renderParticipantsTable(w, r, rows, "", loc)
}

// handleInvitesList renders the invites list page.
func (h *Handler) handleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.InvitesListPage(campaignID, campaignName, loc),
		templates.InvitesListFullPage(campaignID, campaignName, pageCtx),
		htmxLocalizedPageTitle(loc, "title.invites", templates.AppName()),
	)
}

// handleInvitesTable renders the invites table.
func (h *Handler) handleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	inviteClient := h.inviteClient()
	if inviteClient == nil {
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.invite_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   inviteListPageSize,
	})
	if err != nil {
		log.Printf("list invites: %v", err)
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.invites_unavailable"), loc)
		return
	}

	invites := response.GetInvites()
	if len(invites) == 0 {
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.no_invites"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := h.participantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for invites: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetName()
				}
			}
		}
	}

	recipientNames := map[string]string{}
	if authClient := h.authClient(); authClient != nil {
		for _, inv := range invites {
			if inv == nil {
				continue
			}
			recipientID := strings.TrimSpace(inv.GetRecipientUserId())
			if recipientID == "" {
				continue
			}
			if _, ok := recipientNames[recipientID]; ok {
				continue
			}
			userResp, err := authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientID})
			if err != nil {
				log.Printf("get invite recipient: %v", err)
				recipientNames[recipientID] = ""
				continue
			}
			if user := userResp.GetUser(); user != nil {
				recipientNames[recipientID] = user.GetEmail()
			}
		}
	}

	rows := buildInviteRows(invites, participantNames, recipientNames, loc)
	h.renderInvitesTable(w, r, rows, "", loc)
}

// renderInvitesTable renders the invites table component.
func (h *Handler) renderInvitesTable(w http.ResponseWriter, r *http.Request, rows []templates.InviteRow, message string, loc *message.Printer) {
	templ.Handler(templates.InvitesTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildInviteRows formats invite rows for the table.
func buildInviteRows(invites []*statev1.Invite, participantNames map[string]string, recipientNames map[string]string, loc *message.Printer) []templates.InviteRow {
	rows := make([]templates.InviteRow, 0, len(invites))
	for _, inv := range invites {
		if inv == nil {
			continue
		}

		participantLabel := participantNames[inv.GetParticipantId()]
		if participantLabel == "" {
			participantLabel = loc.Sprintf("label.unknown")
		}

		recipientLabel := loc.Sprintf("label.unassigned")
		recipientID := strings.TrimSpace(inv.GetRecipientUserId())
		if recipientID != "" {
			recipientLabel = recipientNames[recipientID]
			if recipientLabel == "" {
				recipientLabel = recipientID
			}
		}

		statusLabel, statusVariant := formatInviteStatus(inv.GetStatus(), loc)

		rows = append(rows, templates.InviteRow{
			ID:            inv.GetId(),
			CampaignID:    inv.GetCampaignId(),
			Participant:   participantLabel,
			Recipient:     recipientLabel,
			Status:        statusLabel,
			StatusVariant: statusVariant,
			CreatedAt:     formatTimestamp(inv.GetCreatedAt()),
			UpdatedAt:     formatTimestamp(inv.GetUpdatedAt()),
		})
	}
	return rows
}

// renderParticipantsTable renders the participants table component.
func (h *Handler) renderParticipantsTable(w http.ResponseWriter, r *http.Request, rows []templates.ParticipantRow, message string, loc *message.Printer) {
	templ.Handler(templates.ParticipantsTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildParticipantRows formats participant rows for the table.
func buildParticipantRows(participants []*statev1.Participant, loc *message.Printer) []templates.ParticipantRow {
	rows := make([]templates.ParticipantRow, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}

		role, roleVariant := formatParticipantRole(participant.GetRole(), loc)
		access, accessVariant := formatParticipantAccess(participant.GetCampaignAccess(), loc)
		controller, controllerVariant := formatParticipantController(participant.GetController(), loc)

		rows = append(rows, templates.ParticipantRow{
			ID:                participant.GetId(),
			Name:              participant.GetName(),
			Role:              role,
			RoleVariant:       roleVariant,
			Access:            access,
			AccessVariant:     accessVariant,
			Controller:        controller,
			ControllerVariant: controllerVariant,
			CreatedDate:       formatCreatedDate(participant.GetCreatedAt()),
		})
	}
	return rows
}

// formatParticipantRole returns a display label and variant for a participant role.
func formatParticipantRole(role statev1.ParticipantRole, loc *message.Printer) (string, string) {
	switch role {
	case statev1.ParticipantRole_GM:
		return loc.Sprintf("label.gm"), "info"
	case statev1.ParticipantRole_PLAYER:
		return loc.Sprintf("label.player"), "success"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatParticipantController returns a display label and variant for a controller type.
func formatParticipantController(controller statev1.Controller, loc *message.Printer) (string, string) {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return loc.Sprintf("label.human"), "success"
	case statev1.Controller_CONTROLLER_AI:
		return loc.Sprintf("label.ai"), "info"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatParticipantAccess returns a display label and variant for campaign access.
func formatParticipantAccess(access statev1.CampaignAccess, loc *message.Printer) (string, string) {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return loc.Sprintf("label.member"), "secondary"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return loc.Sprintf("label.manager"), "info"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return loc.Sprintf("label.owner"), "warning"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// handleSessionDetail renders the session detail page.
