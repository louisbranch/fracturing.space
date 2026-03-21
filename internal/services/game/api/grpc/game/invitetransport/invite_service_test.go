package invitetransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestRevokeInvite_AlreadyClaimed(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusClaimed}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: eventStore},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRevokeInvite_NilRequest(t *testing.T) {
	svc := NewService(Deps{}, nil)
	_, err := svc.RevokeInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeInvite_MissingInviteId(t *testing.T) {
	svc := NewService(Deps{Invite: gametest.NewFakeInviteStore(), Campaign: gametest.NewFakeCampaignStore(), Event: gametest.NewFakeEventStore()}, nil)
	_, err := svc.RevokeInvite(context.Background(), &statev1.RevokeInviteRequest{InviteId: ""})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeInvite_InviteNotFound(t *testing.T) {
	svc := NewService(Deps{Invite: gametest.NewFakeInviteStore(), Campaign: gametest.NewFakeCampaignStore(), Event: gametest.NewFakeEventStore()}, nil)
	_, err := svc.RevokeInvite(context.Background(), &statev1.RevokeInviteRequest{InviteId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRevokeInvite_AlreadyRevoked(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusRevoked}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: gametest.NewFakeEventStore()},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
