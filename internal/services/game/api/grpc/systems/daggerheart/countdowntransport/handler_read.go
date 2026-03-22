package countdowntransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetSceneCountdown returns one stored Daggerheart scene countdown after
// validating campaign and session read access.
func (h *Handler) GetSceneCountdown(ctx context.Context, in *pb.DaggerheartGetSceneCountdownRequest) (*pb.DaggerheartGetSceneCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get scene countdown request is required")
	}
	if err := h.requireReadDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return nil, err
	}
	if h.deps.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if err := h.validateCampaignSessionRead(ctx, campaignID, sessionID, "campaign system does not support daggerheart scene countdowns"); err != nil {
		return nil, err
	}
	countdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if countdown.SessionID != sessionID || countdown.SceneID != sceneID {
		return nil, status.Error(codes.NotFound, "scene countdown was not found")
	}
	return &pb.DaggerheartGetSceneCountdownResponse{Countdown: SceneCountdownToProto(countdown)}, nil
}

// ListSceneCountdowns returns all stored Daggerheart scene countdowns for one
// active scene in stable presentation order.
func (h *Handler) ListSceneCountdowns(ctx context.Context, in *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list scene countdowns request is required")
	}
	if err := h.requireReadDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return nil, err
	}
	if h.deps.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if err := h.validateCampaignSessionRead(ctx, campaignID, sessionID, "campaign system does not support daggerheart scene countdowns"); err != nil {
		return nil, err
	}
	countdowns, err := h.deps.Daggerheart.ListDaggerheartCountdowns(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	resp := &pb.DaggerheartListSceneCountdownsResponse{Countdowns: make([]*pb.DaggerheartSceneCountdown, 0, len(countdowns))}
	for _, countdown := range countdowns {
		if countdown.SessionID != sessionID || countdown.SceneID != sceneID {
			continue
		}
		resp.Countdowns = append(resp.Countdowns, SceneCountdownToProto(countdown))
	}
	return resp, nil
}

// GetCampaignCountdown returns one stored Daggerheart campaign countdown after
// validating campaign read access.
func (h *Handler) GetCampaignCountdown(ctx context.Context, in *pb.DaggerheartGetCampaignCountdownRequest) (*pb.DaggerheartGetCampaignCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign countdown request is required")
	}
	if err := h.requireReadDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return nil, err
	}
	if err := h.validateCampaignRead(ctx, campaignID, "campaign system does not support daggerheart campaign countdowns"); err != nil {
		return nil, err
	}
	countdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if countdown.SessionID != "" || countdown.SceneID != "" {
		return nil, status.Error(codes.NotFound, "campaign countdown was not found")
	}
	return &pb.DaggerheartGetCampaignCountdownResponse{Countdown: CampaignCountdownToProto(countdown)}, nil
}

// ListCampaignCountdowns returns all stored campaign-owned countdowns in
// stable presentation order.
func (h *Handler) ListCampaignCountdowns(ctx context.Context, in *pb.DaggerheartListCampaignCountdownsRequest) (*pb.DaggerheartListCampaignCountdownsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaign countdowns request is required")
	}
	if err := h.requireReadDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	if err := h.validateCampaignRead(ctx, campaignID, "campaign system does not support daggerheart campaign countdowns"); err != nil {
		return nil, err
	}
	countdowns, err := h.deps.Daggerheart.ListDaggerheartCountdowns(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	resp := &pb.DaggerheartListCampaignCountdownsResponse{Countdowns: make([]*pb.DaggerheartCampaignCountdown, 0, len(countdowns))}
	for _, countdown := range countdowns {
		if countdown.SessionID != "" || countdown.SceneID != "" {
			continue
		}
		resp.Countdowns = append(resp.Countdowns, CampaignCountdownToProto(countdown))
	}
	return resp, nil
}

func (h *Handler) requireReadDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	default:
		return nil
	}
}

func (h *Handler) validateCampaignRead(ctx context.Context, campaignID, systemMessage string) error {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, systemMessage); err != nil {
		return err
	}
	return nil
}
