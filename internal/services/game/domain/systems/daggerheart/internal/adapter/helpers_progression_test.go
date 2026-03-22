package adapter

import (
	"context"
	"errors"
	"strings"
	"testing"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestProfileAndHelperBranches(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if err := (*Adapter)(nil).PutCharacterProfile(ctx, "camp-1", "char-1", validCharacterProfile()); err == nil || !strings.Contains(err.Error(), "store is not configured") {
		t.Fatalf("nil PutCharacterProfile() error = %v, want store-not-configured", err)
	}

	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	if err := a.PutCharacterProfile(ctx, "camp-1", "char-1", daggerheartstate.CharacterProfile{}); err == nil || !strings.Contains(err.Error(), "validate daggerheart character profile") {
		t.Fatalf("PutCharacterProfile() validation error = %v, want profile validation error", err)
	}

	store.putCharacterProfileErr = errors.New("profile write failed")
	if err := a.PutCharacterProfile(ctx, "camp-1", "char-1", validCharacterProfile()); err == nil || !strings.Contains(err.Error(), "profile write failed") {
		t.Fatalf("PutCharacterProfile() put error = %v, want put error", err)
	}
	store.putCharacterProfileErr = nil

	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          2,
		Hope:        1,
		HopeMax:     6,
		Stress:      1,
		Armor:       1,
	}
	if err := a.PutCharacterProfile(ctx, "camp-1", "char-1", validCharacterProfile()); err != nil {
		t.Fatalf("PutCharacterProfile() with existing state returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")]; got.Hp != 2 || got.Hope != 1 {
		t.Fatalf("existing state after PutCharacterProfile() = %+v, want unchanged existing state", got)
	}

	delete(store.states, profileKey("camp-1", "char-2"))
	if err := a.PutCharacterProfile(ctx, "camp-1", "char-2", validCharacterProfileWithCompanion()); err != nil {
		t.Fatalf("PutCharacterProfile() with companion returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-2")].CompanionState; got == nil || got.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("companion state after PutCharacterProfile() = %+v, want present companion", got)
	}

	if got := CompanionProjectionStateFromProfile(validCharacterProfile()); got != nil {
		t.Fatalf("CompanionProjectionStateFromProfile() = %+v, want nil without companion sheet", got)
	}

	store.putSnapshotErr = errors.New("snapshot write failed")
	if err := a.PutSnapshot(ctx, "camp-1", 2, 1); err == nil || !strings.Contains(err.Error(), "put daggerheart snapshot: snapshot write failed") {
		t.Fatalf("PutSnapshot() error = %v, want wrapped snapshot error", err)
	}
	store.putSnapshotErr = nil

	store.getSnapshotErr = errors.New("snapshot read failed")
	if got := a.SnapshotShortRests(ctx, "camp-1"); got != 0 {
		t.Fatalf("SnapshotShortRests() with read error = %d, want 0", got)
	}
	store.getSnapshotErr = nil

	store.putCharacterStateErr = errors.New("state write failed")
	if err := a.PutCharacterState(ctx, projectionstore.DaggerheartCharacterState{}); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("PutCharacterState() error = %v, want wrapped state error", err)
	}
	store.putCharacterStateErr = nil

	fallbackArmor, err := a.CharacterArmorMax(ctx, projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "missing",
		Armor:       3,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Amount: 1},
		},
	})
	if err != nil {
		t.Fatalf("CharacterArmorMax() fallback returned error: %v", err)
	}
	if fallbackArmor != 2 {
		t.Fatalf("CharacterArmorMax() fallback = %d, want 2", fallbackArmor)
	}

	store.getCharacterProfileErr = errors.New("profile read failed")
	if _, err := a.CharacterArmorMax(ctx, projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
		t.Fatalf("CharacterArmorMax() error = %v, want wrapped get error", err)
	}
	store.getCharacterProfileErr = nil

	if err := a.ClearRestTemporaryArmor(ctx, "camp-1", "missing", true, true); err != nil {
		t.Fatalf("ClearRestTemporaryArmor() missing state returned error: %v", err)
	}

	store.states[profileKey("camp-1", "char-3")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-3",
		Armor:       4,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Source: "spell", Duration: "short_rest", Amount: 1},
		},
	}
	store.putCharacterStateErr = errors.New("state write failed")
	if err := a.ClearRestTemporaryArmor(ctx, "camp-1", "char-3", true, false); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("ClearRestTemporaryArmor() error = %v, want wrapped state write error", err)
	}
	store.putCharacterStateErr = nil

	store.states[profileKey("camp-1", "char-4")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-4",
		StatModifiers: []projectionstore.DaggerheartStatModifier{{
			ID:            "mod-session",
			Target:        string(rules.StatModifierTargetEvasion),
			Delta:         1,
			ClearTriggers: []string{string(rules.ConditionClearTriggerSessionEnd)},
		}},
	}
	store.putCharacterStateErr = errors.New("should not write")
	if err := a.ClearRestStatModifiers(ctx, "camp-1", "char-4", true, true); err != nil {
		t.Fatalf("ClearRestStatModifiers() no-change returned error: %v", err)
	}
	store.putCharacterStateErr = nil

	store.getCharacterProfileErr = errors.New("profile read failed")
	if err := a.ApplyStatePatch(ctx, "camp-1", "char-1", StatePatch{HP: intPtr(4)}); err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
		t.Fatalf("ApplyStatePatch() error = %v, want wrapped profile read error", err)
	}
	store.getCharacterProfileErr = nil

	store.putCharacterStateErr = errors.New("state write failed")
	if err := a.ApplyConditionPatch(ctx, "camp-1", "char-1", []rules.ConditionState{mustStandardCondition(t, rules.ConditionHidden)}); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("ApplyConditionPatch() error = %v, want wrapped state write error", err)
	}
	store.putCharacterStateErr = nil

	if got := SubclassStateFromProjection(nil); got != nil {
		t.Fatalf("SubclassStateFromProjection(nil) = %#v, want nil", got)
	}
	if got := CompanionStateFromProjection(nil); got != nil {
		t.Fatalf("CompanionStateFromProjection(nil) = %#v, want nil", got)
	}
	if got := StatModifiersFromProjection(nil); got != nil {
		t.Fatalf("StatModifiersFromProjection(nil) = %#v, want nil", got)
	}
}

func TestProgressionAndCharacterHandlerBranches(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = validCharacterProfile().ToStorage("camp-1", "char-1")
	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       4,
	}
	a := NewAdapter(store, func(profile *daggerheartstate.CharacterProfile, _ payload.LevelUpAppliedPayload) {
		profile.Level++
	})

	store.putSnapshotErr = errors.New("snapshot write failed")
	if err := a.HandleRestTaken(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.RestTakenPayload{
		GMFear:       2,
		ShortRests:   1,
		Participants: []ids.CharacterID{ids.CharacterID("char-1")},
	}); err == nil || !strings.Contains(err.Error(), "put daggerheart snapshot: snapshot write failed") {
		t.Fatalf("HandleRestTaken() error = %v, want wrapped snapshot error", err)
	}
	store.putSnapshotErr = nil

	store.getCharacterProfileErr = storage.ErrNotFound
	if err := a.HandleLevelUpApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.LevelUpAppliedPayload{
		CharacterID: ids.CharacterID("missing"),
	}); err != nil {
		t.Fatalf("HandleLevelUpApplied() not-found returned error: %v", err)
	}
	store.getCharacterProfileErr = errors.New("profile read failed")
	if err := a.HandleLevelUpApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.LevelUpAppliedPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character profile for level-up: profile read failed") {
		t.Fatalf("HandleLevelUpApplied() error = %v, want wrapped get error", err)
	}
	store.getCharacterProfileErr = nil

	store.putCharacterProfileErr = errors.New("profile write failed")
	if err := a.HandleGoldUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.GoldUpdatedPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err == nil || !strings.Contains(err.Error(), "profile write failed") {
		t.Fatalf("HandleGoldUpdated() put error = %v, want put error", err)
	}
	store.putCharacterProfileErr = nil

	store.getCharacterProfileErr = storage.ErrNotFound
	if err := a.HandleGoldUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.GoldUpdatedPayload{
		CharacterID: ids.CharacterID("missing"),
	}); err != nil {
		t.Fatalf("HandleGoldUpdated() not-found returned error: %v", err)
	}
	store.getCharacterProfileErr = errors.New("profile read failed")
	if err := a.HandleGoldUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.GoldUpdatedPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character profile for gold update: profile read failed") {
		t.Fatalf("HandleGoldUpdated() get error = %v, want wrapped get error", err)
	}
	store.getCharacterProfileErr = nil

	store.getCharacterProfileErr = storage.ErrNotFound
	if err := a.HandleDomainCardAcquired(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.DomainCardAcquiredPayload{
		CharacterID: ids.CharacterID("missing"),
	}); err != nil {
		t.Fatalf("HandleDomainCardAcquired() not-found returned error: %v", err)
	}
	store.getCharacterProfileErr = errors.New("profile read failed")
	if err := a.HandleDomainCardAcquired(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.DomainCardAcquiredPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character profile for domain card acquire: profile read failed") {
		t.Fatalf("HandleDomainCardAcquired() get error = %v, want wrapped get error", err)
	}
	store.getCharacterProfileErr = nil

	if err := a.HandleEquipmentSwapped(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EquipmentSwappedPayload{}); err != nil {
		t.Fatalf("HandleEquipmentSwapped() blank character returned error: %v", err)
	}

	store.getCharacterProfileErr = errors.New("profile read failed")
	if err := a.HandleEquipmentSwapped(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EquipmentSwappedPayload{
		CharacterID: ids.CharacterID("char-1"),
		ItemType:    "armor",
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character profile for equipment swap: profile read failed") {
		t.Fatalf("HandleEquipmentSwapped() get error = %v, want wrapped get error", err)
	}
	store.getCharacterProfileErr = nil

	store.putCharacterProfileErr = errors.New("profile write failed")
	if err := a.HandleEquipmentSwapped(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EquipmentSwappedPayload{
		CharacterID:     ids.CharacterID("char-1"),
		ItemType:        "armor",
		EquippedArmorID: "armor.full-plate",
	}); err == nil || !strings.Contains(err.Error(), "put daggerheart character profile for equipment swap: profile write failed") {
		t.Fatalf("HandleEquipmentSwapped() put error = %v, want wrapped put error", err)
	}
	store.putCharacterProfileErr = nil

	delete(store.profiles, profileKey("camp-1", "missing"))
	if err := a.HandleEquipmentSwapped(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EquipmentSwappedPayload{
		CharacterID: ids.CharacterID("missing"),
		ItemType:    "armor",
		StressCost:  2,
	}); err != nil {
		t.Fatalf("HandleEquipmentSwapped() missing profile returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "missing")].Stress; got != 2 {
		t.Fatalf("missing-profile equipment swap stress = %d, want 2", got)
	}

	store.getCountdownErr = errors.New("countdown read failed")
	if err := a.HandleCountdownUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.CountdownUpdatedPayload{
		CountdownID: dhids.CountdownID("count-1"),
		Value:       2,
	}); err == nil || !strings.Contains(err.Error(), "countdown read failed") {
		t.Fatalf("HandleCountdownUpdated() get error = %v, want get error", err)
	}
	store.getCountdownErr = nil
	store.countdowns[profileKey("camp-1", "count-1")] = projectionstore.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "count-1",
		Current:     1,
		Max:         4,
		Direction:   "up",
	}
	if err := a.HandleCountdownUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.CountdownUpdatedPayload{
		CountdownID: dhids.CountdownID("count-1"),
		Value:       5,
	}); err == nil {
		t.Fatal("HandleCountdownUpdated() error = nil, want validation error")
	}

	if err := a.HandleDowntimeMoveApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.DowntimeMoveAppliedPayload{}); err != nil {
		t.Fatalf("HandleDowntimeMoveApplied() blank actors returned error: %v", err)
	}

	store.getCharacterStateErr = errors.New("state read failed")
	if err := a.HandleCharacterTemporaryArmorApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.CharacterTemporaryArmorAppliedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Source:      "spell",
		Duration:    "short_rest",
		Amount:      1,
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character state: state read failed") {
		t.Fatalf("HandleCharacterTemporaryArmorApplied() error = %v, want wrapped state read error", err)
	}

	if err := a.HandleBeastformTransformed(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.BeastformTransformedPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character state: state read failed") {
		t.Fatalf("HandleBeastformTransformed() error = %v, want wrapped state read error", err)
	}

	if err := a.HandleBeastformDropped(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.BeastformDroppedPayload{
		CharacterID: ids.CharacterID("char-1"),
	}); err == nil || !strings.Contains(err.Error(), "get daggerheart character state: state read failed") {
		t.Fatalf("HandleBeastformDropped() error = %v, want wrapped state read error", err)
	}
	store.getCharacterStateErr = nil

	delete(store.states, profileKey("camp-1", "statless"))
	if err := a.HandleStatModifierChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.StatModifierChangedPayload{
		CharacterID: ids.CharacterID("statless"),
		Modifiers: []rules.StatModifierState{{
			ID:     "mod-1",
			Target: rules.StatModifierTargetEvasion,
			Delta:  1,
		}},
	}); err != nil {
		t.Fatalf("HandleStatModifierChanged() statless returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "statless")].StatModifiers; len(got) != 1 || got[0].ID != "mod-1" {
		t.Fatalf("stat modifiers after HandleStatModifierChanged() = %#v, want seeded modifier", got)
	}
}

func TestEquipmentSwapUpdatesAllDerivedArmorFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = validCharacterProfile().ToStorage("camp-1", "char-1")
	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       4,
	}
	a := NewAdapter(store, nil)

	evasionAfter := 11
	majorAfter := 9
	severeAfter := 14
	armorScoreAfter := 5
	armorMaxAfter := 5
	spellcastRollBonusAfter := 2
	agilityAfter := 1
	strengthAfter := 2
	finesseAfter := 3
	instinctAfter := 4
	presenceAfter := 5
	knowledgeAfter := 6

	if err := a.HandleEquipmentSwapped(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EquipmentSwappedPayload{
		CharacterID:             ids.CharacterID("char-1"),
		ItemType:                "armor",
		EquippedArmorID:         "armor.scale-mail",
		EvasionAfter:            &evasionAfter,
		MajorThresholdAfter:     &majorAfter,
		SevereThresholdAfter:    &severeAfter,
		ArmorScoreAfter:         &armorScoreAfter,
		ArmorMaxAfter:           &armorMaxAfter,
		SpellcastRollBonusAfter: &spellcastRollBonusAfter,
		AgilityAfter:            &agilityAfter,
		StrengthAfter:           &strengthAfter,
		FinesseAfter:            &finesseAfter,
		InstinctAfter:           &instinctAfter,
		PresenceAfter:           &presenceAfter,
		KnowledgeAfter:          &knowledgeAfter,
	}); err != nil {
		t.Fatalf("HandleEquipmentSwapped() returned error: %v", err)
	}

	got := store.profiles[profileKey("camp-1", "char-1")]
	if got.EquippedArmorID != "armor.scale-mail" ||
		got.Evasion != 11 ||
		got.MajorThreshold != 9 ||
		got.SevereThreshold != 14 ||
		got.ArmorScore != 5 ||
		got.ArmorMax != 5 ||
		got.SpellcastRollBonus != 2 ||
		got.Agility != 1 ||
		got.Strength != 2 ||
		got.Finesse != 3 ||
		got.Instinct != 4 ||
		got.Presence != 5 ||
		got.Knowledge != 6 {
		t.Fatalf("profile after HandleEquipmentSwapped() = %+v, want all derived armor fields updated", got)
	}
}

func TestRestAndModifierHandlersWrapWriteErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = validCharacterProfile().ToStorage("camp-1", "char-1")
	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       4,
		StatModifiers: []projectionstore.DaggerheartStatModifier{{
			ID:            "mod-short",
			Target:        string(rules.StatModifierTargetEvasion),
			Delta:         1,
			ClearTriggers: []string{string(rules.ConditionClearTriggerShortRest)},
		}},
	}
	a := NewAdapter(store, nil)

	store.putCharacterStateErr = errors.New("state write failed")
	if err := a.ClearRestStatModifiers(ctx, "camp-1", "char-1", true, false); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("ClearRestStatModifiers() error = %v, want wrapped state write error", err)
	}

	if err := a.HandleRestTaken(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.RestTakenPayload{
		GMFear:       2,
		ShortRests:   1,
		RefreshRest:  true,
		Participants: []ids.CharacterID{ids.CharacterID("char-1")},
	}); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("HandleRestTaken() error = %v, want wrapped state write error", err)
	}

	if err := a.HandleStatModifierChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.StatModifierChangedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Modifiers: []rules.StatModifierState{{
			ID:     "mod-2",
			Target: rules.StatModifierTargetArmorScore,
			Delta:  1,
		}},
	}); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("HandleStatModifierChanged() error = %v, want wrapped state write error", err)
	}
	store.putCharacterStateErr = nil
}

func TestCharacterTemporaryArmorWrapsPutErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = validCharacterProfile().ToStorage("camp-1", "char-1")
	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     6,
		Stress:      1,
		Armor:       4,
	}
	store.putCharacterStateErr = errors.New("state write failed")
	a := NewAdapter(store, nil)

	if err := a.HandleCharacterTemporaryArmorApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.CharacterTemporaryArmorAppliedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Source:      "spell",
		Duration:    "short_rest",
		Amount:      1,
	}); err == nil || !strings.Contains(err.Error(), "put daggerheart character state: state write failed") {
		t.Fatalf("HandleCharacterTemporaryArmorApplied() error = %v, want wrapped state write error", err)
	}
}

func validCharacterProfile() daggerheartstate.CharacterProfile {
	return daggerheartstate.CharacterProfile{
		Level:           1,
		HpMax:           7,
		StressMax:       6,
		Evasion:         9,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
		ArmorScore:      4,
		StartingArmorID: "armor.chainmail-armor",
	}
}

func validCharacterProfileWithCompanion() daggerheartstate.CharacterProfile {
	profile := validCharacterProfile()
	profile.CompanionSheet = &daggerheartstate.CharacterCompanionSheet{
		AnimalKind: "wolf",
		Name:       "Ash",
		Evasion:    daggerheartstate.CompanionSheetDefaultEvasion,
		Experiences: []daggerheartstate.CharacterCompanionExperience{
			{ExperienceID: "experience.guard", Modifier: daggerheartstate.CompanionSheetExperienceModifier},
			{ExperienceID: "experience.hunt", Modifier: daggerheartstate.CompanionSheetExperienceModifier},
		},
		AttackDescription: "Bite",
		AttackRange:       daggerheartstate.CompanionSheetDefaultAttackRange,
		DamageDieSides:    daggerheartstate.CompanionSheetDefaultDamageDieSides,
		DamageType:        daggerheartstate.CompanionDamageTypePhysical,
	}
	return profile
}
