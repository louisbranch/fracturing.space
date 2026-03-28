package charactertransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestCreateCharacter_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.CreateCharacter(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_MissingCampaignId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		Name: "Hero",
		Kind: statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_CampaignNotFound(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "nonexistent",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateCharacter_CompletedCampaignDisallowed(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.CompletedCampaignRecord("c1")
	ts.Participant = characterManagerParticipantStore("c1")

	svc := NewService(ts.build())
	ctx := requestctx.WithParticipantID(context.Background(), "manager-1")
	_, err := svc.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateCharacter_EmptyName(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)

	svc := NewService(ts.build())
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_InvalidKind(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)

	svc := NewService(ts.build())
	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCharacter_DeniesMissingIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	ts.Participant = characterManagerParticipantStore("c1")

	svc := NewService(ts.build())

	_, err := svc.CreateCharacter(context.Background(), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateCharacter_RequiresDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant = characterManagerParticipantStore("c1")

	svc := NewService(ts.build())
	_, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCharacter_Success_PC(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: testCreateCharacterResults(
		t,
		now,
		"c1",
		"char-123",
		event.ActorTypeSystem,
		"",
		character.CreatePayload{
			CharacterID: "char-123",
			Name:        "Hero",
			Kind:        "pc",
			Notes:       "A brave adventurer",
		},
	)}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("char-123"))

	resp, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.CreateCharacterRequest{
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

	_, err = ts.Character.GetCharacter(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}

	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.created") {
		t.Fatalf("event[0] type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.created"))
	}
}

func TestCreateCharacter_Success_NPC(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusDraft, campaign.GmModeHuman)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: testCreateCharacterResults(
		t,
		now,
		"c1",
		"npc-456",
		event.ActorTypeSystem,
		"",
		character.CreatePayload{
			CharacterID: "npc-456",
			Name:        "Shopkeeper",
			Kind:        "npc",
		},
	)}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("npc-456"))

	resp, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.CreateCharacterRequest{
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
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
}

func TestCreateCharacter_UsesDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: testCreateCharacterResults(
		t,
		now,
		"c1",
		"char-123",
		event.ActorTypeSystem,
		"",
		character.CreatePayload{
			CharacterID: "char-123",
			Name:        "Hero",
			Kind:        "pc",
			Notes:       "A brave adventurer",
		},
	)}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("char-123"))

	resp, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.CreateCharacterRequest{
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
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != commandids.CharacterCreate {
		t.Fatalf("command[0] type = %s, want %s", domain.commands[0].Type, commandids.CharacterCreate)
	}
	if _, err := ts.Character.GetCharacter(context.Background(), "c1", "char-123"); err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.created") {
		t.Fatalf("event[0] type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.created"))
	}
}

func TestCreateCharacter_AssignsOwnerParticipantInCommandPayload(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Date(2026, 2, 20, 18, 30, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: testCreateCharacterResults(
		t,
		now,
		"c1",
		"char-123",
		event.ActorTypeParticipant,
		"manager-1",
		character.CreatePayload{
			CharacterID:        "char-123",
			Name:               "Hero",
			Kind:               "pc",
			OwnerParticipantID: "manager-1",
		},
	)}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("char-123"))

	_, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.CreateCharacterRequest{
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

func TestCreateCharacter_PlayerAssignsOwnerInCommandPayload(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 19, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"player-1": gametest.RoleMemberParticipantRecord("c1", "player-1", participant.RolePlayer),
	}
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: testCreateCharacterResults(
		t,
		now,
		"c1",
		"char-123",
		event.ActorTypeParticipant,
		"player-1",
		character.CreatePayload{
			CharacterID:        "char-123",
			Name:               "Hero",
			Kind:               "pc",
			OwnerParticipantID: "player-1",
		},
	)}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("char-123"))

	resp, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "player-1"), &statev1.CreateCharacterRequest{
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
	if payload.OwnerParticipantID != "player-1" {
		t.Fatalf("owner_participant_id = %q, want %q", payload.OwnerParticipantID, "player-1")
	}
	participantIDValue := resp.GetCharacter().GetOwnerParticipantId()
	if participantIDValue == nil || participantIDValue.GetValue() != "player-1" {
		t.Fatalf("response owner_participant_id = %v, want %q", participantIDValue, "player-1")
	}
	stored, err := ts.Character.GetCharacter(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if stored.OwnerParticipantID != "player-1" {
		t.Fatalf("stored owner_participant_id = %q, want %q", stored.OwnerParticipantID, "player-1")
	}
}

func TestCreateCharacter_GMAssignsOwnerForNPCInCommandPayload(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"gm-1": gametest.RoleMemberParticipantRecord("c1", "gm-1", participant.RoleGM),
	}
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: testCreateCharacterResults(
		t,
		now,
		"c1",
		"char-123",
		event.ActorTypeParticipant,
		"gm-1",
		character.CreatePayload{
			CharacterID:        "char-123",
			Name:               "Guide",
			Kind:               "npc",
			OwnerParticipantID: "gm-1",
		},
	)}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("char-123"))

	resp, err := svc.CreateCharacter(requestctx.WithParticipantID(context.Background(), "gm-1"), &statev1.CreateCharacterRequest{
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
	if payload.OwnerParticipantID != "gm-1" {
		t.Fatalf("owner_participant_id = %q, want %q", payload.OwnerParticipantID, "gm-1")
	}
	participantIDValue := resp.GetCharacter().GetOwnerParticipantId()
	if participantIDValue == nil || participantIDValue.GetValue() != "gm-1" {
		t.Fatalf("response owner_participant_id = %v, want %q", participantIDValue, "gm-1")
	}
	stored, err := ts.Character.GetCharacter(context.Background(), "c1", "char-123")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if stored.OwnerParticipantID != "gm-1" {
		t.Fatalf("stored owner_participant_id = %q, want %q", stored.OwnerParticipantID, "gm-1")
	}
}
