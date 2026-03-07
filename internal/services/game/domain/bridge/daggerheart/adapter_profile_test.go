package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplyProfile_GuardsAndReset(t *testing.T) {
	adapter := NewAdapter(nil)
	if err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected store-not-configured error")
	}

	store := newParityDaggerheartStore()
	adapter = NewAdapter(store)

	if err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", json.RawMessage(`{`)); err == nil {
		t.Fatal("expected decode error")
	}

	err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", json.RawMessage(`{"reset":true}`))
	if err != nil {
		t.Fatalf("reset apply profile: %v", err)
	}
}

func TestApplyProfile_PersistsValidatedProfile(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	payload := json.RawMessage(`{
		"level": 1,
		"hp_max": 6,
		"stress_max": 6,
		"evasion": 10,
		"major_threshold": 3,
		"severe_threshold": 6,
		"proficiency": 1,
		"armor_score": 0,
		"armor_max": 2,
		"agility": 1,
		"strength": 0,
		"finesse": 0,
		"instinct": 0,
		"presence": 0,
		"knowledge": 0,
		"experiences": [{"name":"Scout","modifier":1}],
		"class_id":"class-1",
		"subclass_id":"sub-1",
		"ancestry_id":"anc-1",
		"community_id":"com-1",
		"traits_assigned": true,
		"details_recorded": true,
		"starting_weapon_ids":["w-1"],
		"starting_armor_id":"a-1",
		"starting_potion_item_id":"p-1",
		"background":"bg",
		"domain_card_ids":["d-1"],
		"connections":"conn"
	}`)

	if err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", payload); err != nil {
		t.Fatalf("ApplyProfile: %v", err)
	}

	profile, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 || profile.HpMax != 6 || profile.StressMax != 6 {
		t.Fatalf("unexpected stored profile core stats: %+v", profile)
	}
	if len(profile.Experiences) != 1 || profile.Experiences[0].Name != "Scout" {
		t.Fatalf("unexpected experiences: %+v", profile.Experiences)
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.Hp != 6 || state.Hope != HopeDefault || state.HopeMax != HopeMaxDefault || state.Stress != StressDefault || state.Armor != ArmorDefault || state.LifeState != LifeStateAlive {
		t.Fatalf("unexpected seeded state: %+v", state)
	}
}

func TestApplyProfile_DefaultsLevelWhenZero(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	payload := json.RawMessage(`{
		"level": 0,
		"hp_max": 6,
		"stress_max": 6,
		"evasion": 10,
		"major_threshold": 3,
		"severe_threshold": 6,
		"proficiency": 1,
		"armor_score": 0,
		"armor_max": 2
	}`)
	if err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", payload); err != nil {
		t.Fatalf("ApplyProfile: %v", err)
	}
	profile, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 {
		t.Fatalf("profile level = %d, want 1 default", profile.Level)
	}
}

func TestApplyProfile_DoesNotOverwriteExistingState(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	if err := store.PutDaggerheartCharacterState(context.Background(), storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          2,
		Hope:        1,
		HopeMax:     4,
		Stress:      3,
		Armor:       1,
		LifeState:   LifeStateUnconscious,
	}); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	payload := json.RawMessage(`{
		"level": 1,
		"hp_max": 6,
		"stress_max": 6,
		"evasion": 10,
		"major_threshold": 3,
		"severe_threshold": 6,
		"proficiency": 1,
		"armor_score": 0,
		"armor_max": 2
	}`)
	if err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", payload); err != nil {
		t.Fatalf("ApplyProfile: %v", err)
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.Hp != 2 || state.Hope != 1 || state.HopeMax != 4 || state.Stress != 3 || state.Armor != 1 || state.LifeState != LifeStateUnconscious {
		t.Fatalf("state was overwritten: %+v", state)
	}
}
