package ai

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// ListCampaignDebugTurns returns newest-first debug turn summaries for one session.
func (h *CampaignDebugHandlers) ListCampaignDebugTurns(ctx context.Context, in *aiv1.ListCampaignDebugTurnsRequest) (*aiv1.ListCampaignDebugTurnsResponse, error) {
	if err := requireUnaryRequest(in, "list campaign debug turns request is required"); err != nil {
		return nil, err
	}
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return nil, err
	}
	page, err := h.svc.ListCampaignDebugTurns(ctx, service.ListCampaignDebugTurnsInput{
		CampaignID: strings.TrimSpace(in.GetCampaignId()),
		SessionID:  strings.TrimSpace(in.GetSessionId()),
		PageSize:   clampPageSize(in.GetPageSize()),
		PageToken:  strings.TrimSpace(in.GetPageToken()),
	})
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{Operation: "list campaign debug turns"})
	}
	resp := &aiv1.ListCampaignDebugTurnsResponse{
		Turns:         make([]*aiv1.CampaignDebugTurnSummary, 0, len(page.Turns)),
		NextPageToken: page.NextPageToken,
	}
	for _, turn := range page.Turns {
		resp.Turns = append(resp.Turns, campaignDebugTurnSummaryToProto(turn))
	}
	return resp, nil
}

// GetCampaignDebugTurn returns one turn plus its ordered trace entries.
func (h *CampaignDebugHandlers) GetCampaignDebugTurn(ctx context.Context, in *aiv1.GetCampaignDebugTurnRequest) (*aiv1.GetCampaignDebugTurnResponse, error) {
	if err := requireUnaryRequest(in, "get campaign debug turn request is required"); err != nil {
		return nil, err
	}
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return nil, err
	}
	result, err := h.svc.GetCampaignDebugTurn(ctx, service.GetCampaignDebugTurnInput{
		CampaignID: strings.TrimSpace(in.GetCampaignId()),
		TurnID:     strings.TrimSpace(in.GetTurnId()),
	})
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{Operation: "get campaign debug turn"})
	}
	return &aiv1.GetCampaignDebugTurnResponse{
		Turn: campaignDebugTurnToProto(result.Turn, result.Entries),
	}, nil
}

// SubscribeCampaignDebugUpdates streams future-only debug turn updates for one session.
func (h *CampaignDebugHandlers) SubscribeCampaignDebugUpdates(in *aiv1.SubscribeCampaignDebugUpdatesRequest, stream aiv1.CampaignDebugService_SubscribeCampaignDebugUpdatesServer) error {
	if err := requireUnaryRequest(in, "subscribe campaign debug updates request is required"); err != nil {
		return err
	}
	ctx := stream.Context()
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return err
	}
	updates, unsubscribe, err := h.svc.SubscribeCampaignDebugUpdates(ctx, service.SubscribeCampaignDebugUpdatesInput{
		CampaignID: strings.TrimSpace(in.GetCampaignId()),
		SessionID:  strings.TrimSpace(in.GetSessionId()),
	})
	if unsubscribe != nil {
		defer unsubscribe()
	}
	if err != nil {
		return transportErrorToStatus(err, transportErrorConfig{Operation: "subscribe campaign debug updates"})
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if err := stream.Send(campaignDebugTurnUpdateToProto(update)); err != nil {
				return err
			}
		}
	}
}
