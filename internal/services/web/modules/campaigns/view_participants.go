package campaigns

import (
	"sort"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// mapParticipantsView converts domain participants to template view items.
func mapParticipantsView(items []campaignapp.CampaignParticipant, viewerUserID string) []campaignrender.ParticipantView {
	viewerUserID = strings.TrimSpace(viewerUserID)
	result := make([]campaignrender.ParticipantView, 0, len(items))
	for _, p := range items {
		result = append(result, campaignrender.ParticipantView{
			ID:             p.ID,
			Name:           p.Name,
			Role:           p.Role,
			CampaignAccess: p.CampaignAccess,
			Controller:     p.Controller,
			Pronouns:       p.Pronouns,
			AvatarURL:      p.AvatarURL,
			IsViewer:       viewerUserID != "" && strings.EqualFold(strings.TrimSpace(p.UserID), viewerUserID),
			CanEdit:        p.CanEdit,
			EditReasonCode: p.EditReasonCode,
		})
	}
	return result
}

// mapParticipantEditorView converts domain editor state to template view state.
func mapParticipantEditorView(editor campaignapp.CampaignParticipantEditor) campaignrender.ParticipantEditorView {
	accessOptions := make([]campaignrender.ParticipantAccessOptionView, 0, len(editor.AccessOptions))
	for _, option := range editor.AccessOptions {
		accessOptions = append(accessOptions, campaignrender.ParticipantAccessOptionView{
			Value:   option.Value,
			Allowed: option.Allowed,
		})
	}
	return campaignrender.ParticipantEditorView{
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
func mapParticipantCreatorView(creator campaignapp.CampaignParticipantCreator) campaignrender.ParticipantCreatorView {
	accessOptions := make([]campaignrender.ParticipantAccessOptionView, 0, len(creator.AccessOptions))
	for _, option := range creator.AccessOptions {
		accessOptions = append(accessOptions, campaignrender.ParticipantAccessOptionView{
			Value:   option.Value,
			Allowed: option.Allowed,
		})
	}
	return campaignrender.ParticipantCreatorView{
		Name:           creator.Name,
		Role:           creator.Role,
		CampaignAccess: creator.CampaignAccess,
		AllowGMRole:    creator.AllowGMRole,
		AccessOptions:  accessOptions,
	}
}

// mapInviteSeatOptions converts current participants and invites into available invite seat options.
func mapInviteSeatOptions(participants []campaignapp.CampaignParticipant, invites []campaignapp.CampaignInvite) []campaignrender.InviteSeatOptionView {
	pendingByParticipantID := make(map[string]struct{}, len(invites))
	for _, invite := range invites {
		participantID := strings.TrimSpace(invite.ParticipantID)
		if participantID == "" || !campaignInviteIsPending(invite.Status) {
			continue
		}
		pendingByParticipantID[participantID] = struct{}{}
	}

	result := make([]campaignrender.InviteSeatOptionView, 0, len(participants))
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
		result = append(result, campaignrender.InviteSeatOptionView{
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
