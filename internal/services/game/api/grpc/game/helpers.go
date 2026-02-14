package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Campaign proto conversion helpers

func campaignToProto(c campaign.Campaign) *campaignv1.Campaign {
	return &campaignv1.Campaign{
		Id:               c.ID,
		Name:             c.Name,
		Locale:           platformi18n.NormalizeLocale(c.Locale),
		System:           gameSystemToProto(c.System),
		Status:           campaignStatusToProto(c.Status),
		GmMode:           gmModeToProto(c.GmMode),
		Intent:           campaignIntentToProto(c.Intent),
		AccessPolicy:     campaignAccessPolicyToProto(c.AccessPolicy),
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

func campaignIntentFromProto(intent campaignv1.CampaignIntent) campaign.CampaignIntent {
	switch intent {
	case campaignv1.CampaignIntent_STANDARD:
		return campaign.CampaignIntentStandard
	case campaignv1.CampaignIntent_STARTER:
		return campaign.CampaignIntentStarter
	case campaignv1.CampaignIntent_SANDBOX:
		return campaign.CampaignIntentSandbox
	default:
		return campaign.CampaignIntentUnspecified
	}
}

func campaignIntentToProto(intent campaign.CampaignIntent) campaignv1.CampaignIntent {
	switch intent {
	case campaign.CampaignIntentStandard:
		return campaignv1.CampaignIntent_STANDARD
	case campaign.CampaignIntentStarter:
		return campaignv1.CampaignIntent_STARTER
	case campaign.CampaignIntentSandbox:
		return campaignv1.CampaignIntent_SANDBOX
	default:
		return campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED
	}
}

func campaignAccessPolicyFromProto(policy campaignv1.CampaignAccessPolicy) campaign.CampaignAccessPolicy {
	switch policy {
	case campaignv1.CampaignAccessPolicy_PRIVATE:
		return campaign.CampaignAccessPolicyPrivate
	case campaignv1.CampaignAccessPolicy_RESTRICTED:
		return campaign.CampaignAccessPolicyRestricted
	case campaignv1.CampaignAccessPolicy_PUBLIC:
		return campaign.CampaignAccessPolicyPublic
	default:
		return campaign.CampaignAccessPolicyUnspecified
	}
}

func campaignAccessPolicyToProto(policy campaign.CampaignAccessPolicy) campaignv1.CampaignAccessPolicy {
	switch policy {
	case campaign.CampaignAccessPolicyPrivate:
		return campaignv1.CampaignAccessPolicy_PRIVATE
	case campaign.CampaignAccessPolicyRestricted:
		return campaignv1.CampaignAccessPolicy_RESTRICTED
	case campaign.CampaignAccessPolicyPublic:
		return campaignv1.CampaignAccessPolicy_PUBLIC
	default:
		return campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED
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

func inviteStatusFromProto(status campaignv1.InviteStatus) invite.Status {
	switch status {
	case campaignv1.InviteStatus_PENDING:
		return invite.StatusPending
	case campaignv1.InviteStatus_CLAIMED:
		return invite.StatusClaimed
	case campaignv1.InviteStatus_REVOKED:
		return invite.StatusRevoked
	default:
		return invite.StatusUnspecified
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

func sessionGateToProto(gate storage.SessionGate) (*campaignv1.SessionGate, error) {
	metadata, err := structFromJSON(gate.MetadataJSON)
	if err != nil {
		return nil, err
	}
	resolution, err := structFromJSON(gate.ResolutionJSON)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SessionGate{
		Id:                  gate.GateID,
		CampaignId:          gate.CampaignID,
		SessionId:           gate.SessionID,
		Type:                gate.GateType,
		Status:              sessionGateStatusToProto(gate.Status),
		Reason:              gate.Reason,
		CreatedAt:           timestamppb.New(gate.CreatedAt),
		CreatedByActorType:  gate.CreatedByActorType,
		CreatedByActorId:    gate.CreatedByActorID,
		ResolvedAt:          timestampOrNil(gate.ResolvedAt),
		ResolvedByActorType: gate.ResolvedByActorType,
		ResolvedByActorId:   gate.ResolvedByActorID,
		Metadata:            metadata,
		Resolution:          resolution,
	}, nil
}

func sessionGateStatusToProto(status string) campaignv1.SessionGateStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(session.GateStatusOpen):
		return campaignv1.SessionGateStatus_SESSION_GATE_OPEN
	case string(session.GateStatusResolved):
		return campaignv1.SessionGateStatus_SESSION_GATE_RESOLVED
	case string(session.GateStatusAbandoned):
		return campaignv1.SessionGateStatus_SESSION_GATE_ABANDONED
	default:
		return campaignv1.SessionGateStatus_SESSION_GATE_STATUS_UNSPECIFIED
	}
}

func sessionSpotlightToProto(spotlight storage.SessionSpotlight) *campaignv1.SessionSpotlight {
	return &campaignv1.SessionSpotlight{
		CampaignId:         spotlight.CampaignID,
		SessionId:          spotlight.SessionID,
		Type:               sessionSpotlightTypeToProto(spotlight.SpotlightType),
		CharacterId:        spotlight.CharacterID,
		UpdatedAt:          timestamppb.New(spotlight.UpdatedAt),
		UpdatedByActorType: spotlight.UpdatedByActorType,
		UpdatedByActorId:   spotlight.UpdatedByActorID,
	}
}

func sessionSpotlightTypeToProto(value string) campaignv1.SessionSpotlightType {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case string(session.SpotlightTypeGM):
		return campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM
	case string(session.SpotlightTypeCharacter):
		return campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER
	default:
		return campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED
	}
}

func sessionSpotlightTypeFromProto(value campaignv1.SessionSpotlightType) (session.SpotlightType, error) {
	switch value {
	case campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM:
		return session.SpotlightTypeGM, nil
	case campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER:
		return session.SpotlightTypeCharacter, nil
	default:
		return "", fmt.Errorf("spotlight type is required")
	}
}

// Timestamp helpers

func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(value.UTC())
}

func structToMap(input *structpb.Struct) map[string]any {
	if input == nil {
		return nil
	}
	return input.AsMap()
}

func structFromJSON(data []byte) (*structpb.Struct, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return structpb.NewStruct(payload)
}

func validateStructPayload(values map[string]any) error {
	for key := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("payload keys must be non-empty")
		}
	}
	return nil
}
