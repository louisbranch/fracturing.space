package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OpenSessionGate opens a session gate that blocks action events until resolved.
func (s *SessionService) OpenSessionGate(ctx context.Context, in *campaignv1.OpenSessionGateRequest) (*campaignv1.OpenSessionGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open session gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	gate, err := newSessionApplication(s).OpenSessionGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	pbGate, err := sessionGateToProto(gate)
	if err != nil {
		return nil, grpcerror.Internal("decode session gate", err)
	}

	return &campaignv1.OpenSessionGateResponse{Gate: pbGate}, nil
}

// ResolveSessionGate resolves an open session gate.
func (s *SessionService) ResolveSessionGate(ctx context.Context, in *campaignv1.ResolveSessionGateRequest) (*campaignv1.ResolveSessionGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve session gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	gate, err := newSessionApplication(s).ResolveSessionGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	pbGate, err := sessionGateToProto(gate)
	if err != nil {
		return nil, grpcerror.Internal("decode session gate", err)
	}

	return &campaignv1.ResolveSessionGateResponse{Gate: pbGate}, nil
}

// AbandonSessionGate abandons an open session gate.
func (s *SessionService) AbandonSessionGate(ctx context.Context, in *campaignv1.AbandonSessionGateRequest) (*campaignv1.AbandonSessionGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon session gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	gate, err := newSessionApplication(s).AbandonSessionGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	pbGate, err := sessionGateToProto(gate)
	if err != nil {
		return nil, grpcerror.Internal("decode session gate", err)
	}

	return &campaignv1.AbandonSessionGateResponse{Gate: pbGate}, nil
}
