package adversarytransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
)

// NewHandler builds an adversary transport handler from explicit reads and
// write callbacks.
func NewHandler(deps Dependencies) *Handler {
	if deps.GenerateID == nil {
		deps.GenerateID = id.NewID
	}
	return &Handler{deps: deps}
}

func (h *Handler) GetAdversary(ctx context.Context, in *pb.DaggerheartGetAdversaryRequest) (*pb.DaggerheartGetAdversaryResponse, error) {
	if in == nil {
		return nil, invalidArgument("get adversary request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}
	adversary, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	return &pb.DaggerheartGetAdversaryResponse{Adversary: adversaryToProto(adversary)}, nil
}

func (h *Handler) ListAdversaries(ctx context.Context, in *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	if in == nil {
		return nil, invalidArgument("list adversaries request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}
	sessionID := ""
	if in.SessionId != nil {
		sessionID = strings.TrimSpace(in.SessionId.GetValue())
	}
	adversaries, err := h.deps.Daggerheart.ListDaggerheartAdversaries(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	resp := &pb.DaggerheartListAdversariesResponse{Adversaries: make([]*pb.DaggerheartAdversary, 0, len(adversaries))}
	for _, adversary := range adversaries {
		resp.Adversaries = append(resp.Adversaries, adversaryToProto(adversary))
	}
	return resp, nil
}

func (h *Handler) requireBaseDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return internal("campaign store is not configured")
	case h.deps.Gate == nil:
		return internal("session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return internal("daggerheart store is not configured")
	default:
		return nil
	}
}
