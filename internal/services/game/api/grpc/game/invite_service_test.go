package game

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"google.golang.org/grpc/codes"
)

func TestCreateInvite_NilRequest(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.CreateInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateInvite_MissingStore(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{CampaignId: "campaign-1", ParticipantId: "participant-1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateInvite_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	svc := &InviteService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		clock:       fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		idGenerator: fixedIDGenerator("invite-123"),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	})
	if err != nil {
		t.Fatalf("CreateInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("CreateInvite response has nil invite")
	}
	if resp.Invite.Id != "invite-123" {
		t.Fatalf("invite id = %s, want invite-123", resp.Invite.Id)
	}
	if eventStore.events["campaign-1"][0].Type != event.TypeInviteCreated {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.TypeInviteCreated)
	}
}

func TestRevokeInvite_AlreadyClaimed(t *testing.T) {
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()
	inviteStore.invites["invite-1"] = invite.Invite{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusClaimed}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := &InviteService{
		stores:      Stores{Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: eventStore},
		clock:       fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		idGenerator: fixedIDGenerator("invite-123"),
	}

	ctx := contextWithParticipantID("owner-1")
	_, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateInvite_MissingParticipantIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	svc := &InviteService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		clock:       fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		idGenerator: fixedIDGenerator("invite-123"),
	}

	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestClaimInvite_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = invite.Invite{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		clock: fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	signer := newJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.clock())
	ctx := contextWithUserID("user-1")
	resp, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err != nil {
		t.Fatalf("ClaimInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("ClaimInvite response has nil invite")
	}
	if resp.Participant == nil {
		t.Fatal("ClaimInvite response has nil participant")
	}
	if resp.Invite.Status != statev1.InviteStatus_CLAIMED {
		t.Fatalf("invite status = %v, want CLAIMED", resp.Invite.Status)
	}
	if resp.Participant.UserId != "user-1" {
		t.Fatalf("participant user_id = %s, want user-1", resp.Participant.UserId)
	}

	if len(eventStore.events["campaign-1"]) != 2 {
		t.Fatalf("event count = %d, want 2", len(eventStore.events["campaign-1"]))
	}
	if eventStore.events["campaign-1"][0].Type != event.TypeParticipantBound {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.TypeParticipantBound)
	}
	if eventStore.events["campaign-1"][1].Type != event.TypeInviteClaimed {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][1].Type, event.TypeInviteClaimed)
	}
}

func TestClaimInvite_MissingUserID(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = invite.Invite{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		clock: fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	_, err := svc.ClaimInvite(context.Background(), &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  "grant",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_IdempotentGrant(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = invite.Invite{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		clock: fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	signer := newJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "jti-1", svc.clock())
	ctx := contextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err != nil {
		t.Fatalf("ClaimInvite returned error: %v", err)
	}

	_, err = svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err != nil {
		t.Fatalf("ClaimInvite returned error on retry: %v", err)
	}

	if len(eventStore.events["campaign-1"]) != 2 {
		t.Fatalf("event count = %d, want 2", len(eventStore.events["campaign-1"]))
	}
}

func TestClaimInvite_UserAlreadyClaimed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = campaign.Campaign{ID: "campaign-1", Status: campaign.CampaignStatusDraft}
	participantStore.participants["campaign-1"] = map[string]participant.Participant{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
		"participant-2": {ID: "participant-2", CampaignID: "campaign-1", UserID: "user-1"},
	}
	inviteStore.invites["invite-1"] = invite.Invite{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		clock: fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	signer := newJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "jti-2", svc.clock())
	ctx := contextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	assertStatusCode(t, err, codes.AlreadyExists)
}
