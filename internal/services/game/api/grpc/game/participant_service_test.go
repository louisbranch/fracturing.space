package game

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCreateParticipant_NilRequest(t *testing.T) {
	svc := NewParticipantService(Stores{})
	_, err := svc.CreateParticipant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		Name: "Player 1",
		Role: statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		CampaignId: "nonexistent",
		Name:       "Player 1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateParticipant_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusArchived,
	}

	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Player 1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateParticipant_EmptyName(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusDraft,
	}

	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_InvalidRole(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusDraft,
	}

	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Player 1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusDraft}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: newFakeEventStore()})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Name:       "Game Master",
		Role:       statev1.ParticipantRole_GM,
		Controller: statev1.Controller_CONTROLLER_AI,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateParticipant_Success_GM(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusDraft,
	}

	eventStore := newFakeEventStore()
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
				PayloadJSON: []byte(`{"participant_id":"participant-123","user_id":"","name":"Game Master","role":"GM","controller":"AI","campaign_access":"MEMBER"}`),
			}),
		},
	}}
	svc := &ParticipantService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("participant-123"),
	}

	ctx := contextWithParticipantID("owner-1")
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
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.joined"))
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
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	eventStore := newFakeEventStore()
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
	svc := &ParticipantService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("participant-456"),
	}

	ctx := contextWithParticipantID("owner-1")
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
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.joined"))
	}
}

func TestCreateParticipant_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	eventStore := newFakeEventStore()
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

	svc := &ParticipantService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("participant-123"),
	}

	ctx := contextWithParticipantID("owner-1")
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
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.joined"))
	}
}

func TestUpdateParticipant_NoFields(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, Controller: participant.ControllerHuman},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"name":"Player Uno","controller":"ai"}}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateParticipant_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, Controller: participant.ControllerHuman},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: newFakeEventStore()})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Name:          wrapperspb.String("Player Uno"),
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateParticipant_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, Controller: participant.ControllerHuman},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"name":"Player Uno","controller":"ai"}}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Name:          wrapperspb.String("Player Uno"),
		Controller:    statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant.Name != "Player Uno" {
		t.Errorf("Participant Name = %q, want %q", resp.Participant.Name, "Player Uno")
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}

	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.Name != "Player Uno" {
		t.Errorf("Stored participant Name = %q, want %q", stored.Name, "Player Uno")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.updated"))
	}
}

func TestUpdateParticipant_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, Controller: participant.ControllerHuman},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"name":"Player Uno","controller":"ai"}}`),
			}),
		},
	}}

	svc := &ParticipantService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Name:          wrapperspb.String("Player Uno"),
		Controller:    statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("UpdateParticipant response has nil participant")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.update") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "participant.update")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.updated"))
	}
	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.Name != "Player Uno" {
		t.Fatalf("Stored participant Name = %q, want %q", stored.Name, "Player Uno")
	}
}

func TestUpdateParticipant_CampaignAccess(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", CampaignAccess: participant.CampaignAccessMember},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"campaign_access":"manager"}}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:     "c1",
		ParticipantId:  "p1",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}

	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.CampaignAccess != participant.CampaignAccessManager {
		t.Errorf("Stored participant CampaignAccess = %v, want %v", stored.CampaignAccess, participant.CampaignAccessManager)
	}
}

func TestDeleteParticipant_DeniesMemberWithoutManageAccess(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 2}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {ID: "member-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessMember},
		"p1":       {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("member-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestDeleteParticipant_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 1}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.Participant.Id != "p1" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
	}
	if _, err := participantStore.GetParticipant(context.Background(), "c1", "p1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	updatedCampaign, err := campaignStore.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.ParticipantCount != 0 {
		t.Errorf("ParticipantCount = %d, want 0", updatedCampaign.ParticipantCount)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.left") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.left"))
	}
}

func TestDeleteParticipant_DeniesWhenParticipantOwnsCharacter(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 2}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc","owner_participant_id":"p1"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestDeleteParticipant_DeniesWhenParticipantOwnsCharacterFromActorFallback(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 5, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 2}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestDeleteParticipant_AllowsWhenOwnedCharacterAlreadyDeleted(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 10, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 2}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc","owner_participant_id":"p1"}`),
		},
		{
			Seq:         2,
			CampaignID:  "c1",
			Type:        event.Type("character.deleted"),
			Timestamp:   now.Add(time.Minute),
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","reason":"retired"}`),
		},
	}
	eventStore.nextSeq["c1"] = 3

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.GetParticipant().GetId() != "p1" {
		t.Fatalf("participant id = %q, want %q", resp.GetParticipant().GetId(), "p1")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestDeleteParticipant_AllowsWhenCharacterOwnershipTransferredAway(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 25, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 3}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
		"p2":      {ID: "p2", CampaignID: "c1", Name: "Player Two", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc","owner_participant_id":"p1"}`),
		},
		{
			Seq:         2,
			CampaignID:  "c1",
			Type:        event.Type("character.updated"),
			Timestamp:   now.Add(time.Minute),
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","fields":{"owner_participant_id":"p2"}}`),
		},
	}
	eventStore.nextSeq["c1"] = 3

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore, Domain: domain})
	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.GetParticipant().GetId() != "p1" {
		t.Fatalf("participant id = %q, want %q", resp.GetParticipant().GetId(), "p1")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestDeleteParticipant_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 1}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: newFakeEventStore()})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteParticipant_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, ParticipantCount: 1}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := &ParticipantService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.Participant.Id != "p1" {
		t.Fatalf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.leave") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "participant.leave")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("participant.left") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("participant.left"))
	}
	if _, err := participantStore.GetParticipant(context.Background(), "c1", "p1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	updatedCampaign, err := campaignStore.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.ParticipantCount != 0 {
		t.Fatalf("ParticipantCount = %d, want 0", updatedCampaign.ParticipantCount)
	}
}

func TestListParticipants_NilRequest(t *testing.T) {
	svc := NewParticipantService(Stores{})
	_, err := svc.ListParticipants(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListParticipants_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListParticipants_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListParticipants_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListParticipants_CampaignArchivedAllowed(t *testing.T) {
	// ListParticipants uses CampaignOpRead which is allowed for all campaign statuses,
	// including archived campaigns. This allows viewing historical campaign participants.
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusArchived,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(contextWithParticipantID("p1"), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 1 {
		t.Errorf("ListParticipants returned %d participants, want 1", len(resp.Participants))
	}
}

func TestListParticipants_DeniesNonMember(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(contextWithParticipantID("outsider-1"), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListParticipants_WithParticipants(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
		},
		"p2": {
			ID:             "p2",
			CampaignID:     "c1",
			Name:           "Player 1",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(contextWithParticipantID("p1"), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 2 {
		t.Errorf("ListParticipants returned %d participants, want 2", len(resp.Participants))
	}
}

func TestGetParticipant_NilRequest(t *testing.T) {
	svc := NewParticipantService(Stores{})
	_, err := svc.GetParticipant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{ParticipantId: "p1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_MissingParticipantId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetParticipant_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetParticipant_ParticipantNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(contextWithParticipantID("p1"), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetParticipant_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "Game Master",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.GetParticipant(contextWithParticipantID("p1"), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	if err != nil {
		t.Fatalf("GetParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("GetParticipant response has nil participant")
	}
	if resp.Participant.Id != "p1" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
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
}
