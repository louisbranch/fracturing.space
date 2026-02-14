package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type forkApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newForkApplication(service *ForkService) forkApplication {
	app := forkApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (a forkApplication) ForkCampaign(ctx context.Context, sourceCampaignID string, in *campaignv1.ForkCampaignRequest) (campaign.Campaign, *campaignv1.Lineage, uint64, error) {
	if in.GetCopyParticipants() && a.stores.Participant == nil {
		return campaign.Campaign{}, nil, 0, status.Error(codes.Internal, "participant store is not configured")
	}

	// Get source campaign
	sourceCampaign, err := a.stores.Campaign.Get(ctx, sourceCampaignID)
	if err != nil {
		if isNotFound(err) {
			return campaign.Campaign{}, nil, 0, status.Error(codes.NotFound, "source campaign not found")
		}
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "get source campaign: %v", err)
	}

	// Determine fork point
	forkPoint := forkPointFromProto(in.GetForkPoint())
	forkEventSeq, err := a.resolveForkPoint(ctx, sourceCampaignID, forkPoint)
	if err != nil {
		return campaign.Campaign{}, nil, 0, err
	}

	// Get source campaign's fork metadata to determine origin
	sourceMetadata, err := a.stores.CampaignFork.GetCampaignForkMetadata(ctx, sourceCampaignID)
	if err != nil && !isNotFound(err) {
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "get source fork metadata: %v", err)
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
	}, originCampaignID, forkEventSeq, a.clock, a.idGenerator)
	if err != nil {
		return campaign.Campaign{}, nil, 0, err
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
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	createdEvent, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   f.NewCampaignID,
		Timestamp:    a.clock().UTC(),
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
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "append campaign.created: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, createdEvent); err != nil {
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "apply campaign.created: %v", err)
	}

	forkPayload := event.CampaignForkedPayload{
		ParentCampaignID: sourceCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: originCampaignID,
		CopyParticipants: in.GetCopyParticipants(),
	}
	forkJSON, err := json.Marshal(forkPayload)
	if err != nil {
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	storedFork, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   f.NewCampaignID,
		Timestamp:    a.clock().UTC(),
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
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "append campaign.forked: %v", err)
	}
	if err := applier.Apply(ctx, storedFork); err != nil {
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "apply campaign.forked: %v", err)
	}

	if _, err := a.copyForkEvents(ctx, sourceCampaignID, f.NewCampaignID, forkEventSeq, in.GetCopyParticipants(), applier); err != nil {
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "copy events: %v", err)
	}

	// Calculate depth by walking the parent chain
	depth := calculateDepth(ctx, a.stores.CampaignFork, sourceCampaignID) + 1

	newCampaign, err := a.stores.Campaign.Get(ctx, f.NewCampaignID)
	if err != nil {
		return campaign.Campaign{}, nil, 0, status.Errorf(codes.Internal, "load forked campaign: %v", err)
	}

	lineage := &campaignv1.Lineage{
		CampaignId:       newCampaign.ID,
		ParentCampaignId: sourceCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignId: originCampaignID,
		Depth:            int32(depth),
	}

	return newCampaign, lineage, forkEventSeq, nil
}

func (a forkApplication) copyForkEvents(ctx context.Context, sourceCampaignID, forkCampaignID string, forkEventSeq uint64, copyParticipants bool, applier projection.Applier) (time.Time, error) {
	if forkEventSeq == 0 {
		return time.Time{}, nil
	}

	afterSeq := uint64(0)
	var lastEventAt time.Time
	for {
		events, err := a.stores.Event.ListEvents(ctx, sourceCampaignID, afterSeq, forkEventPageSize)
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
			stored, err := a.stores.Event.AppendEvent(ctx, forked)
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

func (a forkApplication) resolveForkPoint(ctx context.Context, campaignID string, forkPoint fork.ForkPoint) (uint64, error) {
	if forkPoint.IsSessionBoundary() {
		if a.stores.Session == nil {
			return 0, status.Error(codes.Internal, "session store is not configured")
		}
		sessionID := strings.TrimSpace(forkPoint.SessionID)
		if sessionID == "" {
			return 0, status.Error(codes.InvalidArgument, "session id is required for session-based fork points")
		}
		sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
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
			events, err := a.stores.Event.ListEventsBySession(ctx, campaignID, sessionID, afterSeq, forkEventPageSize)
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
		latestSeq, err := a.stores.Event.GetLatestEventSeq(ctx, campaignID)
		if err != nil {
			return 0, status.Errorf(codes.Internal, "get latest event seq: %v", err)
		}
		// If no events exist, fork at seq 0 (start of campaign)
		return latestSeq, nil
	}

	// Validate that the requested event seq exists
	latestSeq, err := a.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "get latest event seq: %v", err)
	}

	if forkPoint.EventSeq > latestSeq {
		return 0, status.Error(codes.FailedPrecondition, "fork point is beyond current campaign state")
	}

	return forkPoint.EventSeq, nil
}
