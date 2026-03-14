package campaigns

import (
	"net/http"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Mutation route handlers ---

// writeMutationError writes a flash error notice and redirects back to the
// originating page so the user stays in context and can retry.
func (h handlers) writeMutationError(w http.ResponseWriter, r *http.Request, err error, fallbackKey, redirectURL string) {
	notice := flash.Notice{Kind: flash.KindError}
	if key := apperrors.LocalizationKey(err); key != "" {
		notice.Key = key
	} else {
		_, lang := h.PageLocalizer(w, r)
		if message := strings.TrimSpace(apperrors.ResolveRichMessage(err, lang)); message != "" {
			notice.Message = message
		} else {
			notice.Key = fallbackKey
		}
	}
	if notice.Key == "" && strings.TrimSpace(notice.Message) == "" {
		notice.Key = fallbackKey
	}
	flash.Write(w, r, notice)
	httpx.WriteRedirect(w, r, redirectURL)
}

// writeMutationSuccess writes a success flash notice and redirects to the
// target page so the user sees confirmation feedback.
func (h handlers) writeMutationSuccess(w http.ResponseWriter, r *http.Request, key, redirectURL string) {
	flash.Write(w, r, flash.NoticeSuccess(key))
	httpx.WriteRedirect(w, r, redirectURL)
}

// handleSessionStart starts a campaign session and redirects to sessions detail.
func (h handlers) handleSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_session_start_form", routepath.AppCampaignSessions(campaignID)) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.StartSession(ctx, campaignID, parseStartSessionInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_start_session", routepath.AppCampaignSessions(campaignID))
		return
	}
	if h.sync != nil {
		h.sync.SessionStarted(ctx, userID, campaignID)
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_session_started", routepath.AppCampaignSessions(campaignID))
}

// handleSessionEnd handles this route in the module transport layer.
func (h handlers) handleSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_session_end_form", routepath.AppCampaignSessions(campaignID)) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.EndSession(ctx, campaignID, parseEndSessionInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_end_session", routepath.AppCampaignSessions(campaignID))
		return
	}
	if h.sync != nil {
		h.sync.SessionEnded(ctx, userID, campaignID)
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_session_ended", routepath.AppCampaignSessions(campaignID))
}

// handleCharacterCreate handles this route in the module transport layer.
func (h handlers) handleCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_create_form", routepath.AppCampaign(campaignID)) {
		return
	}
	input, err := parseCreateCharacterInput(r.Form)
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_character", routepath.AppCampaign(campaignID))
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)
	created, err := h.service.CreateCharacter(ctx, campaignID, input)
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_character", routepath.AppCampaign(campaignID))
		return
	}

	// Redirect to creation page if the campaign has a character creation workflow.
	workspace, err := h.service.CampaignWorkspace(ctx, campaignID)
	if err == nil && h.resolveWorkflow(workspace.System) != nil {
		h.writeMutationSuccess(w, r, "web.campaigns.notice_character_created", routepath.AppCampaignCharacterCreation(campaignID, created.CharacterID))
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_character_created", routepath.AppCampaignCharacter(campaignID, created.CharacterID))
}

// handleCharacterUpdate handles this route in the module transport layer.
func (h handlers) handleCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_update_form", routepath.AppCampaignCharacter(campaignID, characterID)) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateCharacter(ctx, campaignID, characterID, parseUpdateCharacterInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_update_character", routepath.AppCampaignCharacter(campaignID, characterID))
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_character_updated", routepath.AppCampaignCharacter(campaignID, characterID))
}

// handleCharacterControlSet updates the character controller from the detail page.
func (h handlers) handleCharacterControlSet(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_controller_form", redirectURL) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.SetCharacterController(ctx, campaignID, characterID, parseSetCharacterControllerInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_set_character_controller", redirectURL)
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_character_controller_updated", redirectURL)
}

// handleCharacterControlClaim claims an unassigned character for the current participant.
func (h handlers) handleCharacterControlClaim(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_controller_form", redirectURL) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.ClaimCharacterControl(ctx, campaignID, characterID, userID); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_claim_character_control", redirectURL)
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_character_control_claimed", redirectURL)
}

// handleCharacterControlRelease releases the current participant's control.
func (h handlers) handleCharacterControlRelease(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_controller_form", redirectURL) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.ReleaseCharacterControl(ctx, campaignID, characterID, userID); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_release_character_control", redirectURL)
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_character_control_released", redirectURL)
}

// handleCharacterDelete removes a character from the campaign.
func (h handlers) handleCharacterDelete(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	redirectURL := routepath.AppCampaignCharacter(campaignID, characterID)
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_delete_form", redirectURL) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.DeleteCharacter(ctx, campaignID, characterID); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_delete_character", redirectURL)
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_character_deleted", routepath.AppCampaignCharacters(campaignID))
}

// handleInviteCreate handles this route in the module transport layer.
func (h handlers) handleInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_invite_create_form", routepath.AppCampaignInvites(campaignID)) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.CreateInvite(ctx, campaignID, parseCreateInviteInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_invite", routepath.AppCampaignInvites(campaignID))
		return
	}
	if h.sync != nil {
		h.sync.InviteChanged(ctx, []string{userID}, campaignID)
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_invite_created", routepath.AppCampaignInvites(campaignID))
}

// handleParticipantCreate handles this route in the module transport layer.
func (h handlers) handleParticipantCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	redirectURL := routepath.AppCampaignParticipantCreate(campaignID)
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_participant_create_form", redirectURL) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if _, err := h.service.CreateParticipant(ctx, campaignID, parseCreateParticipantInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_participant", redirectURL)
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_participant_created", routepath.AppCampaignInvites(campaignID))
}

// handleInviteRevoke handles this route in the module transport layer.
func (h handlers) handleInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_invite_revoke_form", routepath.AppCampaignInvites(campaignID)) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.RevokeInvite(ctx, campaignID, parseRevokeInviteInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_revoke_invite", routepath.AppCampaignInvites(campaignID))
		return
	}
	if h.sync != nil {
		h.sync.InviteChanged(ctx, []string{userID}, campaignID)
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_invite_revoked", routepath.AppCampaignInvites(campaignID))
}

// handleParticipantUpdate handles this route in the module transport layer.
func (h handlers) handleParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_participant_update_form", routepath.AppCampaignParticipants(campaignID)) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateParticipant(ctx, campaignID, parseUpdateParticipantInput(participantID, r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_update_participant", routepath.AppCampaignParticipants(campaignID))
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_participant_updated", routepath.AppCampaignParticipants(campaignID))
}

// handleCampaignAIBinding handles this route in the module transport layer.
func (h handlers) handleCampaignAIBinding(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_campaign_ai_binding_form", routepath.AppCampaign(campaignID)) {
		return
	}
	input := parseUpdateCampaignAIBindingInput(r.Form)
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateCampaignAIBinding(ctx, campaignID, input); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_update_ai_binding", routepath.AppCampaignParticipantEdit(campaignID, input.ParticipantID))
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_ai_binding_saved", routepath.AppCampaignParticipantEdit(campaignID, input.ParticipantID))
}

// handleCampaignUpdate handles this route in the module transport layer.
func (h handlers) handleCampaignUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_campaign_update_form", routepath.AppCampaign(campaignID)) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateCampaign(ctx, campaignID, parseUpdateCampaignInput(r.Form)); err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_update_campaign", routepath.AppCampaign(campaignID))
		return
	}
	h.writeMutationSuccess(w, r, "web.campaigns.notice_campaign_updated", routepath.AppCampaign(campaignID))
}

// parseAppCharacterKind parses inbound values into package-safe forms.
func parseAppCharacterKind(value string) (campaignapp.CharacterKind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pc", "character_kind_pc":
		return campaignapp.CharacterKindPC, true
	case "npc", "character_kind_npc":
		return campaignapp.CharacterKindNPC, true
	default:
		return campaignapp.CharacterKindUnspecified, false
	}
}
