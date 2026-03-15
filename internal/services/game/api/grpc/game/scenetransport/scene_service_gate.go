package scenetransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OpenSceneGate opens a gate that blocks scene actions until resolved.
func (s *Service) OpenSceneGate(ctx context.Context, in *campaignv1.OpenSceneGateRequest) (*campaignv1.OpenSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open scene gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).OpenSceneGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.OpenSceneGateResponse{}, nil
}

// ResolveSceneGate resolves an open scene gate.
func (s *Service) ResolveSceneGate(ctx context.Context, in *campaignv1.ResolveSceneGateRequest) (*campaignv1.ResolveSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve scene gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).ResolveSceneGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ResolveSceneGateResponse{}, nil
}

// AbandonSceneGate abandons an open scene gate.
func (s *Service) AbandonSceneGate(ctx context.Context, in *campaignv1.AbandonSceneGateRequest) (*campaignv1.AbandonSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon scene gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).AbandonSceneGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.AbandonSceneGateResponse{}, nil
}
