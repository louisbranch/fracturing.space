package daggerheart

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCharacterProfileFromStorage_CopiesAllFields(t *testing.T) {
	stored := projectionstore.DaggerheartCharacterProfile{
		CampaignID:      "camp-1",
		CharacterID:     "char-1",
		Level:           2,
		HpMax:           7,
		StressMax:       6,
		Evasion:         11,
		MajorThreshold:  3,
		SevereThreshold: 6,
		Proficiency:     2,
		ArmorScore:      1,
		ArmorMax:        2,
		Experiences:     []projectionstore.DaggerheartExperience{{Name: "Scout", Modifier: 2}},
		Agility:         2,
		Strength:        1,
		Finesse:         0,
		Instinct:        0,
		Presence:        -1,
		Knowledge:       1,
		ClassID:         "class.guardian",
		SubclassID:      "subclass.stalwart",
		Heritage: projectionstore.DaggerheartHeritageSelection{
			FirstFeatureAncestryID:  "ancestry.human",
			FirstFeatureID:          "ancestry.human.feature-1",
			SecondFeatureAncestryID: "ancestry.human",
			SecondFeatureID:         "ancestry.human.feature-2",
			CommunityID:             "community.highborne",
		},
		TraitsAssigned:       true,
		DetailsRecorded:      true,
		StartingWeaponIDs:    []string{"weapon.longsword"},
		StartingArmorID:      "armor.gambeson-armor",
		StartingPotionItemID: "item.minor-health-potion",
		Background:           "Former sentinel",
		Description:          "Calm and relentless.",
		DomainCardIDs:        []string{"domain-card.valor-bare-bones"},
		Connections:          "Owes the guard captain a favor",
		GoldHandfuls:         1,
		GoldBags:             2,
		GoldChests:           3,
	}

	profile := daggerheartstate.CharacterProfileFromStorage(stored)

	if profile.Level != stored.Level || profile.Description != stored.Description || profile.Connections != stored.Connections {
		t.Fatalf("profile fields not copied: %+v", profile)
	}
	if len(profile.Experiences) != 1 || profile.Experiences[0].Name != "Scout" || profile.Experiences[0].Modifier != 2 {
		t.Fatalf("profile experiences = %+v, want copied experience", profile.Experiences)
	}
	if len(profile.StartingWeaponIDs) != 1 || profile.StartingWeaponIDs[0] != "weapon.longsword" {
		t.Fatalf("starting weapons = %+v, want copied slice", profile.StartingWeaponIDs)
	}
	if len(profile.DomainCardIDs) != 1 || profile.DomainCardIDs[0] != "domain-card.valor-bare-bones" {
		t.Fatalf("domain cards = %+v, want copied slice", profile.DomainCardIDs)
	}

	stored.Experiences[0].Name = "Changed"
	stored.StartingWeaponIDs[0] = "weapon.shortsword"
	stored.DomainCardIDs[0] = "domain-card.changed"
	if profile.Experiences[0].Name != "Scout" {
		t.Fatalf("profile experience mutated with storage slice: %+v", profile.Experiences)
	}
	if profile.StartingWeaponIDs[0] != "weapon.longsword" {
		t.Fatalf("profile starting weapons mutated with storage slice: %+v", profile.StartingWeaponIDs)
	}
	if profile.DomainCardIDs[0] != "domain-card.valor-bare-bones" {
		t.Fatalf("profile domain cards mutated with storage slice: %+v", profile.DomainCardIDs)
	}
}

func TestModuleCharacterReady_ReportsInvalidStateAndMissingProfile(t *testing.T) {
	systemModule := NewModule()

	_, err := systemModule.BindCharacterReadiness("camp-1", map[module.Key]any{
		{ID: SystemID, Version: SystemVersion}: struct{}{},
	})
	if err == nil {
		t.Fatal("expected invalid state bind error")
	}

	evaluator, err := systemModule.BindCharacterReadiness("camp-1", map[module.Key]any{
		{ID: SystemID, Version: SystemVersion}: daggerheartstate.SnapshotState{},
	})
	if err != nil {
		t.Fatalf("BindCharacterReadiness() error = %v", err)
	}
	ready, reason := evaluator.CharacterReady(character.State{CharacterID: "char-1"})
	if ready || reason != "daggerheart profile is missing" {
		t.Fatalf("missing profile readiness = (%t, %q), want false and missing-profile reason", ready, reason)
	}
}

func TestDecideCharacterProfileReplace_NormalizesCharacterIDAndProfile(t *testing.T) {
	now := func() time.Time { return time.Date(2026, time.March, 9, 15, 4, 5, 0, time.FixedZone("EST", -5*60*60)) }
	payloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileReplacePayload{
		CharacterID: ids.CharacterID(" char-1 "),
		Profile: daggerheartstate.CharacterProfile{
			Level:           0,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        1,
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	decision := daggerheartdecider.DecideCharacterProfileReplace(command.Command{
		CampaignID:    ids.CampaignID("camp-1"),
		Type:          daggerheartdecider.CommandTypeCharacterProfileReplace,
		ActorType:     command.ActorTypeParticipant,
		ActorID:       "user-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, now)
	if err := decision.Validate(); err != nil {
		t.Fatalf("decision validate: %v", err)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("decision events = %d, want 1", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != daggerheartpayload.EventTypeCharacterProfileReplaced || evt.EntityID != "char-1" {
		t.Fatalf("event envelope = %+v, want trimmed character profile replaced event", evt)
	}
	if !evt.Timestamp.Equal(now().UTC()) {
		t.Fatalf("event timestamp = %v, want %v", evt.Timestamp, now().UTC())
	}

	var payload daggerheartstate.CharacterProfileReplacedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal emitted payload: %v", err)
	}
	if payload.CharacterID != ids.CharacterID("char-1") {
		t.Fatalf("payload character_id = %q, want trimmed char-1", payload.CharacterID)
	}
	if payload.Profile.Level != 1 {
		t.Fatalf("payload profile level = %d, want normalized default 1", payload.Profile.Level)
	}
}

func TestFoldCharacterProfileReplaced_NormalizesLevelAndSeedsState(t *testing.T) {
	folder := NewFolder()

	payloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileReplacedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Profile: daggerheartstate.CharacterProfile{
			Level:           0,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        1,
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	folded, err := folder.Fold(daggerheartstate.NewSnapshotState("camp-1"), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          daggerheartpayload.EventTypeCharacterProfileReplaced,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}

	state := assertTestSnapshotState(t, folded)
	profile := state.CharacterProfiles[ids.CharacterID("char-1")]
	if profile.Level != 1 {
		t.Fatalf("folded profile level = %d, want normalized default 1", profile.Level)
	}
	characterState := state.CharacterStates[ids.CharacterID("char-1")]
	if characterState.HP != 6 {
		t.Fatalf("folded character hp = %d, want seeded hp_max 6", characterState.HP)
	}
}

func TestAdapterAndFolder_LevelUpAppliedStayInParity(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	folder := NewFolder()

	profileJSON, err := json.Marshal(daggerheartstate.CharacterProfileReplacedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Profile: daggerheartstate.CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  6,
			SevereThreshold: 10,
			Proficiency:     1,
			ArmorScore:      1,
			ArmorMax:        1,
		},
	})
	if err != nil {
		t.Fatalf("marshal profile payload: %v", err)
	}
	replaceEvent := event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          daggerheartpayload.EventTypeCharacterProfileReplaced,
		PayloadJSON:   profileJSON,
	}
	if err := adapter.Apply(context.Background(), replaceEvent); err != nil {
		t.Fatalf("apply replace: %v", err)
	}
	folded, err := folder.Fold(daggerheartstate.NewSnapshotState("camp-1"), replaceEvent)
	if err != nil {
		t.Fatalf("fold replace: %v", err)
	}

	levelUpJSON, err := json.Marshal(daggerheartpayload.LevelUpAppliedPayload{
		CharacterID:    ids.CharacterID("char-1"),
		Level:          2,
		ThresholdDelta: 1,
		Advancements: []daggerheartpayload.LevelUpAdvancementPayload{
			{Type: "add_hp_slots"},
			{Type: "add_stress_slots"},
		},
	})
	if err != nil {
		t.Fatalf("marshal level up payload: %v", err)
	}
	levelUpEvent := event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          daggerheartpayload.EventTypeLevelUpApplied,
		PayloadJSON:   levelUpJSON,
	}
	if err := adapter.Apply(context.Background(), levelUpEvent); err != nil {
		t.Fatalf("apply level up: %v", err)
	}
	folded, err = folder.Fold(folded, levelUpEvent)
	if err != nil {
		t.Fatalf("fold level up: %v", err)
	}

	folderState := canonicalizeSnapshotForParity(assertTestSnapshotState(t, folded))
	adapterState := canonicalizeSnapshotForParity(store.snapshotState("camp-1"))
	if folderState.CharacterProfiles[ids.CharacterID("char-1")].SevereThreshold != 12 {
		t.Fatalf("folded severe threshold = %d, want 12", folderState.CharacterProfiles[ids.CharacterID("char-1")].SevereThreshold)
	}
	if folderState.CharacterProfiles[ids.CharacterID("char-1")].MajorThreshold != 7 {
		t.Fatalf("folded major threshold = %d, want 7", folderState.CharacterProfiles[ids.CharacterID("char-1")].MajorThreshold)
	}
	if !reflect.DeepEqual(folderState, adapterState) {
		t.Fatalf("level-up parity mismatch\nfolder=%#v\nadapter=%#v", folderState, adapterState)
	}
}

func TestDecideCharacterProfileDelete_TrimsReasonAndPreservesActorType(t *testing.T) {
	now := func() time.Time { return time.Date(2026, time.March, 9, 15, 5, 6, 0, time.UTC) }
	payloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileDeletePayload{
		CharacterID: ids.CharacterID("char-1"),
		Reason:      "  reset workflow  ",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	decision := daggerheartdecider.DecideCharacterProfileDelete(command.Command{
		CampaignID:    ids.CampaignID("camp-1"),
		Type:          daggerheartdecider.CommandTypeCharacterProfileDelete,
		ActorType:     command.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, now)
	if err := decision.Validate(); err != nil {
		t.Fatalf("decision validate: %v", err)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("decision events = %d, want 1", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != daggerheartpayload.EventTypeCharacterProfileDeleted || evt.ActorType != event.ActorType(command.ActorTypeGM) {
		t.Fatalf("event envelope = %+v, want character profile deleted with GM actor type", evt)
	}

	var payload daggerheartstate.CharacterProfileDeletedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal emitted payload: %v", err)
	}
	if payload.Reason != "reset workflow" {
		t.Fatalf("payload reason = %q, want trimmed reason", payload.Reason)
	}
}

func TestDecideCharacterProfileCommands_RejectBlankCharacterID(t *testing.T) {
	replacePayloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileReplacePayload{
		CharacterID: ids.CharacterID("   "),
		Profile: daggerheartstate.CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        1,
		},
	})
	if err != nil {
		t.Fatalf("marshal replace payload: %v", err)
	}

	replaceDecision := daggerheartdecider.DecideCharacterProfileReplace(command.Command{
		Type:        daggerheartdecider.CommandTypeCharacterProfileReplace,
		PayloadJSON: replacePayloadJSON,
	}, time.Now)
	if len(replaceDecision.Rejections) != 1 || replaceDecision.Rejections[0].Code != daggerheartdecider.RejectionCodePayloadDecodeFailed {
		t.Fatalf("replace rejection = %+v, want payload decode failure", replaceDecision.Rejections)
	}
	if replaceDecision.Rejections[0].Message != "character_id is required" {
		t.Fatalf("replace rejection message = %q, want character_id is required", replaceDecision.Rejections[0].Message)
	}

	deletePayloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileDeletePayload{CharacterID: ids.CharacterID(" ")})
	if err != nil {
		t.Fatalf("marshal delete payload: %v", err)
	}

	deleteDecision := daggerheartdecider.DecideCharacterProfileDelete(command.Command{
		Type:        daggerheartdecider.CommandTypeCharacterProfileDelete,
		PayloadJSON: deletePayloadJSON,
	}, time.Now)
	if len(deleteDecision.Rejections) != 1 || deleteDecision.Rejections[0].Code != daggerheartdecider.RejectionCodePayloadDecodeFailed {
		t.Fatalf("delete rejection = %+v, want payload decode failure", deleteDecision.Rejections)
	}
	if deleteDecision.Rejections[0].Message != "character_id is required" {
		t.Fatalf("delete rejection message = %q, want character_id is required", deleteDecision.Rejections[0].Message)
	}
}

func TestApplyCharacterProfileEvents_FallBackToEntityIDWhenPayloadCharacterIDMissing(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)

	replacePayloadJSON, err := json.Marshal(daggerheartstate.CharacterProfileReplacedPayload{
		Profile: daggerheartstate.CharacterProfile{
			Level:           1,
			HpMax:           6,
			StressMax:       6,
			Evasion:         10,
			MajorThreshold:  3,
			SevereThreshold: 6,
			Proficiency:     1,
			ArmorScore:      0,
			ArmorMax:        1,
		},
	})
	if err != nil {
		t.Fatalf("marshal replace payload: %v", err)
	}

	if err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-from-event",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          daggerheartpayload.EventTypeCharacterProfileReplaced,
		PayloadJSON:   replacePayloadJSON,
	}); err != nil {
		t.Fatalf("apply replace fallback: %v", err)
	}

	if _, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-from-event"); err != nil {
		t.Fatalf("get fallback profile: %v", err)
	}

	if err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-from-event",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          daggerheartpayload.EventTypeCharacterProfileDeleted,
		PayloadJSON:   []byte(`{}`),
	}); err != nil {
		t.Fatalf("apply delete fallback: %v", err)
	}

	if _, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-from-event"); err != storage.ErrNotFound {
		t.Fatalf("get deleted fallback profile error = %v, want %v", err, storage.ErrNotFound)
	}
}
