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
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			editor, err := h.service.CampaignParticipantEditor(ctx, campaignID, participantID)
			if err != nil {
				return err
			}
			view.ParticipantID = participantID
			view.ParticipantEditor = mapParticipantEditorView(editor)
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
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.CampaignCharacters(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Characters = mapCharactersView(items)
			return nil
		},
	})
}

// handleCharacterDetail handles this route in the module transport layer.
func (h handlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
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
		extra: func(loc webtemplates.Localizer, _ webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
				{Label: sessionID},
			}
		},
		loadData: func(_ context.Context, _ string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			view.SessionID = sessionID
			view.Sessions = mapSessionsView(page.sessions)
			return nil
		},
	})
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
