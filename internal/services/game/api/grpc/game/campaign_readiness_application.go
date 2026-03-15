package game

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type campaignReadinessApplication struct {
	auth   policyDependencies
	stores campaignReadinessStores
}

type campaignReadinessStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Session     storage.SessionStore
	Daggerheart projectionstore.Store
}

func newCampaignReadinessApplication(stores Stores) campaignReadinessApplication {
	return campaignReadinessApplication{
		auth: newPolicyDependencies(stores),
		stores: campaignReadinessStores{
			Campaign:    stores.Campaign,
			Participant: stores.Participant,
			Character:   stores.Character,
			Session:     stores.Session,
			Daggerheart: stores.SystemStores.Daggerheart,
		},
	}
}

func (a campaignReadinessApplication) GetCampaignSessionReadiness(
	ctx context.Context,
	campaignID string,
	requestedLocale commonv1.Locale,
) (*campaignv1.CampaignSessionReadiness, error) {
	record, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicyWithDependencies(ctx, a.auth, record); err != nil {
		return nil, err
	}

	participantsByCampaign, err := a.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("list participants by campaign", err)
	}

	charactersByCampaign, err := listAllCharactersByCampaign(ctx, a.stores.Character, campaignID)
	if err != nil {
		return nil, err
	}

	hasActiveSession, err := campaignHasActiveSession(ctx, a.stores.Session, campaignID)
	if err != nil {
		return nil, err
	}

	state, err := campaignReadinessAggregateState(ctx, a.stores.Daggerheart, record, participantsByCampaign, charactersByCampaign)
	if err != nil {
		return nil, err
	}

	report := readiness.EvaluateSessionStartReport(state, readiness.ReportOptions{
		SystemReadiness:        systemReadinessChecker(systemIDFromCampaignRecord(record), state),
		IncludeSessionBoundary: true,
		HasActiveSession:       hasActiveSession,
	})
	locale := resolveReadinessLocale(requestedLocale, record.Locale)
	readinessProto := &campaignv1.CampaignSessionReadiness{
		Ready: report.Ready(),
	}
	if len(report.Blockers) == 0 {
		return readinessProto, nil
	}

	readinessProto.Blockers = make([]*campaignv1.CampaignSessionReadinessBlocker, 0, len(report.Blockers))
	for _, blocker := range report.Blockers {
		readinessProto.Blockers = append(readinessProto.Blockers, readinessBlockerToProto(locale, blocker))
	}
	return readinessProto, nil
}
