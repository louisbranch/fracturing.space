package charactermutationtransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type contentStoreStub struct {
	armors     map[string]contentstore.DaggerheartArmor
	classes    map[string]contentstore.DaggerheartClass
	subclasses map[string]contentstore.DaggerheartSubclass
	beastforms map[string]contentstore.DaggerheartBeastformEntry
	armorErr   error
	beastErr   error
	classErr   error
	subErr     error
}

func (stub contentStoreStub) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	if stub.armorErr != nil {
		return contentstore.DaggerheartArmor{}, stub.armorErr
	}
	return stub.armors[id], nil
}

func (stub contentStoreStub) GetDaggerheartBeastform(_ context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	if stub.beastErr != nil {
		return contentstore.DaggerheartBeastformEntry{}, stub.beastErr
	}
	return stub.beastforms[id], nil
}

func (stub contentStoreStub) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	if stub.classErr != nil {
		return contentstore.DaggerheartClass{}, stub.classErr
	}
	return stub.classes[id], nil
}

func (stub contentStoreStub) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	if stub.subErr != nil {
		return contentstore.DaggerheartSubclass{}, stub.subErr
	}
	return stub.subclasses[id], nil
}

type statefulDaggerheartStore struct {
	profiles    map[string]projectionstore.DaggerheartCharacterProfile
	stateSeqs   map[string][]projectionstore.DaggerheartCharacterState
	adversaries map[string]projectionstore.DaggerheartAdversary
	stateCalls  map[string]int
}

func (stub *statefulDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, _ string, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	profile, ok := stub.profiles[characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.NotFound, "profile missing")
	}
	return profile, nil
}

func (stub *statefulDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, _ string, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	seq := stub.stateSeqs[characterID]
	if len(seq) == 0 {
		return projectionstore.DaggerheartCharacterState{}, status.Error(codes.NotFound, "state missing")
	}
	if stub.stateCalls == nil {
		stub.stateCalls = make(map[string]int)
	}
	idx := stub.stateCalls[characterID]
	if idx >= len(seq) {
		idx = len(seq) - 1
	}
	stub.stateCalls[characterID]++
	return seq[idx], nil
}

func (stub *statefulDaggerheartStore) GetDaggerheartAdversary(_ context.Context, _ string, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	adversary, ok := stub.adversaries[adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, status.Error(codes.NotFound, "adversary missing")
	}
	return adversary, nil
}

func TestClassStateProjectionHelpers(t *testing.T) {
	projected := classStateFromProjection(projectionstore.DaggerheartClassState{
		AttackBonusUntilRest:       -1,
		FocusTargetID:              " adv-1 ",
		StrangePatternsNumber:      -3,
		RallyDice:                  []int{0, 6, 8},
		PrayerDice:                 []int{-1, 10},
		Unstoppable:                projectionstore.DaggerheartUnstoppableState{CurrentValue: -2, DieSides: -4},
		ActiveBeastform:            &projectionstore.DaggerheartActiveBeastformState{BeastformID: " wolf ", DamageDice: []projectionstore.DaggerheartDamageDie{{Count: 1, Sides: 8}}},
		DifficultyPenaltyUntilRest: 1,
	})
	if projected.AttackBonusUntilRest != 0 || projected.FocusTargetID != "adv-1" || projected.DifficultyPenaltyUntilRest != 0 {
		t.Fatalf("classStateFromProjection() = %#v", projected)
	}
	if len(projected.RallyDice) != 2 || projected.RallyDice[0] != 6 {
		t.Fatalf("rally dice = %#v", projected.RallyDice)
	}
	if projected.ActiveBeastform == nil || projected.ActiveBeastform.BeastformID != "wolf" {
		t.Fatalf("active beastform = %#v", projected.ActiveBeastform)
	}

	ptr := classStatePtr(daggerheartstate.CharacterClassState{FocusTargetID: " adv-2 "})
	if ptr == nil || ptr.FocusTargetID != "adv-2" {
		t.Fatalf("classStatePtr() = %#v", ptr)
	}

	if activeBeastformFromProjection(nil) != nil {
		t.Fatal("activeBeastformFromProjection(nil) = non-nil, want nil")
	}
}

func TestSubclassHelpers(t *testing.T) {
	subclass := subclassStateFromProjection(&projectionstore.DaggerheartSubclassState{
		GiftedPerformerEpicSongUses:       -1,
		TranscendenceActive:               false,
		TranscendenceTraitBonusTarget:     " Presence ",
		TranscendenceTraitBonusValue:      3,
		ElementalChannel:                  "LIGHTNING",
		NemesisTargetID:                   " adv-1 ",
		ContactsEverywhereUsesThisSession: -2,
	})
	if subclass.GiftedPerformerEpicSongUses != 0 || subclass.TranscendenceTraitBonusTarget != "" || subclass.ElementalChannel != "" || subclass.NemesisTargetID != "adv-1" {
		t.Fatalf("subclassStateFromProjection() = %#v", subclass)
	}

	ptr := subclassStatePtr(daggerheartstate.CharacterSubclassState{ElementalChannel: " FIRE "})
	if ptr == nil || ptr.ElementalChannel != "fire" {
		t.Fatalf("subclassStatePtr() = %#v", ptr)
	}

	profile := projectionstore.DaggerheartCharacterProfile{
		SubclassID: "subclass-primary",
		SubclassTracks: []projectionstore.DaggerheartSubclassTrack{
			{SubclassID: "subclass-primary", Rank: projectionstore.DaggerheartSubclassTrackRankSpecialization, Origin: projectionstore.DaggerheartSubclassTrackOriginPrimary},
			{SubclassID: "subclass-multi", Rank: projectionstore.DaggerheartSubclassTrackRankMastery, Origin: projectionstore.DaggerheartSubclassTrackOriginMulticlass},
		},
	}
	if !hasUnlockedSubclassRank(profile, "subclass-primary", "foundation") {
		t.Fatal("primary subclass foundation rank should be unlocked")
	}
	if !hasUnlockedSubclassRank(profile, "subclass-multi", "specialization") {
		t.Fatal("multiclass specialization rank should be unlocked")
	}
	if hasUnlockedSubclassRank(profile, "subclass-primary", "unknown") {
		t.Fatal("unknown rank should not be unlocked")
	}

	ids := uniqueTrimmedIDs([]string{" one ", "", "two", "one", " two "})
	if len(ids) != 2 || ids[0] != "one" || ids[1] != "two" {
		t.Fatalf("uniqueTrimmedIDs() = %#v", ids)
	}

	conditions := projectionConditionStatesToDomain([]projectionstore.DaggerheartConditionState{{
		ID:            "hidden",
		Class:         string(rules.ConditionClassStandard),
		Standard:      rules.ConditionHidden,
		Code:          "hidden",
		Label:         "Hidden",
		ClearTriggers: []string{string(rules.ConditionClearTriggerShortRest)},
	}})
	if len(conditions) != 1 || conditions[0].Label != "Hidden" || len(conditions[0].ClearTriggers) != 1 {
		t.Fatalf("projectionConditionStatesToDomain() = %#v", conditions)
	}

	normalized, added, err := addStandardConditionStateWithOptions(nil, rules.ConditionHidden, rules.WithConditionSource("spell", "spell-1"))
	if err != nil {
		t.Fatalf("addStandardConditionStateWithOptions() error = %v", err)
	}
	if len(normalized) != 1 || len(added) != 1 || added[0].Source != "spell" {
		t.Fatalf("normalized/added = %#v / %#v", normalized, added)
	}

	normalized, added, err = addStandardConditionState(nil, rules.ConditionVulnerable)
	if err != nil {
		t.Fatalf("addStandardConditionState() error = %v", err)
	}
	if len(normalized) != 1 || len(added) != 1 || added[0].Standard != rules.ConditionVulnerable {
		t.Fatalf("normalized/added = %#v / %#v", normalized, added)
	}
}

func TestLevelUpHelperConversionsAndProgression(t *testing.T) {
	if ptr := intPtr(7); ptr == nil || *ptr != 7 {
		t.Fatalf("intPtr() = %#v", ptr)
	}

	tracks := subclassTracksFromProjection([]projectionstore.DaggerheartSubclassTrack{{
		Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
		ClassID:    "class-1",
		SubclassID: "subclass-1",
		Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
	}})
	if len(tracks) != 1 || tracks[0].Origin != daggerheartstate.SubclassTrackOriginPrimary {
		t.Fatalf("subclassTracksFromProjection() = %#v", tracks)
	}

	h := newTestHandler(Dependencies{})
	profile := projectionstore.DaggerheartCharacterProfile{
		SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
			Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
			ClassID:    "class-1",
			SubclassID: "subclass-1",
			Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
		}},
	}

	next, bonuses, err := h.deriveLevelUpSubclassProgression(context.Background(), profile, []daggerheartpayload.LevelUpAdvancementPayload{{
		Type: "upgraded_subclass",
	}})
	if err != nil {
		t.Fatalf("deriveLevelUpSubclassProgression(upgraded_subclass) error = %v", err)
	}
	if len(next) != 1 || next[0].Rank != daggerheartstate.SubclassTrackRankSpecialization || bonuses != (daggerheartstate.SubclassStatBonuses{}) {
		t.Fatalf("next/bonuses = %#v / %#v", next, bonuses)
	}

	_, _, err = h.deriveLevelUpSubclassProgression(context.Background(), profile, []daggerheartpayload.LevelUpAdvancementPayload{{
		Type: "multiclass",
	}})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("multiclass missing payload status = %v", status.Code(err))
	}

	h = newTestHandler(Dependencies{
		Content: contentStoreStub{
			subclasses: map[string]contentstore.DaggerheartSubclass{
				"subclass-1": {
					ID:                 "subclass-1",
					FoundationFeatures: []contentstore.DaggerheartFeature{{Name: "Foundation"}},
					SpecializationFeatures: []contentstore.DaggerheartFeature{{
						Name: "Hardened",
						SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
							Kind:  contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus,
							Bonus: 2,
						},
					}},
				},
			},
		},
	})
	next, bonuses, err = h.deriveLevelUpSubclassProgression(context.Background(), profile, []daggerheartpayload.LevelUpAdvancementPayload{{
		Type: "upgraded_subclass",
	}})
	if err != nil {
		t.Fatalf("deriveLevelUpSubclassProgression(with content) error = %v", err)
	}
	if next[0].Rank != daggerheartstate.SubclassTrackRankSpecialization || bonuses.HpMaxDelta != 2 {
		t.Fatalf("next/bonuses = %#v / %#v", next, bonuses)
	}
}

func TestResolveClassFeaturePayload(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Content: contentStoreStub{
			classes: map[string]contentstore.DaggerheartClass{
				"class-guardian": {
					ID: "class-guardian",
					HopeFeature: contentstore.DaggerheartHopeFeature{
						HopeFeatureRule: &contentstore.DaggerheartHopeFeatureRule{Bonus: 2, HopeCost: 1},
					},
				},
			},
		},
	})

	profile := projectionstore.DaggerheartCharacterProfile{
		ClassID:     "class-guardian",
		ArmorMax:    3,
		Proficiency: 2,
		CharacterID: "char-1",
		CampaignID:  "camp-1",
	}
	state := projectionstore.DaggerheartCharacterState{Hope: 3, Armor: 1}
	classState := daggerheartstate.CharacterClassState{}

	payload, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, state, classState, &pb.DaggerheartApplyClassFeatureRequest{
		CharacterId: "char-1",
		Feature: &pb.DaggerheartApplyClassFeatureRequest_FrontlineTank{
			FrontlineTank: &pb.DaggerheartFrontlineTankFeature{},
		},
	})
	if err != nil {
		t.Fatalf("resolveClassFeaturePayload(frontline_tank) error = %v", err)
	}
	if payload.Feature != "frontline_tank" || len(payload.Targets) != 1 || *payload.Targets[0].HopeAfter != 2 || *payload.Targets[0].ArmorAfter != 3 {
		t.Fatalf("frontline tank payload = %#v", payload)
	}

	payload, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, state, classState, &pb.DaggerheartApplyClassFeatureRequest{
		CharacterId: "char-1",
		Feature: &pb.DaggerheartApplyClassFeatureRequest_RoguesDodge{
			RoguesDodge: &pb.DaggerheartRoguesDodgeFeature{},
		},
	})
	if err != nil {
		t.Fatalf("resolveClassFeaturePayload(rogues_dodge) error = %v", err)
	}
	if payload.Feature != "rogues_dodge" || payload.Targets[0].ClassStateAfter == nil || payload.Targets[0].ClassStateAfter.EvasionBonusUntilHitOrRest != 2 {
		t.Fatalf("rogues dodge payload = %#v", payload)
	}

	_, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, state, classState, &pb.DaggerheartApplyClassFeatureRequest{
		CharacterId: "char-1",
		Feature: &pb.DaggerheartApplyClassFeatureRequest_StrangePatternsChoice{
			StrangePatternsChoice: &pb.DaggerheartStrangePatternsChoice{Number: 13},
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("strange patterns status = %v", status.Code(err))
	}

	_, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{
		Unstoppable: daggerheartstate.CharacterUnstoppableState{Active: true},
	}, &pb.DaggerheartApplyClassFeatureRequest{
		CharacterId: "char-1",
		Feature: &pb.DaggerheartApplyClassFeatureRequest_Unstoppable{
			Unstoppable: &pb.DaggerheartUnstoppableFeature{},
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("unstoppable status = %v", status.Code(err))
	}
}

func TestResolveClassFeaturePayloadAdditionalBranches(t *testing.T) {
	store := &statefulDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-2": {CampaignID: "camp-1", CharacterID: "char-2", HpMax: 8},
			"char-3": {CampaignID: "camp-1", CharacterID: "char-3", HpMax: 6},
		},
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-2": {{
				CampaignID:  "camp-1",
				CharacterID: "char-2",
				Hp:          5,
				Hope:        2,
				HopeMax:     3,
			}},
			"char-3": {{
				CampaignID:  "camp-1",
				CharacterID: "char-3",
				Hp:          6,
				Hope:        1,
				HopeMax:     3,
			}},
		},
	}

	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		Content: contentStoreStub{
			classes: map[string]contentstore.DaggerheartClass{
				"class-guardian": {
					ID: "class-guardian",
					HopeFeature: contentstore.DaggerheartHopeFeature{
						HopeFeatureRule: &contentstore.DaggerheartHopeFeatureRule{Bonus: 2, HopeCost: 1},
					},
				},
				"class-empty": {ID: "class-empty"},
			},
		},
	})

	profile := projectionstore.DaggerheartCharacterProfile{
		ClassID:     "class-guardian",
		ArmorMax:    3,
		Proficiency: 2,
		CharacterID: "char-1",
		CampaignID:  "camp-1",
	}

	t.Run("frontline tank requires rule and enough hope", func(t *testing.T) {
		_, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", projectionstore.DaggerheartCharacterProfile{
			ClassID:     "class-empty",
			CharacterID: "char-1",
			CampaignID:  "camp-1",
		}, projectionstore.DaggerheartCharacterState{Hope: 2}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_FrontlineTank{
				FrontlineTank: &pb.DaggerheartFrontlineTankFeature{},
			},
		})
		if status.Code(err) != codes.Internal {
			t.Fatalf("missing frontline tank rule status = %v", status.Code(err))
		}

		_, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{
			Hope:  0,
			Armor: 1,
		}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_FrontlineTank{
				FrontlineTank: &pb.DaggerheartFrontlineTankFeature{},
			},
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("insufficient hope status = %v", status.Code(err))
		}
	})

	t.Run("no mercy increases attack bonus", func(t *testing.T) {
		payload, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_NoMercy{
				NoMercy: &pb.DaggerheartNoMercyFeature{},
			},
		})
		if err != nil {
			t.Fatalf("resolveClassFeaturePayload(no_mercy) error = %v", err)
		}
		if payload.Feature != "no_mercy" || payload.Targets[0].ClassStateAfter == nil || payload.Targets[0].ClassStateAfter.AttackBonusUntilRest != 2 {
			t.Fatalf("no mercy payload = %#v", payload)
		}
	})

	t.Run("rally spends dice and heals unique targets", func(t *testing.T) {
		payload, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{
			RallyDice: []int{1, 2, 3},
		}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_Rally{
				Rally: &pb.DaggerheartRallyFeature{
					TargetCharacterIds: []string{" char-2 ", "char-2", "char-3"},
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveClassFeaturePayload(rally) error = %v", err)
		}
		if payload.Feature != "rally" || len(payload.Targets) != 3 {
			t.Fatalf("rally payload = %#v", payload)
		}
		if payload.Targets[0].ClassStateAfter == nil || len(payload.Targets[0].ClassStateAfter.RallyDice) != 0 {
			t.Fatalf("source target after rally = %#v", payload.Targets[0])
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HPAfter == nil || *payload.Targets[1].HPAfter != 8 {
			t.Fatalf("char-2 rally target = %#v", payload.Targets[1])
		}
		if payload.Targets[2].CharacterID != "char-3" || payload.Targets[2].HPAfter == nil || *payload.Targets[2].HPAfter != 6 {
			t.Fatalf("char-3 rally target = %#v", payload.Targets[2])
		}
	})

	t.Run("rally validates targets and dice availability", func(t *testing.T) {
		_, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{
			RallyDice: []int{1},
		}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_Rally{
				Rally: &pb.DaggerheartRallyFeature{},
			},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("rally missing targets status = %v", status.Code(err))
		}

		_, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_Rally{
				Rally: &pb.DaggerheartRallyFeature{TargetCharacterIds: []string{"char-2"}},
			},
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("rally missing dice status = %v", status.Code(err))
		}
	})

	t.Run("make a scene shifts hope between characters", func(t *testing.T) {
		payload, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{
			Hope: 2,
		}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_MakeAScene{
				MakeAScene: &pb.DaggerheartMakeASceneFeature{TargetCharacterId: " char-2 "},
			},
		})
		if err != nil {
			t.Fatalf("resolveClassFeaturePayload(make_a_scene) error = %v", err)
		}
		if payload.Feature != "make_a_scene" || len(payload.Targets) != 2 {
			t.Fatalf("make a scene payload = %#v", payload)
		}
		if payload.Targets[0].HopeAfter == nil || *payload.Targets[0].HopeAfter != 1 {
			t.Fatalf("source hope after = %#v", payload.Targets[0])
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HopeAfter == nil || *payload.Targets[1].HopeAfter != 3 {
			t.Fatalf("target hope after = %#v", payload.Targets[1])
		}
	})

	t.Run("make a scene validates target and hope", func(t *testing.T) {
		_, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{
			Hope: 1,
		}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_MakeAScene{
				MakeAScene: &pb.DaggerheartMakeASceneFeature{},
			},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("make a scene missing target status = %v", status.Code(err))
		}

		_, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_MakeAScene{
				MakeAScene: &pb.DaggerheartMakeASceneFeature{TargetCharacterId: "char-2"},
			},
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("make a scene insufficient hope status = %v", status.Code(err))
		}
	})

	t.Run("hunters focus requires target and trims it", func(t *testing.T) {
		_, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_HuntersFocus{
				HuntersFocus: &pb.DaggerheartHuntersFocusFeature{},
			},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("hunters focus missing target status = %v", status.Code(err))
		}

		payload, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_HuntersFocus{
				HuntersFocus: &pb.DaggerheartHuntersFocusFeature{TargetId: " adv-1 "},
			},
		})
		if err != nil {
			t.Fatalf("resolveClassFeaturePayload(hunters_focus) error = %v", err)
		}
		if payload.Feature != "hunters_focus" || payload.Targets[0].ClassStateAfter == nil || payload.Targets[0].ClassStateAfter.FocusTargetID != "adv-1" {
			t.Fatalf("hunters focus payload = %#v", payload)
		}
	})

	t.Run("life support heals target and spends hope", func(t *testing.T) {
		payload, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{
			Hope: 2,
		}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_LifeSupport{
				LifeSupport: &pb.DaggerheartLifeSupportFeature{TargetCharacterId: "char-2"},
			},
		})
		if err != nil {
			t.Fatalf("resolveClassFeaturePayload(life_support) error = %v", err)
		}
		if payload.Feature != "life_support" || len(payload.Targets) != 2 {
			t.Fatalf("life support payload = %#v", payload)
		}
		if payload.Targets[0].HopeAfter == nil || *payload.Targets[0].HopeAfter != 1 {
			t.Fatalf("life support source = %#v", payload.Targets[0])
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HPAfter == nil || *payload.Targets[1].HPAfter != 7 {
			t.Fatalf("life support target = %#v", payload.Targets[1])
		}
	})

	t.Run("life support validates target and hope", func(t *testing.T) {
		_, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{
			Hope: 1,
		}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_LifeSupport{
				LifeSupport: &pb.DaggerheartLifeSupportFeature{},
			},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("life support missing target status = %v", status.Code(err))
		}

		_, err = handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplyClassFeatureRequest_LifeSupport{
				LifeSupport: &pb.DaggerheartLifeSupportFeature{TargetCharacterId: "char-2"},
			},
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("life support insufficient hope status = %v", status.Code(err))
		}
	})

	t.Run("missing feature is invalid", func(t *testing.T) {
		_, err := handler.resolveClassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{}, daggerheartstate.CharacterClassState{}, &pb.DaggerheartApplyClassFeatureRequest{
			CharacterId: "char-1",
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("missing feature status = %v", status.Code(err))
		}
	})
}

func TestResolveSubclassFeaturePayload(t *testing.T) {
	store := &statefulDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 6},
			"char-2": {CampaignID: "camp-1", CharacterID: "char-2", HpMax: 8},
			"char-3": {CampaignID: "camp-1", CharacterID: "char-3", HpMax: 9},
		},
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-1": {{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Hp:          4,
				Hope:        2,
				HopeMax:     6,
				Stress:      3,
				Conditions: []projectionstore.DaggerheartConditionState{{
					ID:       "restrained",
					Class:    string(rules.ConditionClassStandard),
					Standard: rules.ConditionRestrained,
					Code:     rules.ConditionRestrained,
					Label:    "Restrained",
				}},
			}},
			"char-2": {{
				CampaignID:  "camp-1",
				CharacterID: "char-2",
				Hp:          5,
				Hope:        1,
				HopeMax:     6,
				Stress:      2,
			}},
			"char-3": {{
				CampaignID:  "camp-1",
				CharacterID: "char-3",
				Hp:          6,
				Hope:        2,
				HopeMax:     6,
				Stress:      5,
			}},
		},
		adversaries: map[string]projectionstore.DaggerheartAdversary{
			"adv-1": {
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
			},
		},
	}

	handler := newTestHandler(Dependencies{Daggerheart: store})

	t.Run("gifted performer relaxing song heals unique targets", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.troubadour",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.troubadour",
				Rank:       projectionstore.DaggerheartSubclassTrackRankMastery,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 2, HopeMax: 6, Stress: 3}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:               "relaxing_song",
					TargetCharacterIds: []string{" char-2 ", "char-2"},
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(relaxing_song) error = %v", err)
		}
		if payload.Feature != "gifted_performer_relaxing_song" || len(payload.Targets) != 3 {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.Targets[0].SubclassStateAfter == nil || payload.Targets[0].SubclassStateAfter.GiftedPerformerRelaxingSongUses != 1 {
			t.Fatalf("subclass target = %#v", payload.Targets[0])
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HPAfter == nil || *payload.Targets[1].HPAfter != 6 {
			t.Fatalf("char-2 target = %#v", payload.Targets[1])
		}
		if payload.Targets[2].CharacterID != "char-1" || payload.Targets[2].HPAfter == nil || *payload.Targets[2].HPAfter != 5 {
			t.Fatalf("char-1 target = %#v", payload.Targets[2])
		}
	})

	t.Run("gifted performer epic song marks adversary vulnerable", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.troubadour",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.troubadour",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:              "epic_song",
					TargetId:          "adv-1",
					TargetIsAdversary: true,
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(epic_song) error = %v", err)
		}
		if payload.Feature != "gifted_performer_epic_song" || len(payload.AdversaryConditionTargets) != 1 {
			t.Fatalf("payload = %#v", payload)
		}
		if got := payload.AdversaryConditionTargets[0].Added; len(got) != 1 || got[0].Standard != rules.ConditionVulnerable {
			t.Fatalf("added conditions = %#v", got)
		}
	})

	t.Run("elementalist spends hope and sets damage bonus", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.elemental_origin",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.elemental_origin",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 2, HopeMax: 6}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Elementalist{
				Elementalist: &pb.DaggerheartElementalistRequest{Bonus: "damage"},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(elementalist) error = %v", err)
		}
		if payload.Feature != "elementalist" || payload.Targets[0].HopeAfter == nil || *payload.Targets[0].HopeAfter != 1 {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.Targets[0].SubclassStateAfter == nil || payload.Targets[0].SubclassStateAfter.ElementalistDamageBonus != 3 {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
	})

	t.Run("transcendence enables requested bonuses", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.elemental_origin",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.elemental_origin",
				Rank:       projectionstore.DaggerheartSubclassTrackRankSpecialization,
			}},
		}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Transcendence{
				Transcendence: &pb.DaggerheartTranscendenceRequest{
					Bonuses: []string{"trait", "evasion"},
					Trait:   "presence",
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(transcendence) error = %v", err)
		}
		after := payload.Targets[0].SubclassStateAfter
		if payload.Feature != "transcendence" || after == nil || !after.TranscendenceActive || after.TranscendenceEvasionBonus != 2 || after.TranscendenceTraitBonusTarget != "presence" {
			t.Fatalf("payload = %#v", payload)
		}
	})

	t.Run("vanishing act adds cloaked and removes restrained", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.nightwalker",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.nightwalker",
				Rank:       projectionstore.DaggerheartSubclassTrackRankSpecialization,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Stress:      3,
			Conditions: []projectionstore.DaggerheartConditionState{{
				ID:       "restrained",
				Class:    string(rules.ConditionClassStandard),
				Standard: rules.ConditionRestrained,
				Code:     rules.ConditionRestrained,
				Label:    "Restrained",
			}},
		}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_VanishingAct{
				VanishingAct: &emptypb.Empty{},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(vanishing_act) error = %v", err)
		}
		if payload.Feature != "vanishing_act" || payload.Targets[0].StressAfter == nil || *payload.Targets[0].StressAfter != 4 {
			t.Fatalf("payload = %#v", payload)
		}
		if len(payload.CharacterConditionTargets) != 1 {
			t.Fatalf("condition targets = %#v", payload.CharacterConditionTargets)
		}
		change := payload.CharacterConditionTargets[0]
		if len(change.Added) != 1 || change.Added[0].Standard != rules.ConditionCloaked {
			t.Fatalf("added = %#v", change.Added)
		}
		if len(change.Removed) != 1 || change.Removed[0].Standard != rules.ConditionRestrained {
			t.Fatalf("removed = %#v", change.Removed)
		}
	})

	t.Run("battle ritual marks long-rest usage and shifts hope and stress", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.call_of_the_brave",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.call_of_the_brave",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 5, HopeMax: 6, Stress: 1}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_BattleRitual{
				BattleRitual: &emptypb.Empty{},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(battle_ritual) error = %v", err)
		}
		if payload.Feature != "battle_ritual" || len(payload.Targets) != 1 {
			t.Fatalf("payload = %#v", payload)
		}
		target := payload.Targets[0]
		if target.HopeAfter == nil || *target.HopeAfter != 6 || target.StressAfter == nil || *target.StressAfter != 0 {
			t.Fatalf("target = %#v", target)
		}
		if target.SubclassStateAfter == nil || !target.SubclassStateAfter.BattleRitualUsedThisLongRest {
			t.Fatalf("subclass state = %#v", target.SubclassStateAfter)
		}
	})

	t.Run("sparing touch heals target hp and increments uses", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.divine_wielder",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.divine_wielder",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{
					TargetCharacterId: "char-2",
					Clear:             "hp",
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(sparing_touch hp) error = %v", err)
		}
		if payload.Feature != "sparing_touch" || len(payload.Targets) != 2 {
			t.Fatalf("payload = %#v", payload)
		}
		if after := payload.Targets[0].SubclassStateAfter; after == nil || after.SparingTouchUsesThisLongRest != 1 {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HPAfter == nil || *payload.Targets[1].HPAfter != 7 {
			t.Fatalf("target patch = %#v", payload.Targets[1])
		}
	})

	t.Run("clarity of nature distributes stress clear within instinct budget", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.warden_of_renewal",
			Instinct:    4,
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.warden_of_renewal",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature{
				ClarityOfNature: &pb.DaggerheartClarityOfNatureRequest{
					Targets: []*pb.DaggerheartStressClearTarget{
						{CharacterId: "char-2", StressClear: 1},
						{CharacterId: "char-3", StressClear: 3},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(clarity_of_nature) error = %v", err)
		}
		if payload.Feature != "clarity_of_nature" || len(payload.Targets) != 3 {
			t.Fatalf("payload = %#v", payload)
		}
		if after := payload.Targets[0].SubclassStateAfter; after == nil || !after.ClarityOfNatureUsedThisLongRest {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].StressAfter == nil || *payload.Targets[1].StressAfter != 1 {
			t.Fatalf("char-2 target = %#v", payload.Targets[1])
		}
		if payload.Targets[2].CharacterID != "char-3" || payload.Targets[2].StressAfter == nil || *payload.Targets[2].StressAfter != 2 {
			t.Fatalf("char-3 target = %#v", payload.Targets[2])
		}
	})

	t.Run("clarity of nature rejects stress clear above instinct", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.warden_of_renewal",
			Instinct:    2,
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.warden_of_renewal",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}

		_, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature{
				ClarityOfNature: &pb.DaggerheartClarityOfNatureRequest{
					Targets: []*pb.DaggerheartStressClearTarget{{CharacterId: "char-2", StressClear: 3}},
				},
			},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("clarity_of_nature status = %v", status.Code(err))
		}
	})

	t.Run("regeneration spends hope and heals target", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.warden_of_renewal",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.warden_of_renewal",
				Rank:       projectionstore.DaggerheartSubclassTrackRankSpecialization,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 4}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Regeneration{
				Regeneration: &pb.DaggerheartRegenerationRequest{
					TargetCharacterId: "char-2",
					ClearHp:           2,
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(regeneration) error = %v", err)
		}
		if payload.Feature != "regeneration" || len(payload.Targets) != 2 {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.Targets[0].HopeAfter == nil || *payload.Targets[0].HopeAfter != 1 {
			t.Fatalf("actor patch = %#v", payload.Targets[0])
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HPAfter == nil || *payload.Targets[1].HPAfter != 7 {
			t.Fatalf("target patch = %#v", payload.Targets[1])
		}
	})

	t.Run("wardens protection heals unique targets and marks usage", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.warden_of_renewal",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.warden_of_renewal",
				Rank:       projectionstore.DaggerheartSubclassTrackRankMastery,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 4}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_WardensProtection{
				WardensProtection: &pb.DaggerheartWardensProtectionRequest{
					TargetCharacterIds: []string{" char-2 ", "char-3", "char-2"},
				},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(wardens_protection) error = %v", err)
		}
		if payload.Feature != "wardens_protection" || len(payload.Targets) != 3 {
			t.Fatalf("payload = %#v", payload)
		}
		if after := payload.Targets[0].SubclassStateAfter; after == nil || !after.WardensProtectionUsedThisLongRest {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
		if payload.Targets[0].HopeAfter == nil || *payload.Targets[0].HopeAfter != 2 {
			t.Fatalf("actor patch = %#v", payload.Targets[0])
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].HPAfter == nil || *payload.Targets[1].HPAfter != 7 {
			t.Fatalf("char-2 target = %#v", payload.Targets[1])
		}
		if payload.Targets[2].CharacterID != "char-3" || payload.Targets[2].HPAfter == nil || *payload.Targets[2].HPAfter != 8 {
			t.Fatalf("char-3 target = %#v", payload.Targets[2])
		}
	})

	t.Run("elemental incarnation sets channel and stress", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.warden_of_the_elements",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.warden_of_the_elements",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Stress: 3}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ElementalIncarnation{
				ElementalIncarnation: &pb.DaggerheartElementalIncarnationRequest{Channel: daggerheartstate.ElementalChannelAir},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(elemental_incarnation) error = %v", err)
		}
		if payload.Feature != "elemental_incarnation" || payload.Targets[0].StressAfter == nil || *payload.Targets[0].StressAfter != 4 {
			t.Fatalf("payload = %#v", payload)
		}
		if after := payload.Targets[0].SubclassStateAfter; after == nil || after.ElementalChannel != daggerheartstate.ElementalChannelAir {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
	})

	t.Run("elemental incarnation rejects invalid channel", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.warden_of_the_elements",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.warden_of_the_elements",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}

		_, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ElementalIncarnation{
				ElementalIncarnation: &pb.DaggerheartElementalIncarnationRequest{Channel: "lightning"},
			},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("elemental_incarnation status = %v", status.Code(err))
		}
	})

	t.Run("rousing speech clears stress for unique targets", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.wordsmith",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.wordsmith",
				Rank:       projectionstore.DaggerheartSubclassTrackRankFoundation,
			}},
		}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1"}, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_RousingSpeech{
				RousingSpeech: &pb.DaggerheartRousingSpeechRequest{TargetCharacterIds: []string{"char-2", " char-3 ", "char-2"}},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(rousing_speech) error = %v", err)
		}
		if payload.Feature != "rousing_speech" || len(payload.Targets) != 3 {
			t.Fatalf("payload = %#v", payload)
		}
		if after := payload.Targets[0].SubclassStateAfter; after == nil || !after.RousingSpeechUsedThisLongRest {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
		if payload.Targets[1].CharacterID != "char-2" || payload.Targets[1].StressAfter == nil || *payload.Targets[1].StressAfter != 0 {
			t.Fatalf("char-2 target = %#v", payload.Targets[1])
		}
		if payload.Targets[2].CharacterID != "char-3" || payload.Targets[2].StressAfter == nil || *payload.Targets[2].StressAfter != 3 {
			t.Fatalf("char-3 target = %#v", payload.Targets[2])
		}
	})

	t.Run("nemesis spends hope and records adversary target", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{
			CharacterID: "char-1",
			SubclassID:  "subclass.vengeance",
			SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
				Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
				SubclassID: "subclass.vengeance",
				Rank:       projectionstore.DaggerheartSubclassTrackRankMastery,
			}},
		}
		state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 4}

		payload, err := handler.resolveSubclassFeaturePayload(context.Background(), "camp-1", profile, state, daggerheartstate.CharacterClassState{}, daggerheartstate.CharacterSubclassState{}, &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Nemesis{
				Nemesis: &pb.DaggerheartNemesisRequest{AdversaryId: "adv-1"},
			},
		})
		if err != nil {
			t.Fatalf("resolveSubclassFeaturePayload(nemesis) error = %v", err)
		}
		if payload.Feature != "nemesis" || len(payload.Targets) != 1 {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.Targets[0].HopeAfter == nil || *payload.Targets[0].HopeAfter != 2 {
			t.Fatalf("target = %#v", payload.Targets[0])
		}
		if after := payload.Targets[0].SubclassStateAfter; after == nil || after.NemesisTargetID != "adv-1" {
			t.Fatalf("subclass state = %#v", payload.Targets[0].SubclassStateAfter)
		}
	})
}

func TestEnrichArmorSwapPayload(t *testing.T) {
	store := &statefulDaggerheartStore{
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-1": {{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Armor:       1,
			}},
		},
	}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		Content: contentStoreStub{
			armors: map[string]contentstore.DaggerheartArmor{
				"old-armor": {
					ID:         "old-armor",
					ArmorScore: 2,
					Rules:      contentstore.DaggerheartArmorRules{EvasionDelta: 1, AllTraitsDelta: 1},
				},
				"new-armor": {
					ID:                  "new-armor",
					ArmorScore:          4,
					BaseMajorThreshold:  8,
					BaseSevereThreshold: 16,
					Rules: contentstore.DaggerheartArmorRules{
						EvasionDelta:       -1,
						PresenceDelta:      2,
						SpellcastRollBonus: 3,
					},
				},
			},
		},
	})

	payload := &daggerheartpayload.EquipmentSwapPayload{
		CharacterID: ids.CharacterID("char-1"),
		ItemID:      "new-armor",
		ItemType:    "armor",
		To:          "active",
	}
	profile := projectionstore.DaggerheartCharacterProfile{
		CharacterID:     "char-1",
		Level:           1,
		EquippedArmorID: "old-armor",
		ArmorMax:        2,
		Evasion:         11,
		Agility:         3,
		Strength:        3,
		Finesse:         3,
		Instinct:        3,
		Presence:        2,
		Knowledge:       3,
	}

	if err := handler.enrichArmorSwapPayload(context.Background(), "camp-1", profile, payload); err != nil {
		t.Fatalf("enrichArmorSwapPayload() error = %v", err)
	}
	if payload.EquippedArmorID != "new-armor" {
		t.Fatalf("equipped armor = %q", payload.EquippedArmorID)
	}
	if payload.ArmorScoreAfter == nil || *payload.ArmorScoreAfter != 4 || payload.ArmorMaxAfter == nil || *payload.ArmorMaxAfter != 4 {
		t.Fatalf("armor score/max = %#v / %#v", payload.ArmorScoreAfter, payload.ArmorMaxAfter)
	}
	if payload.EvasionAfter == nil || *payload.EvasionAfter != 9 {
		t.Fatalf("evasion after = %#v", payload.EvasionAfter)
	}
	if payload.PresenceAfter == nil || *payload.PresenceAfter != 3 {
		t.Fatalf("presence after = %#v", payload.PresenceAfter)
	}
	if payload.SpellcastRollBonusAfter == nil || *payload.SpellcastRollBonusAfter != 3 {
		t.Fatalf("spellcast bonus after = %#v", payload.SpellcastRollBonusAfter)
	}
	if payload.ArmorAfter == nil || *payload.ArmorAfter != 3 {
		t.Fatalf("armor after = %#v", payload.ArmorAfter)
	}
}

func TestApplyClassFeature(t *testing.T) {
	store := &statefulDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				ClassID:     "class-guardian",
				ArmorMax:    3,
			},
		},
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-1": {
				{CampaignID: "camp-1", CharacterID: "char-1", Hope: 3, Armor: 1},
				{CampaignID: "camp-1", CharacterID: "char-1", Hope: 2, Armor: 3},
			},
		},
	}

	var recorded CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		Content: contentStoreStub{
			classes: map[string]contentstore.DaggerheartClass{
				"class-guardian": {
					ID: "class-guardian",
					HopeFeature: contentstore.DaggerheartHopeFeature{
						HopeFeatureRule: &contentstore.DaggerheartHopeFeatureRule{Bonus: 2, HopeCost: 1},
					},
				},
			},
		},
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			recorded = in
			return nil
		},
	})

	resp, err := handler.ApplyClassFeature(testContext(), &pb.DaggerheartApplyClassFeatureRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Feature: &pb.DaggerheartApplyClassFeatureRequest_FrontlineTank{
			FrontlineTank: &pb.DaggerheartFrontlineTankFeature{},
		},
	})
	if err != nil {
		t.Fatalf("ApplyClassFeature() error = %v", err)
	}
	if recorded.CommandType != commandids.DaggerheartClassFeatureApply {
		t.Fatalf("command type = %q", recorded.CommandType)
	}

	var payload daggerheartpayload.ClassFeatureApplyPayload
	if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Feature != "frontline_tank" || payload.Targets[0].ArmorAfter == nil || *payload.Targets[0].ArmorAfter != 3 {
		t.Fatalf("payload = %#v", payload)
	}
	if resp.GetCharacterId() != "char-1" || resp.GetState() == nil || resp.GetState().GetArmor() != 3 {
		t.Fatalf("response = %#v", resp)
	}
}

func TestApplySubclassFeature(t *testing.T) {
	store := &statefulDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				SubclassID:  "subclass.syndicate",
				SubclassTracks: []projectionstore.DaggerheartSubclassTrack{{
					Origin:     projectionstore.DaggerheartSubclassTrackOriginPrimary,
					SubclassID: "subclass.syndicate",
					Rank:       projectionstore.DaggerheartSubclassTrackRankSpecialization,
				}},
			},
		},
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-1": {
				{CampaignID: "camp-1", CharacterID: "char-1"},
				{
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					SubclassState: &projectionstore.DaggerheartSubclassState{
						ContactsEverywhereUsesThisSession: 1,
						ContactsEverywhereActionDieBonus:  3,
					},
				},
			},
		},
	}

	var recorded CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			recorded = in
			return nil
		},
	})

	resp, err := handler.ApplySubclassFeature(testContext(), &pb.DaggerheartApplySubclassFeatureRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Feature: &pb.DaggerheartApplySubclassFeatureRequest_ContactsEverywhere{
			ContactsEverywhere: &pb.DaggerheartContactsEverywhereRequest{Option: "next_action_bonus"},
		},
	})
	if err != nil {
		t.Fatalf("ApplySubclassFeature() error = %v", err)
	}
	if recorded.CommandType != commandids.DaggerheartSubclassFeatureApply {
		t.Fatalf("command type = %q", recorded.CommandType)
	}

	var payload daggerheartpayload.SubclassFeatureApplyPayload
	if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Feature != "contacts_everywhere" || payload.Targets[0].SubclassStateAfter == nil || payload.Targets[0].SubclassStateAfter.ContactsEverywhereActionDieBonus != 3 {
		t.Fatalf("payload = %#v", payload)
	}
	if resp.GetCharacterId() != "char-1" || resp.GetState() == nil || resp.GetState().GetSubclassState().GetContactsEverywhereActionDieBonus() != 3 {
		t.Fatalf("response = %#v", resp)
	}
}

func TestTransformAndDropBeastform(t *testing.T) {
	t.Run("transform", func(t *testing.T) {
		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					ClassID:     "class.druid",
					Level:       1,
					StressMax:   3,
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{CampaignID: "camp-1", CharacterID: "char-1", Stress: 0},
					{
						CampaignID:  "camp-1",
						CharacterID: "char-1",
						Stress:      1,
						ClassState: projectionstore.DaggerheartClassState{
							ActiveBeastform: &projectionstore.DaggerheartActiveBeastformState{BeastformID: "wolf"},
						},
					},
				},
			},
		}

		var recorded CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			Content: contentStoreStub{
				beastforms: map[string]contentstore.DaggerheartBeastformEntry{
					"wolf": {
						ID:         "wolf",
						Tier:       1,
						Trait:      "agility",
						TraitBonus: 1,
						Attack: contentstore.DaggerheartBeastformAttack{
							Range:      "melee",
							DamageDice: []contentstore.DaggerheartDamageDie{{Count: 1, Sides: 8}},
							DamageType: "physical",
						},
						Features: []contentstore.DaggerheartBeastformFeature{{Name: "Fragile"}},
					},
				},
			},
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				recorded = in
				return nil
			},
		})

		resp, err := handler.TransformBeastform(testContext(), &pb.DaggerheartTransformBeastformRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			BeastformId: "wolf",
		})
		if err != nil {
			t.Fatalf("TransformBeastform() error = %v", err)
		}
		if recorded.CommandType != commandids.DaggerheartBeastformTransform {
			t.Fatalf("command type = %q", recorded.CommandType)
		}

		var payload daggerheartpayload.BeastformTransformPayload
		if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.BeastformID != "wolf" || payload.StressAfter == nil || *payload.StressAfter != 1 || payload.ClassStateAfter == nil || payload.ClassStateAfter.ActiveBeastform == nil {
			t.Fatalf("payload = %#v", payload)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil || resp.GetState().GetStress() != 1 {
			t.Fatalf("response = %#v", resp)
		}
	})

	t.Run("drop", func(t *testing.T) {
		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					ClassID:     "class.druid",
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{
						CampaignID:  "camp-1",
						CharacterID: "char-1",
						ClassState: projectionstore.DaggerheartClassState{
							ActiveBeastform: &projectionstore.DaggerheartActiveBeastformState{BeastformID: "wolf"},
						},
					},
					{CampaignID: "camp-1", CharacterID: "char-1"},
				},
			},
		}

		var recorded CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				recorded = in
				return nil
			},
		})

		resp, err := handler.DropBeastform(testContext(), &pb.DaggerheartDropBeastformRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
		})
		if err != nil {
			t.Fatalf("DropBeastform() error = %v", err)
		}
		if recorded.CommandType != commandids.DaggerheartBeastformDrop {
			t.Fatalf("command type = %q", recorded.CommandType)
		}

		var payload daggerheartpayload.BeastformDropPayload
		if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.BeastformID != "wolf" || payload.ClassStateAfter == nil || payload.ClassStateAfter.ActiveBeastform != nil {
			t.Fatalf("payload = %#v", payload)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil || resp.GetState().GetClassState().GetActiveBeastform() != nil {
			t.Fatalf("response = %#v", resp)
		}
	})
}

func TestCompanionLifecycle(t *testing.T) {
	t.Run("begin experience", func(t *testing.T) {
		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
						Experiences: []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "exp-1"}},
					},
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{CampaignID: "camp-1", CharacterID: "char-1"},
					{
						CampaignID:     "camp-1",
						CharacterID:    "char-1",
						CompanionState: &projectionstore.DaggerheartCompanionState{Status: "away", ActiveExperienceID: "exp-1"},
					},
				},
			},
		}

		var recorded CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				recorded = in
				return nil
			},
		})

		resp, err := handler.BeginCompanionExperience(testContext(), &pb.DaggerheartBeginCompanionExperienceRequest{
			CampaignId:   "camp-1",
			CharacterId:  "char-1",
			ExperienceId: "exp-1",
		})
		if err != nil {
			t.Fatalf("BeginCompanionExperience() error = %v", err)
		}
		if recorded.CommandType != commandids.DaggerheartCompanionExperienceBegin {
			t.Fatalf("command type = %q", recorded.CommandType)
		}

		var payload daggerheartpayload.CompanionExperienceBeginPayload
		if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.ExperienceID != "exp-1" || payload.CompanionStateAfter == nil || payload.CompanionStateAfter.ActiveExperienceID != "exp-1" {
			t.Fatalf("payload = %#v", payload)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil || resp.GetState().GetCompanionState().GetActiveExperienceId() != "exp-1" {
			t.Fatalf("response = %#v", resp)
		}
	})

	t.Run("return companion", func(t *testing.T) {
		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
						Experiences: []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "exp-1"}},
					},
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{
						CampaignID:     "camp-1",
						CharacterID:    "char-1",
						Stress:         1,
						CompanionState: &projectionstore.DaggerheartCompanionState{Status: "away", ActiveExperienceID: "exp-1"},
					},
					{
						CampaignID:     "camp-1",
						CharacterID:    "char-1",
						Stress:         0,
						CompanionState: &projectionstore.DaggerheartCompanionState{Status: "present"},
					},
				},
			},
		}

		var recorded CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				recorded = in
				return nil
			},
		})

		resp, err := handler.ReturnCompanion(testContext(), &pb.DaggerheartReturnCompanionRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			Resolution:  pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EXPERIENCE_COMPLETED,
		})
		if err != nil {
			t.Fatalf("ReturnCompanion() error = %v", err)
		}
		if recorded.CommandType != commandids.DaggerheartCompanionReturn {
			t.Fatalf("command type = %q", recorded.CommandType)
		}

		var payload daggerheartpayload.CompanionReturnPayload
		if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.Resolution != "experience_completed" || payload.StressAfter == nil || *payload.StressAfter != 0 {
			t.Fatalf("payload = %#v", payload)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil || resp.GetState().GetStress() != 0 {
			t.Fatalf("response = %#v", resp)
		}
	})
}

func TestApplyCharacterStatePatch(t *testing.T) {
	store := &statefulDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1"},
		},
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-1": {
				{CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 2, Stress: 1, Armor: 0},
				{CampaignID: "camp-1", CharacterID: "char-1", Hp: 4, Hope: 1, Stress: 2, Armor: 3},
			},
		},
	}

	var recorded CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			recorded = in
			return nil
		},
	})

	hp := int32(4)
	hope := int32(1)
	stress := int32(2)
	armor := int32(3)
	resp, err := handler.ApplyCharacterStatePatch(testContext(), &pb.DaggerheartApplyCharacterStatePatchRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Hp:          &hp,
		Hope:        &hope,
		Stress:      &stress,
		Armor:       &armor,
		Source:      "gm.adjustment",
		MutationSource: &pb.DaggerheartMutationSource{
			Type:        pb.DaggerheartMutationSourceType_DAGGERHEART_MUTATION_SOURCE_TYPE_GM_ADJUSTMENT,
			Description: "Manual correction",
			SourceId:    "adj-1",
		},
	})
	if err != nil {
		t.Fatalf("ApplyCharacterStatePatch() error = %v", err)
	}
	if recorded.CommandType != commandids.DaggerheartCharacterStatePatch {
		t.Fatalf("command type = %q", recorded.CommandType)
	}

	var payload daggerheartpayload.CharacterStatePatchPayload
	if err := json.Unmarshal(recorded.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Source != "gm.adjustment" || payload.MutationSource == nil || payload.MutationSource.SourceID != "adj-1" {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.HPBefore == nil || *payload.HPBefore != 5 || payload.HPAfter == nil || *payload.HPAfter != 4 {
		t.Fatalf("hp patch = %#v", payload)
	}
	if resp.GetCharacterId() != "char-1" || resp.GetHp() != 4 || resp.GetHope() != 1 || resp.GetStress() != 2 || resp.GetArmor() != 3 {
		t.Fatalf("response = %#v", resp)
	}
}
