package sessiontransport

import (
	"fmt"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SessionToProto converts a session projection record into its gRPC read model.
func SessionToProto(sess storage.SessionRecord) *campaignv1.Session {
	pb := &campaignv1.Session{
		Id:         sess.ID,
		CampaignId: sess.CampaignID,
		Name:       sess.Name,
		Status:     SessionStatusToProto(sess.Status),
		StartedAt:  timestamppb.New(sess.StartedAt),
		UpdatedAt:  timestamppb.New(sess.UpdatedAt),
	}
	if sess.EndedAt != nil {
		pb.EndedAt = timestamppb.New(*sess.EndedAt)
	}
	return pb
}

// SessionRecapToProto converts a session recap projection into its protobuf
// read model.
func SessionRecapToProto(recap storage.SessionRecap) *campaignv1.SessionRecap {
	return &campaignv1.SessionRecap{
		CampaignId: recap.CampaignID,
		SessionId:  recap.SessionID,
		Markdown:   recap.Markdown,
		UpdatedAt:  timestamppb.New(recap.UpdatedAt),
	}
}

// SessionStatusToProto converts a domain session status to its protobuf enum.
func SessionStatusToProto(status session.Status) campaignv1.SessionStatus {
	switch status {
	case session.StatusActive:
		return campaignv1.SessionStatus_SESSION_ACTIVE
	case session.StatusEnded:
		return campaignv1.SessionStatus_SESSION_ENDED
	default:
		return campaignv1.SessionStatus_SESSION_STATUS_UNSPECIFIED
	}
}

// ActiveUserSessionToProto converts an active-session listing record for a user.
func ActiveUserSessionToProto(campaign storage.CampaignRecord, sess storage.SessionRecord) *campaignv1.ActiveUserSession {
	return &campaignv1.ActiveUserSession{
		CampaignId:   campaign.ID,
		CampaignName: campaign.Name,
		SessionId:    sess.ID,
		SessionName:  sess.Name,
		StartedAt:    timestamppb.New(sess.StartedAt),
	}
}

// GateToProto converts a session gate projection record into its protobuf read model.
func GateToProto(gate storage.SessionGate) (*campaignv1.SessionGate, error) {
	metadata, err := structFromMap(gate.Metadata)
	if err != nil {
		return nil, err
	}
	resolution, err := structFromMap(gate.Resolution)
	if err != nil {
		return nil, err
	}
	progress, err := structFromValue(gate.Progress)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SessionGate{
		Id:                  gate.GateID,
		CampaignId:          gate.CampaignID,
		SessionId:           gate.SessionID,
		Type:                gate.GateType,
		Status:              GateStatusToProto(gate.Status),
		Reason:              gate.Reason,
		CreatedAt:           timestamppb.New(gate.CreatedAt),
		CreatedByActorType:  gate.CreatedByActorType,
		CreatedByActorId:    gate.CreatedByActorID,
		ResolvedAt:          handler.TimestampOrNil(gate.ResolvedAt),
		ResolvedByActorType: gate.ResolvedByActorType,
		ResolvedByActorId:   gate.ResolvedByActorID,
		Metadata:            metadata,
		Progress:            progress,
		Resolution:          resolution,
	}, nil
}

// GateStatusToProto converts a domain gate status to its protobuf enum.
func GateStatusToProto(status session.GateStatus) campaignv1.SessionGateStatus {
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

// SpotlightToProto converts a session spotlight projection into its protobuf read model.
func SpotlightToProto(spotlight storage.SessionSpotlight) *campaignv1.SessionSpotlight {
	return &campaignv1.SessionSpotlight{
		CampaignId:         spotlight.CampaignID,
		SessionId:          spotlight.SessionID,
		Type:               SpotlightTypeToProto(spotlight.SpotlightType),
		CharacterId:        spotlight.CharacterID,
		UpdatedAt:          timestamppb.New(spotlight.UpdatedAt),
		UpdatedByActorType: spotlight.UpdatedByActorType,
		UpdatedByActorId:   spotlight.UpdatedByActorID,
	}
}

// SpotlightTypeToProto converts a domain spotlight type to its protobuf enum.
func SpotlightTypeToProto(value session.SpotlightType) campaignv1.SessionSpotlightType {
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

// SpotlightTypeFromProto converts a protobuf spotlight type to the domain value.
func SpotlightTypeFromProto(value campaignv1.SessionSpotlightType) (session.SpotlightType, error) {
	switch value {
	case campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM:
		return session.SpotlightTypeGM, nil
	case campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER:
		return session.SpotlightTypeCharacter, nil
	default:
		return "", fmt.Errorf("spotlight type is required")
	}
}

func structFromMap(payload map[string]any) (*structpb.Struct, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	values, err := session.JSONMapFromValue(payload)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(values)
}

func structFromValue(payload any) (*structpb.Struct, error) {
	if payload == nil {
		return nil, nil
	}
	values, err := session.JSONMapFromValue(payload)
	if err != nil {
		return nil, err
	}
	return structFromMap(values)
}
