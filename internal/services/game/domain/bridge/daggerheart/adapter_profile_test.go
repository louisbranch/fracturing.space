package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestApplyCharacterProfileEvents_GuardsAndDelete(t *testing.T) {
	adapter := NewAdapter(nil)
	if err := adapter.Apply(context.Background(), event.Event{}); err == nil {
		t.Fatal("expected store-not-configured error")
	}

	store := newParityDaggerheartStore()
	adapter = NewAdapter(store)

	if err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileReplaced,
		PayloadJSON:   []byte(`{`),
	}); err == nil {
		t.Fatal("expected decode error")
	}

	err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileDeleted,
		PayloadJSON:   []byte(`{"character_id":"char-1"}`),
	})
	if err != nil {
		t.Fatalf("delete apply profile: %v", err)
	}
}

func TestApplyCharacterProfileReplaced_PersistsValidatedProfile(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	profilePayload, err := json.Marshal(CharacterProfileReplacedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Profile: CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        2,
			Agility:         1,
			Strength:        0,
			Finesse:         0,
			Instinct:        0,
			Presence:        0,
			Knowledge:       0,
			Experiences: []CharacterProfileExperience{
				{Name: "Scout", Modifier: 1},
			},
			ClassID:    "class-1",
			SubclassID: "sub-1",
			Heritage: CharacterHeritage{
				FirstFeatureAncestryID:  "anc-1",
				FirstFeatureID:          "anc-1.feature-1",
				SecondFeatureAncestryID: "anc-1",
				SecondFeatureID:         "anc-1.feature-2",
				CommunityID:             "com-1",
			},
			TraitsAssigned:       true,
			DetailsRecorded:      true,
			StartingWeaponIDs:    []string{"w-1"},
			StartingArmorID:      "a-1",
			StartingPotionItemID: "p-1",
			Background:           "bg",
			Description:          "Tall, patient, and heavily armored.",
			DomainCardIDs:        []string{"d-1"},
			Connections:          "conn",
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileReplaced,
		PayloadJSON:   profilePayload,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
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
	if profile.Description != "Tall, patient, and heavily armored." {
		t.Fatalf("description = %q, want %q", profile.Description, "Tall, patient, and heavily armored.")
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.Hp != 6 || state.Hope != HopeDefault || state.HopeMax != HopeMaxDefault || state.Stress != StressDefault || state.Armor != 2 || state.LifeState != LifeStateAlive {
		t.Fatalf("unexpected seeded state: %+v", state)
	}
}

func TestApplyCharacterProfileReplaced_DefaultsLevelWhenZero(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	profilePayload, err := json.Marshal(CharacterProfileReplacedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Profile: CharacterProfile{
			Level:           0,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        2,
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileReplaced,
		PayloadJSON:   profilePayload,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	profile, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 {
		t.Fatalf("profile level = %d, want 1 default", profile.Level)
	}
}

func TestApplyCharacterProfileReplaced_DoesNotOverwriteExistingState(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
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

	profilePayload, err := json.Marshal(CharacterProfileReplacedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Profile: CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        2,
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileReplaced,
		PayloadJSON:   profilePayload,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.Hp != 2 || state.Hope != 1 || state.HopeMax != 4 || state.Stress != 3 || state.Armor != 1 || state.LifeState != LifeStateUnconscious {
		t.Fatalf("state was overwritten: %+v", state)
	}
}
