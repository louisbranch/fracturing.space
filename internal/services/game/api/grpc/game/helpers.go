package game

import (
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Campaign proto conversion helpers

func campaignToProto(c campaign.Campaign) *campaignv1.Campaign {
	return &campaignv1.Campaign{
		Id:               c.ID,
		Name:             c.Name,
		System:           gameSystemToProto(c.System),
		Status:           campaignStatusToProto(c.Status),
		GmMode:           gmModeToProto(c.GmMode),
		ParticipantCount: int32(c.ParticipantCount),
		CharacterCount:   int32(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CreatedAt:        timestamppb.New(c.CreatedAt),
		UpdatedAt:        timestamppb.New(c.UpdatedAt),
		CompletedAt:      timestampOrNil(c.CompletedAt),
		ArchivedAt:       timestampOrNil(c.ArchivedAt),
	}
}

func campaignStatusToProto(status campaign.CampaignStatus) campaignv1.CampaignStatus {
	switch status {
	case campaign.CampaignStatusDraft:
		return campaignv1.CampaignStatus_DRAFT
	case campaign.CampaignStatusActive:
		return campaignv1.CampaignStatus_ACTIVE
	case campaign.CampaignStatusCompleted:
		return campaignv1.CampaignStatus_COMPLETED
	case campaign.CampaignStatusArchived:
		return campaignv1.CampaignStatus_ARCHIVED
	default:
		return campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED
	}
}

func gmModeFromProto(mode campaignv1.GmMode) campaign.GmMode {
	switch mode {
	case campaignv1.GmMode_HUMAN:
		return campaign.GmModeHuman
	case campaignv1.GmMode_AI:
		return campaign.GmModeAI
	case campaignv1.GmMode_HYBRID:
		return campaign.GmModeHybrid
	default:
		return campaign.GmModeUnspecified
	}
}

func gmModeToProto(mode campaign.GmMode) campaignv1.GmMode {
	switch mode {
	case campaign.GmModeHuman:
		return campaignv1.GmMode_HUMAN
	case campaign.GmModeAI:
		return campaignv1.GmMode_AI
	case campaign.GmModeHybrid:
		return campaignv1.GmMode_HYBRID
	default:
		return campaignv1.GmMode_GM_MODE_UNSPECIFIED
	}
}

func gameSystemToProto(system commonv1.GameSystem) commonv1.GameSystem {
	return system
}

func gameSystemFromProto(system commonv1.GameSystem) commonv1.GameSystem {
	return system
}

// Participant proto conversion helpers

func participantToProto(p participant.Participant) *campaignv1.Participant {
	return &campaignv1.Participant{
		Id:             p.ID,
		CampaignId:     p.CampaignID,
		UserId:         p.UserID,
		DisplayName:    p.DisplayName,
		Role:           participantRoleToProto(p.Role),
		CampaignAccess: campaignAccessToProto(p.CampaignAccess),
		Controller:     controllerToProto(p.Controller),
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
	}
}

func participantRoleFromProto(role campaignv1.ParticipantRole) participant.ParticipantRole {
	switch role {
	case campaignv1.ParticipantRole_GM:
		return participant.ParticipantRoleGM
	case campaignv1.ParticipantRole_PLAYER:
		return participant.ParticipantRolePlayer
	default:
		return participant.ParticipantRoleUnspecified
	}
}

func participantRoleToProto(role participant.ParticipantRole) campaignv1.ParticipantRole {
	switch role {
	case participant.ParticipantRoleGM:
		return campaignv1.ParticipantRole_GM
	case participant.ParticipantRolePlayer:
		return campaignv1.ParticipantRole_PLAYER
	default:
		return campaignv1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

func controllerFromProto(controller campaignv1.Controller) participant.Controller {
	switch controller {
	case campaignv1.Controller_CONTROLLER_HUMAN:
		return participant.ControllerHuman
	case campaignv1.Controller_CONTROLLER_AI:
		return participant.ControllerAI
	default:
		return participant.ControllerUnspecified
	}
}

func controllerToProto(controller participant.Controller) campaignv1.Controller {
	switch controller {
	case participant.ControllerHuman:
		return campaignv1.Controller_CONTROLLER_HUMAN
	case participant.ControllerAI:
		return campaignv1.Controller_CONTROLLER_AI
	default:
		return campaignv1.Controller_CONTROLLER_UNSPECIFIED
	}
}

func campaignAccessFromProto(access campaignv1.CampaignAccess) participant.CampaignAccess {
	switch access {
	case campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return participant.CampaignAccessMember
	case campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return participant.CampaignAccessManager
	case campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return participant.CampaignAccessOwner
	default:
		return participant.CampaignAccessUnspecified
	}
}

func campaignAccessToProto(access participant.CampaignAccess) campaignv1.CampaignAccess {
	switch access {
	case participant.CampaignAccessMember:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER
	case participant.CampaignAccessManager:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER
	case participant.CampaignAccessOwner:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
	default:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED
	}
}

// Invite proto conversion helpers

func inviteToProto(inv invite.Invite) *campaignv1.Invite {
	return &campaignv1.Invite{
		Id:                     inv.ID,
		CampaignId:             inv.CampaignID,
		ParticipantId:          inv.ParticipantID,
		RecipientUserId:        inv.RecipientUserID,
		Status:                 inviteStatusToProto(inv.Status),
		CreatedByParticipantId: inv.CreatedByParticipantID,
		CreatedAt:              timestamppb.New(inv.CreatedAt),
		UpdatedAt:              timestamppb.New(inv.UpdatedAt),
	}
}

func inviteStatusToProto(status invite.Status) campaignv1.InviteStatus {
	switch status {
	case invite.StatusPending:
		return campaignv1.InviteStatus_PENDING
	case invite.StatusClaimed:
		return campaignv1.InviteStatus_CLAIMED
	case invite.StatusRevoked:
		return campaignv1.InviteStatus_REVOKED
	default:
		return campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED
	}
}

// Character proto conversion helpers

func characterToProto(ch character.Character) *campaignv1.Character {
	pb := &campaignv1.Character{
		Id:         ch.ID,
		CampaignId: ch.CampaignID,
		Name:       ch.Name,
		Kind:       characterKindToProto(ch.Kind),
		Notes:      ch.Notes,
		CreatedAt:  timestamppb.New(ch.CreatedAt),
		UpdatedAt:  timestamppb.New(ch.UpdatedAt),
	}
	if strings.TrimSpace(ch.ParticipantID) != "" {
		pb.ParticipantId = wrapperspb.String(ch.ParticipantID)
	}
	return pb
}

func characterKindFromProto(kind campaignv1.CharacterKind) character.CharacterKind {
	switch kind {
	case campaignv1.CharacterKind_PC:
		return character.CharacterKindPC
	case campaignv1.CharacterKind_NPC:
		return character.CharacterKindNPC
	default:
		return character.CharacterKindUnspecified
	}
}

func characterKindToProto(kind character.CharacterKind) campaignv1.CharacterKind {
	switch kind {
	case character.CharacterKindPC:
		return campaignv1.CharacterKind_PC
	case character.CharacterKindNPC:
		return campaignv1.CharacterKind_NPC
	default:
		return campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

// Session proto conversion helpers

func sessionToProto(sess session.Session) *campaignv1.Session {
	pb := &campaignv1.Session{
		Id:         sess.ID,
		CampaignId: sess.CampaignID,
		Name:       sess.Name,
		Status:     sessionStatusToProto(sess.Status),
		StartedAt:  timestamppb.New(sess.StartedAt),
		UpdatedAt:  timestamppb.New(sess.UpdatedAt),
	}
	if sess.EndedAt != nil {
		pb.EndedAt = timestamppb.New(*sess.EndedAt)
	}
	return pb
}

func sessionStatusToProto(status session.SessionStatus) campaignv1.SessionStatus {
	switch status {
	case session.SessionStatusActive:
		return campaignv1.SessionStatus_SESSION_ACTIVE
	case session.SessionStatusEnded:
		return campaignv1.SessionStatus_SESSION_ENDED
	default:
		return campaignv1.SessionStatus_SESSION_STATUS_UNSPECIFIED
	}
}

// Timestamp helpers

func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(value.UTC())
}
