package game

import (
	"fmt"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Session proto conversion helpers.
func sessionToProto(sess storage.SessionRecord) *campaignv1.Session {
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

func sessionStatusToProto(status session.Status) campaignv1.SessionStatus {
	switch status {
	case session.StatusActive:
		return campaignv1.SessionStatus_SESSION_ACTIVE
	case session.StatusEnded:
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

func sessionGateStatusToProto(status session.GateStatus) campaignv1.SessionGateStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
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

func sessionSpotlightTypeToProto(value session.SpotlightType) campaignv1.SessionSpotlightType {
	trimmed := strings.ToLower(strings.TrimSpace(string(value)))
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
