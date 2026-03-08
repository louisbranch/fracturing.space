package campaigns

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Mutation route handlers ---

// handleSessionStart starts a campaign session and redirects to sessions detail.
func (h handlers) handleSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_session_start_form", "failed to parse session start form") {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.StartSession(ctx, campaignID, parseStartSessionInput(r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	if h.sync != nil {
		h.sync.SessionStarted(ctx, userID, campaignID)
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignSessions(campaignID))
}

// handleSessionEnd handles this route in the module transport layer.
func (h handlers) handleSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_session_end_form", "failed to parse session end form") {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.EndSession(ctx, campaignID, parseEndSessionInput(r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	if h.sync != nil {
		h.sync.SessionEnded(ctx, userID, campaignID)
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignSessions(campaignID))
}

// handleCharacterCreate handles this route in the module transport layer.
func (h handlers) handleCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_create_form", "failed to parse character create form") {
		return
	}
	input, err := parseCreateCharacterInput(r.Form)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)
	created, err := h.service.CreateCharacter(ctx, campaignID, input)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Redirect to creation page if the campaign has a character creation workflow.
	workspace, err := h.service.CampaignWorkspace(ctx, campaignID)
	if err == nil && h.service.ResolveWorkflow(workspace.System) != nil {
		httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, created.CharacterID))
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, created.CharacterID))
}

// handleCharacterUpdate handles this route in the module transport layer.
func (h handlers) handleCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_update_form", "failed to parse character update form") {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateCharacter(ctx, campaignID, characterID, parseUpdateCharacterInput(r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, characterID))
}

// handleInviteCreate handles this route in the module transport layer.
func (h handlers) handleInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_invite_create_form", "failed to parse invite create form") {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.CreateInvite(ctx, campaignID, parseCreateInviteInput(r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignInvites(campaignID))
}

// handleInviteRevoke handles this route in the module transport layer.
func (h handlers) handleInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_invite_revoke_form", "failed to parse invite revoke form") {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.RevokeInvite(ctx, campaignID, parseRevokeInviteInput(r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignInvites(campaignID))
}

// handleParticipantUpdate handles this route in the module transport layer.
func (h handlers) handleParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_participant_update_form", "failed to parse participant update form") {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateParticipant(ctx, campaignID, parseUpdateParticipantInput(participantID, r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignParticipants(campaignID))
}

// handleCampaignAIBinding handles this route in the module transport layer.
func (h handlers) handleCampaignAIBinding(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_campaign_ai_binding_form", "failed to parse campaign AI binding form") {
		return
	}
	input := parseUpdateCampaignAIBindingInput(r.Form)
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateCampaignAIBinding(ctx, campaignID, input); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignParticipantEdit(campaignID, input.ParticipantID))
}

// handleCampaignUpdate handles this route in the module transport layer.
func (h handlers) handleCampaignUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_campaign_update_form", "failed to parse campaign update form") {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateCampaign(ctx, campaignID, parseUpdateCampaignInput(r.Form)); err != nil {
		h.WriteError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaign(campaignID))
}

// parseAppCharacterKind parses inbound values into package-safe forms.
func parseAppCharacterKind(value string) (CharacterKind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pc", "character_kind_pc":
		return CharacterKindPC, true
	case "npc", "character_kind_npc":
		return CharacterKindNPC, true
	default:
		return CharacterKindUnspecified, false
	}
}
