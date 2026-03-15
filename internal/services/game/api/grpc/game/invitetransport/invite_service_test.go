package invitetransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testRuntime is a shared write-path runtime configured once for all tests.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = gametest.SetupRuntime()
	os.Exit(m.Run())
}

// assertStatusCode verifies the gRPC status code for an error.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		err = grpcerror.HandleDomainError(err)
		statusErr, ok = status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T", err)
		}
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}

type fakeDomainEngine struct {
	store         storage.EventStore
	result        engine.Result
	resultsByType map[command.Type]engine.Result
	calls         int
	lastCommand   command.Command
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.lastCommand = cmd
	f.commands = append(f.commands, cmd)

	result := f.result
	if len(f.resultsByType) > 0 {
		if selected, ok := f.resultsByType[cmd.Type]; ok {
			result = selected
		}
	}
	if f.store == nil {
		return result, nil
	}
	if len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := f.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

type eventAppender interface {
	AppendEvent(context.Context, event.Event) (event.Event, error)
}

func seedParticipantJoinedEvent(t *testing.T, store eventAppender, record storage.ParticipantRecord, stamp time.Time) {
	t.Helper()

	role := record.Role
	if role == "" {
		role = participant.RolePlayer
	}
	controller := record.Controller
	if controller == "" {
		controller = participant.ControllerHuman
	}
	access := record.CampaignAccess
	if access == "" {
		access = participant.CampaignAccessMember
	}
	name := record.Name
	if name == "" {
		name = record.ID
	}
	payloadJSON, err := json.Marshal(participant.JoinPayload{
		ParticipantID:  ids.ParticipantID(record.ID),
		UserID:         ids.UserID(record.UserID),
		Name:           name,
		Role:           string(role),
		Controller:     string(controller),
		CampaignAccess: string(access),
		AvatarSetID:    record.AvatarSetID,
		AvatarAssetID:  record.AvatarAssetID,
		Pronouns:       record.Pronouns,
	})
	if err != nil {
		t.Fatalf("marshal participant join payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(record.CampaignID),
		Type:        participant.EventTypeJoined,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    record.ID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append participant join event: %v", err)
	}
}

func seedInviteCreatedEvent(t *testing.T, store eventAppender, record storage.InviteRecord, stamp time.Time) {
	t.Helper()

	status := record.Status
	if status == invite.StatusUnspecified {
		status = invite.StatusPending
	}
	payloadJSON, err := json.Marshal(invite.CreatePayload{
		InviteID:               ids.InviteID(record.ID),
		ParticipantID:          ids.ParticipantID(record.ParticipantID),
		RecipientUserID:        ids.UserID(record.RecipientUserID),
		CreatedByParticipantID: ids.ParticipantID(record.CreatedByParticipantID),
		Status:                 string(status),
	})
	if err != nil {
		t.Fatalf("marshal invite create payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(record.CampaignID),
		Type:        invite.EventTypeCreated,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "invite",
		EntityID:    record.ID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append invite create event: %v", err)
	}
}

func seedParticipantLeftEvent(t *testing.T, store eventAppender, campaignID string, participantID string, stamp time.Time) {
	t.Helper()

	payloadJSON, err := json.Marshal(participant.LeavePayload{
		ParticipantID: ids.ParticipantID(participantID),
	})
	if err != nil {
		t.Fatalf("marshal participant leave payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Type:        participant.EventTypeLeft,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    participantID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append participant left event: %v", err)
	}
}

func seedParticipantUnboundEvent(t *testing.T, store eventAppender, campaignID string, participantID string, userID string, stamp time.Time) {
	t.Helper()

	payloadJSON, err := json.Marshal(participant.UnbindPayload{
		ParticipantID: ids.ParticipantID(participantID),
		UserID:        ids.UserID(userID),
	})
	if err != nil {
		t.Fatalf("marshal participant unbind payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Type:        participant.EventTypeUnbound,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    participantID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append participant unbound event: %v", err)
	}
}

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
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-2"}}

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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-2"}}

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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-2"}}
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
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
	authClient := &gametest.FakeAuthClient{User: &authv1.User{Id: "user-2"}}
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
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
	socialClient := &gametest.FakeSocialClient{Profile: &socialv1.UserProfile{
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
	socialClient := &gametest.FakeSocialClient{Profile: &socialv1.UserProfile{
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

func TestRevokeInvite_Success(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	eventStore := gametest.NewFakeEventStore()
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

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: eventStore, Write: domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime}},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	if len(eventStore.Events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.Events["campaign-1"]))
	}
	if eventStore.Events["campaign-1"][0].Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, event.Type("invite.revoked"))
	}
}

func TestRevokeInvite_RequiresDomainEngine(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
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
	assertStatusCode(t, err, codes.Internal)
}

func TestRevokeInvite_UsesDomainEngine(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	eventStore := gametest.NewFakeEventStore()
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

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Invite:      inviteStore,
			Participant: participantStore,
			Campaign:    campaignStore,
			Event:       eventStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	if len(eventStore.Events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.Events["campaign-1"]))
	}
	if eventStore.Events["campaign-1"][0].Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, event.Type("invite.revoked"))
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

func TestListPendingInvites_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner, UserID: "user-1"},
		"seat-1":  {ID: "seat-1", CampaignID: "campaign-1", Name: "Seat 1", Role: participant.RolePlayer},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:                     "invite-1",
		CampaignID:             "campaign-1",
		ParticipantID:          "seat-1",
		Status:                 invite.StatusPending,
		CreatedByParticipantID: "owner-1",
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:            "invite-2",
		CampaignID:    "campaign-1",
		ParticipantID: "seat-1",
		Status:        invite.StatusClaimed,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		&gametest.FakeAuthClient{User: &authv1.User{Id: "user-1", Username: "owner"}},
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	svc := NewService(Deps{}, nil)
	_, err := svc.GetInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetInvite_MissingInviteId(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetInvite_InviteNotFound(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{InviteId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetInvite_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	campaignStore := gametest.NewFakeCampaignStore()
	inviteStore := gametest.NewFakeInviteStore()
	participantStore := gametest.NewFakeParticipantStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListInvites_NilRequest(t *testing.T) {
	svc := NewService(Deps{}, nil)
	_, err := svc.ListInvites(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.ListInvites(context.Background(), &statev1.ListInvitesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.ListInvites(context.Background(), &statev1.ListInvitesRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListInvites_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:         "invite-2",
		CampaignID: "campaign-1",
		Status:     invite.StatusClaimed,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 2 {
		t.Fatalf("invite count = %d, want 2", len(resp.Invites))
	}
}

func TestListInvites_WithStatusFilter(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:         "invite-2",
		CampaignID: "campaign-1",
		Status:     invite.StatusClaimed,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
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
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 0 {
		t.Fatalf("invite count = %d, want 0", len(resp.Invites))
	}
}

func TestListPendingInvitesForUser_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {ID: "seat-1", CampaignID: "campaign-1", Name: "Seat 1", Role: participant.RolePlayer},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:              "invite-1",
		CampaignID:      "campaign-1",
		ParticipantID:   "seat-1",
		RecipientUserID: "user-1",
		Status:          invite.StatusPending,
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:              "invite-2",
		CampaignID:      "campaign-1",
		ParticipantID:   "seat-1",
		RecipientUserID: "user-2",
		Status:          invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithUserID("user-1")
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
