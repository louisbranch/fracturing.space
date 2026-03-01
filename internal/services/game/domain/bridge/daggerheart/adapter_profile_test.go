package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
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
