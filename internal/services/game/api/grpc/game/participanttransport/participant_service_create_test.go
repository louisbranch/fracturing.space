package participanttransport

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
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
)

// newParticipantServiceForTest is a convenience wrapper matching the old test helper signature.
func newParticipantServiceForTest(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
) *Service {
	return newServiceWithDependencies(deps, clock, idGenerator, authClient)
}

func TestCreateParticipant_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.CreateParticipant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		Name: "Player 1",
		Role: statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		CampaignId: "nonexistent",
		Name:       "Player 1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateParticipant_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Player 1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateParticipant_EmptyName(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.DraftCampaignRecord("c1")

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_InvalidRole(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.DraftCampaignRecord("c1")

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Player 1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_DomainRejectsAIInvariant(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.DraftCampaignRecord("c1")
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Reject(command.Rejection{
				Code:    "PARTICIPANT_AI_ROLE_REQUIRED",
				Message: "ai-controlled participants must use gm role",
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "AI Seat",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_AI,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestCreateParticipant_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.DraftCampaignRecord("c1")

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Game Master",
		Role:       statev1.ParticipantRole_GM,
		Controller: statev1.Controller_CONTROLLER_AI,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateParticipant_Success_GM(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.DraftCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"","name":"Game Master","role":"GM","controller":"AI","campaign_access":"MANAGER"}`),
			}),
		},
	}}
	svc := newParticipantServiceForTest(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Game Master",
		Role:       statev1.ParticipantRole_GM,
		Controller: statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("CreateParticipant response has nil participant")
	}
	if resp.Participant.Id != "participant-123" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "participant-123")
	}
	if resp.Participant.Name != "Game Master" {
		t.Errorf("Participant Name = %q, want %q", resp.Participant.Name, "Game Master")
	}
	if resp.Participant.Role != statev1.ParticipantRole_GM {
		t.Errorf("Participant Role = %v, want %v", resp.Participant.Role, statev1.ParticipantRole_GM)
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.joined"))
	}

	// Verify persisted
	stored, err := participantStore.GetParticipant(context.Background(), "c1", "participant-123")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.Name != "Game Master" {
		t.Errorf("Stored participant Name = %q, want %q", stored.Name, "Game Master")
	}
}

func TestCreateParticipant_Success_Player(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-456",
				PayloadJSON: []byte(`{"participant_id":"participant-456","user_id":"","name":"Player One","role":"PLAYER","controller":"HUMAN","campaign_access":"MEMBER"}`),
			}),
		},
	}}
	svc := newParticipantServiceForTest(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-456"),
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Player One",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if resp.Participant.Role != statev1.ParticipantRole_PLAYER {
		t.Errorf("Participant Role = %v, want %v", resp.Participant.Role, statev1.ParticipantRole_PLAYER)
	}
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER)
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.joined"))
	}
}

func TestCreateParticipant_Success_ManagerAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-manager",
				PayloadJSON: []byte(`{"participant_id":"participant-manager","user_id":"","name":"Quartermaster","role":"PLAYER","controller":"HUMAN","campaign_access":"MANAGER"}`),
			}),
		},
	}}
	svc := newParticipantServiceForTest(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-manager"),
		nil,
	)

	resp, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId:     "c1",
		Name:           "Quartermaster",
		Role:           statev1.ParticipantRole_PLAYER,
		Controller:     statev1.Controller_CONTROLLER_HUMAN,
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if resp.Participant.GetCampaignAccess() != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatalf("Participant CampaignAccess = %v, want %v", resp.Participant.GetCampaignAccess(), statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}
}

func TestCreateParticipant_DeniesManagerAssigningOwnerAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":   gametest.OwnerParticipantRecord("c1", "owner-1"),
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("manager-1"), &statev1.CreateParticipantRequest{
		CampaignId:     "c1",
		Name:           "Pending Owner",
		Role:           statev1.ParticipantRole_PLAYER,
		Controller:     statev1.Controller_CONTROLLER_HUMAN,
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateParticipant_DeniesHumanGMForAIGMCampaign(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		GmMode: campaign.GmModeAI,
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Human GM",
		Role:       statev1.ParticipantRole_GM,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Player One","role":"player","controller":"human","campaign_access":"member"}`),
			}),
		},
	}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Player One",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
		UserId:     "user-123",
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("CreateParticipant response has nil participant")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.join") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "participant.join")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.joined"))
	}
}

func TestCreateParticipant_UserLinkedRequestFieldsTakePrecedenceOverSocial(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Request Name","role":"player","controller":"human","campaign_access":"member","avatar_set_id":"people-v1","avatar_asset_id":"request-avatar","pronouns":"request-pronouns"}`),
			}),
		},
	}}
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-123",
		Name:          "Social Name",
		Pronouns:      sharedpronouns.ToProto("social-pronouns"),
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
			Social:      socialClient,
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		nil,
	)

	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId:    "c1",
		UserId:        "user-123",
		Name:          "Request Name",
		Role:          statev1.ParticipantRole_PLAYER,
		Controller:    statev1.Controller_CONTROLLER_HUMAN,
		AvatarSetId:   "people-v1",
		AvatarAssetId: "request-avatar",
		Pronouns:      sharedpronouns.ToProto("request-pronouns"),
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if socialClient.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfile calls = %d, want %d", socialClient.GetUserProfileCalls, 1)
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Name != "Request Name" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "Request Name")
	}
	if payload.AvatarSetID != "people-v1" {
		t.Fatalf("payload avatar_set_id = %q, want %q", payload.AvatarSetID, "people-v1")
	}
	if payload.AvatarAssetID != "request-avatar" {
		t.Fatalf("payload avatar_asset_id = %q, want %q", payload.AvatarAssetID, "request-avatar")
	}
	if payload.Pronouns != "request-pronouns" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Pronouns, "request-pronouns")
	}
}

func TestCreateParticipant_UserLinkedMissingFieldsHydrateFromSocial(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Social Name","role":"player","controller":"human","campaign_access":"member","avatar_set_id":"creatures-v1","avatar_asset_id":"social-avatar","pronouns":"social-pronouns"}`),
			}),
		},
	}}
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-123",
		Name:          "Social Name",
		Pronouns:      sharedpronouns.ToProto("social-pronouns"),
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
			Social:      socialClient,
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		nil,
	)

	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		UserId:     "user-123",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Name != "Social Name" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "Social Name")
	}
	if payload.AvatarSetID != "creatures-v1" {
		t.Fatalf("payload avatar_set_id = %q, want %q", payload.AvatarSetID, "creatures-v1")
	}
	if payload.AvatarAssetID != "social-avatar" {
		t.Fatalf("payload avatar_asset_id = %q, want %q", payload.AvatarAssetID, "social-avatar")
	}
	if payload.Pronouns != "social-pronouns" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Pronouns, "social-pronouns")
	}
}

func TestCreateParticipant_UserLinkedMissingNameFallsBackToAuthUsername(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"user-handle","role":"player","controller":"human","campaign_access":"member"}`),
			}),
		},
	}}
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "user-handle"}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		authClient,
	)

	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		UserId:     "user-123",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Name != "user-handle" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "user-handle")
	}
	if payload.AvatarSetID != "" {
		t.Fatalf("payload avatar_set_id = %q, want empty", payload.AvatarSetID)
	}
	if payload.AvatarAssetID != "" {
		t.Fatalf("payload avatar_asset_id = %q, want empty", payload.AvatarAssetID)
	}
	if payload.Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Pronouns, "they/them")
	}
	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-123" {
		t.Fatalf("GetUser request = %#v, want user-123", authClient.LastGetUserRequest)
	}
}

func TestCreateParticipant_UserLinkedMissingPronounsFallsBackToTheyThem(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"Player One","role":"player","controller":"human","campaign_access":"member","pronouns":"they/them"}`),
			}),
		},
	}}
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-123",
		AvatarSetId:   "creatures-v1",
		AvatarAssetId: "social-avatar",
	}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
			Social:      socialClient,
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		nil,
	)

	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		UserId:     "user-123",
		Name:       "Player One",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Pronouns, "they/them")
	}
	if payload.Name != "Player One" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "Player One")
	}
	if payload.AvatarSetID != "creatures-v1" {
		t.Fatalf("payload avatar_set_id = %q, want %q", payload.AvatarSetID, "creatures-v1")
	}
	if payload.AvatarAssetID != "social-avatar" {
		t.Fatalf("payload avatar_asset_id = %q, want %q", payload.AvatarAssetID, "social-avatar")
	}
}

func TestCreateParticipant_UserLinkedMissingNameFallsBackToAuthUsernameForLocale(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
	}
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		Locale: "pt-BR",
	}

	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.join"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.joined"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "participant-123",
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"user-123","name":"apelido","role":"player","controller":"human","campaign_access":"member"}`),
			}),
		},
	}}
	authClient := &testclients.FakeAuthClient{User: &authv1.User{Id: "user-123", Username: "apelido"}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("participant-123"),
		authClient,
	)

	_, err := svc.CreateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		UserId:     "user-123",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}

	var payload participant.JoinPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Name != "apelido" {
		t.Fatalf("payload name = %q, want %q", payload.Name, "apelido")
	}
	if payload.Pronouns != "they/them" {
		t.Fatalf("payload pronouns = %q, want %q", payload.Pronouns, "they/them")
	}
	if authClient.LastGetUserRequest == nil || authClient.LastGetUserRequest.GetUserId() != "user-123" {
		t.Fatalf("GetUser request = %#v, want user-123", authClient.LastGetUserRequest)
	}
}
