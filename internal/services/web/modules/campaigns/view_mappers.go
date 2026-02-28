package campaigns

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignWorkspaceMenu builds the side navigation menu for a campaign workspace page.
func campaignWorkspaceMenu(campaignID string, currentPath string, loc webtemplates.Localizer) *webtemplates.AppSideMenu {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil
	}
	overviewURL := routepath.AppCampaign(campaignID)
	participantsURL := routepath.AppCampaignParticipants(campaignID)
	charactersURL := routepath.AppCampaignCharacters(campaignID)
	return &webtemplates.AppSideMenu{
		CurrentPath: strings.TrimSpace(currentPath),
		Items: []webtemplates.AppSideMenuItem{
			{
				Label:       webtemplates.T(loc, "game.campaign.menu.overview"),
				URL:         overviewURL,
				MatchPrefix: overviewURL,
				MatchExact:  true,
				IconID:      commonv1.IconId_ICON_ID_CAMPAIGN,
			},
			{
				Label:       webtemplates.T(loc, "game.participants.title"),
				URL:         participantsURL,
				MatchPrefix: participantsURL,
				IconID:      commonv1.IconId_ICON_ID_PARTICIPANT,
			},
			{
				Label:       webtemplates.T(loc, "game.characters.title"),
				URL:         charactersURL,
				MatchPrefix: charactersURL,
				IconID:      commonv1.IconId_ICON_ID_CHARACTER,
			},
		},
	}
}

// mapCampaignListItems converts domain summaries to template list items.
func mapCampaignListItems(items []CampaignSummary) []webtemplates.CampaignListItem {
	result := make([]webtemplates.CampaignListItem, 0, len(items))
	for _, item := range items {
		result = append(result, webtemplates.CampaignListItem{
			ID:               item.ID,
			Name:             item.Name,
			Theme:            item.Theme,
			CoverImageURL:    item.CoverImageURL,
			ParticipantCount: item.ParticipantCount,
			CharacterCount:   item.CharacterCount,
		})
	}
	return result
}

// mapParticipantsView converts domain participants to template view items.
func mapParticipantsView(items []CampaignParticipant) []webtemplates.CampaignParticipantView {
	result := make([]webtemplates.CampaignParticipantView, 0, len(items))
	for _, p := range items {
		result = append(result, webtemplates.CampaignParticipantView{
			ID:             p.ID,
			Name:           p.Name,
			Role:           p.Role,
			CampaignAccess: p.CampaignAccess,
			Controller:     p.Controller,
			Pronouns:       p.Pronouns,
			AvatarURL:      p.AvatarURL,
		})
	}
	return result
}

// mapCharactersView converts domain characters to template view items.
func mapCharactersView(items []CampaignCharacter) []webtemplates.CampaignCharacterView {
	result := make([]webtemplates.CampaignCharacterView, 0, len(items))
	for _, c := range items {
		result = append(result, webtemplates.CampaignCharacterView{
			ID:             c.ID,
			Name:           c.Name,
			Kind:           c.Kind,
			Controller:     c.Controller,
			Pronouns:       c.Pronouns,
			Aliases:        append([]string(nil), c.Aliases...),
			AvatarURL:      c.AvatarURL,
			CanEdit:        c.CanEdit,
			EditReasonCode: c.EditReasonCode,
		})
	}
	return result
}

// mapSessionsView converts domain sessions to template view items.
func mapSessionsView(items []CampaignSession) []webtemplates.CampaignSessionView {
	result := make([]webtemplates.CampaignSessionView, 0, len(items))
	for _, s := range items {
		result = append(result, webtemplates.CampaignSessionView{
			ID:        s.ID,
			Name:      s.Name,
			Status:    s.Status,
			StartedAt: s.StartedAt,
			UpdatedAt: s.UpdatedAt,
			EndedAt:   s.EndedAt,
		})
	}
	return result
}

// mapInvitesView converts domain invites to template view items.
func mapInvitesView(items []CampaignInvite) []webtemplates.CampaignInviteView {
	result := make([]webtemplates.CampaignInviteView, 0, len(items))
	for _, inv := range items {
		result = append(result, webtemplates.CampaignInviteView{
			ID:              inv.ID,
			ParticipantID:   inv.ParticipantID,
			RecipientUserID: inv.RecipientUserID,
			Status:          inv.Status,
		})
	}
	return result
}
