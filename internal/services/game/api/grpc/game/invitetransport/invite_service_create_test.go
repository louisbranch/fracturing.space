package invitetransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/testclients"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestCreateInvite_NilRequest(t *testing.T) {
	svc := NewService(Deps{}, nil)
	_, err := svc.CreateInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateInvite_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateInvite_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
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

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	if eventStore.Events["campaign-1"][0].Type != event.Type("invite.created") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, event.Type("invite.created"))
	}
}

func TestCreateInvite_UsesResolvedActorParticipantIDWhenOnlyUserIdentityIsPresent(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {
			ID:             "owner-1",
			CampaignID:     "campaign-1",
			UserID:         "user-1",
			CampaignAccess: participant.CampaignAccessOwner,
		},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "invite",
			EntityID:    "invite-123",
			PayloadJSON: []byte(`{"invite_id":"invite-123","participant_id":"participant-1","status":"pending","created_by_participant_id":"owner-1"}`),
		}),
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithUserID("user-1")
	if _, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	}); err != nil {
		t.Fatalf("CreateInvite returned error: %v", err)
	}

	var payload invite.CreatePayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal domain command payload: %v", err)
	}
	if payload.CreatedByParticipantID != ids.ParticipantID("owner-1") {
		t.Fatalf("created_by_participant_id = %q, want %q", payload.CreatedByParticipantID, "owner-1")
	}
}

func TestCreateInvite_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
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

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	if len(eventStore.Events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.Events["campaign-1"]))
	}
	if eventStore.Events["campaign-1"][0].Type != event.Type("invite.created") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, event.Type("invite.created"))
	}
}

func TestCreateInvite_PersistsCreatorFromResolvedUserActor(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", UserID: "user-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "invite",
			EntityID:    "invite-123",
			PayloadJSON: []byte(`{"invite_id":"invite-123","participant_id":"participant-1","status":"pending","created_by_participant_id":"owner-1"}`),
		}),
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	resp, err := svc.CreateInvite(gametest.ContextWithUserID("user-1"), &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	})
	if err != nil {
		t.Fatalf("CreateInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("CreateInvite response has nil invite")
	}
	if got := resp.Invite.GetCreatedByParticipantId(); got != "owner-1" {
		t.Fatalf("created by participant id = %q, want %q", got, "owner-1")
	}

	var payload invite.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal command payload: %v", err)
	}
	if got := payload.CreatedByParticipantID.String(); got != "owner-1" {
		t.Fatalf("command payload created_by_participant_id = %q, want %q", got, "owner-1")
	}
}

func TestCreateInvite_RecipientAlreadyParticipant(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-2"}}

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
		"participant-2": {ID: "participant-2", CampaignID: "campaign-1", UserID: "user-2"},
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("invite-123"),
		authClient,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      "campaign-1",
		ParticipantId:   "participant-1",
		RecipientUserId: "user-2",
	})
	assertStatusCode(t, err, codes.AlreadyExists)

	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-2" {
		t.Fatalf("GetUser request = %#v, want user-2", authClient.LastGetUserRequest)
	}
	if participantStore.ListCampaignIDsByUserCalls != 1 {
		t.Fatalf("ListCampaignIDsByUser calls = %d, want 1", participantStore.ListCampaignIDsByUserCalls)
	}
	if len(eventStore.Events["campaign-1"]) != 0 {
		t.Fatalf("event count = %d, want 0", len(eventStore.Events["campaign-1"]))
	}
}

func TestCreateInvite_RecipientAlreadyHasPendingInvite(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-2"}}

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
		"participant-2": {ID: "participant-2", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-existing"] = storage.InviteRecord{
		ID:              "invite-existing",
		CampaignID:      "campaign-1",
		ParticipantID:   "participant-2",
		RecipientUserID: "user-2",
		Status:          invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("invite-123"),
		authClient,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      "campaign-1",
		ParticipantId:   "participant-1",
		RecipientUserId: "user-2",
	})
	assertStatusCode(t, err, codes.AlreadyExists)

	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-2" {
		t.Fatalf("GetUser request = %#v, want user-2", authClient.LastGetUserRequest)
	}
	if participantStore.ListCampaignIDsByUserCalls != 1 {
		t.Fatalf("ListCampaignIDsByUser calls = %d, want 1", participantStore.ListCampaignIDsByUserCalls)
	}
	if len(eventStore.Events["campaign-1"]) != 0 {
		t.Fatalf("event count = %d, want 0", len(eventStore.Events["campaign-1"]))
	}
}

func TestCreateInvite_RecipientParticipantInOtherCampaignDoesNotBlock(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-2"}}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	participantStore.Participants["campaign-2"] = map[string]storage.ParticipantRecord{
		"participant-2": {ID: "participant-2", CampaignID: "campaign-2", UserID: "user-2"},
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
			PayloadJSON: []byte(`{"invite_id":"invite-123","participant_id":"participant-1","recipient_user_id":"user-2","status":"pending","created_by_participant_id":"owner-1"}`),
		}),
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("invite-123"),
		authClient,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      "campaign-1",
		ParticipantId:   "participant-1",
		RecipientUserId: "user-2",
	})
	if err != nil {
		t.Fatalf("CreateInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("CreateInvite response has nil invite")
	}
	if resp.Invite.GetRecipientUserId() != "user-2" {
		t.Fatalf("recipient user id = %q, want user-2", resp.Invite.GetRecipientUserId())
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
}

func TestCreateInvite_RecipientPendingInviteInOtherCampaignDoesNotBlock(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-2"}}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}
	inviteStore.Invites["invite-other"] = storage.InviteRecord{
		ID:              "invite-other",
		CampaignID:      "campaign-2",
		ParticipantID:   "participant-2",
		RecipientUserID: "user-2",
		Status:          invite.StatusPending,
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
			PayloadJSON: []byte(`{"invite_id":"invite-123","participant_id":"participant-1","recipient_user_id":"user-2","status":"pending","created_by_participant_id":"owner-1"}`),
		}),
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
			Event:       eventStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("invite-123"),
		authClient,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      "campaign-1",
		ParticipantId:   "participant-1",
		RecipientUserId: "user-2",
	})
	if err != nil {
		t.Fatalf("CreateInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("CreateInvite response has nil invite")
	}
	if resp.Invite.GetRecipientUserId() != "user-2" {
		t.Fatalf("recipient user id = %q, want user-2", resp.Invite.GetRecipientUserId())
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
}

func TestCreateInvite_MissingParticipantIdentity(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()
	eventStore := gametest.NewFakeBatchEventStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1":       {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
		"participant-1": {ID: "participant-1", CampaignID: "campaign-1"},
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Invite: inviteStore, Event: eventStore},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("invite-123"),
		nil,
		nil,
	)

	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{
		CampaignId:    "campaign-1",
		ParticipantId: "participant-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateInvite_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Invite:      gametest.NewFakeInviteStore(),
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}, nil)
	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{ParticipantId: "p1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateInvite_MissingParticipantId(t *testing.T) {
	svc := NewService(Deps{
		Invite:      gametest.NewFakeInviteStore(),
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}, nil)
	_, err := svc.CreateInvite(context.Background(), &statev1.CreateInviteRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}
