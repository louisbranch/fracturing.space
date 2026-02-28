package campaigns

import (
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignWorkspaceMenu builds the side navigation menu for a campaign workspace page.
func campaignWorkspaceMenu(workspace CampaignWorkspace, currentPath string, loc webtemplates.Localizer) *webtemplates.AppSideMenu {
	campaignID := strings.TrimSpace(workspace.ID)
	if campaignID == "" {
		return nil
	}
	participantCount := strings.TrimSpace(workspace.ParticipantCount)
	if participantCount == "" {
		participantCount = "0"
	}
	characterCount := strings.TrimSpace(workspace.CharacterCount)
	if characterCount == "" {
		characterCount = "0"
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
				Badge:       participantCount,
				IconID:      commonv1.IconId_ICON_ID_PARTICIPANT,
			},
			{
				Label:       webtemplates.T(loc, "game.characters.title"),
				URL:         charactersURL,
				MatchPrefix: charactersURL,
				Badge:       characterCount,
				IconID:      commonv1.IconId_ICON_ID_CHARACTER,
			},
		},
	}
}

// mapCampaignListItems converts domain summaries to template list items.
func mapCampaignListItems(items []CampaignSummary, now time.Time, loc webtemplates.Localizer) []webtemplates.CampaignListItem {
	result := make([]webtemplates.CampaignListItem, 0, len(items))
	for _, item := range items {
		result = append(result, webtemplates.CampaignListItem{
			ID:               item.ID,
			Name:             item.Name,
			Theme:            item.Theme,
			CoverImageURL:    item.CoverImageURL,
			ParticipantCount: item.ParticipantCount,
			CharacterCount:   item.CharacterCount,
			UpdatedAt:        campaignListItemUpdatedAt(item.UpdatedAtUnixNano, now, loc),
		})
	}
	return result
}

func campaignListItemUpdatedAt(updatedAtUnixNano int64, now time.Time, loc webtemplates.Localizer) string {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	updatedAt := time.Unix(0, updatedAtUnixNano).UTC()
	if updatedAtUnixNano <= 0 {
		return webtemplates.T(loc, "game.campaigns.updated_at", webtemplates.T(loc, "game.notifications.time.just_now"))
	}
	if updatedAt.IsZero() {
		return webtemplates.T(loc, "game.campaigns.updated_at", webtemplates.T(loc, "game.notifications.time.just_now"))
	}

	delta := now.Sub(updatedAt)
	if delta < 0 {
		delta = 0
	}

	var updatedLabel string
	switch {
	case delta < time.Minute:
		updatedLabel = webtemplates.T(loc, "game.notifications.time.just_now")
	case delta < time.Hour:
		minutes := int(delta / time.Minute)
		if minutes <= 1 {
			updatedLabel = webtemplates.T(loc, "game.notifications.time.minute_ago")
		} else {
			updatedLabel = webtemplates.T(loc, "game.notifications.time.minutes_ago", minutes)
		}
	case delta < 24*time.Hour:
		hours := int(delta / time.Hour)
		if hours <= 1 {
			updatedLabel = webtemplates.T(loc, "game.notifications.time.hour_ago")
		} else {
			updatedLabel = webtemplates.T(loc, "game.notifications.time.hours_ago", hours)
		}
	default:
		days := int(delta / (24 * time.Hour))
		if days <= 1 {
			updatedLabel = webtemplates.T(loc, "game.notifications.time.day_ago")
		} else {
			updatedLabel = webtemplates.T(loc, "game.notifications.time.days_ago", days)
		}
	}
	return webtemplates.T(loc, "game.campaigns.updated_at", updatedLabel)
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
