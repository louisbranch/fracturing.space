package daggerhearttools

import "testing"

func TestResolveAttackTargetIDUsesBoardDiagnostics(t *testing.T) {
	t.Run("single adversary infers target", func(t *testing.T) {
		targetID, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{
			Status:      "READY",
			Adversaries: []adversarySummary{{ID: "adv-1"}},
		})
		if err != nil {
			t.Fatalf("resolveAttackTargetID: %v", err)
		}
		if targetID != "adv-1" {
			t.Fatalf("target_id = %q, want adv-1", targetID)
		}
	})

	t.Run("no active scene returns corrective guidance", func(t *testing.T) {
		_, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{Status: "NO_ACTIVE_SCENE"})
		if err == nil || err.Error() != "cannot infer target_id because the combat board has no active scene; use interaction_state_read or interaction_activate_scene, then retry" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("multiple adversaries require explicit target", func(t *testing.T) {
		_, err := resolveAttackTargetID("", daggerheartCombatBoardPayload{
			Status:      "READY",
			Adversaries: []adversarySummary{{ID: "adv-1"}, {ID: "adv-2"}},
		})
		if err == nil || err.Error() != "target_id is required when the combat board has multiple visible adversaries; read daggerheart_combat_board_read and specify the intended target_id" {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestInferredAttackProfiles(t *testing.T) {
	t.Run("primary weapon inference", func(t *testing.T) {
		profile := inferredPrimaryWeaponAttackProfile(characterSheetPayload{
			Daggerheart: &daggerheartCharacterSheetState{
				Equipment: &equipmentSummary{
					PrimaryWeapon: &weaponSummary{
						Name:       "Longsword",
						Trait:      "Strength",
						Range:      "MELEE",
						DamageDice: "1d10",
						DamageType: "PHYSICAL",
					},
				},
			},
		}, "char-1")
		if profile == nil || profile.Standard == nil {
			t.Fatalf("profile = %#v", profile)
		}
		if profile.Standard.Trait != "Strength" || len(profile.Standard.DamageDice) != 1 || profile.Standard.DamageDice[0].Sides != 10 {
			t.Fatalf("standard attack = %#v", profile.Standard)
		}
		if profile.Damage == nil || profile.Damage.Source != "Longsword" || profile.Damage.DamageType != "PHYSICAL" {
			t.Fatalf("damage = %#v", profile.Damage)
		}
	})

	t.Run("active beastform inference", func(t *testing.T) {
		profile := inferredBeastformAttackProfile(characterSheetPayload{
			Daggerheart: &daggerheartCharacterSheetState{
				ClassState: &classStateSummary{
					ActiveBeastform: &beastformSummary{
						BeastformID: "beastform.wolf",
						AttackTrait: "Agility",
						AttackRange: "MELEE",
						DamageDice:  []damageDieSpec{{Count: 2, Sides: 8}},
						DamageType:  "PHYSICAL",
					},
				},
			},
		}, "char-1")
		if profile == nil || profile.Beastform == nil {
			t.Fatalf("profile = %#v", profile)
		}
		if profile.Damage == nil || profile.Damage.Source != "beastform.wolf" {
			t.Fatalf("damage = %#v", profile.Damage)
		}
	})
}

func TestParseDamageDiceString(t *testing.T) {
	dice, ok := parseDamageDiceString("2d8 + d6")
	if !ok {
		t.Fatal("expected parse success")
	}
	if len(dice) != 2 {
		t.Fatalf("dice length = %d, want 2", len(dice))
	}
	if dice[0] != (rollDiceSpec{Count: 2, Sides: 8}) || dice[1] != (rollDiceSpec{Count: 1, Sides: 6}) {
		t.Fatalf("dice = %#v", dice)
	}
}

func TestMergeAttackDamageSpecFillsInferredDefaults(t *testing.T) {
	merged := mergeAttackDamageSpec(&attackDamageSpecInput{
		Source: "Player override",
	}, &attackDamageSpecInput{
		DamageType:         "PHYSICAL",
		Source:             "Longsword",
		SourceCharacterIDs: []string{"char-1"},
	})
	if merged == nil {
		t.Fatal("expected merged damage")
	}
	if merged.Source != "Player override" {
		t.Fatalf("source = %q, want Player override", merged.Source)
	}
	if merged.DamageType != "PHYSICAL" {
		t.Fatalf("damage_type = %q, want PHYSICAL", merged.DamageType)
	}
	if len(merged.SourceCharacterIDs) != 1 || merged.SourceCharacterIDs[0] != "char-1" {
		t.Fatalf("source_character_ids = %#v", merged.SourceCharacterIDs)
	}
}

func TestNormalizeAttackProfilesPrefersMeaningfulStandardAttack(t *testing.T) {
	standard, beastform := normalizeAttackProfiles(&standardAttackProfileInput{
		Trait:       "Agility",
		DamageDice:  []rollDiceSpec{{Count: 1, Sides: 10}},
		AttackRange: "MELEE",
	}, &beastformAttackProfileInput{})
	if standard == nil || beastform != nil {
		t.Fatalf("normalized profiles = (%#v, %#v)", standard, beastform)
	}

	standard, beastform = normalizeAttackProfiles(&standardAttackProfileInput{}, &beastformAttackProfileInput{})
	if standard != nil || beastform == nil {
		t.Fatalf("zero standard should fall back to beastform: (%#v, %#v)", standard, beastform)
	}
}
