package campaigns

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Character creation workflow routes ---

// handleCharacterCreationStep applies the next character creation workflow step.
func (h handlers) handleCharacterCreationStep(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	if !h.requireParsedForm(w, r, "error.web.message.failed_to_parse_character_creation_form", routepath.AppCampaignCharacterCreation(campaignID, characterID)) {
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)

	workspace, err := h.service.CampaignWorkspace(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	workflow := h.resolveWorkflow(workspace.System)
	if workflow == nil {
		h.WriteNotFound(w, r)
		return
	}

	progress, err := h.service.CampaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if progress.Ready {
		h.writeCreationStepError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_already_complete", "character creation workflow is already complete"), campaignID, characterID)
		return
	}

	stepInput, err := workflow.ParseStepInput(r.Form, progress.NextStep)
	if err != nil {
		h.writeCreationStepError(w, r, err, campaignID, characterID)
		return
	}
	if err := h.service.ApplyCharacterCreationStep(ctx, campaignID, characterID, stepInput); err != nil {
		h.writeCreationStepError(w, r, err, campaignID, characterID)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}

// writeCreationStepError writes a step validation error as a flash notice and
// redirects back to the creation page instead of rendering a full error page.
func (h handlers) writeCreationStepError(w http.ResponseWriter, r *http.Request, err error, campaignID, characterID string) {
	key := apperrors.LocalizationKey(err)
	if key == "" {
		key = "error.web.message.character_creation_step_failed"
	}
	flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: key})
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}

// handleCharacterCreationReset handles this route in the module transport layer.
func (h handlers) handleCharacterCreationReset(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.service.ResetCharacterCreationWorkflow(ctx, campaignID, characterID); err != nil {
		h.writeCreationStepError(w, r, err, campaignID, characterID)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}
