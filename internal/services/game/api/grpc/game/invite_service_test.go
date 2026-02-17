package game

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestCreateInvite_NilRequest(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.CreateInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateInvite_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	svc := &InviteService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		clock:       fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		idGenerator: fixedIDGenerator("invite-123"),
	}

	ctx := contextWithParticipantID("owner-1")
	_, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateInvite_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "invite",
			EntityID:    "invite-123",
			PayloadJSON: []byte(`{"invite_id":"invite-123","participant_id":"participant-1","status":"pending","created_by_participant_id":"owner-1"}`),
		}),
	}}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
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
	if eventStore.events["campaign-1"][0].Type != event.Type("invite.created") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.Type("invite.created"))
	}
}

func TestCreateInvite_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "invite",
			EntityID:    "invite-123",
			PayloadJSON: []byte(`{"invite_id":"invite-123","participant_id":"participant-1","status":"pending","created_by_participant_id":"owner-1"}`),
		}),
	}}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
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
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("invite.create") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "invite.create")
	}
	if domain.commands[0].EntityID != "invite-123" {
		t.Fatalf("command entity id = %s, want %s", domain.commands[0].EntityID, "invite-123")
	}
	if len(eventStore.events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.events["campaign-1"]))
	}
	if eventStore.events["campaign-1"][0].Type != event.Type("invite.created") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.Type("invite.created"))
	}
}

func TestRevokeInvite_AlreadyClaimed(t *testing.T) {
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()
	inviteStore.invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusClaimed}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
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

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
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

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.bind"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("participant.bound"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-1",
				PayloadJSON: []byte(`{"participant_id":"participant-1","user_id":"user-1"}`),
			}),
		},
		command.Type("invite.claim"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("invite.claimed"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "invite",
				EntityID:    "invite-1",
				PayloadJSON: []byte(`{"invite_id":"invite-1","participant_id":"participant-1","user_id":"user-1","jti":"jwt-1"}`),
			}),
		},
	}}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
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
	if eventStore.events["campaign-1"][0].Type != event.Type("participant.bound") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.Type("participant.bound"))
	}
	if eventStore.events["campaign-1"][1].Type != event.Type("invite.claimed") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][1].Type, event.Type("invite.claimed"))
	}
}

func TestClaimInvite_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
		},
		clock: fixedClock(now),
	}

	signer := newJoinGrantSigner(t)
	joinGrant := signer.Token(t, "campaign-1", "invite-1", "user-1", "", svc.clock())
	ctx := contextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: "campaign-1",
		InviteId:   "invite-1",
		JoinGrant:  joinGrant,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestClaimInvite_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
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
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-1",
				PayloadJSON: []byte(`{"participant_id":"participant-1","user_id":"user-1"}`),
			}),
		},
		command.Type("invite.claim"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("invite.claimed"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "invite",
				EntityID:    "invite-1",
				PayloadJSON: []byte(`{"invite_id":"invite-1","participant_id":"participant-1","user_id":"user-1","jti":"jwt-1"}`),
			}),
		},
	}}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
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
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.bind") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "participant.bind")
	}
	if domain.commands[1].Type != command.Type("invite.claim") {
		t.Fatalf("command type = %s, want %s", domain.commands[1].Type, "invite.claim")
	}
	if len(eventStore.events["campaign-1"]) != 2 {
		t.Fatalf("event count = %d, want 2", len(eventStore.events["campaign-1"]))
	}
	if eventStore.events["campaign-1"][0].Type != event.Type("participant.bound") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.Type("participant.bound"))
	}
	if eventStore.events["campaign-1"][1].Type != event.Type("invite.claimed") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][1].Type, event.Type("invite.claimed"))
	}
}

func TestClaimInvite_MissingUserID(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
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

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
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

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:            "invite-1",
		CampaignID:    "campaign-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.bind"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("participant.bound"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "participant-1",
				PayloadJSON: []byte(`{"participant_id":"participant-1","user_id":"user-1"}`),
			}),
		},
		command.Type("invite.claim"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "campaign-1",
				Type:        event.Type("invite.claimed"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "invite",
				EntityID:    "invite-1",
				PayloadJSON: []byte(`{"invite_id":"invite-1","participant_id":"participant-1","user_id":"user-1","jti":"jti-1"}`),
			}),
		},
	}}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
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

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
		"participant-2": {ID: "participant-2", CampaignID: "campaign-1", UserID: "user-1"},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
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

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Domain:      domain,
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

func TestRevokeInvite_NilRequest(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.RevokeInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeInvite_MissingInviteId(t *testing.T) {
	svc := NewInviteService(Stores{Invite: newFakeInviteStore(), Campaign: newFakeCampaignStore(), Event: newFakeEventStore()})
	_, err := svc.RevokeInvite(context.Background(), &statev1.RevokeInviteRequest{InviteId: ""})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeInvite_InviteNotFound(t *testing.T) {
	svc := NewInviteService(Stores{Invite: newFakeInviteStore(), Campaign: newFakeCampaignStore(), Event: newFakeEventStore()})
	_, err := svc.RevokeInvite(context.Background(), &statev1.RevokeInviteRequest{InviteId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRevokeInvite_AlreadyRevoked(t *testing.T) {
	inviteStore := newFakeInviteStore()
	inviteStore.invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusRevoked}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := &InviteService{
		stores:      Stores{Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: newFakeEventStore()},
		clock:       fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		idGenerator: fixedIDGenerator("x"),
	}

	ctx := contextWithParticipantID("owner-1")
	_, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRevokeInvite_Success(t *testing.T) {
	inviteStore := newFakeInviteStore()
	inviteStore.invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.revoked"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "invite",
			EntityID:    "invite-1",
			PayloadJSON: []byte(`{"invite_id":"invite-1"}`),
		}),
	}}

	svc := &InviteService{
		stores:      Stores{Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("x"),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	if err != nil {
		t.Fatalf("RevokeInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("RevokeInvite response has nil invite")
	}
	if resp.Invite.Status != statev1.InviteStatus_REVOKED {
		t.Fatalf("invite status = %v, want REVOKED", resp.Invite.Status)
	}
	if len(eventStore.events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.events["campaign-1"]))
	}
	if eventStore.events["campaign-1"][0].Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.Type("invite.revoked"))
	}
}

func TestRevokeInvite_RequiresDomainEngine(t *testing.T) {
	inviteStore := newFakeInviteStore()
	inviteStore.invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := &InviteService{
		stores:      Stores{Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: newFakeEventStore()},
		clock:       fixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		idGenerator: fixedIDGenerator("x"),
	}

	ctx := contextWithParticipantID("owner-1")
	_, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestRevokeInvite_UsesDomainEngine(t *testing.T) {
	inviteStore := newFakeInviteStore()
	inviteStore.invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore := newFakeParticipantStore()
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.revoked"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "invite",
			EntityID:    "invite-1",
			PayloadJSON: []byte(`{"invite_id":"invite-1"}`),
		}),
	}}

	svc := &InviteService{
		stores: Stores{
			Invite:      inviteStore,
			Participant: participantStore,
			Campaign:    campaignStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("x"),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	if err != nil {
		t.Fatalf("RevokeInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("RevokeInvite response has nil invite")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("invite.revoke") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "invite.revoke")
	}
	if len(eventStore.events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.events["campaign-1"]))
	}
	if eventStore.events["campaign-1"][0].Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", eventStore.events["campaign-1"][0].Type, event.Type("invite.revoked"))
	}
}

func TestClaimInvite_NilRequest(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.ClaimInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_MissingCampaignId(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:      newFakeInviteStore(),
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Event:       newFakeEventStore(),
	})
	ctx := contextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{InviteId: "inv-1", JoinGrant: "grant"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_MissingInviteId(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:      newFakeInviteStore(),
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Event:       newFakeEventStore(),
	})
	ctx := contextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{CampaignId: "c1", JoinGrant: "grant"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_MissingJoinGrant(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:      newFakeInviteStore(),
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Event:       newFakeEventStore(),
	})
	ctx := contextWithUserID("user-1")
	_, err := svc.ClaimInvite(ctx, &statev1.ClaimInviteRequest{CampaignId: "c1", InviteId: "inv-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateInvite_MissingCampaignId(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:      newFakeInviteStore(),
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{ParticipantId: "p1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateInvite_MissingParticipantId(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:      newFakeInviteStore(),
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListPendingInvites_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner, UserID: "user-1"},
		"seat-1":  {ID: "seat-1", CampaignID: "campaign-1", Name: "Seat 1", Role: participant.RolePlayer},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:                     "invite-1",
		CampaignID:             "campaign-1",
		ParticipantID:          "seat-1",
		Status:                 invite.StatusPending,
		CreatedByParticipantID: "owner-1",
	}
	inviteStore.invites["invite-2"] = storage.InviteRecord{
		ID:            "invite-2",
		CampaignID:    "campaign-1",
		ParticipantID: "seat-1",
		Status:        invite.StatusClaimed,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		authClient: &fakeAuthClient{user: &authv1.User{Id: "user-1", Username: "owner"}},
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.ListPendingInvites(ctx, &statev1.ListPendingInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListPendingInvites returned error: %v", err)
	}
	if len(resp.Invites) != 1 {
		t.Fatalf("pending invite count = %d, want 1", len(resp.Invites))
	}
	entry := resp.Invites[0]
	if entry.Invite.Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", entry.Invite.Id)
	}
	if entry.Participant == nil || entry.Participant.Id != "seat-1" {
		t.Fatalf("participant id = %v, want seat-1", entry.Participant)
	}
	if entry.CreatedByUser == nil || entry.CreatedByUser.Id != "user-1" {
		t.Fatalf("created_by_user id = %v, want user-1", entry.CreatedByUser)
	}
}

func TestGetInvite_NilRequest(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.GetInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetInvite_MissingInviteId(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:   newFakeInviteStore(),
		Campaign: newFakeCampaignStore(),
	})
	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetInvite_InviteNotFound(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:   newFakeInviteStore(),
		Campaign: newFakeCampaignStore(),
	})
	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{InviteId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetInvite_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.GetInvite(ctx, &statev1.GetInviteRequest{InviteId: "invite-1"})
	if err != nil {
		t.Fatalf("GetInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("GetInvite response has nil invite")
	}
	if resp.Invite.Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", resp.Invite.Id)
	}
}

func TestGetInvite_MissingParticipantIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	inviteStore := newFakeInviteStore()
	participantStore := newFakeParticipantStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
	}

	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListInvites_NilRequest(t *testing.T) {
	svc := NewInviteService(Stores{})
	_, err := svc.ListInvites(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_MissingCampaignId(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:   newFakeInviteStore(),
		Campaign: newFakeCampaignStore(),
	})
	_, err := svc.ListInvites(context.Background(), &statev1.ListInvitesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_CampaignNotFound(t *testing.T) {
	svc := NewInviteService(Stores{
		Invite:   newFakeInviteStore(),
		Campaign: newFakeCampaignStore(),
	})
	_, err := svc.ListInvites(context.Background(), &statev1.ListInvitesRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListInvites_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}
	inviteStore.invites["invite-2"] = storage.InviteRecord{
		ID:         "invite-2",
		CampaignID: "campaign-1",
		Status:     invite.StatusClaimed,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 2 {
		t.Fatalf("invite count = %d, want 2", len(resp.Invites))
	}
}

func TestListInvites_WithStatusFilter(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}
	inviteStore.invites["invite-2"] = storage.InviteRecord{
		ID:         "invite-2",
		CampaignID: "campaign-1",
		Status:     invite.StatusClaimed,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: "campaign-1",
		Status:     statev1.InviteStatus_PENDING,
	})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 1 {
		t.Fatalf("invite count = %d, want 1", len(resp.Invites))
	}
	if resp.Invites[0].Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", resp.Invites[0].Id)
	}
}

func TestListInvites_EmptyResult(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 0 {
		t.Fatalf("invite count = %d, want 0", len(resp.Invites))
	}
}

func TestListPendingInvitesForUser_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	inviteStore := newFakeInviteStore()

	campaignStore.campaigns["campaign-1"] = storage.CampaignRecord{ID: "campaign-1", Status: campaign.StatusDraft}
	participantStore.participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {ID: "seat-1", CampaignID: "campaign-1", Name: "Seat 1", Role: participant.RolePlayer},
	}
	inviteStore.invites["invite-1"] = storage.InviteRecord{
		ID:              "invite-1",
		CampaignID:      "campaign-1",
		ParticipantID:   "seat-1",
		RecipientUserID: "user-1",
		Status:          invite.StatusPending,
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	inviteStore.invites["invite-2"] = storage.InviteRecord{
		ID:              "invite-2",
		CampaignID:      "campaign-1",
		ParticipantID:   "seat-1",
		RecipientUserID: "user-2",
		Status:          invite.StatusPending,
	}

	svc := &InviteService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
	}

	ctx := contextWithUserID("user-1")
	resp, err := svc.ListPendingInvitesForUser(ctx, &statev1.ListPendingInvitesForUserRequest{})
	if err != nil {
		t.Fatalf("ListPendingInvitesForUser returned error: %v", err)
	}
	if len(resp.Invites) != 1 {
		t.Fatalf("pending invite count = %d, want 1", len(resp.Invites))
	}
	entry := resp.Invites[0]
	if entry.Invite.Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", entry.Invite.Id)
	}
	if entry.Campaign == nil || entry.Campaign.Id != "campaign-1" {
		t.Fatalf("campaign id = %v, want campaign-1", entry.Campaign)
	}
	if entry.Participant == nil || entry.Participant.Id != "seat-1" {
		t.Fatalf("participant id = %v, want seat-1", entry.Participant)
	}
}
