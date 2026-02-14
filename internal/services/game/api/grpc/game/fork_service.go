package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/fork"
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

	newCampaign, lineage, forkEventSeq, err := newForkApplication(s).ForkCampaign(ctx, sourceCampaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ForkCampaignResponse{
		Campaign:     campaignToProto(newCampaign),
		Lineage:      lineage,
		ForkEventSeq: forkEventSeq,
	}, nil
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
		depth = calculateDepth(ctx, s.stores.CampaignFork, metadata.ParentCampaignID) + 1
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

// calculateDepth calculates the fork depth by walking up the parent chain.
func calculateDepth(ctx context.Context, store storage.CampaignForkStore, campaignID string) int {
	depth := 0
	currentID := campaignID

	for i := 0; i < 100; i++ { // Limit to prevent infinite loops
		metadata, err := store.GetCampaignForkMetadata(ctx, currentID)
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
