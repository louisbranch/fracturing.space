package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OpenSceneGate opens a gate that blocks scene actions until resolved.
func (s *SceneService) OpenSceneGate(ctx context.Context, in *campaignv1.OpenSceneGateRequest) (*campaignv1.OpenSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open scene gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).OpenSceneGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.OpenSceneGateResponse{}, nil
}

// ResolveSceneGate resolves an open scene gate.
func (s *SceneService) ResolveSceneGate(ctx context.Context, in *campaignv1.ResolveSceneGateRequest) (*campaignv1.ResolveSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve scene gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).ResolveSceneGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ResolveSceneGateResponse{}, nil
}

// AbandonSceneGate abandons an open scene gate.
func (s *SceneService) AbandonSceneGate(ctx context.Context, in *campaignv1.AbandonSceneGateRequest) (*campaignv1.AbandonSceneGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon scene gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).AbandonSceneGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.AbandonSceneGateResponse{}, nil
}
