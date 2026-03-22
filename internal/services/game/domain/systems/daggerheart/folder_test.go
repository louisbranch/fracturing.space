package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartfolder "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/folder"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// assertTestSnapshotState extracts a *daggerheartstate.SnapshotState from an any value returned
// by a fold, accepting both value and pointer types for test convenience.
func assertTestSnapshotState(t *testing.T, v any) daggerheartstate.SnapshotState {
	t.Helper()
	switch typed := v.(type) {
	case daggerheartstate.SnapshotState:
		return typed
	case *daggerheartstate.SnapshotState:
		if typed != nil {
			return *typed
		}
	}
	t.Fatalf("expected daggerheartstate.SnapshotState or *daggerheartstate.SnapshotState, got %T", v)
	return daggerheartstate.SnapshotState{}
}

func TestSnapshotOrDefault_NilReturnsFactoryDefaults(t *testing.T) {
	s, hasState := daggerheartstate.SnapshotOrDefault(nil)
	if hasState {
		t.Fatal("expected hasState=false for nil input")
	}
	if s.GMFear != daggerheartstate.GMFearDefault {
		t.Fatalf("gm fear = %d, want %d", s.GMFear, daggerheartstate.GMFearDefault)
	}
	if s.CharacterStates == nil {
		t.Fatal("CharacterStates should be initialized")
	}
	if s.AdversaryStates == nil {
		t.Fatal("AdversaryStates should be initialized")
	}
	if s.CountdownStates == nil {
		t.Fatal("CountdownStates should be initialized")
	}
}

func TestSnapshotOrDefault_ValueReturnsState(t *testing.T) {
	input := daggerheartstate.SnapshotState{CampaignID: "camp-1", GMFear: 5}
	s, hasState := daggerheartstate.SnapshotOrDefault(input)
	if !hasState {
		t.Fatal("expected hasState=true for value input")
	}
	if s.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want camp-1", s.CampaignID)
	}
	if s.GMFear != 5 {
		t.Fatalf("gm fear = %d, want 5", s.GMFear)
	}
}

func TestSnapshotOrDefault_PointerReturnsState(t *testing.T) {
	input := &daggerheartstate.SnapshotState{CampaignID: "camp-2", GMFear: 3}
	s, hasState := daggerheartstate.SnapshotOrDefault(input)
	if !hasState {
		t.Fatal("expected hasState=true for pointer input")
	}
	if s.CampaignID != "camp-2" {
		t.Fatalf("campaign id = %s, want camp-2", s.CampaignID)
	}
	if s.GMFear != 3 {
		t.Fatalf("gm fear = %d, want 3", s.GMFear)
	}
}

func TestSnapshotOrDefault_NilPointerReturnsFactoryDefaults(t *testing.T) {
	var input *daggerheartstate.SnapshotState
	s, hasState := daggerheartstate.SnapshotOrDefault(input)
	if hasState {
		t.Fatal("expected hasState=false for nil pointer input")
	}
	if s.GMFear != daggerheartstate.GMFearDefault {
		t.Fatalf("gm fear = %d, want %d", s.GMFear, daggerheartstate.GMFearDefault)
	}
	if s.CharacterStates == nil {
		t.Fatal("CharacterStates should be initialized")
	}
}

func TestFolderApplyGMFearChanged_UpdatesState(t *testing.T) {
	projector := NewFolder()
	state := daggerheartstate.SnapshotState{CampaignID: "camp-1", GMFear: 2}

	payload, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 5, Reason: "shift"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	updated, err := projector.Fold(state, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	if snapshot.GMFear != 5 {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, 5)
	}
	if snapshot.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", snapshot.CampaignID, "camp-1")
	}
}

func TestFolderApplyCharacterStatePatched_StoresCharacterState(t *testing.T) {
	projector := NewFolder()
	hp := 6
	hope := 2
	payload, err := json.Marshal(daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		HP:          &hp,
		Hope:        &hope,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(daggerheartstate.SnapshotState{CampaignID: "camp-1"}, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeCharacterStatePatched,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	character, ok := snapshot.CharacterStates["char-1"]
	if !ok {
		t.Fatal("expected character state")
	}
	if character.CampaignID != "camp-1" {
		t.Fatalf("character campaign id = %s, want %s", character.CampaignID, "camp-1")
	}
	if character.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", character.CharacterID, "char-1")
	}
	if character.HP != hp {
		t.Fatalf("hp = %d, want %d", character.HP, hp)
	}
	if character.Hope != hope {
		t.Fatalf("hope = %d, want %d", character.Hope, hope)
	}
}

func TestFolderApplyCharacterStatePatched_DoesNotMutateWithoutAfterFields(t *testing.T) {
	projector := NewFolder()
	payload, err := json.Marshal(daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "char-1",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          0,
			},
		},
	}, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeCharacterStatePatched,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	character, ok := snapshot.CharacterStates["char-1"]
	if !ok {
		t.Fatal("expected character state")
	}
	if character.HP != 0 {
		t.Fatalf("hp = %d, want %d", character.HP, 0)
	}
}

func TestFolderApplyAdversaryUpdated_AppliesZeroAndEmptyValues(t *testing.T) {
	projector := NewFolder()
	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[dhids.AdversaryID]daggerheartstate.AdversaryState{
			"adv-1": {
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				Name:        "Goblin",
				Kind:        "bruiser",
				SessionID:   "sess-1",
				Notes:       "old notes",
				HP:          6,
				HPMax:       8,
				Stress:      3,
				StressMax:   3,
				Evasion:     2,
				Major:       2,
				Severe:      3,
				Armor:       1,
			},
		},
	}

	payload, err := json.Marshal(daggerheartpayload.AdversaryUpdatedPayload{
		AdversaryID: "adv-1",
		Name:        "Goblin",
		Kind:        "",
		SessionID:   "",
		Notes:       "",
		HP:          0,
		HPMax:       8,
		Stress:      0,
		StressMax:   3,
		Evasion:     0,
		Major:       0,
		Severe:      0,
		Armor:       0,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(state, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeAdversaryUpdated,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	adversary, ok := snapshot.AdversaryStates["adv-1"]
	if !ok {
		t.Fatal("expected adversary state")
	}
	if adversary.Kind != "" {
		t.Fatalf("kind = %q, want empty", adversary.Kind)
	}
	if adversary.SessionID != "" {
		t.Fatalf("session id = %q, want empty", adversary.SessionID)
	}
	if adversary.Notes != "" {
		t.Fatalf("notes = %q, want empty", adversary.Notes)
	}
	if adversary.HP != 0 {
		t.Fatalf("hp = %d, want 0", adversary.HP)
	}
	if adversary.Stress != 0 {
		t.Fatalf("stress = %d, want 0", adversary.Stress)
	}
	if adversary.Evasion != 0 {
		t.Fatalf("evasion = %d, want 0", adversary.Evasion)
	}
	if adversary.Major != 0 {
		t.Fatalf("major = %d, want 0", adversary.Major)
	}
	if adversary.Severe != 0 {
		t.Fatalf("severe = %d, want 0", adversary.Severe)
	}
	if adversary.Armor != 0 {
		t.Fatalf("armor = %d, want 0", adversary.Armor)
	}
}

func TestFoldEquipmentSwapped_ArmorUpdatesProfileAndState(t *testing.T) {
	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterProfiles: map[ids.CharacterID]daggerheartstate.CharacterProfile{
			"char-1": {
				Evasion:         10,
				MajorThreshold:  3,
				SevereThreshold: 6,
				ArmorScore:      1,
				ArmorMax:        2,
				Agility:         1,
				Strength:        1,
				Finesse:         1,
				Instinct:        1,
				Presence:        1,
				Knowledge:       1,
			},
		},
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Armor:       2,
				Stress:      1,
			},
		},
	}

	err := daggerheartfolder.FoldEquipmentSwapped(&state, daggerheartpayload.EquipmentSwappedPayload{
		CharacterID:             "char-1",
		ItemType:                "armor",
		EquippedArmorID:         "armor.chainmail-armor",
		EvasionAfter:            intPtr(8),
		MajorThresholdAfter:     intPtr(7),
		SevereThresholdAfter:    intPtr(15),
		ArmorScoreAfter:         intPtr(4),
		ArmorMaxAfter:           intPtr(4),
		SpellcastRollBonusAfter: intPtr(1),
		AgilityAfter:            intPtr(0),
		StrengthAfter:           intPtr(0),
		FinesseAfter:            intPtr(0),
		InstinctAfter:           intPtr(0),
		PresenceAfter:           intPtr(0),
		KnowledgeAfter:          intPtr(0),
		ArmorAfter:              intPtr(4),
		StressCost:              2,
	})
	if err != nil {
		t.Fatalf("foldEquipmentSwapped: %v", err)
	}

	profile := state.CharacterProfiles["char-1"]
	if profile.EquippedArmorID != "armor.chainmail-armor" ||
		profile.Evasion != 8 ||
		profile.MajorThreshold != 7 ||
		profile.SevereThreshold != 15 ||
		profile.ArmorScore != 4 ||
		profile.ArmorMax != 4 ||
		profile.SpellcastRollBonus != 1 {
		t.Fatalf("profile after fold = %+v", profile)
	}
	if profile.Agility != 0 || profile.Strength != 0 || profile.Finesse != 0 ||
		profile.Instinct != 0 || profile.Presence != 0 || profile.Knowledge != 0 {
		t.Fatalf("profile traits after fold = %+v", profile)
	}

	character := state.CharacterStates["char-1"]
	if character.Armor != 4 || character.Stress != 3 {
		t.Fatalf("state after fold = %+v, want armor=4 stress=3", character)
	}
}

func TestFolderApplyGoldUpdated_UpdatesProfileWhenPresent(t *testing.T) {
	projector := NewFolder()
	payload, err := json.Marshal(daggerheartpayload.GoldUpdatedPayload{
		CharacterID: "char-1",
		Handfuls:    3,
		Bags:        2,
		Chests:      1,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterProfiles: map[ids.CharacterID]daggerheartstate.CharacterProfile{
			"char-1": {},
		},
	}, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeGoldUpdated,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}

	snapshot := assertTestSnapshotState(t, updated)
	profile := snapshot.CharacterProfiles["char-1"]
	if profile.GoldHandfuls != 3 || profile.GoldBags != 2 || profile.GoldChests != 1 {
		t.Fatalf("gold profile = %+v, want handfuls=3 bags=2 chests=1", profile)
	}
}

func TestFolderApplyDomainCardAcquired_AppendsToProfileWhenPresent(t *testing.T) {
	projector := NewFolder()
	payload, err := json.Marshal(daggerheartpayload.DomainCardAcquiredPayload{
		CharacterID: "char-1",
		CardID:      "card-2",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterProfiles: map[ids.CharacterID]daggerheartstate.CharacterProfile{
			"char-1": {
				DomainCardIDs: []string{"card-1"},
			},
		},
	}, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeDomainCardAcquired,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}

	snapshot := assertTestSnapshotState(t, updated)
	profile := snapshot.CharacterProfiles["char-1"]
	if len(profile.DomainCardIDs) != 2 || profile.DomainCardIDs[0] != "card-1" || profile.DomainCardIDs[1] != "card-2" {
		t.Fatalf("domain cards = %v, want [card-1 card-2]", profile.DomainCardIDs)
	}
}

func TestFolderApplyRestTaken_RejectsOutOfRangeGMFear(t *testing.T) {
	projector := NewFolder()
	payload, err := json.Marshal(daggerheartpayload.RestTakenPayload{
		RestType:     "short",
		GMFear:       daggerheartstate.GMFearMax + 1,
		Participants: []ids.CharacterID{"char-1"},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	_, err = projector.Fold(daggerheartstate.SnapshotState{CampaignID: "camp-1"}, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeRestTaken,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err == nil {
		t.Fatal("expected gm fear range validation error")
	}
}

func TestFolderApplyHandlesAllRegisteredEvents(t *testing.T) {
	projector := NewFolder()
	for _, def := range daggerheartEventDefinitions {
		if def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		t.Run(string(def.Type), func(t *testing.T) {
			payloadJSON := []byte(`{}`)
			if def.Type == daggerheartpayload.EventTypeGMFearChanged {
				payload, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 2})
				if err != nil {
					t.Fatalf("marshal payload: %v", err)
				}
				payloadJSON = payload
			}

			updated, err := projector.Fold(daggerheartstate.SnapshotState{CampaignID: "camp-1", GMFear: 1}, event.Event{
				CampaignID:    "camp-1",
				Type:          def.Type,
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   payloadJSON,
			})
			if err != nil {
				t.Fatalf("projector apply %s: %v", def.Type, err)
			}
			assertTestSnapshotState(t, updated)
		})
	}
}

func TestFolderApply_RejectsAggregateState(t *testing.T) {
	// System folders should only receive their own state type, not the
	// full aggregate.State. The aggregate folder extracts the system-specific
	// state before calling RouteEvent.
	folder := NewFolder()
	aggState := aggregate.State{
		Systems: map[module.Key]any{
			{ID: SystemID, Version: SystemVersion}: daggerheartstate.SnapshotState{
				CampaignID: "camp-1",
				GMFear:     3,
			},
		},
	}
	_, err := folder.Fold(aggState, event.Event{
		CampaignID:    "camp-1",
		Type:          daggerheartpayload.EventTypeGMFearChanged,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"before":3,"after":5}`),
	})
	if err == nil {
		t.Fatal("expected error when passing aggregate.State to projector")
	}
}

func TestFolderApplyUnknownEventReturnsError(t *testing.T) {
	projector := NewFolder()
	_, err := projector.Fold(daggerheartstate.SnapshotState{CampaignID: "camp-1"}, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.unknown"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}
