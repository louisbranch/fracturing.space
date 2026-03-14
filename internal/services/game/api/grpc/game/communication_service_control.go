package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OpenCommunicationGate opens a new active-session communication gate for GM-managed control workflows.
func (s *CommunicationService) OpenCommunicationGate(ctx context.Context, in *campaignv1.OpenCommunicationGateRequest) (*campaignv1.OpenCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).OpenCommunicationGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.OpenCommunicationGateResponse{Context: contextState}, nil
}

// RequestGMHandoff opens or reuses the active session's GM handoff gate for a participant-driven control action.
func (s *CommunicationService) RequestGMHandoff(ctx context.Context, in *campaignv1.RequestGMHandoffRequest) (*campaignv1.RequestGMHandoffResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request gm handoff request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).RequestGMHandoff(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.RequestGMHandoffResponse{Context: contextState}, nil
}

// ResolveCommunicationGate resolves the active session's current communication gate.
func (s *CommunicationService) ResolveCommunicationGate(ctx context.Context, in *campaignv1.ResolveCommunicationGateRequest) (*campaignv1.ResolveCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).ResolveCommunicationGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResolveCommunicationGateResponse{Context: contextState}, nil
}

// RespondToCommunicationGate records one participant response against the active session gate.
func (s *CommunicationService) RespondToCommunicationGate(ctx context.Context, in *campaignv1.RespondToCommunicationGateRequest) (*campaignv1.RespondToCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "respond to communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).RespondToCommunicationGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.RespondToCommunicationGateResponse{Context: contextState}, nil
}

// ResolveGMHandoff resolves the active session's GM handoff gate and returns refreshed communication context.
func (s *CommunicationService) ResolveGMHandoff(ctx context.Context, in *campaignv1.ResolveGMHandoffRequest) (*campaignv1.ResolveGMHandoffResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve gm handoff request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).ResolveGMHandoff(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResolveGMHandoffResponse{Context: contextState}, nil
}

// AbandonCommunicationGate abandons the active session's current communication gate.
func (s *CommunicationService) AbandonCommunicationGate(ctx context.Context, in *campaignv1.AbandonCommunicationGateRequest) (*campaignv1.AbandonCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).AbandonCommunicationGate(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.AbandonCommunicationGateResponse{Context: contextState}, nil
}

// AbandonGMHandoff abandons the active session's GM handoff gate and returns refreshed communication context.
func (s *CommunicationService) AbandonGMHandoff(ctx context.Context, in *campaignv1.AbandonGMHandoffRequest) (*campaignv1.AbandonGMHandoffResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon gm handoff request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := newCommunicationApplication(s).AbandonGMHandoff(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}
	return &campaignv1.AbandonGMHandoffResponse{Context: contextState}, nil
}
