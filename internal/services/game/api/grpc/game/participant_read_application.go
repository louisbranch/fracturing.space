package game

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type participantListPage struct {
	participants  []storage.ParticipantRecord
	nextPageToken string
}

func (c participantApplication) ListParticipants(ctx context.Context, campaignID, pageToken string, pageSize int32) (participantListPage, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return participantListPage{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return participantListPage{}, err
	}
	if err := requireReadPolicyWithDependencies(ctx, c.auth, campaignRecord); err != nil {
		return participantListPage{}, err
	}

	resolvedPageSize := pagination.ClampPageSize(pageSize, pagination.PageSizeConfig{
		Default: defaultListParticipantsPageSize,
		Max:     maxListParticipantsPageSize,
	})
	page, err := c.stores.Participant.ListParticipants(ctx, campaignID, resolvedPageSize, pageToken)
	if err != nil {
		return participantListPage{}, grpcerror.Internal("list participants", err)
	}
	return participantListPage{
		participants:  page.Participants,
		nextPageToken: page.NextPageToken,
	}, nil
}

func (c participantApplication) GetParticipant(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := requireReadPolicyWithDependencies(ctx, c.auth, campaignRecord); err != nil {
		return storage.ParticipantRecord{}, err
	}

	record, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	return record, nil
}
