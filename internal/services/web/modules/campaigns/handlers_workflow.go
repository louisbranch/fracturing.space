package campaigns

import (
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Character creation workflow routes ---

func (h handlers) handleCharacterCreationStepRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	characterID, ok := h.routeCharacterID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_character_creation_form", "failed to parse character creation form"))
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)

	workspace, err := h.service.campaignWorkspace(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	workflow := h.service.resolveWorkflow(workspace.System)
	if workflow == nil {
		h.WriteNotFound(w, r)
		return
	}

	progress, err := h.service.campaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if progress.Ready {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_already_complete", "character creation workflow is already complete"))
		return
	}

	stepInput, err := workflow.ParseStepInput(r, progress.NextStep)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if err := h.service.applyCharacterCreationStep(ctx, strings.TrimSpace(campaignID), strings.TrimSpace(characterID), stepInput); err != nil {
		h.WriteError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, characterID))
}

func (h handlers) handleCharacterCreationResetRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	characterID, ok := h.routeCharacterID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.resetCharacterCreationWorkflow(ctx, strings.TrimSpace(campaignID), strings.TrimSpace(characterID)); err != nil {
		h.WriteError(w, r, err)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacter(campaignID, characterID))
}
