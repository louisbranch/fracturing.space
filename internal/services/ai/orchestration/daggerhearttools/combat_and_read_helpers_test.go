package daggerhearttools

import (
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCombatSummaryHelpersShapeProtoResponses(t *testing.T) {
	action := actionRollSummaryFromProto(&pb.SessionActionRollResponse{
		RollSeq:    7,
		HopeDie:    10,
		FearDie:    3,
		Total:      13,
		Difficulty: 12,
		Success:    true,
		Flavor:     " fear ",
		Rng:        &commonv1.RngResponse{SeedUsed: 5},
	})
	if action == nil || action.Outcome != pb.Outcome_SUCCESS_WITH_FEAR.String() || action.Flavor != "fear" {
		t.Fatalf("actionRollSummaryFromProto() = %#v", action)
	}

	outcome := rollOutcomeSummaryFromProto(&pb.ApplyRollOutcomeResponse{
		RollSeq:              7,
		RequiresComplication: true,
		Updated: &pb.OutcomeUpdated{
			GmFear: int32Ptr(2),
		},
	})
	if outcome == nil || !outcome.RequiresComplication || outcome.Updated == nil || outcome.Updated.GMFear == nil {
		t.Fatalf("rollOutcomeSummaryFromProto() = %#v", outcome)
	}

	attack := attackOutcomeSummaryFromProto(&pb.DaggerheartApplyAttackOutcomeResponse{
		RollSeq:     9,
		CharacterId: "char-1",
		Targets:     []string{" adv-1 ", "", "adv-1"},
		Result: &pb.DaggerheartAttackOutcomeResult{
			Outcome: pb.Outcome_SUCCESS_WITH_HOPE,
			Success: true,
			Flavor:  " hope ",
		},
	})
	if attack == nil || len(attack.Targets) != 1 || attack.Result.Outcome != pb.Outcome_SUCCESS_WITH_HOPE.String() {
		t.Fatalf("attackOutcomeSummaryFromProto() = %#v", attack)
	}

	adversaryAttack := adversaryAttackOutcomeSummaryFromProto(&pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
		RollSeq:     11,
		AdversaryId: "adv-1",
		Targets:     []string{"char-1"},
		Result: &pb.DaggerheartAdversaryAttackOutcomeResult{
			Success:    true,
			Roll:       8,
			Total:      10,
			Difficulty: 9,
		},
	})
	if adversaryAttack == nil || adversaryAttack.Result.Total != 10 {
		t.Fatalf("adversaryAttackOutcomeSummaryFromProto() = %#v", adversaryAttack)
	}

	damage := damageRollSummaryFromProto(&pb.SessionDamageRollResponse{
		RollSeq:       12,
		BaseTotal:     5,
		Modifier:      2,
		CriticalBonus: 4,
		Total:         11,
		Critical:      true,
		Rolls: []*pb.DiceRoll{
			{Sides: 8, Results: []int32{3, 4}, Total: 7},
		},
	})
	if damage == nil || len(damage.Rolls) != 1 || damage.Rolls[0].Results[1] != 4 {
		t.Fatalf("damageRollSummaryFromProto() = %#v", damage)
	}

	reaction := reactionOutcomeSummaryFromProto(&pb.DaggerheartApplyReactionOutcomeResponse{
		RollSeq:     13,
		CharacterId: "char-1",
		Result: &pb.DaggerheartReactionOutcomeResult{
			Outcome:            pb.Outcome_FAILURE_WITH_FEAR,
			EffectsNegated:     true,
			CritNegatesEffects: true,
		},
	})
	if reaction == nil || !reaction.Result.CritNegatesEffects {
		t.Fatalf("reactionOutcomeSummaryFromProto() = %#v", reaction)
	}

	damageApplied := characterDamageAppliedSummaryFromProto(&pb.DaggerheartApplyDamageResponse{
		CharacterId: "char-1",
		State: &pb.DaggerheartCharacterState{
			Hp:        4,
			Hope:      2,
			Stress:    1,
			Armor:     3,
			LifeState: pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
		},
	})
	if damageApplied == nil || damageApplied.State.LifeState != "UNCONSCIOUS" {
		t.Fatalf("characterDamageAppliedSummaryFromProto() = %#v", damageApplied)
	}

	adversaryApplied := adversaryDamageAppliedSummaryFromProto(&pb.DaggerheartApplyAdversaryDamageResponse{
		AdversaryId: "adv-1",
		Adversary: &pb.DaggerheartAdversary{
			Id:   "adv-1",
			Name: "Warden",
		},
	})
	if adversaryApplied == nil || adversaryApplied.Adversary.Name != "Warden" {
		t.Fatalf("adversaryDamageAppliedSummaryFromProto() = %#v", adversaryApplied)
	}
}

func TestAttackProfileAndInputHelpers(t *testing.T) {
	if err := applyAttackProfile(nil, nil, nil); err == nil || err.Error() != "attack flow request is required" {
		t.Fatalf("applyAttackProfile(nil) error = %v", err)
	}

	req := &pb.SessionAttackFlowRequest{}
	if err := applyAttackProfile(req, nil, nil); err == nil || !contains(err.Error(), "no attack profile is available") {
		t.Fatalf("applyAttackProfile(no profile) error = %v", err)
	}

	req = &pb.SessionAttackFlowRequest{}
	if err := applyAttackProfile(req, &standardAttackProfileInput{Trait: "Agility", DamageDice: []rollDiceSpec{{Count: 1, Sides: 8}}, AttackRange: "melee"}, &beastformAttackProfileInput{}); err != nil || req.GetStandardAttack() == nil || req.GetBeastformAttack() != nil {
		t.Fatalf("applyAttackProfile(standard wins) req=%#v err=%v", req, err)
	}

	req = &pb.SessionAttackFlowRequest{}
	if err := applyAttackProfile(req, &standardAttackProfileInput{}, &beastformAttackProfileInput{}); err != nil || req.GetBeastformAttack() == nil {
		t.Fatalf("applyAttackProfile(zero standard falls back to beastform) req=%#v err=%v", req, err)
	}
	if err := applyAttackProfile(req, &standardAttackProfileInput{}, nil); err == nil || err.Error() != "standard_attack.trait is required" {
		t.Fatalf("applyAttackProfile(missing trait) error = %v", err)
	}
	if err := applyAttackProfile(req, &standardAttackProfileInput{Trait: "Agility"}, nil); err == nil || err.Error() != "standard_attack.damage_dice are required" {
		t.Fatalf("applyAttackProfile(missing damage dice) error = %v", err)
	}
	if err := applyAttackProfile(req, &standardAttackProfileInput{Trait: "Agility", DamageDice: []rollDiceSpec{{Count: 1, Sides: 8}}}, nil); err == nil || err.Error() != "standard_attack.attack_range is required" {
		t.Fatalf("applyAttackProfile(missing range) error = %v", err)
	}
	if err := applyAttackProfile(req, nil, &beastformAttackProfileInput{}); err != nil || req.GetBeastformAttack() == nil {
		t.Fatalf("applyAttackProfile(beastform) req = %#v err=%v", req, err)
	}

	req = &pb.SessionAttackFlowRequest{}
	if err := applyAttackProfile(req, &standardAttackProfileInput{
		Trait:          " Agility ",
		DamageDice:     []rollDiceSpec{{Count: 1, Sides: 8}, {Count: 0, Sides: 10}},
		DamageModifier: 2,
		AttackRange:    "ranged",
		DamageCritical: true,
	}, nil); err != nil {
		t.Fatalf("applyAttackProfile(standard) error = %v", err)
	}
	if req.GetStandardAttack().GetTrait() != "Agility" || len(req.GetStandardAttack().GetDamageDice()) != 1 {
		t.Fatalf("standard attack req = %#v", req.GetStandardAttack())
	}

	damage := attackDamageSpecToProto(&attackDamageSpecInput{
		DamageType:         "magic",
		ResistMagic:        true,
		Source:             "  Arc blade ",
		SourceCharacterIDs: []string{" char-1 ", "", "char-1"},
	})
	if damage == nil || damage.GetDamageType() != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC || len(damage.GetSourceCharacterIds()) != 1 {
		t.Fatalf("attackDamageSpecToProto() = %#v", damage)
	}

	if incomingAttackArmorReactionToProto(nil) != nil {
		t.Fatal("incomingAttackArmorReactionToProto(nil) = non-nil, want nil")
	}
	if incomingAttackArmorReactionToProto(&incomingAttackArmorReactionInput{
		Shifting:    &struct{}{},
		Timeslowing: &timeslowingArmorReactionInput{},
	}) != nil {
		t.Fatal("incomingAttackArmorReactionToProto(multi) = non-nil, want nil")
	}
	seed := uint64(17)
	reaction := incomingAttackArmorReactionToProto(&incomingAttackArmorReactionInput{
		Timeslowing: &timeslowingArmorReactionInput{Rng: &rngRequest{Seed: &seed, RollMode: "replay"}},
	})
	if reaction == nil || reaction.GetTimeslowing() == nil || reaction.GetTimeslowing().GetRng().GetSeed() != 17 {
		t.Fatalf("incomingAttackArmorReactionToProto() = %#v", reaction)
	}

	supporters, err := groupActionSupportersToProto([]groupActionSupporterInput{{Trait: "Agility"}})
	if err == nil || err.Error() != "supporters[0].character_id is required" {
		t.Fatalf("groupActionSupportersToProto(missing character) = (%#v, %v)", supporters, err)
	}
	supporters, err = groupActionSupportersToProto([]groupActionSupporterInput{{CharacterID: "char-2", Trait: "Presence", Context: "move_silently"}})
	if err != nil || len(supporters) != 1 || supporters[0].GetContext() != pb.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY {
		t.Fatalf("groupActionSupportersToProto() = (%#v, %v)", supporters, err)
	}

	if _, err := tagTeamParticipantToProto("first", nil); err == nil || err.Error() != "first is required" {
		t.Fatalf("tagTeamParticipantToProto(nil) error = %v", err)
	}
	participant, err := tagTeamParticipantToProto("second", &tagTeamParticipantInput{CharacterID: "char-3", Trait: "Finesse"})
	if err != nil || participant.GetCharacterId() != "char-3" {
		t.Fatalf("tagTeamParticipantToProto() = (%#v, %v)", participant, err)
	}

	dice := diceSpecsToProto([]rollDiceSpec{{Count: 2, Sides: 6}, {Count: 0, Sides: 8}})
	if len(dice) != 1 || dice[0].GetCount() != 2 {
		t.Fatalf("diceSpecsToProto() = %#v", dice)
	}
}

func TestTargetInferenceAndDamageHelpers(t *testing.T) {
	if _, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{Status: "NO_ACTIVE_SCENE"}); err == nil || !contains(err.Error(), "no active scene") {
		t.Fatalf("resolveAttackTargetID(NO_ACTIVE_SCENE) error = %v", err)
	}
	if _, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{Status: "EMPTY_BOARD"}); err == nil || !contains(err.Error(), "combat board is empty") {
		t.Fatalf("resolveAttackTargetID(EMPTY_BOARD) error = %v", err)
	}
	if _, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{Status: "NO_VISIBLE_ADVERSARY"}); err == nil || !contains(err.Error(), "no visible adversary") {
		t.Fatalf("resolveAttackTargetID(NO_VISIBLE_ADVERSARY) error = %v", err)
	}
	if targetID, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{Adversaries: []adversarySummary{{ID: "adv-1"}}}); err != nil || targetID != "adv-1" {
		t.Fatalf("resolveAttackTargetID(single) = (%q, %v)", targetID, err)
	}
	if _, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{Adversaries: []adversarySummary{{ID: "adv-1"}, {ID: "adv-2"}}}); err == nil || !contains(err.Error(), "multiple visible adversaries") {
		t.Fatalf("resolveAttackTargetID(multi) error = %v", err)
	}

	inferred := &attackDamageSpecInput{DamageType: "physical", Source: "Sword", SourceCharacterIDs: []string{"char-1"}}
	merged := mergeAttackDamageSpec(&attackDamageSpecInput{Direct: true}, inferred)
	if merged.DamageType != "physical" || merged.Source != "Sword" || !merged.Direct {
		t.Fatalf("mergeAttackDamageSpec() = %#v", merged)
	}
	if mergeAttackDamageSpec(nil, nil) != nil {
		t.Fatal("mergeAttackDamageSpec(nil, nil) = non-nil, want nil")
	}

	dice, ok := parseDamageDiceString("2d8 + d6")
	if !ok || len(dice) != 2 || dice[0].Count != 2 || dice[1].Sides != 6 {
		t.Fatalf("parseDamageDiceString(valid) = (%#v, %v)", dice, ok)
	}
	if _, ok := parseDamageDiceString("invalid"); ok {
		t.Fatal("parseDamageDiceString(invalid) = ok, want false")
	}

	if firstNonEmpty("", "  alpha ", "beta") != "alpha" {
		t.Fatal("firstNonEmpty() did not return first trimmed value")
	}
	if !boolDefaultTrue(nil) {
		t.Fatal("boolDefaultTrue(nil) = false, want true")
	}
	value := false
	if boolDefaultTrue(&value) {
		t.Fatal("boolDefaultTrue(false) = true, want false")
	}
	board := daggerheartCombatBoardPayload{Adversaries: []adversarySummary{{ID: " adv-1 "}}}
	if !board.hasAdversary("adv-1") || board.hasAdversary("adv-2") {
		t.Fatalf("hasAdversary board = %#v", board)
	}
}

func TestReadSurfaceHelperMappings(t *testing.T) {
	profile := &pb.DaggerheartProfile{
		Agility:              wrapperspb.Int32(2),
		Finesse:              wrapperspb.Int32(1),
		HpMax:                12,
		StressMax:            wrapperspb.Int32(6),
		ArmorMax:             wrapperspb.Int32(3),
		Evasion:              wrapperspb.Int32(14),
		ArmorScore:           wrapperspb.Int32(2),
		Proficiency:          wrapperspb.Int32(1),
		MajorThreshold:       wrapperspb.Int32(8),
		SevereThreshold:      wrapperspb.Int32(14),
		SpellcastRollBonus:   wrapperspb.Int32(1),
		StartingPotionItemId: "item:minor-health-potion",
		PrimaryWeapon: &pb.DaggerheartSheetWeaponSummary{
			Id:         "weapon-1",
			Name:       "Longbow",
			Trait:      "Agility",
			Range:      "far",
			DamageDice: "1d8",
			DamageType: "physical",
		},
		ActiveArmor: &pb.DaggerheartSheetArmorSummary{
			Id:        "armor-1",
			Name:      "Leather",
			BaseScore: 1,
		},
		DomainCardIds: []string{"domain:valor-rally", "  "},
		ActiveClassFeatures: []*pb.DaggerheartActiveClassFeature{{
			Id:          "feature-1",
			Name:        "Battle Cry",
			Description: "Boost the line.",
			Level:       2,
			HopeFeature: true,
		}},
		ActiveSubclassFeatures: []*pb.DaggerheartActiveSubclassTrackFeatures{{
			Track: &pb.DaggerheartSubclassTrack{
				Origin:     pb.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY,
				ClassId:    "class:ranger",
				SubclassId: "subclass:beastbound",
				DomainId:   "domain:valor",
			},
			FoundationFeatures: []*pb.DaggerheartActiveSubclassFeature{{
				Id:          "sub-1",
				Name:        "Pack Link",
				Description: "Coordinate with a beast.",
				Level:       1,
			}},
		}},
	}
	state := &pb.DaggerheartCharacterState{
		Hp:        9,
		Hope:      2,
		HopeMax:   6,
		Stress:    1,
		Armor:     2,
		LifeState: pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
	}
	if resources := resourcesFromProto(profile, state); resources == nil || resources.LifeState != "ALIVE" || resources.HPMax != 12 {
		t.Fatalf("resourcesFromProto() = %#v", resources)
	}
	if defenses := defensesFromProto(profile); defenses == nil || defenses.Evasion == nil || *defenses.Evasion != 14 {
		t.Fatalf("defensesFromProto() = %#v", defenses)
	}
	if equipment := equipmentFromProto(profile); equipment == nil || equipment.PrimaryWeapon.Name != "Longbow" || len(equipment.Consumables) != 1 {
		t.Fatalf("equipmentFromProto() = %#v", equipment)
	}
	cards := domainCardsFromProto(profile.GetDomainCardIds())
	if len(cards) != 1 || cards[0].Name != "Rally" || cards[0].Domain != "Valor" {
		t.Fatalf("domainCardsFromProto() = %#v", cards)
	}
	classFeatures := activeClassFeaturesFromProto(profile.GetActiveClassFeatures())
	if len(classFeatures) != 1 || !classFeatures[0].HopeFeature {
		t.Fatalf("activeClassFeaturesFromProto() = %#v", classFeatures)
	}
	subclassFeatures := activeSubclassFeaturesFromProto(profile.GetActiveSubclassFeatures())
	if len(subclassFeatures) != 1 || subclassFeatures[0].Origin != "PRIMARY" || subclassFeatures[0].Class != "Ranger" {
		t.Fatalf("activeSubclassFeaturesFromProto() = %#v", subclassFeatures)
	}

	companion := companionFromProto(&pb.DaggerheartCompanionSheet{
		Name:              "Moss",
		AnimalKind:        "Wolf",
		Evasion:           13,
		AttackDescription: "Bite",
		AttackRange:       "melee",
		DamageDieSides:    8,
		DamageType:        "physical",
		Experiences: []*pb.DaggerheartCompanionExperience{{
			Name:     "Tracker",
			Modifier: 2,
		}},
	}, &pb.DaggerheartCompanionState{
		Status:             "READY",
		ActiveExperienceId: "tracker",
	})
	if companion == nil || companion.Name != "Moss" || len(companion.Experiences) != 1 {
		t.Fatalf("companionFromProto() = %#v", companion)
	}

	classState := classStateFromProto(&pb.DaggerheartClassState{
		FocusTargetId: "adv-1",
		RallyDice:     []int32{6},
		Unstoppable: &pb.DaggerheartUnstoppableState{
			Active:       true,
			CurrentValue: 3,
		},
		ActiveBeastform: &pb.DaggerheartActiveBeastformState{
			BeastformId: "wolf",
			AttackTrait: "Agility",
			AttackRange: "melee",
			DamageDice:  []*pb.DaggerheartBeastformAttackDie{{Count: 1, Sides: 8}},
			DamageType:  "physical",
		},
	})
	if classState == nil || classState.Unstoppable == nil || classState.ActiveBeastform == nil {
		t.Fatalf("classStateFromProto() = %#v", classState)
	}

	subclassState := subclassStateFromProto(&pb.DaggerheartSubclassState{
		TranscendenceActive:           true,
		TranscendenceTraitBonusTarget: "Presence",
		ElementalChannel:              "Fire",
	})
	if subclassState == nil || !subclassState.TranscendenceActive || subclassState.ElementalChannel != "Fire" {
		t.Fatalf("subclassStateFromProto() = %#v", subclassState)
	}

	armor := temporaryArmorFromProto([]*pb.DaggerheartTemporaryArmorBucket{{Source: "Spell", Amount: 2}})
	if len(armor) != 1 || armor[0].Amount != 2 {
		t.Fatalf("temporaryArmorFromProto() = %#v", armor)
	}
	modifiers := statModifiersFromProto([]*pb.DaggerheartStatModifier{{Target: "Agility", Delta: 1, Label: "Blessing"}})
	if len(modifiers) != 1 || modifiers[0].Label != "Blessing" {
		t.Fatalf("statModifiersFromProto() = %#v", modifiers)
	}

	if label := contentLabelFromID("class:ranger"); label != "Ranger" {
		t.Fatalf("contentLabelFromID() = %q", label)
	}
	name, domain := domainCardLabelFromID("domain:valor-rally")
	if name != "Rally" || domain != "Valor" {
		t.Fatalf("domainCardLabelFromID() = (%q, %q)", name, domain)
	}
	if slug := humanizeContentSlug("stone_guard"); slug != "Stone Guard" {
		t.Fatalf("humanizeContentSlug() = %q", slug)
	}
	if ptr := intPtrIfNonZero(0); ptr != nil {
		t.Fatalf("intPtrIfNonZero(0) = %#v, want nil", ptr)
	}
	if ptr := intPtrFromWrapper(wrapperspb.Int32(5)); ptr == nil || *ptr != 5 {
		t.Fatalf("intPtrFromWrapper() = %#v", ptr)
	}
	if kind := characterKindToString(statev1.CharacterKind_PC); kind != "PC" {
		t.Fatalf("characterKindToString() = %q", kind)
	}
	if spotlight := sessionSpotlightTypeToString(statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER); spotlight != "CHARACTER" {
		t.Fatalf("sessionSpotlightTypeToString() = %q", spotlight)
	}
	if origin := daggerheartSubclassTrackOriginToString(pb.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS); origin != "MULTICLASS" {
		t.Fatalf("daggerheartSubclassTrackOriginToString() = %q", origin)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
