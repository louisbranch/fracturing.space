package forktransport

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetLineage_RequiresCampaignReadPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
	}

	participantStore := gametest.NewFakeParticipantStore()
	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Participant:  participantStore,
	}, nil, nil)

	_, err := svc.GetLineage(context.Background(), &statev1.GetLineageRequest{CampaignId: "camp-1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListForks_NilRequest(t *testing.T) {
	svc := newServiceForTest(Deps{}, nil, nil)
	_, err := svc.ListForks(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListForks_MissingSourceCampaignId(t *testing.T) {
	svc := newServiceForTest(Deps{Campaign: gametest.NewFakeCampaignStore()}, nil, nil)
	_, err := svc.ListForks(context.Background(), &statev1.ListForksRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListForks_Unimplemented(t *testing.T) {
	svc := newServiceForTest(Deps{Campaign: gametest.NewFakeCampaignStore()}, nil, nil)
	_, err := svc.ListForks(context.Background(), &statev1.ListForksRequest{SourceCampaignId: "camp-1"})
	assertStatusCode(t, err, codes.Unimplemented)
}

func TestForkPointFromProto(t *testing.T) {
	fp := forkPointFromProto(nil)
	if fp.EventSeq != 0 || fp.SessionID != "" {
		t.Fatalf("expected zero ForkPoint for nil input, got %+v", fp)
	}

	fp = forkPointFromProto(&statev1.ForkPoint{EventSeq: 42, SessionId: "sess-1"})
	if fp.EventSeq != 42 || fp.SessionID != "sess-1" {
		t.Fatalf("ForkPoint = %+v, want EventSeq=42 SessionID=sess-1", fp)
	}
}
