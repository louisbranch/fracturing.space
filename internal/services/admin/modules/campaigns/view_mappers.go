package campaigns

import (
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func buildCampaignRows(campaigns []*statev1.Campaign, loc *message.Printer) []templates.CampaignRow {
	rows := make([]templates.CampaignRow, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		rows = append(rows, templates.CampaignRow{
			ID:               campaign.GetId(),
			Name:             campaign.GetName(),
			System:           formatGameSystem(campaign.GetSystem(), loc),
			GMMode:           formatGmMode(campaign.GetGmMode(), loc),
			ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			ThemePrompt:      truncateText(campaign.GetThemePrompt(), campaignThemePromptLimit),
			CreatedDate:      formatCreatedDate(campaign.GetCreatedAt()),
		})
	}
	return rows
}

func buildCampaignDetail(campaign *statev1.Campaign, loc *message.Printer) templates.CampaignDetail {
	if campaign == nil {
		return templates.CampaignDetail{}
	}
	return templates.CampaignDetail{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		System:           formatGameSystem(campaign.GetSystem(), loc),
		GMMode:           formatGmMode(campaign.GetGmMode(), loc),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		ThemePrompt:      campaign.GetThemePrompt(),
		CreatedAt:        eventview.FormatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        eventview.FormatTimestamp(campaign.GetUpdatedAt()),
	}
}

func buildCampaignSessionRows(sessions []*statev1.Session, loc *message.Printer) []templates.CampaignSessionRow {
	rows := make([]templates.CampaignSessionRow, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		statusBadge := "secondary"
		if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
			statusBadge = "success"
		}
		row := templates.CampaignSessionRow{
			ID:          session.GetId(),
			CampaignID:  session.GetCampaignId(),
			Name:        session.GetName(),
			Status:      formatSessionStatus(session.GetStatus(), loc),
			StatusBadge: statusBadge,
			StartedAt:   eventview.FormatTimestamp(session.GetStartedAt()),
		}
		if session.GetEndedAt() != nil {
			row.EndedAt = eventview.FormatTimestamp(session.GetEndedAt())
		}
		rows = append(rows, row)
	}
	return rows
}

func buildCharacterRows(characters []*statev1.Character, participantNames map[string]string, loc *message.Printer) []templates.CharacterRow {
	rows := make([]templates.CharacterRow, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}

		controller := formatCharacterController(character, participantNames, loc)

		rows = append(rows, templates.CharacterRow{
			ID:         character.GetId(),
			CampaignID: character.GetCampaignId(),
			Name:       character.GetName(),
			Kind:       formatCharacterKind(character.GetKind(), loc),
			Controller: controller,
		})
	}
	return rows
}

func buildCharacterSheet(campaignID string, campaignName string, character *statev1.Character, recentEvents []templates.EventRow, controller string, loc *message.Printer) templates.CharacterSheetView {
	return templates.CharacterSheetView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Character:    character,
		Controller:   controller,
		CreatedAt:    eventview.FormatTimestamp(character.GetCreatedAt()),
		UpdatedAt:    eventview.FormatTimestamp(character.GetUpdatedAt()),
		RecentEvents: recentEvents,
	}
}

func buildInviteRows(invites []*invitev1.Invite, participantNames map[string]string, recipientNames map[string]string, loc *message.Printer) []templates.InviteRow {
	rows := make([]templates.InviteRow, 0, len(invites))
	for _, inv := range invites {
		if inv == nil {
			continue
		}

		participantLabel := participantNames[inv.GetParticipantId()]
		if participantLabel == "" {
			participantLabel = loc.Sprintf("label.unknown")
		}

		recipientLabel := loc.Sprintf("label.unassigned")
		recipientID := strings.TrimSpace(inv.GetRecipientUserId())
		if recipientID != "" {
			recipientLabel = recipientNames[recipientID]
			if recipientLabel == "" {
				recipientLabel = recipientID
			}
		}

		statusLabel, statusVariant := formatInviteStatus(inv.GetStatus(), loc)

		rows = append(rows, templates.InviteRow{
			ID:            inv.GetId(),
			CampaignID:    inv.GetCampaignId(),
			Participant:   participantLabel,
			Recipient:     recipientLabel,
			Status:        statusLabel,
			StatusVariant: statusVariant,
			CreatedAt:     eventview.FormatTimestamp(inv.GetCreatedAt()),
			UpdatedAt:     eventview.FormatTimestamp(inv.GetUpdatedAt()),
		})
	}
	return rows
}

func buildParticipantRows(participants []*statev1.Participant, loc *message.Printer) []templates.ParticipantRow {
	rows := make([]templates.ParticipantRow, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}

		role, roleVariant := formatParticipantRole(participant.GetRole(), loc)
		access, accessVariant := formatParticipantAccess(participant.GetCampaignAccess(), loc)
		controller, controllerVariant := formatParticipantController(participant.GetController(), loc)

		rows = append(rows, templates.ParticipantRow{
			ID:                participant.GetId(),
			Name:              participant.GetName(),
			Role:              role,
			RoleVariant:       roleVariant,
			Access:            access,
			AccessVariant:     accessVariant,
			Controller:        controller,
			ControllerVariant: controllerVariant,
			CreatedDate:       formatCreatedDate(participant.GetCreatedAt()),
		})
	}
	return rows
}

func buildSessionDetail(campaignID string, campaignName string, session *statev1.Session, eventCount int32, loc *message.Printer) templates.SessionDetail {
	if session == nil {
		return templates.SessionDetail{}
	}

	status := formatSessionStatus(session.GetStatus(), loc)
	statusBadge := "secondary"
	if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
		statusBadge = "success"
	}

	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           session.GetId(),
		Name:         session.GetName(),
		Status:       status,
		StatusBadge:  statusBadge,
		StartedAt:    eventview.FormatTimestamp(session.GetStartedAt()),
		EventCount:   eventCount,
	}

	if session.GetEndedAt() != nil {
		detail.EndedAt = eventview.FormatTimestamp(session.GetEndedAt())
	}

	return detail
}

func formatGmMode(mode statev1.GmMode, loc *message.Printer) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return loc.Sprintf("label.human")
	case statev1.GmMode_AI:
		return loc.Sprintf("label.ai")
	case statev1.GmMode_HYBRID:
		return loc.Sprintf("label.hybrid")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatGameSystem(system commonv1.GameSystem, loc *message.Printer) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return loc.Sprintf("label.daggerheart")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatSessionStatus(status statev1.SessionStatus, loc *message.Printer) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return loc.Sprintf("label.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return loc.Sprintf("label.ended")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatInviteStatus(status invitev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case invitev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case invitev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case invitev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatCreatedDate(createdAt *timestamppb.Timestamp) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.AsTime().Format("2006-01-02")
}

func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func formatParticipantRole(role statev1.ParticipantRole, loc *message.Printer) (string, string) {
	switch role {
	case statev1.ParticipantRole_GM:
		return loc.Sprintf("label.gm"), "info"
	case statev1.ParticipantRole_PLAYER:
		return loc.Sprintf("label.player"), "success"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatParticipantController(controller statev1.Controller, loc *message.Printer) (string, string) {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return loc.Sprintf("label.human"), "success"
	case statev1.Controller_CONTROLLER_AI:
		return loc.Sprintf("label.ai"), "info"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatParticipantAccess(access statev1.CampaignAccess, loc *message.Printer) (string, string) {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return loc.Sprintf("label.member"), "secondary"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return loc.Sprintf("label.manager"), "info"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return loc.Sprintf("label.owner"), "warning"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatCharacterController(character *statev1.Character, participantNames map[string]string, loc *message.Printer) string {
	if character == nil {
		return loc.Sprintf("label.unassigned")
	}
	participantID := ""
	if character.GetOwnerParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetOwnerParticipantId().GetValue())
	}
	if participantID == "" {
		return loc.Sprintf("label.unassigned")
	}
	if name, ok := participantNames[participantID]; ok {
		return name
	}
	return loc.Sprintf("label.unknown")
}

func formatCharacterKind(kind statev1.CharacterKind, loc *message.Printer) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return loc.Sprintf("label.pc")
	case statev1.CharacterKind_NPC:
		return loc.Sprintf("label.npc")
	default:
		return loc.Sprintf("label.unspecified")
	}
}
