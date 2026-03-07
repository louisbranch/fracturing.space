package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
