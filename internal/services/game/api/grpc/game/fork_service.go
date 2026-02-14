package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListForksPageSize = 10
	maxListForksPageSize     = 50
	forkEventPageSize        = 200
	forkSnapshotPageSize     = 200
)

// ForkService implements the game.v1.ForkService gRPC API.
type ForkService struct {
	campaignv1.UnimplementedForkServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewForkService creates a ForkService with default dependencies.
func NewForkService(stores Stores) *ForkService {
	return &ForkService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// ForkCampaign creates a new campaign by forking an existing campaign at a specific point.
func (s *ForkService) ForkCampaign(ctx context.Context, in *campaignv1.ForkCampaignRequest) (*campaignv1.ForkCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "fork campaign request is required")
	}

	sourceCampaignID := strings.TrimSpace(in.GetSourceCampaignId())
	if sourceCampaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "source campaign id is required")
	}

	if in.GetCopyParticipants() && s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	// Get source campaign
	sourceCampaign, err := s.stores.Campaign.Get(ctx, sourceCampaignID)
	if err != nil {
		if isNotFound(err) {
			return nil, status.Error(codes.NotFound, "source campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "get source campaign: %v", err)
	}

	// Determine fork point
	forkPoint := forkPointFromProto(in.GetForkPoint())
	forkEventSeq, err := s.resolveForkPoint(ctx, sourceCampaignID, forkPoint)
	if err != nil {
		return nil, err
	}

	// Get source campaign's fork metadata to determine origin
	sourceMetadata, err := s.stores.CampaignFork.GetCampaignForkMetadata(ctx, sourceCampaignID)
	if err != nil && !isNotFound(err) {
		return nil, status.Errorf(codes.Internal, "get source fork metadata: %v", err)
	}

	originCampaignID := sourceMetadata.OriginCampaignID
	if originCampaignID == "" {
		originCampaignID = sourceCampaignID
	}

	// Create the fork record
	f, err := fork.CreateFork(fork.CreateForkInput{
		SourceCampaignID: sourceCampaignID,
		ForkPoint:        forkPoint,
		NewCampaignName:  in.GetNewCampaignName(),
		CopyParticipants: in.GetCopyParticipants(),
	}, originCampaignID, forkEventSeq, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	newCampaignName := in.GetNewCampaignName()
	if newCampaignName == "" {
		newCampaignName = fmt.Sprintf("%s (Fork)", sourceCampaign.Name)
	}
	campaignPayload := event.CampaignCreatedPayload{
		Name:        newCampaignName,
		GameSystem:  sourceCampaign.System.String(),
		GmMode:      gmModeToProto(sourceCampaign.GmMode).String(),
		ThemePrompt: sourceCampaign.ThemePrompt,
	}
	campaignJSON, err := json.Marshal(campaignPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	createdEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   f.NewCampaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeCampaignCreated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "campaign",
		EntityID:     f.NewCampaignID,
		PayloadJSON:  campaignJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append campaign.created: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, createdEvent); err != nil {
		return nil, status.Errorf(codes.Internal, "apply campaign.created: %v", err)
	}

	forkPayload := event.CampaignForkedPayload{
		ParentCampaignID: sourceCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: originCampaignID,
		CopyParticipants: in.GetCopyParticipants(),
	}
	forkJSON, err := json.Marshal(forkPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	storedFork, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   f.NewCampaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeCampaignForked,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "campaign",
		EntityID:     f.NewCampaignID,
		PayloadJSON:  forkJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append campaign.forked: %v", err)
	}
	if err := applier.Apply(ctx, storedFork); err != nil {
		return nil, status.Errorf(codes.Internal, "apply campaign.forked: %v", err)
	}

	if _, err := s.copyForkEvents(ctx, sourceCampaignID, f.NewCampaignID, forkEventSeq, in.GetCopyParticipants(), applier); err != nil {
		return nil, status.Errorf(codes.Internal, "copy events: %v", err)
	}

	// Calculate depth by walking the parent chain
	depth := s.calculateDepth(ctx, sourceCampaignID) + 1

	newCampaign, err := s.stores.Campaign.Get(ctx, f.NewCampaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load forked campaign: %v", err)
	}

	lineage := &campaignv1.Lineage{
		CampaignId:       newCampaign.ID,
		ParentCampaignId: sourceCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignId: originCampaignID,
		Depth:            int32(depth),
	}

	return &campaignv1.ForkCampaignResponse{
		Campaign:     campaignToProto(newCampaign),
		Lineage:      lineage,
		ForkEventSeq: forkEventSeq,
	}, nil
}

func (s *ForkService) copyForkEvents(ctx context.Context, sourceCampaignID, forkCampaignID string, forkEventSeq uint64, copyParticipants bool, applier projection.Applier) (time.Time, error) {
	if forkEventSeq == 0 {
		return time.Time{}, nil
	}

	afterSeq := uint64(0)
	var lastEventAt time.Time
	for {
		events, err := s.stores.Event.ListEvents(ctx, sourceCampaignID, afterSeq, forkEventPageSize)
		if err != nil {
			return lastEventAt, fmt.Errorf("list events: %w", err)
		}
		if len(events) == 0 {
			return lastEventAt, nil
		}

		for _, evt := range events {
			if evt.Seq > forkEventSeq {
				return lastEventAt, nil
			}
			lastEventAt = evt.Timestamp
			shouldCopy, err := shouldCopyForkEvent(evt, copyParticipants)
			if err != nil {
				return lastEventAt, fmt.Errorf("filter forked event: %w", err)
			}
			if !shouldCopy {
				afterSeq = evt.Seq
				continue
			}

			forked := forkEventForCampaign(evt, forkCampaignID)
			stored, err := s.stores.Event.AppendEvent(ctx, forked)
			if err != nil {
				return lastEventAt, fmt.Errorf("append forked event: %w", err)
			}
			if err := applier.Apply(ctx, stored); err != nil {
				return lastEventAt, fmt.Errorf("apply forked event: %w", err)
			}
			afterSeq = evt.Seq
		}

		if len(events) < forkEventPageSize {
			return lastEventAt, nil
		}
	}
}

func shouldCopyForkEvent(evt event.Event, copyParticipants bool) (bool, error) {
	switch evt.Type {
	case event.TypeCampaignCreated, event.TypeCampaignForked:
		return false, nil
	case event.TypeParticipantJoined, event.TypeParticipantUpdated, event.TypeParticipantLeft:
		return copyParticipants, nil
	case event.TypeCharacterUpdated:
		if copyParticipants {
			return true, nil
		}
		var payload event.CharacterUpdatedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return false, fmt.Errorf("decode character.updated payload: %w", err)
		}
		participantValue, hasParticipant := payload.Fields["participant_id"]
		if !hasParticipant {
			return true, nil
		}
		participantID, ok := participantValue.(string)
		if !ok {
			return false, fmt.Errorf("character.updated participant_id must be string")
		}
		if strings.TrimSpace(participantID) == "" {
			return true, nil
		}
		if len(payload.Fields) == 1 {
			return false, nil
		}
		return true, nil
	default:
		return true, nil
	}
}

func forkEventForCampaign(evt event.Event, campaignID string) event.Event {
	forked := evt
	forked.CampaignID = campaignID
	forked.Seq = 0
	forked.Hash = ""
	if strings.EqualFold(evt.EntityType, "campaign") {
		forked.EntityID = campaignID
	}
	return forked
}

// GetLineage returns the lineage (ancestry) of a campaign.
func (s *ForkService) GetLineage(ctx context.Context, in *campaignv1.GetLineageRequest) (*campaignv1.GetLineageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get lineage request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	// Verify campaign exists
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if isNotFound(err) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "get campaign: %v", err)
	}

	metadata, err := s.stores.CampaignFork.GetCampaignForkMetadata(ctx, campaignID)
	if err != nil && !isNotFound(err) {
		return nil, status.Errorf(codes.Internal, "get fork metadata: %v", err)
	}

	// Calculate depth by walking up the chain
	depth := 0
	if metadata.ParentCampaignID != "" {
		depth = s.calculateDepth(ctx, metadata.ParentCampaignID) + 1
	}

	originID := metadata.OriginCampaignID
	if originID == "" {
		originID = campaignID
	}

	return &campaignv1.GetLineageResponse{
		Lineage: &campaignv1.Lineage{
			CampaignId:       campaignID,
			ParentCampaignId: metadata.ParentCampaignID,
			ForkEventSeq:     metadata.ForkEventSeq,
			OriginCampaignId: originID,
			Depth:            int32(depth),
		},
	}, nil
}

// ListForks returns campaigns forked from a given campaign.
func (s *ForkService) ListForks(ctx context.Context, in *campaignv1.ListForksRequest) (*campaignv1.ListForksResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list forks request is required")
	}

	sourceCampaignID := strings.TrimSpace(in.GetSourceCampaignId())
	if sourceCampaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "source campaign id is required")
	}

	// Listing forks requires querying campaigns by parent_campaign_id,
	// which is not yet implemented in the storage layer.
	return nil, status.Error(codes.Unimplemented, "list forks not yet implemented")
}

// resolveForkPoint determines the actual event sequence for a fork point.
func (s *ForkService) resolveForkPoint(ctx context.Context, campaignID string, forkPoint fork.ForkPoint) (uint64, error) {
	if forkPoint.IsSessionBoundary() {
		if s.stores.Session == nil {
			return 0, status.Error(codes.Internal, "session store is not configured")
		}
		sessionID := strings.TrimSpace(forkPoint.SessionID)
		if sessionID == "" {
			return 0, status.Error(codes.InvalidArgument, "session id is required for session-based fork points")
		}
		sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
		if err != nil {
			if isNotFound(err) {
				return 0, status.Error(codes.NotFound, "session not found")
			}
			return 0, status.Errorf(codes.Internal, "get session: %v", err)
		}
		if sess.Status != session.SessionStatusEnded {
			return 0, status.Error(codes.FailedPrecondition, "session has not ended")
		}

		lastSeq := uint64(0)
		afterSeq := uint64(0)
		for {
			events, err := s.stores.Event.ListEventsBySession(ctx, campaignID, sessionID, afterSeq, forkEventPageSize)
			if err != nil {
				return 0, status.Errorf(codes.Internal, "list session events: %v", err)
			}
			if len(events) == 0 {
				if lastSeq == 0 {
					return 0, status.Error(codes.FailedPrecondition, "session has no events to fork at")
				}
				return lastSeq, nil
			}
			for _, evt := range events {
				lastSeq = evt.Seq
				afterSeq = evt.Seq
			}
			if len(events) < forkEventPageSize {
				return lastSeq, nil
			}
		}
	}

	// If event seq is 0, use the latest event
	if forkPoint.EventSeq == 0 {
		latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
		if err != nil {
			return 0, status.Errorf(codes.Internal, "get latest event seq: %v", err)
		}
		// If no events exist, fork at seq 0 (start of campaign)
		return latestSeq, nil
	}

	// Validate that the requested event seq exists
	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "get latest event seq: %v", err)
	}

	if forkPoint.EventSeq > latestSeq {
		return 0, status.Error(codes.FailedPrecondition, "fork point is beyond current campaign state")
	}

	return forkPoint.EventSeq, nil
}

// calculateDepth calculates the fork depth by walking up the parent chain.
func (s *ForkService) calculateDepth(ctx context.Context, campaignID string) int {
	depth := 0
	currentID := campaignID

	for i := 0; i < 100; i++ { // Limit to prevent infinite loops
		metadata, err := s.stores.CampaignFork.GetCampaignForkMetadata(ctx, currentID)
		if err != nil || metadata.ParentCampaignID == "" {
			break
		}
		depth++
		currentID = metadata.ParentCampaignID
	}

	return depth
}

// forkPointFromProto converts a proto ForkPoint to domain ForkPoint.
func forkPointFromProto(pb *campaignv1.ForkPoint) fork.ForkPoint {
	if pb == nil {
		return fork.ForkPoint{}
	}
	return fork.ForkPoint{
		EventSeq:  pb.GetEventSeq(),
		SessionID: pb.GetSessionId(),
	}
}

// isNotFound reports whether err is a not-found error.
func isNotFound(err error) bool {
	return err == storage.ErrNotFound
}
