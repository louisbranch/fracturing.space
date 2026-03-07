package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCampaignSessionReadiness returns deterministic readiness blockers for session start.
func (s *CampaignService) GetCampaignSessionReadiness(ctx context.Context, in *campaignv1.GetCampaignSessionReadinessRequest) (*campaignv1.GetCampaignSessionReadinessResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign session readiness request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	record, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, record); err != nil {
		return nil, err
	}

	participantsByCampaign, err := s.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants by campaign: %v", err)
	}

	charactersByCampaign, err := listAllCharactersByCampaign(ctx, s.stores.Character, campaignID)
	if err != nil {
		return nil, err
	}

	hasActiveSession, err := campaignHasActiveSession(ctx, s.stores.Session, campaignID)
	if err != nil {
		return nil, err
	}

	state, err := campaignReadinessAggregateState(ctx, s.stores, record, participantsByCampaign, charactersByCampaign)
	if err != nil {
		return nil, err
	}

	report := readiness.EvaluateSessionStartReport(state, readiness.ReportOptions{
		SystemReadiness:        systemReadinessChecker(record.System),
		IncludeSessionBoundary: true,
		HasActiveSession:       hasActiveSession,
	})
	locale := resolveReadinessLocale(in.GetLocale(), record.Locale)
	readinessProto := &campaignv1.CampaignSessionReadiness{
		Ready: report.Ready(),
	}
	if len(report.Blockers) > 0 {
		readinessProto.Blockers = make([]*campaignv1.CampaignSessionReadinessBlocker, 0, len(report.Blockers))
		for _, blocker := range report.Blockers {
			readinessProto.Blockers = append(readinessProto.Blockers, readinessBlockerToProto(locale, blocker))
		}
	}

	return &campaignv1.GetCampaignSessionReadinessResponse{
		Readiness: readinessProto,
	}, nil
}
