package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *DaggerheartService) runCreateAdversary(ctx context.Context, in *pb.DaggerheartCreateAdversaryRequest) (*pb.DaggerheartCreateAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create adversary request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
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
	sceneID := strings.TrimSpace(in.GetSceneId())

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
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}

	if sessionID != "" {
		if err := s.requireDependencies(dependencySessionStore); err != nil {
			return nil, err
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

	payload := daggerheart.AdversaryCreatePayload{
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

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartAdversaryCreate,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("adversary create did not emit an event", "apply adversary created event"))
	if err != nil {
		return nil, err
	}

	created, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load adversary: %v", err)
	}

	return &pb.DaggerheartCreateAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(created),
	}, nil
}

func (s *DaggerheartService) runUpdateAdversary(ctx context.Context, in *pb.DaggerheartUpdateAdversaryRequest) (*pb.DaggerheartUpdateAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update adversary request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
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
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
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
	sceneID := strings.TrimSpace(in.GetSceneId())
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
		if err := s.requireDependencies(dependencySessionStore); err != nil {
			return nil, err
		}
		if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
			return nil, handleDomainError(err)
		}
		if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
			return nil, err
		}
	}

	payload := daggerheart.AdversaryUpdatePayload{
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

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartAdversaryUpdate,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("adversary update did not emit an event", "apply adversary updated event"))
	if err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load adversary: %v", err)
	}

	return &pb.DaggerheartUpdateAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(updated),
	}, nil
}

func (s *DaggerheartService) runDeleteAdversary(ctx context.Context, in *pb.DaggerheartDeleteAdversaryRequest) (*pb.DaggerheartDeleteAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete adversary request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
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
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}

	current, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	sessionID := strings.TrimSpace(current.SessionID)
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sessionID != "" {
		if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
			return nil, err
		}
	}

	payload := daggerheart.AdversaryDeletePayload{
		AdversaryID: adversaryID,
		Reason:      strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartAdversaryDelete,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("adversary delete did not emit an event", "apply adversary deleted event"))
	if err != nil {
		return nil, err
	}

	return &pb.DaggerheartDeleteAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(current),
	}, nil
}

func (s *DaggerheartService) runGetAdversary(ctx context.Context, in *pb.DaggerheartGetAdversaryRequest) (*pb.DaggerheartGetAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get adversary request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore); err != nil {
		return nil, err
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
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}

	adversary, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &pb.DaggerheartGetAdversaryResponse{
		Adversary: daggerheartAdversaryToProto(adversary),
	}, nil
}

func (s *DaggerheartService) runListAdversaries(ctx context.Context, in *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list adversaries request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore); err != nil {
		return nil, err
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
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
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
