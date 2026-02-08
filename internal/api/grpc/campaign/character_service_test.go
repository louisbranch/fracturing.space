package campaign

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/campaign"
	"github.com/louisbranch/fracturing.space/internal/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCreateCharacter_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.CreateCharacter(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_MissingCharacterStore(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCharacter_MissingEventStore(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:    newFakeCampaignStore(),
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCharacter_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:    newFakeCampaignStore(),
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		Name: "Hero",
		Kind: statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:    newFakeCampaignStore(),
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "nonexistent",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateCharacter_EmptyName(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_InvalidKind(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_Success_PC(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := &CharacterService{
		stores: Stores{
			Campaign:    campaignStore,
			Character:   characterStore,
			Daggerheart: dhStore,
			Event:       eventStore,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	resp, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
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
	if eventStore.events["c1"][0].Type != event.TypeCharacterCreated {
		t.Fatalf("event[0] type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeCharacterCreated)
	}
	if eventStore.events["c1"][1].Type != event.TypeProfileUpdated {
		t.Fatalf("event[1] type = %s, want %s", eventStore.events["c1"][1].Type, event.TypeProfileUpdated)
	}
	if eventStore.events["c1"][2].Type != event.TypeCharacterStateChanged {
		t.Fatalf("event[2] type = %s, want %s", eventStore.events["c1"][2].Type, event.TypeCharacterStateChanged)
	}
}

func TestCreateCharacter_Success_NPC(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusDraft,
	}

	svc := &CharacterService{
		stores: Stores{
			Campaign:    campaignStore,
			Character:   characterStore,
			Daggerheart: dhStore,
			Event:       eventStore,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("npc-456"),
	}

	resp, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
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

func TestUpdateCharacter_NoFields(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.CharacterKindPC},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Event: eventStore})
	_, err := svc.UpdateCharacter(context.Background(), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCharacter_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.CharacterKindPC, Notes: "old"},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Event: eventStore})
	resp, err := svc.UpdateCharacter(context.Background(), &statev1.UpdateCharacterRequest{
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
	if eventStore.events["c1"][0].Type != event.TypeCharacterUpdated {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeCharacterUpdated)
	}
}

func TestDeleteCharacter_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive, CharacterCount: 1}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.CharacterKindPC},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore, Event: eventStore})
	resp, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
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
	if eventStore.events["c1"][0].Type != event.TypeCharacterDeleted {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeCharacterDeleted)
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

func TestListCharacters_EmptyList(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore})
	resp, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{CampaignId: "c1"})
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
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.CharacterKindPC, CreatedAt: now},
		"ch2": {ID: "ch2", CampaignID: "c1", Name: "Sidekick", Kind: character.CharacterKindNPC, CreatedAt: now},
	}

	svc := NewCharacterService(Stores{Campaign: campaignStore, Character: characterStore})
	resp, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{CampaignId: "c1"})
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
		Campaign:       newFakeCampaignStore(),
		Character:      newFakeCharacterStore(),
		ControlDefault: newFakeControlDefaultStore(),
		Event:          newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CharacterId: "ch1",
		Controller:  &statev1.CharacterController{Controller: &statev1.CharacterController_Gm{Gm: &statev1.GmController{}}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_MissingCharacterId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewCharacterService(Stores{
		Campaign:       campaignStore,
		Character:      newFakeCharacterStore(),
		ControlDefault: newFakeControlDefaultStore(),
		Event:          newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId: "c1",
		Controller: &statev1.CharacterController{Controller: &statev1.CharacterController_Gm{Gm: &statev1.GmController{}}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:       newFakeCampaignStore(),
		Character:      newFakeCharacterStore(),
		ControlDefault: newFakeControlDefaultStore(),
		Event:          newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
		Controller:  &statev1.CharacterController{Controller: &statev1.CharacterController_Gm{Gm: &statev1.GmController{}}},
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_CharacterNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewCharacterService(Stores{
		Campaign:       campaignStore,
		Character:      newFakeCharacterStore(),
		ControlDefault: newFakeControlDefaultStore(),
		Event:          newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
		Controller:  &statev1.CharacterController{Controller: &statev1.CharacterController_Gm{Gm: &statev1.GmController{}}},
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_MissingController(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}

	svc := NewCharacterService(Stores{
		Campaign:       campaignStore,
		Character:      characterStore,
		ControlDefault: newFakeControlDefaultStore(),
		Event:          newFakeEventStore(),
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

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}

	svc := NewCharacterService(Stores{
		Campaign:       campaignStore,
		Character:      characterStore,
		ControlDefault: newFakeControlDefaultStore(),
		Participant:    participantStore,
		Event:          newFakeEventStore(),
	})
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Controller: &statev1.CharacterController{
			Controller: &statev1.CharacterController_Participant{
				Participant: &statev1.ParticipantController{ParticipantId: "nonexistent"},
			},
		},
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_Success_GM(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	controlStore := newFakeControlDefaultStore()
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}

	svc := NewCharacterService(Stores{
		Campaign:       campaignStore,
		Character:      characterStore,
		ControlDefault: controlStore,
		Event:          eventStore,
	})

	resp, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Controller:  &statev1.CharacterController{Controller: &statev1.CharacterController_Gm{Gm: &statev1.GmController{}}},
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

	// Verify persisted
	ctrl, err := controlStore.GetControlDefault(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Control default not persisted: %v", err)
	}
	if !ctrl.IsGM {
		t.Error("Controller should be GM")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeControllerAssigned {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeControllerAssigned)
	}
}

func TestSetDefaultControl_Success_Participant(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	participantStore := newFakeParticipantStore()
	controlStore := newFakeControlDefaultStore()
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"p1": {ID: "p1", CampaignID: "c1", DisplayName: "Player 1", CreatedAt: now},
	}

	svc := NewCharacterService(Stores{
		Campaign:       campaignStore,
		Character:      characterStore,
		Participant:    participantStore,
		ControlDefault: controlStore,
		Event:          eventStore,
	})

	resp, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Controller: &statev1.CharacterController{
			Controller: &statev1.CharacterController_Participant{
				Participant: &statev1.ParticipantController{ParticipantId: "p1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}

	// Verify persisted
	ctrl, err := controlStore.GetControlDefault(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Control default not persisted: %v", err)
	}
	if ctrl.IsGM {
		t.Error("Controller should not be GM")
	}
	if ctrl.ParticipantID != "p1" {
		t.Errorf("Controller ParticipantID = %q, want %q", ctrl.ParticipantID, "p1")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeControllerAssigned {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeControllerAssigned)
	}
	_ = resp
}

func TestGetCharacterSheet_NilRequest(t *testing.T) {
	svc := NewCharacterService(Stores{})
	_, err := svc.GetCharacterSheet(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_MissingCampaignId(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:    newFakeCampaignStore(),
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{CharacterId: "ch1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_MissingCharacterId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_CampaignNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{
		Campaign:    newFakeCampaignStore(),
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCharacterSheet_CharacterNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   newFakeCharacterStore(),
		Daggerheart: newFakeDaggerheartStore(),
	})
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCharacterSheet_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.CharacterKindPC, CreatedAt: now},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10, Agility: 2, Strength: 1},
	}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}

	svc := NewCharacterService(Stores{
		Campaign:    campaignStore,
		Character:   characterStore,
		Daggerheart: dhStore,
	})

	resp, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
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
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 18 {
		t.Errorf("Profile HpMax = %d, want %d", dh.GetHpMax(), 18)
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
	svc := NewCharacterService(Stores{Daggerheart: newFakeDaggerheartStore(), Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_MissingCharacterId(t *testing.T) {
	svc := NewCharacterService(Stores{Daggerheart: newFakeDaggerheartStore(), Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterProfile_ProfileNotFound(t *testing.T) {
	svc := NewCharacterService(Stores{Daggerheart: newFakeDaggerheartStore(), Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterProfile_NegativeHpMax(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18},
	}

	svc := NewCharacterService(Stores{Daggerheart: dhStore, Event: newFakeEventStore()})
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
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}
	eventStore := newFakeEventStore()

	svc := NewCharacterService(Stores{Daggerheart: dhStore, Event: eventStore})
	resp, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 0}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	// HpMax should remain unchanged at 18
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 18 {
		t.Errorf("Profile HpMax = %d, want %d (unchanged)", dh.GetHpMax(), 18)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeProfileUpdated {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeProfileUpdated)
	}
}

func TestPatchCharacterProfile_Success(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10},
	}
	eventStore := newFakeEventStore()

	svc := NewCharacterService(Stores{Daggerheart: dhStore, Event: eventStore})
	resp, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{HpMax: 24, StressMax: wrapperspb.Int32(8)}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	if resp.Profile == nil {
		t.Fatal("PatchCharacterProfile response has nil profile")
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 24 {
		t.Errorf("Profile HpMax = %d, want %d", dh.GetHpMax(), 24)
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
	if eventStore.events["c1"][0].Type != event.TypeProfileUpdated {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeProfileUpdated)
	}
}

func TestPatchCharacterProfile_UpdateTraits(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	// Set initial values for all 6 traits
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {
			CampaignID:      "c1",
			CharacterID:     "ch1",
			HpMax:           18,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  5,
			SevereThreshold: 10,
			Agility:         2,
			Strength:        0, // Initial value (legitimately zero)
			Finesse:         1,
			Instinct:        -1, // Negative trait
			Presence:        3,
			Knowledge:       2,
		},
	}

	eventStore := newFakeEventStore()
	svc := NewCharacterService(Stores{Daggerheart: dhStore, Event: eventStore})
	// Patch only Agility and Strength, leaving other 4 traits unchanged
	resp, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			Agility:  wrapperspb.Int32(3),
			Strength: wrapperspb.Int32(1),
		}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterProfile returned error: %v", err)
	}
	dh := resp.Profile.GetDaggerheart()
	if dh == nil {
		t.Fatal("Expected Daggerheart profile, got nil")
	}

	// Verify patched traits have new values
	if dh.GetAgility().GetValue() != 3 {
		t.Errorf("Profile Agility = %d, want %d", dh.GetAgility().GetValue(), 3)
	}
	if dh.GetStrength().GetValue() != 1 {
		t.Errorf("Profile Strength = %d, want %d", dh.GetStrength().GetValue(), 1)
	}

	// Verify unpatched traits retain original values
	if dh.GetFinesse().GetValue() != 1 {
		t.Errorf("Profile Finesse = %d, want %d (unchanged)", dh.GetFinesse().GetValue(), 1)
	}
	if dh.GetInstinct().GetValue() != -1 {
		t.Errorf("Profile Instinct = %d, want %d (unchanged)", dh.GetInstinct().GetValue(), -1)
	}
	if dh.GetPresence().GetValue() != 3 {
		t.Errorf("Profile Presence = %d, want %d (unchanged)", dh.GetPresence().GetValue(), 3)
	}
	if dh.GetKnowledge().GetValue() != 2 {
		t.Errorf("Profile Knowledge = %d, want %d (unchanged)", dh.GetKnowledge().GetValue(), 2)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeProfileUpdated {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeProfileUpdated)
	}
}

func TestPatchCharacterProfile_NegativeStressMax(t *testing.T) {
	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewCharacterService(Stores{Daggerheart: dhStore, Event: newFakeEventStore()})
	_, err := svc.PatchCharacterProfile(context.Background(), &statev1.PatchCharacterProfileRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		SystemProfilePatch: &statev1.PatchCharacterProfileRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{StressMax: wrapperspb.Int32(-1)}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
