package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *DaggerheartService) CreateAdversary(ctx context.Context, in *pb.DaggerheartCreateAdversaryRequest) (*pb.DaggerheartCreateAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create adversary request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	kind := strings.TrimSpace(in.GetKind())
	notes := strings.TrimSpace(in.GetNotes())
	var sessionID string
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}

	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		HP:            in.Hp,
		HPMax:         in.HpMax,
		Stress:        in.Stress,
		StressMax:     in.StressMax,
		Evasion:       in.Evasion,
		Major:         in.MajorThreshold,
		Severe:        in.SevereThreshold,
		Armor:         in.Armor,
		RequireFields: false,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversaries")
	}

	if sessionID != "" {
		if s.stores.Session == nil {
			return nil, status.Error(codes.Internal, "session store is not configured")
		}
		if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
			return nil, handleDomainError(err)
		}
		if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
			return nil, err
		}
	}

	adversaryID, err := id.NewID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate adversary id: %v", err)
	}

	payload := daggerheart.AdversaryCreatedPayload{
		AdversaryID: adversaryID,
		Name:        name,
		Kind:        kind,
		SessionID:   sessionID,
		Notes:       notes,
		HP:          stats.HP,
		HPMax:       stats.HPMax,
		Stress:      stats.Stress,
		StressMax:   stats.StressMax,
		Evasion:     stats.Evasion,
		Major:       stats.Major,
		Severe:      stats.Severe,
		Armor:       stats.Armor,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeAdversaryCreated,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append adversary created event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply adversary created event: %v", err)
	}

	created, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load adversary: %v", err)
	}

	return &pb.DaggerheartCreateAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(created),
	}, nil
}

func (s *DaggerheartService) UpdateAdversary(ctx context.Context, in *pb.DaggerheartUpdateAdversaryRequest) (*pb.DaggerheartUpdateAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update adversary request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}
	if in.Name == nil && in.Kind == nil && in.SessionId == nil && in.Notes == nil {
		if in.Hp == nil && in.HpMax == nil && in.Stress == nil && in.StressMax == nil && in.Evasion == nil && in.MajorThreshold == nil && in.SevereThreshold == nil && in.Armor == nil {
			return nil, status.Error(codes.InvalidArgument, "at least one field is required")
		}
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversaries")
	}

	current, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	currentSessionID := strings.TrimSpace(current.SessionID)
	if currentSessionID != "" {
		if err := s.ensureNoOpenSessionGate(ctx, campaignID, currentSessionID); err != nil {
			return nil, err
		}
	}

	name := current.Name
	if in.Name != nil {
		name = strings.TrimSpace(in.Name.GetValue())
		if name == "" {
			return nil, status.Error(codes.InvalidArgument, "name is required")
		}
	}
	kind := current.Kind
	if in.Kind != nil {
		kind = strings.TrimSpace(in.Kind.GetValue())
	}
	sessionID := current.SessionID
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}
	notes := current.Notes
	if in.Notes != nil {
		notes = strings.TrimSpace(in.Notes.GetValue())
	}

	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		HP:            in.Hp,
		HPMax:         in.HpMax,
		Stress:        in.Stress,
		StressMax:     in.StressMax,
		Evasion:       in.Evasion,
		Major:         in.MajorThreshold,
		Severe:        in.SevereThreshold,
		Armor:         in.Armor,
		RequireFields: false,
		Current:       &current,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if sessionID != "" {
		if s.stores.Session == nil {
			return nil, status.Error(codes.Internal, "session store is not configured")
		}
		if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
			return nil, handleDomainError(err)
		}
		if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
			return nil, err
		}
	}

	payload := daggerheart.AdversaryUpdatedPayload{
		AdversaryID: adversaryID,
		Name:        name,
		Kind:        kind,
		SessionID:   sessionID,
		Notes:       notes,
		HP:          stats.HP,
		HPMax:       stats.HPMax,
		Stress:      stats.Stress,
		StressMax:   stats.StressMax,
		Evasion:     stats.Evasion,
		Major:       stats.Major,
		Severe:      stats.Severe,
		Armor:       stats.Armor,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeAdversaryUpdated,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append adversary updated event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply adversary updated event: %v", err)
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load adversary: %v", err)
	}

	return &pb.DaggerheartUpdateAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(updated),
	}, nil
}

func (s *DaggerheartService) DeleteAdversary(ctx context.Context, in *pb.DaggerheartDeleteAdversaryRequest) (*pb.DaggerheartDeleteAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete adversary request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversaries")
	}

	current, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	sessionID := strings.TrimSpace(current.SessionID)
	if sessionID != "" {
		if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
			return nil, err
		}
	}

	payload := daggerheart.AdversaryDeletedPayload{
		AdversaryID: adversaryID,
		Reason:      strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary payload: %v", err)
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:    campaignID,
		Timestamp:     time.Now().UTC(),
		Type:          daggerheart.EventTypeAdversaryDeleted,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ActorType:     event.ActorTypeSystem,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      c.System.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append adversary deleted event: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	if err := adapter.ApplyEvent(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply adversary deleted event: %v", err)
	}

	return &pb.DaggerheartDeleteAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(current),
	}, nil
}

func (s *DaggerheartService) GetAdversary(ctx context.Context, in *pb.DaggerheartGetAdversaryRequest) (*pb.DaggerheartGetAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get adversary request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversaries")
	}

	adversary, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &pb.DaggerheartGetAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(adversary),
	}, nil
}

func (s *DaggerheartService) ListAdversaries(ctx context.Context, in *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list adversaries request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversaries")
	}

	sessionID := ""
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}

	adversaries, err := s.stores.Daggerheart.ListDaggerheartAdversaries(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	response := &pb.DaggerheartListAdversariesResponse{
		Adversaries: make([]*pb.DaggerheartAdversary, 0, len(adversaries)),
	}
	for _, adversary := range adversaries {
		response.Adversaries = append(response.Adversaries, daggerheartAdversaryToProto(adversary))
	}

	return response, nil
}

func (s *DaggerheartService) loadAdversaryForSession(ctx context.Context, campaignID, sessionID, adversaryID string) (storage.DaggerheartAdversary, error) {
	adversary, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.DaggerheartAdversary{}, status.Error(codes.NotFound, "adversary not found")
		}
		return storage.DaggerheartAdversary{}, status.Errorf(codes.Internal, "load adversary: %v", err)
	}
	if adversary.SessionID != "" && adversary.SessionID != sessionID {
		return storage.DaggerheartAdversary{}, status.Error(codes.FailedPrecondition, "adversary is not in session")
	}
	return adversary, nil
}

func daggerheartAdversaryToProto(adversary storage.DaggerheartAdversary) *pb.DaggerheartAdversary {
	var sessionID *wrapperspb.StringValue
	if strings.TrimSpace(adversary.SessionID) != "" {
		sessionID = wrapperspb.String(adversary.SessionID)
	}
	return &pb.DaggerheartAdversary{
		Id:              adversary.AdversaryID,
		CampaignId:      adversary.CampaignID,
		Name:            adversary.Name,
		Kind:            adversary.Kind,
		SessionId:       sessionID,
		Notes:           adversary.Notes,
		Hp:              int32(adversary.HP),
		HpMax:           int32(adversary.HPMax),
		Stress:          int32(adversary.Stress),
		StressMax:       int32(adversary.StressMax),
		Evasion:         int32(adversary.Evasion),
		MajorThreshold:  int32(adversary.Major),
		SevereThreshold: int32(adversary.Severe),
		Armor:           int32(adversary.Armor),
		Conditions:      daggerheartConditionsToProto(adversary.Conditions),
		CreatedAt:       timestamppb.New(adversary.CreatedAt),
		UpdatedAt:       timestamppb.New(adversary.UpdatedAt),
	}
}

type adversaryStatsInput struct {
	HP            *wrapperspb.Int32Value
	HPMax         *wrapperspb.Int32Value
	Stress        *wrapperspb.Int32Value
	StressMax     *wrapperspb.Int32Value
	Evasion       *wrapperspb.Int32Value
	Major         *wrapperspb.Int32Value
	Severe        *wrapperspb.Int32Value
	Armor         *wrapperspb.Int32Value
	RequireFields bool
	Current       *storage.DaggerheartAdversary
}

type adversaryStats struct {
	HP        int
	HPMax     int
	Stress    int
	StressMax int
	Evasion   int
	Major     int
	Severe    int
	Armor     int
}

func normalizeAdversaryStats(input adversaryStatsInput) (adversaryStats, error) {
	stats := adversaryStats{
		HP:        6,
		HPMax:     6,
		Stress:    0,
		StressMax: 6,
		Evasion:   10,
		Major:     8,
		Severe:    12,
		Armor:     0,
	}
	if input.Current != nil {
		stats = adversaryStats{
			HP:        input.Current.HP,
			HPMax:     input.Current.HPMax,
			Stress:    input.Current.Stress,
			StressMax: input.Current.StressMax,
			Evasion:   input.Current.Evasion,
			Major:     input.Current.Major,
			Severe:    input.Current.Severe,
			Armor:     input.Current.Armor,
		}
	}

	if input.HPMax != nil {
		stats.HPMax = int(input.HPMax.GetValue())
	}
	if input.HP != nil {
		stats.HP = int(input.HP.GetValue())
	} else if input.HPMax != nil && input.Current == nil {
		stats.HP = stats.HPMax
	} else if input.HPMax != nil && input.Current != nil && stats.HP > stats.HPMax {
		stats.HP = stats.HPMax
	}

	if input.StressMax != nil {
		stats.StressMax = int(input.StressMax.GetValue())
	}
	if input.Stress != nil {
		stats.Stress = int(input.Stress.GetValue())
	} else if input.StressMax != nil && input.Current == nil {
		stats.Stress = stats.StressMax
	} else if input.StressMax != nil && input.Current != nil && stats.Stress > stats.StressMax {
		stats.Stress = stats.StressMax
	}

	if input.Evasion != nil {
		stats.Evasion = int(input.Evasion.GetValue())
	}
	if input.Major != nil {
		stats.Major = int(input.Major.GetValue())
	}
	if input.Severe != nil {
		stats.Severe = int(input.Severe.GetValue())
	}
	if input.Armor != nil {
		stats.Armor = int(input.Armor.GetValue())
	}

	if stats.HPMax <= 0 {
		return adversaryStats{}, fmt.Errorf("hp_max must be positive")
	}
	if stats.HP < 0 || stats.HP > stats.HPMax {
		return adversaryStats{}, fmt.Errorf("hp must be in range 0..%d", stats.HPMax)
	}
	if stats.StressMax < 0 {
		return adversaryStats{}, fmt.Errorf("stress_max must be non-negative")
	}
	if stats.Stress < 0 || stats.Stress > stats.StressMax {
		return adversaryStats{}, fmt.Errorf("stress must be in range 0..%d", stats.StressMax)
	}
	if stats.Evasion < 0 {
		return adversaryStats{}, fmt.Errorf("evasion must be non-negative")
	}
	if stats.Major < 0 || stats.Severe < 0 {
		return adversaryStats{}, fmt.Errorf("thresholds must be non-negative")
	}
	if stats.Severe < stats.Major {
		return adversaryStats{}, fmt.Errorf("severe_threshold must be >= major_threshold")
	}
	if stats.Armor < 0 {
		return adversaryStats{}, fmt.Errorf("armor must be non-negative")
	}

	if input.RequireFields && (input.HP == nil || input.HPMax == nil) {
		return adversaryStats{}, fmt.Errorf("hp and hp_max are required")
	}

	return stats, nil
}
