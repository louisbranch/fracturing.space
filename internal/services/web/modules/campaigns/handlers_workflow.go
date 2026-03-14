package campaigns

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/forminput"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Character creation workflow routes ---

// handleCharacterCreationStep applies the next character creation workflow step.
func (h handlers) handleCharacterCreationStep(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	if !forminput.ParseOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_character_creation_form", routepath.AppCampaignCharacterCreation(campaignID, characterID)) {
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)

	workspace, err := h.workspace.CampaignWorkspace(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if !h.creationPages.Enabled(workspace.System) {
		h.WriteNotFound(w, r)
		return
	}
	if err := h.creationMutation.ApplyStep(ctx, campaignID, characterID, workspace.System, r.Form); err != nil {
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
	if err := h.creationMutation.Reset(ctx, campaignID, characterID); err != nil {
		h.writeCreationStepError(w, r, err, campaignID, characterID)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaignCharacterCreation(campaignID, characterID))
}
