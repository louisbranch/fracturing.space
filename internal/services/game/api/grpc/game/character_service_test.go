package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func characterManagerParticipantStore(campaignID string) *fakeParticipantStore {
	store := newFakeParticipantStore()
	store.participants[campaignID] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     campaignID,
			CampaignAccess: participant.CampaignAccessManager,
		},
	}
	return store
}

// activeCampaignStore returns a campaign store with an active campaign for the given ID.
func activeCampaignStore(campaignID string) *fakeCampaignStore {
	store := newFakeCampaignStore()
	store.campaigns[campaignID] = storage.CampaignRecord{ID: campaignID, Status: campaign.StatusActive}
	return store
}

func TestCreateCharacter_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.CreateCharacter(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:     newFakeCampaignStore(),
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		Name: "Hero",
		Kind: statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:     newFakeCampaignStore(),
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "nonexistent",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateCharacter_CompletedCampaignDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusCompleted,
	}
	participantStore := characterManagerParticipantStore("c1")

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})
	ctx := contextWithParticipantID("manager-1")
	_, err := svc.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateCharacter_EmptyName(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_InvalidKind(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Participant:  characterManagerParticipantStore("c1"),
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})

	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateCharacter_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Event:        newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCharacter_Success_PC(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","name":"Hero","kind":"pc","notes":"A brave adventurer"}`),
			}),
		},
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      "char-123",
				SystemID:      "daggerheart",
				SystemVersion: "1.0.0",
				PayloadJSON:   []byte(`{"character_id":"char-123","hp_after":6}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
			Event:        eventStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	resp, err := svc.CreateCharacter(contextWithParticipantID("manager-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
		Notes:      "A brave adventurer",
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if resp.Character == nil {
		t.Fatal("CreateCharacter response has nil character")
	}
	if resp.Character.Id != "char-123" {
		t.Errorf("Character ID = %q, want %q", resp.Character.Id, "char-123")
	}
	if resp.Character.Name != "Hero" {
		t.Errorf("Character Name = %q, want %q", resp.Character.Name, "Hero")
	}
	if resp.Character.Kind != statev1.CharacterKind_PC {
		t.Errorf("Character Kind = %v, want %v", resp.Character.Kind, statev1.CharacterKind_PC)
	}

	// Verify character persisted
	_, err = characterStore.GetCharacter(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}

	// Verify Daggerheart profile persisted
	_, err = dhStore.GetDaggerheartCharacterProfile(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Daggerheart profile not persisted: %v", err)
	}

	// Verify Daggerheart state persisted
	_, err = dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Daggerheart state not persisted: %v", err)
	}

	if got := len(eventStore.events["c1"]); got != 3 {
		t.Fatalf("expected 3 events, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.created") {
		t.Fatalf("event[0] type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.created"))
	}
	if eventStore.events["c1"][1].Type != event.Type("character.profile_updated") {
		t.Fatalf("event[1] type = %s, want %s", eventStore.events["c1"][1].Type, event.Type("character.profile_updated"))
	}
	if eventStore.events["c1"][2].Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event[2] type = %s, want %s", eventStore.events["c1"][2].Type, event.Type("sys.daggerheart.character_state_patched"))
	}
}

func TestCreateCharacter_Success_NPC(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusDraft,
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "npc-456",
				PayloadJSON: []byte(`{"character_id":"npc-456","name":"Shopkeeper","kind":"npc"}`),
			}),
		},
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "npc-456",
				PayloadJSON: []byte(`{"character_id":"npc-456","system_profile":{"daggerheart":{"hp_max":6}}}`),
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      "npc-456",
				SystemID:      "daggerheart",
				SystemVersion: "1.0.0",
				PayloadJSON:   []byte(`{"character_id":"npc-456","hp_after":3}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
			Event:        eventStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("npc-456"),
	}

	resp, err := svc.CreateCharacter(contextWithParticipantID("manager-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Shopkeeper",
		Kind:       statev1.CharacterKind_NPC,
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if resp.Character.Kind != statev1.CharacterKind_NPC {
		t.Errorf("Character Kind = %v, want %v", resp.Character.Kind, statev1.CharacterKind_NPC)
	}
	if got := len(eventStore.events["c1"]); got != 3 {
		t.Fatalf("expected 3 events, got %d", got)
	}
}

func TestCreateCharacter_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","name":"Hero","kind":"pc","notes":"A brave adventurer"}`),
			}),
		},
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      "char-123",
				SystemID:      "daggerheart",
				SystemVersion: "1.0.0",
				PayloadJSON:   []byte(`{"character_id":"char-123","hp_after":6}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
			Event:        eventStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	resp, err := svc.CreateCharacter(contextWithParticipantID("manager-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
		Notes:      "A brave adventurer",
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if resp.Character == nil {
		t.Fatal("CreateCharacter response has nil character")
	}
	if domain.calls != 3 {
		t.Fatalf("expected domain to be called three times, got %d", domain.calls)
	}
	if len(domain.commands) != 3 {
		t.Fatalf("expected 3 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.create") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.create")
	}
	if domain.commands[1].Type != command.Type("character.profile_update") {
		t.Fatalf("command type = %s, want %s", domain.commands[1].Type, "character.profile_update")
	}
	if domain.commands[2].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command type = %s, want %s", domain.commands[2].Type, "sys.daggerheart.character_state.patch")
	}
	if _, err := characterStore.GetCharacter(context.Background(), "c1", "char-123"); err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if _, err := dhStore.GetDaggerheartCharacterProfile(context.Background(), "c1", "char-123"); err != nil {
		t.Fatalf("Daggerheart profile not persisted: %v", err)
	}
	if _, err := dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "char-123"); err != nil {
		t.Fatalf("Daggerheart state not persisted: %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 3 {
		t.Fatalf("expected 3 events, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.created") {
		t.Fatalf("event[0] type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.created"))
	}
	if eventStore.events["c1"][1].Type != event.Type("character.profile_updated") {
		t.Fatalf("event[1] type = %s, want %s", eventStore.events["c1"][1].Type, event.Type("character.profile_updated"))
	}
	if eventStore.events["c1"][2].Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event[2] type = %s, want %s", eventStore.events["c1"][2].Type, event.Type("sys.daggerheart.character_state_patched"))
	}
}

func TestCreateCharacter_AssignsOwnerParticipantInCommandPayload(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 18, 30, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "manager-1",
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","name":"Hero","kind":"pc","owner_participant_id":"manager-1"}`),
			}),
		},
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "manager-1",
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeParticipant,
				ActorID:       "manager-1",
				EntityType:    "character",
				EntityID:      "char-123",
				SystemID:      "daggerheart",
				SystemVersion: "1.0.0",
				PayloadJSON:   []byte(`{"character_id":"char-123","hp_after":6}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
			Event:        eventStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	_, err := svc.CreateCharacter(contextWithParticipantID("manager-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if len(domain.commands) == 0 {
		t.Fatal("expected character.create command")
	}
	var payload character.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal create payload: %v", err)
	}
	if payload.OwnerParticipantID != "manager-1" {
		t.Fatalf("owner_participant_id = %q, want %q", payload.OwnerParticipantID, "manager-1")
	}
}

func TestCreateCharacter_PlayerAssignsControllerInCommandPayload(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"player-1": {
			ID:             "player-1",
			CampaignID:     "c1",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "player-1",
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","name":"Hero","kind":"pc","owner_participant_id":"player-1","participant_id":"player-1"}`),
			}),
		},
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "player-1",
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeParticipant,
				ActorID:       "player-1",
				EntityType:    "character",
				EntityID:      "char-123",
				SystemID:      "daggerheart",
				SystemVersion: "1.0.0",
				PayloadJSON:   []byte(`{"character_id":"char-123","hp_after":6}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
			Event:        eventStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	resp, err := svc.CreateCharacter(contextWithParticipantID("player-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if len(domain.commands) == 0 {
		t.Fatal("expected character.create command")
	}
	var payload character.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal create payload: %v", err)
	}
	if payload.ParticipantID != "player-1" {
		t.Fatalf("participant_id = %q, want %q", payload.ParticipantID, "player-1")
	}
	participantIDValue := resp.GetCharacter().GetParticipantId()
	if participantIDValue == nil || participantIDValue.GetValue() != "player-1" {
		t.Fatalf("response participant_id = %v, want %q", participantIDValue, "player-1")
	}
	stored, err := characterStore.GetCharacter(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if stored.ParticipantID != "player-1" {
		t.Fatalf("stored participant_id = %q, want %q", stored.ParticipantID, "player-1")
	}
}

func TestCreateCharacter_GMAssignsControllerForNPCInCommandPayload(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"gm-1": {
			ID:             "gm-1",
			CampaignID:     "c1",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "gm-1",
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","name":"Guide","kind":"npc","owner_participant_id":"gm-1","participant_id":"gm-1"}`),
			}),
		},
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "gm-1",
				EntityType:  "character",
				EntityID:    "char-123",
				PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeParticipant,
				ActorID:       "gm-1",
				EntityType:    "character",
				EntityID:      "char-123",
				SystemID:      "daggerheart",
				SystemVersion: "1.0.0",
				PayloadJSON:   []byte(`{"character_id":"char-123","hp_after":6}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
			Event:        eventStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	resp, err := svc.CreateCharacter(contextWithParticipantID("gm-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Guide",
		Kind:       statev1.CharacterKind_NPC,
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if len(domain.commands) == 0 {
		t.Fatal("expected character.create command")
	}
	var payload character.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal create payload: %v", err)
	}
	if payload.ParticipantID != "gm-1" {
		t.Fatalf("participant_id = %q, want %q", payload.ParticipantID, "gm-1")
	}
	participantIDValue := resp.GetCharacter().GetParticipantId()
	if participantIDValue == nil || participantIDValue.GetValue() != "gm-1" {
		t.Fatalf("response participant_id = %v, want %q", participantIDValue, "gm-1")
	}
	stored, err := characterStore.GetCharacter(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if stored.ParticipantID != "gm-1" {
		t.Fatalf("stored participant_id = %q, want %q", stored.ParticipantID, "gm-1")
	}
}

func TestUpdateCharacter_NoFields(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-owner", Name: "Hero", Kind: character.KindPC},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Event: eventStore})
	_, err := svc.UpdateCharacter(context.Background(), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCharacter_DeniesMemberWhenNotOwner(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 18, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
		"member-owner": {
			ID:             "member-owner",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-owner", Name: "Hero", Kind: character.KindPC},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"Renamed"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	_, err := svc.UpdateCharacter(contextWithParticipantID("member-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestUpdateCharacter_DeniesControllerWhenOwnershipUnresolved(t *testing.T) {
	// A member whose participant ID matches ParticipantID (controller) but has
	// no ownership events should be denied — controller is not owner.
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"controller-1": {
			ID:             "controller-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, ParticipantID: "controller-1"},
	}
	// No events — ownership cannot be resolved from the journal.
	eventStore.nextSeq["c1"] = 1

	domain := &fakeDomainEngine{store: eventStore}
	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	_, err := svc.UpdateCharacter(contextWithParticipantID("controller-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestUpdateCharacter_AllowsMemberWhenOwner(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 18, 35, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-1", Name: "Hero", Kind: character.KindPC},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-1"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"Renamed"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	resp, err := svc.UpdateCharacter(contextWithParticipantID("member-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.GetCharacter().GetName() != "Renamed" {
		t.Fatalf("character name = %q, want %q", resp.GetCharacter().GetName(), "Renamed")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestUpdateCharacter_AllowsMemberWhenOwnershipTransferred(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 30, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
		"member-owner": {
			ID:             "member-owner",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-1", Name: "Hero", Kind: character.KindPC},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
		{
			Seq:         2,
			CampaignID:  "c1",
			Type:        event.Type("character.updated"),
			Timestamp:   now.Add(time.Minute),
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","fields":{"owner_participant_id":"member-1"}}`),
		},
	}
	eventStore.nextSeq["c1"] = 3

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"Renamed"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	resp, err := svc.UpdateCharacter(contextWithParticipantID("member-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.GetCharacter().GetName() != "Renamed" {
		t.Fatalf("character name = %q, want %q", resp.GetCharacter().GetName(), "Renamed")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestUpdateCharacter_DeniesManagerOwnershipTransfer(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 35, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessManager,
		},
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "manager-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"manager-1"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now.Add(time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "manager-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"owner_participant_id":"member-1"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	_, err := svc.UpdateCharacter(contextWithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		OwnerParticipantId: wrapperspb.String("member-1"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestUpdateCharacter_AllowsOwnerOwnershipTransfer(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 40, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {
			ID:             "owner-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessOwner,
		},
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"owner-1"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now.Add(time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"owner_participant_id":"member-1"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	resp, err := svc.UpdateCharacter(contextWithParticipantID("owner-1"), &statev1.UpdateCharacterRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		OwnerParticipantId: wrapperspb.String("member-1"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.GetCharacter().GetId() != "ch1" {
		t.Fatalf("character id = %q, want %q", resp.GetCharacter().GetId(), "ch1")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("domain command count = %d, want 1", len(domain.commands))
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["owner_participant_id"] != "member-1" {
		t.Fatalf("owner_participant_id = %q, want %q", payload.Fields["owner_participant_id"], "member-1")
	}
}

func TestUpdateCharacter_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Event: newFakeEventStore()})
	_, err := svc.UpdateCharacter(context.Background(), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("New Hero"),
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateCharacter_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, Notes: "old"},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"New Hero","kind":"npc","notes":"updated"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Participant: participantStore, Character: characterStore, Event: eventStore, Domain: domain})
	resp, err := svc.UpdateCharacter(contextWithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("New Hero"),
		Kind:        statev1.CharacterKind_NPC,
		Notes:       wrapperspb.String("updated"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.Character.Name != "New Hero" {
		t.Errorf("Character Name = %q, want %q", resp.Character.Name, "New Hero")
	}
	if resp.Character.Kind != statev1.CharacterKind_NPC {
		t.Errorf("Character Kind = %v, want %v", resp.Character.Kind, statev1.CharacterKind_NPC)
	}
	if resp.Character.Notes != "updated" {
		t.Errorf("Character Notes = %q, want %q", resp.Character.Notes, "updated")
	}

	stored, err := characterStore.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("character not persisted: %v", err)
	}
	if stored.Name != "New Hero" {
		t.Errorf("Stored Name = %q, want %q", stored.Name, "New Hero")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.updated"))
	}
}

func TestUpdateCharacter_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, Notes: "old"},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"New Hero","kind":"npc","notes":"updated"}}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Character:   characterStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.UpdateCharacter(contextWithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("New Hero"),
		Kind:        statev1.CharacterKind_NPC,
		Notes:       wrapperspb.String("updated"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.Character == nil {
		t.Fatal("UpdateCharacter response has nil character")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.update") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.update")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.updated"))
	}
	stored, err := characterStore.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("character not persisted: %v", err)
	}
	if stored.Name != "New Hero" {
		t.Fatalf("Stored Name = %q, want %q", stored.Name, "New Hero")
	}
}

func TestDeleteCharacter_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, CharacterCount: 1}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.deleted"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","reason":"retired"}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Participant: participantStore, Character: characterStore, Event: eventStore, Domain: domain})
	resp, err := svc.DeleteCharacter(contextWithParticipantID("manager-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Reason:      "retired",
	})
	if err != nil {
		t.Fatalf("DeleteCharacter returned error: %v", err)
	}
	if resp.Character.Id != "ch1" {
		t.Errorf("Character ID = %q, want %q", resp.Character.Id, "ch1")
	}
	if _, err := characterStore.GetCharacter(context.Background(), "c1", "ch1"); err == nil {
		t.Fatal("expected character to be deleted")
	}
	updatedCampaign, err := campaignStore.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.CharacterCount != 0 {
		t.Errorf("CharacterCount = %d, want 0", updatedCampaign.CharacterCount)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.deleted") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.deleted"))
	}
}

func TestDeleteCharacter_DeniesMemberWhenNotOwner(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 18, 10, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, CharacterCount: 1}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
		"member-owner": {
			ID:             "member-owner",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.deleted"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","reason":"retired"}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})
	_, err := svc.DeleteCharacter(contextWithParticipantID("member-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Reason:      "retired",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestDeleteCharacter_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, CharacterCount: 1}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Event: newFakeEventStore()})
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteCharacter_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive, CharacterCount: 1}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.deleted"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","reason":"retired"}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Character:   characterStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.DeleteCharacter(contextWithParticipantID("manager-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Reason:      "retired",
	})
	if err != nil {
		t.Fatalf("DeleteCharacter returned error: %v", err)
	}
	if resp.Character.Id != "ch1" {
		t.Fatalf("Character ID = %q, want %q", resp.Character.Id, "ch1")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.delete") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.delete")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.deleted") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.deleted"))
	}
	if _, err := characterStore.GetCharacter(context.Background(), "c1", "ch1"); err == nil {
		t.Fatal("expected character to be deleted")
	}
	updatedCampaign, err := campaignStore.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.CharacterCount != 0 {
		t.Fatalf("CharacterCount = %d, want 0", updatedCampaign.CharacterCount)
	}
}

func TestListCharacters_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.ListCharacters(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCharacters_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:  newFakeCampaignStore(),
		Character: newFakeCharacterStore(),
	})
	_, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCharacters_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:  newFakeCampaignStore(),
		Character: newFakeCharacterStore(),
	})
	_, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListCharacters_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", Name: "GM", Role: participant.RoleGM, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Participant: participantStore})
	_, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListCharacters_EmptyList(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", Name: "GM", Role: participant.RoleGM, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Participant: participantStore})
	resp, err := svc.ListCharacters(contextWithParticipantID("p1"), &statev1.ListCharactersRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListCharacters returned error: %v", err)
	}
	if len(resp.Characters) != 0 {
		t.Errorf("ListCharacters returned %d characters, want 0", len(resp.Characters))
	}
}

func TestListCharacters_WithCharacters(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, CreatedAt: now},
		"ch2": {ID: "ch2", CampaignID: "c1", Name: "Sidekick", Kind: character.KindNPC, CreatedAt: now},
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", Name: "GM", Role: participant.RoleGM, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Participant: participantStore})
	resp, err := svc.ListCharacters(contextWithParticipantID("p1"), &statev1.ListCharactersRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListCharacters returned error: %v", err)
	}
	if len(resp.Characters) != 2 {
		t.Errorf("ListCharacters returned %d characters, want 2", len(resp.Characters))
	}
}

func TestSetDefaultControl_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.SetDefaultControl(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:  newFakeCampaignStore(),
		Character: newFakeCharacterStore(),
		Event:     newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_MissingCharacterId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCharacterService(Stores{
		Campaign:  campaignStore,
		Character: newFakeCharacterStore(),
		Event:     newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:  newFakeCampaignStore(),
		Character: newFakeCharacterStore(),
		Event:     newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "nonexistent",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_CharacterNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCharacterService(Stores{
		Campaign:  campaignStore,
		Character: newFakeCharacterStore(),
		Event:     newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "nonexistent",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero"},
	}

	svc := NewCharacterService(Stores{
		Campaign:  campaignStore,
		Character: characterStore,
		Event:     newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSetDefaultControl_MissingParticipantId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}

	svc := NewCharacterService(Stores{
		Campaign:  campaignStore,
		Character: characterStore,
		Event:     newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_ParticipantNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   characterStore,
		Participant: participantStore,
		Event:       newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String("nonexistent"),
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_Success_Unassigned(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":""}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
		Domain:      domain,
	})

	resp, err := svc.SetDefaultControl(contextWithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}
	if resp.CampaignId != "c1" {
		t.Errorf("Response CampaignId = %q, want %q", resp.CampaignId, "c1")
	}
	if resp.CharacterId != "ch1" {
		t.Errorf("Response CharacterId = %q, want %q", resp.CharacterId, "ch1")
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["participant_id"] != "" {
		t.Fatalf("participant_id = %q, want empty", payload.Fields["participant_id"])
	}
	if payload.Fields["avatar_set_id"] != assetcatalog.AvatarSetBlankV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], assetcatalog.AvatarSetBlankV1)
	}
	if payload.Fields["avatar_asset_id"] != "" {
		t.Fatalf("avatar_asset_id = %q, want empty", payload.Fields["avatar_asset_id"])
	}

	// Verify persisted
	updated, err := characterStore.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if updated.ParticipantID != "" {
		t.Fatalf("ParticipantID = %q, want empty", updated.ParticipantID)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.updated"))
	}
}

func TestSetDefaultControl_Success_Participant(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := characterManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     "c1",
			Name:           "Manager 1",
			CampaignAccess: participant.CampaignAccessManager,
			CreatedAt:      now,
		},
		"p1": {
			ID:            "p1",
			CampaignID:    "c1",
			Name:          "Player 1",
			AvatarSetID:   assetcatalog.AvatarSetPeopleV1,
			AvatarAssetID: "009",
			CreatedAt:     now,
		},
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":"p1"}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   characterStore,
		Participant: participantStore,
		Event:       eventStore,
		Domain:      domain,
	})

	resp, err := svc.SetDefaultControl(contextWithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String("p1"),
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["participant_id"] != "p1" {
		t.Fatalf("participant_id = %q, want %q", payload.Fields["participant_id"], "p1")
	}
	if payload.Fields["avatar_set_id"] != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], assetcatalog.AvatarSetPeopleV1)
	}
	if payload.Fields["avatar_asset_id"] != "009" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "009")
	}

	// Verify persisted
	ctrl, err := characterStore.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if ctrl.ParticipantID != "p1" {
		t.Errorf("ParticipantID = %q, want %q", ctrl.ParticipantID, "p1")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.updated"))
	}
	_ = resp
}

func TestSetDefaultControl_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := characterManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     "c1",
			Name:           "Manager 1",
			CampaignAccess: participant.CampaignAccessManager,
			CreatedAt:      now,
		},
		"p1": {ID: "p1", CampaignID: "c1", Name: "Player 1", CreatedAt: now},
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":"p1"}}`),
			}),
		},
	}}

	svc := &CharacterService{
		stores: Stores{
			Campaign:    campaignStore,
			Character:   characterStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.SetDefaultControl(contextWithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String("p1"),
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}
	if resp.CharacterId != "ch1" {
		t.Fatalf("Response CharacterId = %q, want %q", resp.CharacterId, "ch1")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.update") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.update")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.updated"))
	}
	updated, err := characterStore.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if updated.ParticipantID != "p1" {
		t.Fatalf("ParticipantID = %q, want %q", updated.ParticipantID, "p1")
	}
}

func TestGetCharacterSheet_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.GetCharacterSheet(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:     newFakeCampaignStore(),
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{CharacterId: "ch1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_MissingCharacterId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:     newFakeCampaignStore(),
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCharacterSheet_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", Name: "GM", Role: participant.RoleGM, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Participant:  participantStore,
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCharacterSheet_CharacterNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", Name: "GM", Role: participant.RoleGM, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    newFakeCharacterStore(),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()},
		Participant:  participantStore,
	})
	_, err := svc.GetCharacterSheet(contextWithParticipantID("p1"), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCharacterSheet_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, CreatedAt: now},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10, Agility: 2, Strength: 1},
	}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", Name: "GM", Role: participant.RoleGM, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Character:    characterStore,
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
		Participant:  participantStore,
	})

	resp, err := svc.GetCharacterSheet(contextWithParticipantID("p1"), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	if err != nil {
		t.Fatalf("GetCharacterSheet returned error: %v", err)
	}
	if resp.Character == nil {
		t.Fatal("GetCharacterSheet response has nil character")
	}
	if resp.Profile == nil {
		t.Fatal("GetCharacterSheet response has nil profile")
	}
	if resp.State == nil {
		t.Fatal("GetCharacterSheet response has nil state")
	}
	if resp.Character.Name != "Hero" {
		t.Errorf("Character Name = %q, want %q", resp.Character.Name, "Hero")
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 12 {
		t.Errorf("Profile HpMax = %d, want %d", dh.GetHpMax(), 12)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHope() != 3 {
		t.Errorf("State Hope = %d, want %d", dh.GetHope(), 3)
	}
}

func TestPatchCharacterProfile_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.PatchCharacterProfile(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_MissingCharacterId(t *testing.T) {
	svc := NewCharacterService(Stores{SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{Campaign: newFakeCampaignStore(), SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterProfile_ProfileNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	svc := NewCharacterService(Stores{Campaign: campaignStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterProfile_CompletedCampaignDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusCompleted,
	}
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}

	svc := NewCharacterService(Stores{
		Campaign:     campaignStore,
		Participant:  characterManagerParticipantStore("c1"),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
		Event:        newFakeEventStore(),
	})
	ctx := contextWithParticipantID("manager-1")
	_, err := svc.PatchCharacterProfile(ctx, &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestPatchCharacterProfile_DeniesMissingIdentity(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","system_profile":{"daggerheart":{"hp_max":10}}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:     activeCampaignStore("c1"),
		Participant:  characterManagerParticipantStore("c1"),
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
		Event:        eventStore,
		Domain:       domain,
	})

	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestPatchCharacterProfile_DeniesMemberWhenNotOwner(t *testing.T) {
	participantStore := newFakeParticipantStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 18, 20, 0, 0, time.UTC)

	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
		"member-owner": {
			ID:             "member-owner",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	characterStore := newFakeCharacterStore()
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-owner", Name: "Hero", Kind: character.KindPC},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 6, StressMax: 6},
	}
	eventStore.events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
	}
	eventStore.nextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","system_profile":{"daggerheart":{"hp_max":10}}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{
		Campaign:     activeCampaignStore("c1"),
		Participant:  participantStore,
		Character:    characterStore,
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
		Event:        eventStore,
		Domain:       domain,
	})
	_, err := svc.PatchCharacterProfile(contextWithParticipantID("member-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10},
		},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestPatchCharacterProfile_NegativeHpMax(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12},
	}

	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: -1}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_ZeroHpMaxNoChange(t *testing.T) {
	// In proto3 patch semantics, HpMax=0 means "don't change" since 0 is the default value.
	// The original HpMax should be preserved.
	participantStore := characterManagerParticipantStore("c1")
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	profileJSON, err := json.Marshal(map[string]any{
		"character_id": "ch1",
		"system_profile": map[string]any{
			"daggerheart": map[string]any{
				"hp_max":     12,
				"stress_max": 6,
			},
		},
	})
	if err != nil {
		t.Fatalf("encode profile payload: %v", err)
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: profileJSON,
			}),
		},
	}}

	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), Participant: participantStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: eventStore, Domain: domain})
	resp, err := svc.PatchCharacterProfile(contextWithParticipantID("manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 0}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	// HpMax should remain unchanged at 12
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 12 {
		t.Errorf("Profile HpMax = %d, want %d (unchanged)", dh.GetHpMax(), 12)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.profile_updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.profile_updated"))
	}
}

func TestPatchCharacterProfile_Success(t *testing.T) {
	participantStore := characterManagerParticipantStore("c1")
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10},
	}
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","system_profile":{"daggerheart":{"hp_max":10,"stress_max":8}}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), Participant: participantStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: eventStore, Domain: domain})
	resp, err := svc.PatchCharacterProfile(contextWithParticipantID("manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10, StressMax: wrapperspb.Int32(8)}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if resp.Profile == nil {
		t.Fatal("PatchCharacterProfile response has nil profile")
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 10 {
		t.Errorf("Profile HpMax = %d, want %d", dh.GetHpMax(), 10)
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetStressMax().GetValue() != 8 {
		t.Errorf("Profile StressMax = %d, want %d", dh.GetStressMax().GetValue(), 8)
	}

	// Verify unchanged fields preserved
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetEvasion().GetValue() != 10 {
		t.Errorf("Profile Evasion = %d, want %d (unchanged)", dh.GetEvasion().GetValue(), 10)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.profile_updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.profile_updated"))
	}
}

func TestPatchCharacterProfile_UsesDomainEngine(t *testing.T) {
	participantStore := characterManagerParticipantStore("c1")
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10},
	}
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("character.profile_update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.profile_updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","system_profile":{"daggerheart":{"hp_max":10,"stress_max":8}}}`),
			}),
		},
	}}

	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), Participant: participantStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: eventStore, Domain: domain})
	resp, err := svc.PatchCharacterProfile(contextWithParticipantID("manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10, StressMax: wrapperspb.Int32(8)}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if resp.Profile == nil {
		t.Fatal("PatchCharacterProfile response has nil profile")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.profile_update") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.profile_update")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("character.profile_updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("character.profile_updated"))
	}
}

func TestPatchCharacterProfile_RejectsCreationWorkflowFields(t *testing.T) {
	participantStore := characterManagerParticipantStore("c1")
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {
			CampaignID:  "c1",
			CharacterID: "ch1",
			HpMax:       12,
			StressMax:   6,
		},
	}

	svc := NewCharacterService(Stores{
		Campaign:     activeCampaignStore("c1"),
		Participant:  participantStore,
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
		Event:        newFakeEventStore(),
		Domain:       &fakeDomainEngine{},
	})

	_, err := svc.PatchCharacterProfile(contextWithParticipantID("manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			Agility: wrapperspb.Int32(3),
		}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDaggerheartExperiencesToProto(t *testing.T) {
	// Nil/empty input
	result := daggerheartExperiencesToProto(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input, got %v", result)
	}
	result = daggerheartExperiencesToProto([]storage.DaggerheartExperience{})
	if result != nil {
		t.Fatalf("expected nil for empty input, got %v", result)
	}

	// Normal conversion
	result = daggerheartExperiencesToProto([]storage.DaggerheartExperience{
		{Name: "Stealth", Modifier: 3},
		{Name: "Insight", Modifier: -1},
	})
	if len(result) != 2 {
		t.Fatalf("expected 2 experiences, got %d", len(result))
	}
	if result[0].GetName() != "Stealth" || result[0].GetModifier() != 3 {
		t.Fatalf("experience 0 mismatch: %v", result[0])
	}
	if result[1].GetName() != "Insight" || result[1].GetModifier() != -1 {
		t.Fatalf("experience 1 mismatch: %v", result[1])
	}
}

func TestDeleteCharacter_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.DeleteCharacter(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCharacter_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{Campaign: newFakeCampaignStore(), Character: newFakeCharacterStore(), Event: newFakeEventStore()})
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCharacter_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{Campaign: newFakeCampaignStore(), Character: newFakeCharacterStore(), Event: newFakeEventStore()})
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestDeleteCharacter_MissingCharacterId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: newFakeCharacterStore(), Event: newFakeEventStore()})
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCharacter_CharacterNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: newFakeCharacterStore(), Event: newFakeEventStore()})
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterProfile_HpMaxTooHigh(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 13}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_StressMaxTooHigh(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{StressMax: wrapperspb.Int32(13)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeEvasion(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{Evasion: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeMajorThreshold(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{MajorThreshold: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeSevereThreshold(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{SevereThreshold: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeProficiency(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{Proficiency: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_RequiresDomainEngine(t *testing.T) {
	participantStore := characterManagerParticipantStore("c1")
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12},
	}

	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), Participant: participantStore, SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(contextWithParticipantID("manager-1"), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 10}},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestPatchCharacterProfile_NegativeArmorScore(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{ArmorScore: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_ArmorMaxTooHigh(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{ArmorMax: wrapperspb.Int32(13)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeArmorMax(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{ArmorMax: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_EmptyExperienceName(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}
	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			Experiences: []*daggerheartv1.DaggerheartExperience{{Name: "", Modifier: 1}},
		}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_NegativeStressMax(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6},
	}

	svc := NewCharacterService(Stores{Campaign: activeCampaignStore("c1"), SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore}, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{StressMax: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
