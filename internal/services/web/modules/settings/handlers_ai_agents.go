package settings

import (
	"context"
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleAIAgentsGet handles this route in the module transport layer.
func (h handlers) handleAIAgentsGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	form := webtemplates.SettingsAIAgentsForm{
		CredentialID: parseAIAgentCredentialSelectionInput(r.URL.Query()),
	}
	if form.CredentialID != "" {
		models, err := h.aiAgents.ListAIProviderModels(ctx, userID, form.CredentialID)
		if err != nil {
			statusCode := apperrors.HTTPStatus(err)
			if statusCode == http.StatusBadRequest || statusCode == http.StatusConflict {
				loc, lang := h.PageLocalizer(w, r)
				h.renderAIAgentsPage(w, r, ctx, userID, statusCode, form, webi18n.LocalizeError(loc, err, lang))
				return
			}
			h.WriteError(w, r, err)
			return
		}
		form.ModelOptions = mapAIModelTemplateOptions(models)
	}
	h.renderAIAgentsPage(w, r, ctx, userID, http.StatusOK, form, "")
}

// handleAIAgentsCreate handles this route in the module transport layer.
func (h handlers) handleAIAgentsCreate(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_ai_agent_form", "failed to parse ai agent form"))
		return
	}
	input := parseAIAgentCreateInput(r.PostForm)
	if err := h.aiAgents.CreateAIAgent(ctx, userID, input); err != nil {
		statusCode := apperrors.HTTPStatus(err)
		if statusCode == http.StatusBadRequest || statusCode == http.StatusConflict {
			loc, lang := h.PageLocalizer(w, r)
			form := webtemplates.SettingsAIAgentsForm{
				Label:        input.Label,
				CredentialID: input.CredentialID,
				Model:        input.Model,
				Instructions: input.Instructions,
			}
			if form.CredentialID != "" {
				models, modelErr := h.aiAgents.ListAIProviderModels(ctx, userID, form.CredentialID)
				if modelErr == nil {
					form.ModelOptions = mapAIModelTemplateOptions(models)
				}
			}
			h.renderAIAgentsPage(w, r, ctx, userID, statusCode, form, webi18n.LocalizeError(loc, err, lang))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_agents.notice_created"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIAgents)
}

// handleAIAgentDelete deletes one AI agent from the settings surface.
func (h handlers) handleAIAgentDelete(w http.ResponseWriter, r *http.Request, agentID string) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.aiAgents.DeleteAIAgent(ctx, userID, agentID); err != nil {
		statusCode := apperrors.HTTPStatus(err)
		if statusCode == http.StatusBadRequest || statusCode == http.StatusConflict {
			h.writeFlashNotice(w, r, flashnotice.Notice{Kind: flashnotice.KindError, Key: apperrors.LocalizationKey(err)})
			httpx.WriteRedirect(w, r, routepath.AppSettingsAIAgents)
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_agents.notice_deleted"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIAgents)
}

// renderAIAgentsPage centralizes this web behavior in one helper seam.
func (h handlers) renderAIAgentsPage(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	userID string,
	statusCode int,
	form webtemplates.SettingsAIAgentsForm,
	errorMessage string,
) {
	loc, _ := h.PageLocalizer(w, r)
	credentialOptions, err := h.loadAIAgentCredentialOptions(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	agentRows, err := h.loadAIAgentRows(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	form.ErrorMessage = errorMessage
	form.CredentialOptions = credentialOptions
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsAIAgents,
		webtemplates.T(loc, "web.settings.page_ai_agents_title"),
		webtemplates.SettingsAIAgentsFragment(form, agentRows, loc),
	)
}
