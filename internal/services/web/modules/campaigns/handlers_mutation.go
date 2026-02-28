package campaigns

import (
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Mutation route handlers ---

func (h handlers) handleSessionStartRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.startSession(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignSessions(campaignID))
}

func (h handlers) handleSessionEndRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.endSession(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignSessions(campaignID))
}

func (h handlers) handleParticipantUpdateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.updateParticipants(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignParticipants(campaignID))
}

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
	created, err := h.service.createCharacter(ctx, campaignID, CreateCharacterInput{
		Name: name,
		Kind: kind,
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, created.CharacterID))
}

func (h handlers) handleCharacterUpdateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.updateCharacter(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacters(campaignID))
}

func (h handlers) handleCharacterControlRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.controlCharacter(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacters(campaignID))
}

func (h handlers) handleInviteCreateRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.createInvite(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignInvites(campaignID))
}

func (h handlers) handleInviteRevokeRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.revokeInvite(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignInvites(campaignID))
}

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
