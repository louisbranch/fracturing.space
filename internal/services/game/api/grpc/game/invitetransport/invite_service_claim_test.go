package invitetransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/testclients"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
)

func TestClaimInvite_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
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

	if len(eventStore.Events["campaign-1"]) != 4 {
		t.Fatalf("event count = %d, want 4", len(eventStore.Events["campaign-1"]))
	}
	if eventStore.Events["campaign-1"][0].Type != participant.EventTypeJoined {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, participant.EventTypeJoined)
	}
	if eventStore.Events["campaign-1"][1].Type != invite.EventTypeCreated {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][1].Type, invite.EventTypeCreated)
	}
	if eventStore.Events["campaign-1"][2].Type != participant.EventTypeBound {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][2].Type, participant.EventTypeBound)
	}
	if eventStore.Events["campaign-1"][3].Type != invite.EventTypeClaimed {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][3].Type, invite.EventTypeClaimed)
	}
}

func TestClaimInvite_UsesReplayStateForSeatOccupancy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seedParticipantJoinedEvent(t, eventStore, storage.ParticipantRecord{
		ID:         "participant-1",
		CampaignID: "campaign-1",
		UserID:     "user-existing",
	}, now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-new", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-new")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if len(eventStore.Events["campaign-1"]) != 2 {
		t.Fatalf("event count = %d, want 2", len(eventStore.Events["campaign-1"]))
	}
}

func TestClaimInvite_DetectsExistingUserFromReplayState(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
		"participant-2": {ID: "participant-2", CampaignID: "campaign-1", UserID: "user-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-2"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	resp, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err == nil {
		t.Fatalf("ClaimInvite() expected conflict, got response %+v", resp)
	}
	assertStatusCode(t, err, codes.AlreadyExists)
	if len(eventStore.Events["campaign-1"]) != 3 {
		t.Fatalf("event count = %d, want 3", len(eventStore.Events["campaign-1"]))
	}
}

func TestClaimInvite_IgnoresLeftParticipantWhenCheckingExistingUser(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedParticipantJoinedEvent(t, eventStore, storage.ParticipantRecord{
		ID:         "participant-2",
		CampaignID: "campaign-1",
		UserID:     "user-1",
	}, now)
	seedParticipantLeftEvent(t, eventStore, "campaign-1", "participant-2", now.Add(time.Second))
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	resp, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err != nil {
		t.Fatalf("ClaimInvite returned error: %v", err)
	}
	if resp.Participant.GetUserId() != "user-1" {
		t.Fatalf("participant user_id = %q, want %q", resp.Participant.GetUserId(), "user-1")
	}
	if len(eventStore.Events["campaign-1"]) != 6 {
		t.Fatalf("event count = %d, want 6", len(eventStore.Events["campaign-1"]))
	}
}

func TestClaimInvite_IgnoresUnboundParticipantWhenCheckingExistingUser(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedParticipantJoinedEvent(t, eventStore, storage.ParticipantRecord{
		ID:         "participant-2",
		CampaignID: "campaign-1",
		UserID:     "user-1",
	}, now)
	seedParticipantUnboundEvent(t, eventStore, "campaign-1", "participant-2", "user-1", now.Add(time.Second))
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	resp, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err != nil {
		t.Fatalf("ClaimInvite returned error: %v", err)
	}
	if resp.Participant.GetUserId() != "user-1" {
		t.Fatalf("participant user_id = %q, want %q", resp.Participant.GetUserId(), "user-1")
	}
	if len(eventStore.Events["campaign-1"]) != 6 {
		t.Fatalf("event count = %d, want 6", len(eventStore.Events["campaign-1"]))
	}
}

func TestClaimInvite_RejectsAIControlledSeatBinding(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {
			ID:             "participant-1",
			CampaignID:     "campaign-1",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)

	if len(eventStore.Events["campaign-1"]) != 2 {
		t.Fatalf("event count = %d, want 2", len(eventStore.Events["campaign-1"]))
	}
}

func TestClaimInvite_MissingUserID(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.bind"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("participant.bound"),
				Timestamp:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-1",
				PayloadJSON: []byte(`{"participant_id":"participant-1","user_id":"user-1"}`),
			}),
		},
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		nil,
		nil,
		nil,
	)

	_, err := svc.ClaimInvite(context.Background(), &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  "grant",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_IdempotentGrant(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "jti-1", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
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

	if len(eventStore.Events["campaign-1"]) != 4 {
		t.Fatalf("event count = %d, want 4", len(eventStore.Events["campaign-1"]))
	}
	boundCount := 0
	claimedCount := 0
	for _, evt := range eventStore.Events["campaign-1"] {
		if evt.Type == participant.EventTypeBound {
			boundCount++
		}
		if evt.Type == invite.EventTypeClaimed {
			claimedCount++
		}
	}
	if boundCount != 1 {
		t.Fatalf("participant.bound count = %d, want 1", boundCount)
	}
	if claimedCount != 1 {
		t.Fatalf("invite.claimed count = %d, want 1", claimedCount)
	}
}

func TestClaimInvite_UserAlreadyClaimed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
		"participant-2": {ID: "participant-2", CampaignID: "campaign-1", UserID: "user-1"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-2"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "jti-2", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	assertStatusCode(t, err, codes.AlreadyExists)
	if len(eventStore.Events["campaign-1"]) != 3 {
		t.Fatalf("event count = %d, want 3", len(eventStore.Events["campaign-1"]))
	}
}

func TestClaimInvite_HydratesParticipantFromSocialProfile(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		Name:          "Ariadne",
		Pronouns:      sharedpronouns.ToProto("she/her"),
		AvatarSetId:   "avatar-set-1",
		AvatarAssetId: "avatar-asset-1",
	}}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1", Name: "Pending Seat"},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-1",
				PayloadJSON: []byte(`{"participant_id":"participant-1","fields":{"name":"Ariadne","pronouns":"she/her","avatar_set_id":"avatar-set-1","avatar_asset_id":"avatar-asset-1"}}`),
			}),
		},
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Social:      socialClient,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	resp, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	if err != nil {
		t.Fatalf("ClaimInvite returned error: %v", err)
	}
	if resp.Participant.GetName() != "Ariadne" {
		t.Fatalf("participant name = %q, want %q", resp.Participant.GetName(), "Ariadne")
	}
	if resp.Participant.GetAvatarSetId() != "avatar-set-1" {
		t.Fatalf("participant avatar set = %q, want %q", resp.Participant.GetAvatarSetId(), "avatar-set-1")
	}
	if resp.Participant.GetAvatarAssetId() != "avatar-asset-1" {
		t.Fatalf("participant avatar asset = %q, want %q", resp.Participant.GetAvatarAssetId(), "avatar-asset-1")
	}
	if socialClient.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfile calls = %d, want 1", socialClient.GetUserProfileCalls)
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
	if len(domain.commands) != 1 || domain.commands[0].Type != command.Type("participant.update") {
		t.Fatalf("commands = %#v, want participant.update only", domain.commands)
	}
}

func TestClaimInvite_ResyncsControlledCharacterAvatarFromClaimedSeat(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		Name:          "Ariadne",
		Pronouns:      sharedpronouns.ToProto("she/her"),
		AvatarSetId:   "avatar-set-1",
		AvatarAssetId: "avatar-asset-1",
	}}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1", Name: "Pending Seat"},
	}
	characterStore.Characters["campaign-1"] = map[string]storage.CharacterRecord{
		"char-1": {
			ID:            "char-1",
			CampaignID:    "campaign-1",
			ParticipantID: "participant-1",
			Name:          "Hero",
			AvatarSetID:   "old-set",
			AvatarAssetID: "old-asset",
			Pronouns:      "xe/xem",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	seedParticipantJoinedEvent(t, eventStore, participantStore.Participants["campaign-1"]["participant-1"], now)
	seedInviteCreatedEvent(t, eventStore, inviteStore.Invites["invite-1"], now)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-1",
				PayloadJSON: []byte(`{"participant_id":"participant-1","fields":{"name":"Ariadne","pronouns":"she/her","avatar_set_id":"avatar-set-1","avatar_asset_id":"avatar-asset-1"}}`),
			}),
		},
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","fields":{"avatar_set_id":"avatar-set-1","avatar_asset_id":"avatar-asset-1"}}`),
			}),
		},
	}}

	svc := newServiceWithDependencies(
		Deps{
			Campaign:    campaignStore,
			Participant: participantStore,
			Character:   characterStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Social:      socialClient,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	signer := gametest.NewJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.app.clock())
	ctx := gametest.ContextWithUserID("user-1")
	if _, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	}); err != nil {
		t.Fatalf("ClaimInvite returned error: %v", err)
	}

	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.update") {
		t.Fatalf("command[0] type = %s, want participant.update", domain.commands[0].Type)
	}
	if domain.commands[1].Type != command.Type("character.update") {
		t.Fatalf("command[1] type = %s, want character.update", domain.commands[1].Type)
	}

	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode character update payload: %v", err)
	}
	if payload.Fields["avatar_set_id"] != "avatar-set-1" {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], "avatar-set-1")
	}
	if payload.Fields["avatar_asset_id"] != "avatar-asset-1" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "avatar-asset-1")
	}
	if _, ok := payload.Fields["pronouns"]; ok {
		t.Fatalf("pronouns field should be omitted, got %q", payload.Fields["pronouns"])
	}

	updated, err := characterStore.GetCharacter(context.Background(), "campaign-1", "char-1")
	if err != nil {
		t.Fatalf("load updated character: %v", err)
	}
	if updated.AvatarSetID != "avatar-set-1" || updated.AvatarAssetID != "avatar-asset-1" {
		t.Fatalf("character avatar = %q/%q, want avatar-set-1/avatar-asset-1", updated.AvatarSetID, updated.AvatarAssetID)
	}
	if updated.Pronouns != "xe/xem" {
		t.Fatalf("character pronouns = %q, want %q", updated.Pronouns, "xe/xem")
	}
}

func TestClaimInvite_NilRequest(t *testing.T) {
	svc := NewService(Deps{}, nil)
	_, err := svc.ClaimInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Invite:      gametest.NewFakeInviteStore(),
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}, nil)
	ctx := gametest.ContextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{InviteId: "inv-1", JoinGrant: "grant"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_MissingInviteId(t *testing.T) {
	svc := NewService(Deps{
		Invite:      gametest.NewFakeInviteStore(),
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}, nil)
	ctx := gametest.ContextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{CampaignId: "c1", JoinGrant: "grant"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_MissingJoinGrant(t *testing.T) {
	svc := NewService(Deps{
		Invite:      gametest.NewFakeInviteStore(),
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}, nil)
	ctx := gametest.ContextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{CampaignId: "c1", InviteId: "inv-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}
