package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartsys "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestParseCharacterKind(t *testing.T) {
	tests := []struct {
		input string
		want  character.Kind
		err   bool
	}{
		{"pc", character.KindPC, false},
		{"NPC", character.KindNPC, false},
		{"CHARACTER_KIND_PC", character.KindPC, false},
		{"CHARACTER_KIND_NPC", character.KindNPC, false},
		{"PC", character.KindPC, false},
		{"npc", character.KindNPC, false},
		{"", character.KindUnspecified, true},
		{"enemy", character.KindUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseCharacterKind(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseCharacterKind(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseCharacterKind(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Fake character store ---

func TestApplyCharacterCreated(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 0}
	applier := Applier{Character: charStore, Campaign: cStore}

	payload := testevent.CharacterCreatedPayload{Name: "Aragorn", Kind: "PC", Notes: "A ranger"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 17, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	ch, err := charStore.GetCharacter(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if ch.Name != "Aragorn" {
		t.Fatalf("Name = %q, want %q", ch.Name, "Aragorn")
	}
	if ch.Kind != character.KindPC {
		t.Fatalf("Kind = %v, want PC", ch.Kind)
	}
	c, _ := cStore.Get(ctx, "camp-1")
	// Count is derived from actual store records (1 character created).
	if c.CharacterCount != 1 {
		t.Fatalf("CharacterCount = %d, want 1", c.CharacterCount)
	}
}

func TestApplyCharacterCreated_IdempotentCount(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 0}
	applier := Applier{Character: charStore, Campaign: cStore}

	payload := testevent.CharacterCreatedPayload{Name: "Aragorn", Kind: "PC"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 17, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data, Timestamp: stamp}

	// Apply the same event twice (idempotent replay).
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	c, _ := cStore.Get(ctx, "camp-1")
	if c.CharacterCount != 1 {
		t.Fatalf("CharacterCount = %d, want 1 (idempotent)", c.CharacterCount)
	}
}

func TestApplyCharacterCreated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterCreatedPayload{Name: "A", Kind: "PC"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyCharacterCreated_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterCreatedPayload{Name: "A", Kind: "PC"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyCharacterUpdated tests ---

func TestApplyCharacterUpdated(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{
		ID: "char-1", CampaignID: "camp-1", Name: "Old", Kind: character.KindPC,
	}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: cStore}

	payload := testevent.CharacterUpdatedPayload{Fields: map[string]any{
		"name":                 "New Name",
		"kind":                 "NPC",
		"notes":                "Some notes",
		"owner_participant_id": "part-1",
	}}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 18, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := charStore.GetCharacter(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Kind != character.KindNPC {
		t.Fatalf("Kind = %v, want NPC", updated.Kind)
	}
	if updated.Notes != "Some notes" {
		t.Fatalf("Notes = %q, want %q", updated.Notes, "Some notes")
	}
	if updated.OwnerParticipantID != "part-1" {
		t.Fatalf("OwnerParticipantID = %q, want %q", updated.OwnerParticipantID, "part-1")
	}
}

func TestApplyCharacterUpdated_EmptyFields(t *testing.T) {
	ctx := context.Background()
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.CharacterUpdatedPayload{Fields: map[string]any{}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply with empty fields should succeed: %v", err)
	}
}

func TestApplyCharacterUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"name": "x"}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

// --- applyCharacterDeleted tests ---

func TestApplyCharacterDeleted(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 3}
	applier := Applier{Character: charStore, Campaign: cStore}

	data, _ := json.Marshal(testevent.CharacterDeletedPayload{})
	stamp := time.Date(2026, 2, 11, 18, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := charStore.GetCharacter(ctx, "camp-1", "char-1"); err == nil {
		t.Fatal("expected character to be deleted")
	}
	c, _ := cStore.Get(ctx, "camp-1")
	// Count is derived from actual store records (0 remaining), not arithmetic.
	if c.CharacterCount != 0 {
		t.Fatalf("CharacterCount = %d, want 0", c.CharacterCount)
	}
}

func TestApplyCharacterDeleted_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterDeletedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyCharacterDeleted_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterDeletedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeCharacterDeleted, PayloadJSON: data}
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyDaggerheartCharacterProfileReplaced tests ---

func TestApplyDaggerheartCharacterProfileReplaced(t *testing.T) {
	ctx := context.Background()
	dhStore := newProjectionDaggerheartStore()
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(dhStore))
	applier := Applier{Adapters: adapters}

	payload := daggerheartstate.CharacterProfileReplacedPayload{
		CharacterID: "char-1",
		Profile: daggerheartstate.CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  4,
			SevereThreshold: 8,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        0,
			Agility:         1,
			Strength:        0,
			Finesse:         2,
			Instinct:        1,
			Presence:        0,
			Knowledge:       -1,
			Experiences: []daggerheartstate.CharacterProfileExperience{
				{Name: "Ranger", Modifier: 2},
			},
			Description: "Tall, patient, and heavily armored.",
		},
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   data,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	profile, err := dhStore.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 || profile.HpMax != 6 {
		t.Fatalf("profile level=%d hpMax=%d, want 1/6", profile.Level, profile.HpMax)
	}
	if profile.Agility != 1 || profile.Knowledge != -1 {
		t.Fatalf("traits agility=%d knowledge=%d, want 1/-1", profile.Agility, profile.Knowledge)
	}
	if profile.Description != "Tall, patient, and heavily armored." {
		t.Fatalf("description = %q, want %q", profile.Description, "Tall, patient, and heavily armored.")
	}
}

func TestApplyDaggerheartCharacterProfileReplaced_RoutedThroughAdapter(t *testing.T) {
	ctx := context.Background()
	dhStore := newProjectionDaggerheartStore()
	adapters := bridge.NewAdapterRegistry()
	if err := adapters.Register(daggerheartsys.NewAdapter(dhStore)); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: adapters}

	payload := daggerheartstate.CharacterProfileReplacedPayload{
		CharacterID: "char-1",
		Profile: daggerheartstate.CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  4,
			SevereThreshold: 8,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        0,
			Agility:         1,
			Strength:        0,
			Finesse:         2,
			Instinct:        1,
			Presence:        0,
			Knowledge:       -1,
			Experiences: []daggerheartstate.CharacterProfileExperience{
				{Name: "Ranger", Modifier: 2},
			},
		},
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   data,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply via adapter: %v", err)
	}
	profile, err := dhStore.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 || profile.HpMax != 6 {
		t.Fatalf("profile level=%d hpMax=%d, want 1/6", profile.Level, profile.HpMax)
	}
}

func TestApplyDaggerheartCharacterProfileReplaced_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(daggerheartstate.CharacterProfileReplacedPayload{
		CharacterID: "char-1",
		Profile:     daggerheartstate.CharacterProfile{},
	})
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   data,
	}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyDaggerheartCharacterProfileReplaced_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(daggerheartstate.CharacterProfileReplacedPayload{
		CharacterID: "char-1",
		Profile:     daggerheartstate.CharacterProfile{},
	})
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   data,
	}
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applySessionGateOpened tests ---

func TestApplyCharacterUpdated_InvalidNameType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"name": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid name type")
	}
}

func TestApplyCharacterUpdated_EmptyName(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"name": "  "}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty character name")
	}
}

func TestApplyCharacterUpdated_InvalidKindType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"kind": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid kind type")
	}
}

func TestApplyCharacterUpdated_InvalidNotesType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"notes": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid notes type")
	}
}

func TestApplyCharacterUpdated_InvalidOwnerParticipantIDType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"owner_participant_id": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid owner_participant_id type")
	}
}

// --- marshalOptionalMap tests ---

// --- applyParticipantLeft missing branches ---

func TestApplyCharacterCreated_MissingCampaignStore(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore()}
	payload := testevent.CharacterCreatedPayload{Name: "Hero", Kind: "PC"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCharacterCreated_MissingCampaignID(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.CharacterCreatedPayload{Name: "Hero", Kind: "PC"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "  ", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyCharacterCreated_InvalidJSON(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCharacterCreated_InvalidKind(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.CharacterCreatedPayload{Name: "Hero", Kind: "ALIEN"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

// --- applyCharacterDeleted missing branches ---

func TestApplyCharacterDeleted_MissingCampaignStore(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCharacterDeleted_MissingCampaignID(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyCharacterDeleted_ZeroCount(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 0}
	applier := Applier{Character: charStore, Campaign: campaignStore}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: []byte("{}"), Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.CharacterCount != 0 {
		t.Fatalf("CharacterCount = %d, want 0", c.CharacterCount)
	}
}

// --- applyDaggerheartCharacterProfileReplaced missing branches ---

func TestApplyDaggerheartCharacterProfileReplaced_MissingCampaignID(t *testing.T) {
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	evt := testevent.Event{
		CampaignID:    "  ",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   []byte("{}"),
	}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyDaggerheartCharacterProfileReplaced_InvalidJSON(t *testing.T) {
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   []byte("{"),
	}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionSpotlightSet missing branches ---

func TestApplyCharacterUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCharacterUpdated_InvalidKind(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1", Name: "Hero"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}
	payload := testevent.CharacterUpdatedPayload{Fields: map[string]any{"kind": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

// --- applyParticipantUpdated missing branches ---

func TestApplyDaggerheartCharacterProfileReplaced_InvalidProfileData(t *testing.T) {
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	payload := map[string]any{"character_id": "char-1", "profile": "not-an-object"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   data,
	}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid profile data")
	}
}

func TestApplyDaggerheartCharacterProfileReplaced_DefaultLevel(t *testing.T) {
	ctx := context.Background()
	daggerheartStore := newProjectionDaggerheartStore()
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(daggerheartStore))
	applier := Applier{Adapters: adapters}
	payload := map[string]any{
		"character_id": "char-1",
		"profile": map[string]any{
			"hp_max":           float64(6),
			"stress_max":       float64(6),
			"evasion":          float64(10),
			"major_threshold":  float64(5),
			"severe_threshold": float64(10),
			"proficiency":      float64(0),
			"armor_score":      float64(0),
			"armor_max":        float64(0),
			"agility":          float64(1),
			"strength":         float64(0),
			"finesse":          float64(0),
			"instinct":         float64(0),
			"presence":         float64(0),
			"knowledge":        float64(0),
			"experiences":      []any{},
		},
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:    "camp-1",
		EntityID:      "char-1",
		Type:          testevent.TypeDaggerheartCharacterProfileReplaced,
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		PayloadJSON:   data,
	}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	profile, _ := daggerheartStore.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if profile.Level != 1 {
		t.Fatalf("Level = %d, want 1 (default)", profile.Level)
	}
}
