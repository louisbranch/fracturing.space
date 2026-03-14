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
func campaignWorkspaceMenu(workspace campaignapp.CampaignWorkspace, currentPath string, sessions []campaignapp.CampaignSession, loc webtemplates.Localizer) *webtemplates.AppSideMenu {
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
			{
				Label:       webtemplates.T(loc, "game.sessions.title"),
				URL:         sessionsURL,
				MatchPrefix: sessionsURL,
				Badge:       strconv.Itoa(campaignSessionMenuCount(sessions)),
				IconID:      commonv1.IconId_ICON_ID_SESSION,
				SubItems:    sessionSubItems,
			},
			{
				Label:       webtemplates.T(loc, "game.campaign_invites.title"),
				URL:         invitesURL,
				MatchPrefix: invitesURL,
				IconID:      commonv1.IconId_ICON_ID_INVITES,
			},
		},
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

// campaignSessionMenuItemName returns a session menu label with safe fallback copy.
func campaignSessionMenuItemName(session campaignapp.CampaignSession, loc webtemplates.Localizer) string {
	name := strings.TrimSpace(session.Name)
	sessionID := strings.TrimSpace(session.ID)
	if name != "" && (sessionID == "" || !strings.EqualFold(name, sessionID)) {
		return name
	}
	return webtemplates.T(loc, "game.sessions.menu.unnamed")
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
func mapCampaignListItems(items []campaignapp.CampaignSummary, now time.Time, loc webtemplates.Localizer) []webtemplates.CampaignListItem {
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

// mapParticipantsView converts domain participants to template view items.
func mapParticipantsView(items []campaignapp.CampaignParticipant) []webtemplates.CampaignParticipantView {
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
			CanEdit:        p.CanEdit,
			EditReasonCode: p.EditReasonCode,
		})
	}
	return result
}

// mapParticipantEditorView converts domain editor state to template view state.
func mapParticipantEditorView(editor campaignapp.CampaignParticipantEditor) webtemplates.CampaignParticipantEditorView {
	accessOptions := make([]webtemplates.CampaignParticipantAccessOptionView, 0, len(editor.AccessOptions))
	for _, option := range editor.AccessOptions {
		accessOptions = append(accessOptions, webtemplates.CampaignParticipantAccessOptionView{
			Value:   option.Value,
			Allowed: option.Allowed,
		})
	}
	return webtemplates.CampaignParticipantEditorView{
		ID:             editor.Participant.ID,
		Name:           editor.Participant.Name,
		Role:           editor.Participant.Role,
		Controller:     editor.Participant.Controller,
		Pronouns:       editor.Participant.Pronouns,
		CampaignAccess: editor.Participant.CampaignAccess,
		AllowGMRole:    editor.AllowGMRole,
		RoleReadOnly:   editor.RoleReadOnly,
		AccessOptions:  accessOptions,
		AccessReadOnly: editor.AccessReadOnly,
	}
}

// mapParticipantCreatorView converts domain creator state to template view state.
func mapParticipantCreatorView(creator campaignapp.CampaignParticipantCreator) webtemplates.CampaignParticipantCreatorView {
	accessOptions := make([]webtemplates.CampaignParticipantAccessOptionView, 0, len(creator.AccessOptions))
	for _, option := range creator.AccessOptions {
		accessOptions = append(accessOptions, webtemplates.CampaignParticipantAccessOptionView{
			Value:   option.Value,
			Allowed: option.Allowed,
		})
	}
	return webtemplates.CampaignParticipantCreatorView{
		Name:           creator.Name,
		Role:           creator.Role,
		CampaignAccess: creator.CampaignAccess,
		AllowGMRole:    creator.AllowGMRole,
		AccessOptions:  accessOptions,
	}
}

// mapAIBindingEditorView converts domain AI-binding editor state to template view state.
func mapAIBindingEditorView(editor campaignapp.CampaignAIBindingEditor) webtemplates.CampaignAIBindingEditorView {
	options := make([]webtemplates.CampaignAIAgentOptionView, 0, len(editor.Options))
	for _, option := range editor.Options {
		options = append(options, webtemplates.CampaignAIAgentOptionView{
			ID:       option.ID,
			Name:     option.Label,
			Enabled:  option.Enabled,
			Selected: option.Selected,
		})
	}
	return webtemplates.CampaignAIBindingEditorView{
		Visible:     editor.Visible,
		Enabled:     editor.Enabled,
		Unavailable: editor.Unavailable,
		CurrentID:   editor.CurrentID,
		Options:     options,
	}
}

// mapCharactersView converts domain characters to template view items.
func mapCharactersView(items []campaignapp.CampaignCharacter) []webtemplates.CampaignCharacterView {
	result := make([]webtemplates.CampaignCharacterView, 0, len(items))
	for _, c := range items {
		result = append(result, webtemplates.CampaignCharacterView{
			ID:                      c.ID,
			Name:                    c.Name,
			Kind:                    c.Kind,
			Controller:              c.Controller,
			ControllerParticipantID: c.ControllerParticipantID,
			Pronouns:                c.Pronouns,
			Aliases:                 append([]string(nil), c.Aliases...),
			AvatarURL:               c.AvatarURL,
			CanEdit:                 c.CanEdit,
			EditReasonCode:          c.EditReasonCode,
		})
	}
	return result
}

// mapCharacterEditorView converts domain character editor state to template view state.
func mapCharacterEditorView(editor campaignapp.CampaignCharacterEditor) webtemplates.CampaignCharacterEditorView {
	return webtemplates.CampaignCharacterEditorView{
		ID:       editor.Character.ID,
		Name:     editor.Character.Name,
		Pronouns: editor.Character.Pronouns,
		Kind:     editor.Character.Kind,
	}
}

// mapCharacterControlView converts domain control state to template view state.
func mapCharacterControlView(control campaignapp.CampaignCharacterControl) webtemplates.CampaignCharacterControlView {
	options := make([]webtemplates.CampaignCharacterControlOptionView, 0, len(control.Options))
	for _, option := range control.Options {
		options = append(options, webtemplates.CampaignCharacterControlOptionView{
			ParticipantID: option.ParticipantID,
			Label:         option.Label,
			Selected:      option.Selected,
		})
	}
	return webtemplates.CampaignCharacterControlView{
		CurrentParticipantName: control.CurrentParticipantName,
		CanSelfClaim:           control.CanSelfClaim,
		CanSelfRelease:         control.CanSelfRelease,
		CanManageControl:       control.CanManageControl,
		Options:                options,
	}
}

// mapSessionsView converts domain sessions to template view items.
func mapSessionsView(items []campaignapp.CampaignSession) []webtemplates.CampaignSessionView {
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

// mapSessionReadinessView converts domain session readiness to template view.
func mapSessionReadinessView(readiness campaignapp.CampaignSessionReadiness) webtemplates.CampaignSessionReadinessView {
	result := webtemplates.CampaignSessionReadinessView{
		Ready:    readiness.Ready,
		Blockers: make([]webtemplates.CampaignSessionReadinessBlockerView, 0, len(readiness.Blockers)),
	}
	for _, blocker := range readiness.Blockers {
		result.Blockers = append(result.Blockers, webtemplates.CampaignSessionReadinessBlockerView{
			Code:    blocker.Code,
			Message: blocker.Message,
		})
	}
	return result
}

// mapInvitesView converts domain invites to template view items.
func mapInvitesView(items []campaignapp.CampaignInvite) []webtemplates.CampaignInviteView {
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

// mapInviteSeatOptions converts eligible invite targets to template select options.
func mapInviteSeatOptions(participants []campaignapp.CampaignParticipant, invites []campaignapp.CampaignInvite) []webtemplates.CampaignInviteSeatOptionView {
	pendingByParticipantID := make(map[string]struct{}, len(invites))
	for _, invite := range invites {
		participantID := strings.TrimSpace(invite.ParticipantID)
		if participantID == "" || !campaignInviteIsPending(invite.Status) {
			continue
		}
		pendingByParticipantID[participantID] = struct{}{}
	}

	result := make([]webtemplates.CampaignInviteSeatOptionView, 0, len(participants))
	for _, participant := range participants {
		participantID := strings.TrimSpace(participant.ID)
		if participantID == "" {
			continue
		}
		if campaignInviteSeatController(participant.Controller) != "human" {
			continue
		}
		if strings.TrimSpace(participant.UserID) != "" {
			continue
		}
		if _, exists := pendingByParticipantID[participantID]; exists {
			continue
		}

		label := strings.TrimSpace(participant.Name)
		if label == "" {
			label = participantID
		}
		result = append(result, webtemplates.CampaignInviteSeatOptionView{
			ParticipantID: participantID,
			Label:         label,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		leftLabel := strings.ToLower(strings.TrimSpace(result[i].Label))
		rightLabel := strings.ToLower(strings.TrimSpace(result[j].Label))
		if leftLabel == rightLabel {
			return strings.TrimSpace(result[i].ParticipantID) < strings.TrimSpace(result[j].ParticipantID)
		}
		return leftLabel < rightLabel
	})

	return result
}

// campaignInviteIsPending normalizes invite status checks for selector eligibility.
func campaignInviteIsPending(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "invite_status_pending":
		return true
	default:
		return false
	}
}

// campaignInviteSeatController canonicalizes controller labels for invite-seat filtering.
func campaignInviteSeatController(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human", "controller_human":
		return "human"
	case "ai", "controller_ai":
		return "ai"
	default:
		return ""
	}
}
