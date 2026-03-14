package campaigns

import (
	"context"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// --- Per-sub-page detail handlers ---

// handleOverview renders the default campaign detail overview section.
func (h handlers) handleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerOverview,
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			if err := h.service.RequireManageCampaign(ctx, campaignID); err == nil {
				view.CanEditCampaign = true
			}
			return nil
		},
	})
}

// handleCampaignEdit handles this route in the module transport layer.
func (h handlers) handleCampaignEdit(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCampaignEdit,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
				{Label: webtemplates.T(loc, "game.campaign.action_edit")},
			}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			if err := h.service.RequireManageCampaign(ctx, campaignID); err != nil {
				return err
			}
			view.CanEditCampaign = true
			view.LocaleValue = campaignWorkspaceLocaleFormValue(view.Locale)
			return nil
		},
	})
}

// handleParticipants handles this route in the module transport layer.
func (h handlers) handleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerParticipants,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.participants.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.CampaignParticipants(ctx, campaignID)
			if err != nil {
				return err
			}
			if err := h.service.RequireManageParticipants(ctx, campaignID); err == nil {
				view.CanManageParticipants = true
			}
			view.Participants = mapParticipantsView(items)
			return nil
		},
	})
}

// handleParticipantEdit handles this route in the module transport layer.
func (h handlers) handleParticipantEdit(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerParticipantEdit,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
				{Label: webtemplates.T(loc, "game.participants.action_edit")},
			}
		},
		loadData: func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			editor, err := h.service.CampaignParticipantEditor(ctx, campaignID, participantID)
			if err != nil {
				return err
			}
			view.ParticipantID = editor.Participant.ID
			view.ParticipantEditor = mapParticipantEditorView(editor)
			if strings.TrimSpace(view.ParticipantID) == "" {
				view.ParticipantID = participantID
			}
			if strings.EqualFold(strings.TrimSpace(editor.Participant.Controller), "AI") {
				aiBinding, err := h.service.CampaignAIBindingEditor(ctx, campaignID, page.workspace.AIAgentID)
				if err != nil {
					return err
				}
				view.AIBindingEditor = mapAIBindingEditorView(aiBinding)
			}
			return nil
		},
	})
}

// handleCharacters handles this route in the module transport layer.
func (h handlers) handleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacters,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.characters.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.CampaignCharacters(ctx, campaignID)
			if err != nil {
				return err
			}
			if err := h.service.RequireMutateCharacters(ctx, campaignID); err == nil {
				view.CanCreateCharacter = true
			}
			view.Characters = mapCharactersView(items)
			view.CharacterCreationEnabled = h.resolveWorkflow(page.workspace.System) != nil
			return nil
		},
	})
}

// handleCharacterCreatePage handles this route in the module transport layer.
func (h handlers) handleCharacterCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacterCreate,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
				{Label: webtemplates.T(loc, "game.characters.submit_create")},
			}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			if err := h.service.RequireMutateCharacters(ctx, campaignID); err != nil {
				return err
			}
			view.CanCreateCharacter = true
			view.CharacterEditor = webtemplates.CampaignCharacterEditorView{
				Kind: "PC",
			}
			return nil
		},
	})
}

// handleCharacterEdit handles this route in the module transport layer.
func (h handlers) handleCharacterEdit(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacterEdit,
		extra: func(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
				{Label: campaignCharacterEditBreadcrumbLabel(loc, view), URL: routepath.AppCampaignCharacter(campaignID, characterID)},
				{Label: webtemplates.T(loc, "game.characters.action_edit_page")},
			}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			editor, err := h.service.CampaignCharacterEditor(ctx, campaignID, characterID)
			if err != nil {
				return err
			}
			view.CharacterID = characterID
			view.CharacterEditor = mapCharacterEditorView(editor)
			if strings.TrimSpace(view.CharacterID) == "" {
				view.CharacterID = characterID
			}
			view.Characters = []webtemplates.CampaignCharacterView{{
				ID:                      editor.Character.ID,
				Name:                    editor.Character.Name,
				Kind:                    editor.Character.Kind,
				Controller:              editor.Character.Controller,
				ControllerParticipantID: editor.Character.ControllerParticipantID,
				Pronouns:                editor.Character.Pronouns,
				CanEdit:                 true,
				EditReasonCode:          editor.Character.EditReasonCode,
			}}
			return nil
		},
	})
}

// handleCharacterDetail handles this route in the module transport layer.
func (h handlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	userID := h.RequestUserID(r)
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacterDetail,
		extra: func(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
				{Label: campaignCharacterBreadcrumbLabel(loc, view)},
			}
		},
		loadData: func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			characterItems, err := h.service.CampaignCharacters(ctx, campaignID)
			if err != nil {
				return err
			}
			view.CharacterID = characterID
			view.Characters = mapCharactersView(characterItems)
			control, err := h.service.CampaignCharacterControl(ctx, campaignID, characterID, userID)
			if err != nil {
				return err
			}
			view.CharacterControl = mapCharacterControlView(control)
			workflow := h.resolveWorkflow(page.workspace.System)
			view.CharacterCreationEnabled = workflow != nil
			if view.CharacterCreationEnabled {
				creation, err := h.service.CampaignCharacterCreation(ctx, campaignID, characterID, page.locale, workflow)
				if err != nil {
					return err
				}
				view.CharacterCreation = workflow.CreationView(creation)
			}
			return nil
		},
	})
}

// handleSessions handles this route in the module transport layer.
func (h handlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerSessions,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.sessions.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			view.Sessions = mapSessionsView(page.sessions)
			readiness, err := h.service.CampaignSessionReadiness(ctx, campaignID, page.locale)
			if err != nil {
				return err
			}
			view.SessionReadiness = mapSessionReadinessView(readiness)
			return nil
		},
	})
}

// handleSessionDetail handles this route in the module transport layer.
func (h handlers) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerSessionDetail,
		extra: func(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
				{Label: campaignSessionBreadcrumbLabel(loc, view)},
			}
		},
		loadData: func(_ context.Context, _ string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			view.SessionID = sessionID
			view.Sessions = mapSessionsView(page.sessions)
			return nil
		},
	})
}

// campaignSessionBreadcrumbLabel resolves the selected session breadcrumb label.
func campaignSessionBreadcrumbLabel(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) string {
	selectedSessionID := strings.TrimSpace(view.SessionID)
	if selectedSessionID == "" {
		return webtemplates.T(loc, "game.sessions.title")
	}
	for _, session := range view.Sessions {
		if strings.TrimSpace(session.ID) != selectedSessionID {
			continue
		}
		sessionName := strings.TrimSpace(session.Name)
		if sessionName != "" {
			return sessionName
		}
		break
	}
	return webtemplates.T(loc, "game.sessions.menu.unnamed")
}

// handleInvites handles this route in the module transport layer.
func (h handlers) handleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerInvites,
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.campaign_invites.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.CampaignInvites(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Invites = mapInvitesView(items)
			return nil
		},
	})
}

// campaignCharacterBreadcrumbLabel resolves the selected character breadcrumb label.
func campaignCharacterBreadcrumbLabel(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) string {
	selectedCharacterID := strings.TrimSpace(view.CharacterID)
	if selectedCharacterID == "" {
		return webtemplates.T(loc, "game.character_detail.title")
	}
	for _, character := range view.Characters {
		if strings.TrimSpace(character.ID) != selectedCharacterID {
			continue
		}
		characterName := strings.TrimSpace(character.Name)
		if characterName != "" {
			return characterName
		}
		break
	}
	return webtemplates.T(loc, "game.character_detail.title")
}

// campaignCharacterEditBreadcrumbLabel resolves the selected character label for edit breadcrumbs.
func campaignCharacterEditBreadcrumbLabel(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) string {
	if name := strings.TrimSpace(view.CharacterEditor.Name); name != "" {
		return name
	}
	return campaignCharacterBreadcrumbLabel(loc, view)
}
