package charactertransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func newCharacterServiceForTest(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *Service {
	return newServiceWithDependencies(deps, clock, idGenerator)
}

func characterManagerParticipantStore(campaignID string) *gametest.FakeParticipantStore {
	store := gametest.NewFakeParticipantStore()
	store.Participants[campaignID] = map[string]storage.ParticipantRecord{
		"manager-1": gametest.ManagerParticipantRecord(campaignID, "manager-1"),
	}
	return store
}

// activeCampaignStore returns a campaign store with an active campaign for the given ID.
func activeCampaignStore(campaignID string) *gametest.FakeCampaignStore {
	store := gametest.NewFakeCampaignStore()
	store.Campaigns[campaignID] = gametest.ActiveCampaignRecord(campaignID)
	return store
}

func TestUpdateCharacter_AllowsMemberWhenOwner(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 18, 35, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": gametest.MemberParticipantRecord("c1", "member-1"),
	}
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-1", Name: "Hero", Kind: character.KindPC},
	}
	ts.Event.Events["c1"] = []event.Event{
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
	ts.Event.NextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
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

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.UpdateCharacter(requestctx.WithParticipantID("member-1"), &statev1.UpdateCharacterRequest{
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

func TestUpdateCharacter_Success(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, Notes: "old"},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
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

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.UpdateCharacter(requestctx.WithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
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

	stored, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("character not persisted: %v", err)
	}
	if stored.Name != "New Hero" {
		t.Errorf("Stored Name = %q, want %q", stored.Name, "New Hero")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.updated"))
	}
}

func TestUpdateCharacter_UsesDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, Notes: "old"},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
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

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	resp, err := svc.UpdateCharacter(requestctx.WithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
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
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.updated"))
	}
	stored, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("character not persisted: %v", err)
	}
	if stored.Name != "New Hero" {
		t.Fatalf("Stored Name = %q, want %q", stored.Name, "New Hero")
	}
}
