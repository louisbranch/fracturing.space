package forktransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestForkCampaign_RequiresCampaignManagePolicy(t *testing.T) {
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:     "source",
		Name:   "Source Campaign",
		Status: campaign.StatusActive,
	}

	participantStore := gametest.NewFakeParticipantStore()
	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(context.Background(), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestForkCampaign_AllowsManagerManagePolicy(t *testing.T) {
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:     "source",
		Name:   "Source Campaign",
		Status: campaign.StatusActive,
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"manager-1": {ID: "manager-1", CampaignID: "source", CampaignAccess: participant.CampaignAccessManager},
	}

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(gametest.ContextWithParticipantID("manager-1"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestForkCampaign_DeniesMemberManagePolicy(t *testing.T) {
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:     "source",
		Name:   "Source Campaign",
		Status: campaign.StatusActive,
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"member-1": {ID: "member-1", CampaignID: "source", CampaignAccess: participant.CampaignAccessMember},
	}

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(gametest.ContextWithParticipantID("member-1"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}
