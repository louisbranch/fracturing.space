package campaigns

import (
	"context"
	"net/http"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// --- Per-sub-page detail handlers ---

func (h handlers) handleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{marker: markerOverview})
}

func (h handlers) handleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerParticipants,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
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

func (h handlers) handleCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacters,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
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

func (h handlers) handleCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID, characterID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerCharacterDetail,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.characters.title"), URL: routepath.AppCampaignCharacters(campaignID)},
				{Label: characterID},
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

func (h handlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerSessions,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{{Label: webtemplates.T(loc, "game.sessions.title")}}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.CampaignSessions(ctx, campaignID)
			if err != nil {
				return err
			}
			view.Sessions = mapSessionsView(items)
			return nil
		},
	})
}

func (h handlers) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerSessionDetail,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
			return []sharedtemplates.BreadcrumbItem{
				{Label: webtemplates.T(loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
				{Label: sessionID},
			}
		},
		loadData: func(ctx context.Context, campaignID string, _ *campaignPageContext, view *webtemplates.CampaignDetailView) error {
			items, err := h.service.CampaignSessions(ctx, campaignID)
			if err != nil {
				return err
			}
			view.SessionID = sessionID
			view.Sessions = mapSessionsView(items)
			return nil
		},
	})
}

func (h handlers) handleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.renderCampaignDetail(w, r, campaignID, campaignDetailSpec{
		marker: markerInvites,
		extra: func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem {
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
