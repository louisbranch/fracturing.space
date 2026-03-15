package campaigns

import (
	"sort"
	"strconv"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignWorkspaceMenu builds the side navigation menu for a campaign workspace page.
func campaignWorkspaceMenu(
	workspace campaignapp.CampaignWorkspace,
	currentPath string,
	sessions []campaignapp.CampaignSession,
	canManageInvites bool,
	loc webtemplates.Localizer,
) *webtemplates.AppSideMenu {
	campaignID := strings.TrimSpace(workspace.ID)
	if campaignID == "" {
		return nil
	}
	sessionSubItems := campaignSessionMenuSubItems(campaignID, sessions, loc)
	participantCount := strings.TrimSpace(workspace.ParticipantCount)
	if participantCount == "" {
		participantCount = "0"
	}
	characterCount := strings.TrimSpace(workspace.CharacterCount)
	if characterCount == "" {
		characterCount = "0"
	}
	overviewURL := routepath.AppCampaign(campaignID)
	sessionsURL := routepath.AppCampaignSessions(campaignID)
	participantsURL := routepath.AppCampaignParticipants(campaignID)
	invitesURL := routepath.AppCampaignInvites(campaignID)
	charactersURL := routepath.AppCampaignCharacters(campaignID)
	items := []webtemplates.AppSideMenuItem{
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
		{
			Label:       webtemplates.T(loc, "game.sessions.title"),
			URL:         sessionsURL,
			MatchPrefix: sessionsURL,
			Badge:       strconv.Itoa(campaignSessionMenuCount(sessions)),
			IconID:      commonv1.IconId_ICON_ID_SESSION,
			SubItems:    sessionSubItems,
		},
	}
	if canManageInvites {
		items = append(items, webtemplates.AppSideMenuItem{
			Label:       webtemplates.T(loc, "game.campaign_invites.title"),
			URL:         invitesURL,
			MatchPrefix: invitesURL,
			IconID:      commonv1.IconId_ICON_ID_INVITES,
		})
	}
	return &webtemplates.AppSideMenu{
		CurrentPath: strings.TrimSpace(currentPath),
		Items:       items,
	}
}

const campaignSessionTimestampLayout = "2006-01-02 15:04 UTC"

// campaignSessionMenuSubItems builds campaign session subitems for workspace navigation.
// Only active sessions are shown in the side menu, with a "Join Game" link.
func campaignSessionMenuSubItems(campaignID string, sessions []campaignapp.CampaignSession, loc webtemplates.Localizer) []webtemplates.AppSideMenuSubItem {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || len(sessions) == 0 {
		return []webtemplates.AppSideMenuSubItem{}
	}

	startLabel := webtemplates.T(loc, "game.sessions.menu.start")
	result := make([]webtemplates.AppSideMenuSubItem, 0)
	for _, session := range sessions {
		sessionID := strings.TrimSpace(session.ID)
		if sessionID == "" || !campaignSessionMenuIsActive(session) {
			continue
		}

		startValue := strings.TrimSpace(session.StartedAt)
		if startValue == "" {
			startValue = "-"
		}

		result = append(result, webtemplates.AppSideMenuSubItem{
			Label:         campaignSessionMenuItemName(session, loc),
			URL:           routepath.AppCampaignSession(campaignID, sessionID),
			StartDetail:   startLabel + ": " + startValue,
			ActiveSession: true,
			JoinURL:       routepath.AppCampaignGame(campaignID),
			JoinLabel:     webtemplates.T(loc, "game.sessions.action_join_game"),
		})
	}

	return result
}

// campaignSessionMenuCount returns the total number of sessions represented in the menu badge.
func campaignSessionMenuCount(sessions []campaignapp.CampaignSession) int {
	count := 0
	for _, session := range sessions {
		if strings.TrimSpace(session.ID) == "" {
			continue
		}
		count++
	}
	return count
}

// campaignSessionMenuItemName returns the stored session name for menu labels.
func campaignSessionMenuItemName(session campaignapp.CampaignSession, _ webtemplates.Localizer) string {
	return strings.TrimSpace(session.Name)
}

// campaignSessionMenuStartTime parses session start timestamps used for deterministic ordering.
func campaignSessionMenuStartTime(session campaignapp.CampaignSession) (time.Time, bool) {
	startedAt := strings.TrimSpace(session.StartedAt)
	if startedAt == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(campaignSessionTimestampLayout, startedAt)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

// campaignSessionMenuIsActive marks session rows that should receive active-session styling.
func campaignSessionMenuIsActive(session campaignapp.CampaignSession) bool {
	return strings.EqualFold(strings.TrimSpace(session.Status), "active")
}

// campaignWorkspaceLocaleFormValue maps campaign locale labels/tags to form values.
func campaignWorkspaceLocaleFormValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pt", "pt-br", "portuguese (brazil)":
		return "pt-BR"
	default:
		return "en-US"
	}
}

// mapCampaignListItems converts domain summaries to template list items.
func mapCampaignListItems(items []campaignapp.CampaignSummary, now time.Time, loc webtemplates.Localizer) []CampaignListItem {
	result := make([]CampaignListItem, 0, len(items))
	for _, item := range items {
		result = append(result, CampaignListItem{
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

// campaignListItemUpdatedAt centralizes this web behavior in one helper seam.
func campaignListItemUpdatedAt(updatedAtUnixNano int64, now time.Time, loc webtemplates.Localizer) string {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	updatedAt := time.Unix(0, updatedAtUnixNano).UTC()
	if updatedAtUnixNano <= 0 || updatedAt.IsZero() {
		return webtemplates.T(loc, "game.campaigns.updated_at", webtemplates.T(loc, "game.notifications.time.just_now"))
	}

	delta := now.Sub(updatedAt)
	return webtemplates.T(loc, "game.campaigns.updated_at", webtemplates.RelativeTimeLabel(delta, loc))
}

// sortedActiveSessions returns active sessions ordered by most recent start time first.
func sortedActiveSessions(items []campaignapp.CampaignSession) []campaignapp.CampaignSession {
	active := make([]campaignapp.CampaignSession, 0, len(items))
	for _, session := range items {
		if !campaignSessionMenuIsActive(session) {
			continue
		}
		active = append(active, session)
	}
	sort.SliceStable(active, func(i, j int) bool {
		iTime, iOK := campaignSessionMenuStartTime(active[i])
		jTime, jOK := campaignSessionMenuStartTime(active[j])
		switch {
		case iOK && jOK:
			if !iTime.Equal(jTime) {
				return iTime.After(jTime)
			}
		case iOK:
			return true
		case jOK:
			return false
		}
		return strings.TrimSpace(active[i].ID) < strings.TrimSpace(active[j].ID)
	})
	return active
}
