package campaigns

import (
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Mutation route handlers ---

// handleSessionStartRoute starts a campaign session and redirects to sessions detail.
func (h handlers) handleSessionStartRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_session_start_form", "failed to parse session start form"))
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.StartSession(ctx, campaignID, StartSessionInput{
		Name: strings.TrimSpace(r.Form.Get("name")),
	}); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignSessions(campaignID))
}

// handleSessionEndRoute handles this route in the module transport layer.
func (h handlers) handleSessionEndRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_session_end_form", "failed to parse session end form"))
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.EndSession(ctx, campaignID, EndSessionInput{
		SessionID: strings.TrimSpace(r.Form.Get("session_id")),
	}); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignSessions(campaignID))
}

// handleCharacterCreateRoute handles this route in the module transport layer.
func (h handlers) handleCharacterCreateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_character_create_form", "failed to parse character create form"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	kindValue := strings.TrimSpace(r.FormValue("kind"))
	if kindValue == "" {
		kindValue = "pc"
	}
	kind, ok := parseAppCharacterKind(kindValue)
	if !ok {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid"))
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)
	created, err := h.service.CreateCharacter(ctx, campaignID, CreateCharacterInput{
		Name: name,
		Kind: kind,
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, created.CharacterID))
}

// handleInviteCreateRoute handles this route in the module transport layer.
func (h handlers) handleInviteCreateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_invite_create_form", "failed to parse invite create form"))
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.CreateInvite(ctx, campaignID, CreateInviteInput{
		ParticipantID:   strings.TrimSpace(r.Form.Get("participant_id")),
		RecipientUserID: strings.TrimSpace(r.Form.Get("recipient_user_id")),
	}); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignInvites(campaignID))
}

// handleInviteRevokeRoute handles this route in the module transport layer.
func (h handlers) handleInviteRevokeRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_invite_revoke_form", "failed to parse invite revoke form"))
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.RevokeInvite(ctx, campaignID, RevokeInviteInput{
		InviteID: strings.TrimSpace(r.Form.Get("invite_id")),
	}); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignInvites(campaignID))
}

// handleParticipantUpdateRoute handles this route in the module transport layer.
func (h handlers) handleParticipantUpdateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	participantID, ok := h.routeParticipantID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_participant_update_form", "failed to parse participant update form"))
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.UpdateParticipant(ctx, campaignID, UpdateParticipantInput{
		ParticipantID:  participantID,
		Name:           strings.TrimSpace(r.Form.Get("name")),
		Role:           strings.TrimSpace(r.Form.Get("role")),
		Pronouns:       strings.TrimSpace(r.Form.Get("pronouns")),
		CampaignAccess: strings.TrimSpace(r.Form.Get("campaign_access")),
	}); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignParticipants(campaignID))
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
