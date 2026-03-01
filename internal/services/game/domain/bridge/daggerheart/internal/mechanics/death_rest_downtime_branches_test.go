package mechanics

import "testing"

func TestNormalizeDeathMoveValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "trim and lowercase", value: "  Avoid_Death  ", want: DeathMoveAvoidDeath},
		{name: "empty", value: " ", wantErr: true},
		{name: "unsupported", value: "stand_up", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeDeathMove(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDeathMove: %v", err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeDeathMove = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeLifeStateValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "trim and lowercase", value: "  UNCONSCIOUS ", want: LifeStateUnconscious},
		{name: "empty", value: " ", wantErr: true},
		{name: "unsupported", value: "wounded", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeLifeState(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeLifeState: %v", err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeLifeState = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveDeathMoveRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input DeathMoveInput
	}{
		{
			name: "invalid move",
			input: DeathMoveInput{
				Move: "unknown",
			},
		},
		{
			name: "level must be positive",
			input: DeathMoveInput{
				Move:  DeathMoveAvoidDeath,
				Level: 0,
			},
		},
		{
			name: "hp max must be non negative",
			input: DeathMoveInput{
				Move:  DeathMoveAvoidDeath,
				Level: 1,
				HPMax: -1,
			},
		},
		{
			name: "stress max must be non negative",
			input: DeathMoveInput{
				Move:      DeathMoveAvoidDeath,
				Level:     1,
				HPMax:     6,
				StressMax: -1,
			},
		},
		{
			name: "hope max must be in range",
			input: DeathMoveInput{
				Move:    DeathMoveAvoidDeath,
				Level:   1,
				HPMax:   6,
				HopeMax: HopeMax + 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ResolveDeathMove(tc.input); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestResolveDeathMoveAvoidDeathBranches(t *testing.T) {
	t.Run("scar gained", func(t *testing.T) {
		outcome, err := ResolveDeathMove(DeathMoveInput{
			Move:      DeathMoveAvoidDeath,
			Level:     12,
			HP:        0,
			HPMax:     6,
			Hope:      5,
			HopeMax:   5,
			Stress:    2,
			StressMax: 6,
			Seed:      10,
		})
		if err != nil {
			t.Fatalf("ResolveDeathMove: %v", err)
		}
		if !outcome.ScarGained {
			t.Fatalf("expected ScarGained=true")
		}
		if outcome.HopeMaxAfter != 4 {
			t.Fatalf("HopeMaxAfter = %d, want 4", outcome.HopeMaxAfter)
		}
		if outcome.HopeAfter != 4 {
			t.Fatalf("HopeAfter = %d, want 4", outcome.HopeAfter)
		}
	})

	t.Run("no scar gained", func(t *testing.T) {
		seed, outcome := findAvoidDeathOutcome(t, func(o DeathMoveOutcome) bool {
			return !o.ScarGained
		})
		if outcome.HopeDie == nil {
			t.Fatalf("expected HopeDie, seed=%d", seed)
		}
		if outcome.LifeState != LifeStateUnconscious {
			t.Fatalf("LifeState = %q, want %q", outcome.LifeState, LifeStateUnconscious)
		}
	})
}

func TestResolveDeathMoveRiskItAllBranches(t *testing.T) {
	base := DeathMoveInput{
		Move:      DeathMoveRiskItAll,
		Level:     1,
		HP:        2,
		HPMax:     6,
		Hope:      3,
		HopeMax:   6,
		Stress:    3,
		StressMax: 6,
	}

	t.Run("tie restores to full", func(t *testing.T) {
		seed, outcome := findRiskItAllOutcome(t, base, func(o DeathMoveOutcome) bool {
			return o.HopeDie != nil && o.FearDie != nil && *o.HopeDie == *o.FearDie
		})
		if outcome.LifeState != LifeStateAlive {
			t.Fatalf("LifeState = %q, want %q (seed=%d)", outcome.LifeState, LifeStateAlive, seed)
		}
		if outcome.HPAfter != base.HPMax || outcome.StressAfter != 0 {
			t.Fatalf("unexpected tie outcome: %+v (seed=%d)", outcome, seed)
		}
	})

	t.Run("hope beats fear uses default clear", func(t *testing.T) {
		seed, outcome := findRiskItAllOutcome(t, base, func(o DeathMoveOutcome) bool {
			return o.HopeDie != nil && o.FearDie != nil && *o.HopeDie > *o.FearDie
		})
		if outcome.LifeState != LifeStateAlive {
			t.Fatalf("LifeState = %q, want %q (seed=%d)", outcome.LifeState, LifeStateAlive, seed)
		}
		if outcome.HPCleared != *outcome.HopeDie || outcome.StressCleared != 0 {
			t.Fatalf("unexpected default clear outcome: %+v (seed=%d)", outcome, seed)
		}
	})

	t.Run("hope beats fear uses explicit clear values", func(t *testing.T) {
		seed, outcome := findRiskItAllOutcome(t, base, func(o DeathMoveOutcome) bool {
			return o.HopeDie != nil && o.FearDie != nil && *o.HopeDie > *o.FearDie && *o.HopeDie >= 2
		})
		hpClear := 1
		stressClear := 1
		explicit := base
		explicit.Seed = seed
		explicit.RiskItAllHPClear = &hpClear
		explicit.RiskItAllStClear = &stressClear
		got, err := ResolveDeathMove(explicit)
		if err != nil {
			t.Fatalf("ResolveDeathMove: %v", err)
		}
		if got.HPCleared != hpClear || got.StressCleared != stressClear {
			t.Fatalf("explicit clear outcome = %+v, want hp=%d stress=%d", got, hpClear, stressClear)
		}
		if outcome.HopeDie == nil {
			t.Fatalf("expected HopeDie for seed=%d", seed)
		}
	})

	t.Run("risk_it_all rejects invalid clear values", func(t *testing.T) {
		seed, outcome := findRiskItAllOutcome(t, base, func(o DeathMoveOutcome) bool {
			return o.HopeDie != nil && o.FearDie != nil && *o.HopeDie > *o.FearDie
		})

		negative := base
		negative.Seed = seed
		hpNeg := -1
		negative.RiskItAllHPClear = &hpNeg
		if _, err := ResolveDeathMove(negative); err == nil {
			t.Fatalf("expected error for negative clear values")
		}

		exceed := base
		exceed.Seed = seed
		hp := *outcome.HopeDie
		stress := 1
		exceed.RiskItAllHPClear = &hp
		exceed.RiskItAllStClear = &stress
		if _, err := ResolveDeathMove(exceed); err == nil {
			t.Fatalf("expected error for clear values exceeding hope die")
		}
	})

	t.Run("fear beats hope means death", func(t *testing.T) {
		seed, outcome := findRiskItAllOutcome(t, base, func(o DeathMoveOutcome) bool {
			return o.HopeDie != nil && o.FearDie != nil && *o.HopeDie < *o.FearDie
		})
		if outcome.LifeState != LifeStateDead {
			t.Fatalf("LifeState = %q, want %q (seed=%d)", outcome.LifeState, LifeStateDead, seed)
		}
	})
}

func TestResolveRestOutcomeBranches(t *testing.T) {
	t.Run("short rest interrupted is not applied", func(t *testing.T) {
		outcome, err := ResolveRestOutcome(RestState{ConsecutiveShortRests: 1}, RestTypeShort, true, 11, 3)
		if err != nil {
			t.Fatalf("ResolveRestOutcome: %v", err)
		}
		if outcome.Applied {
			t.Fatalf("Applied = true, want false")
		}
		if outcome.State.ConsecutiveShortRests != 1 {
			t.Fatalf("ConsecutiveShortRests = %d, want 1", outcome.State.ConsecutiveShortRests)
		}
	})

	t.Run("short rest applies and increments short-rest chain", func(t *testing.T) {
		outcome, err := ResolveRestOutcome(RestState{ConsecutiveShortRests: 1}, RestTypeShort, false, 22, 4)
		if err != nil {
			t.Fatalf("ResolveRestOutcome: %v", err)
		}
		if !outcome.Applied {
			t.Fatalf("Applied = false, want true")
		}
		if outcome.EffectiveType != RestTypeShort {
			t.Fatalf("EffectiveType = %d, want %d", outcome.EffectiveType, RestTypeShort)
		}
		if outcome.State.ConsecutiveShortRests != 2 {
			t.Fatalf("ConsecutiveShortRests = %d, want 2", outcome.State.ConsecutiveShortRests)
		}
		if outcome.AdvanceCountdown {
			t.Fatalf("AdvanceCountdown = true, want false for short rest")
		}
	})

	t.Run("interrupted long rest downgrades to short rest", func(t *testing.T) {
		outcome, err := ResolveRestOutcome(RestState{ConsecutiveShortRests: 2}, RestTypeLong, true, 33, 4)
		if err != nil {
			t.Fatalf("ResolveRestOutcome: %v", err)
		}
		if outcome.EffectiveType != RestTypeShort {
			t.Fatalf("EffectiveType = %d, want %d", outcome.EffectiveType, RestTypeShort)
		}
		if outcome.State.ConsecutiveShortRests != 3 {
			t.Fatalf("ConsecutiveShortRests = %d, want 3", outcome.State.ConsecutiveShortRests)
		}
	})

	t.Run("long rest resets short-rest chain and clamps negative party size", func(t *testing.T) {
		outcome, err := ResolveRestOutcome(RestState{ConsecutiveShortRests: 3}, RestTypeLong, false, 44, -10)
		if err != nil {
			t.Fatalf("ResolveRestOutcome: %v", err)
		}
		if outcome.EffectiveType != RestTypeLong {
			t.Fatalf("EffectiveType = %d, want %d", outcome.EffectiveType, RestTypeLong)
		}
		if outcome.State.ConsecutiveShortRests != 0 {
			t.Fatalf("ConsecutiveShortRests = %d, want 0", outcome.State.ConsecutiveShortRests)
		}
		if !outcome.AdvanceCountdown {
			t.Fatalf("AdvanceCountdown = false, want true")
		}
		if !outcome.RefreshLongRest {
			t.Fatalf("RefreshLongRest = false, want true")
		}
	})
}

func TestApplyDowntimeMoveBranches(t *testing.T) {
	t.Run("clear all stress", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Stress:      4,
			StressMax:   6,
		})
		result := ApplyDowntimeMove(state, DowntimeClearAllStress, DowntimeOptions{})
		if result.StressAfter != 0 || state.Stress != 0 {
			t.Fatalf("stress after clear = %d, want 0", result.StressAfter)
		}
	})

	t.Run("prepare move gains hope based on group option", func(t *testing.T) {
		stateSolo := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Hope:        1,
			HopeMax:     6,
		})
		solo := ApplyDowntimeMove(stateSolo, DowntimePrepare, DowntimeOptions{PrepareWithGroup: false})
		if solo.HopeAfter != 2 {
			t.Fatalf("solo prepare HopeAfter = %d, want 2", solo.HopeAfter)
		}

		stateGroup := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Hope:        1,
			HopeMax:     6,
		})
		group := ApplyDowntimeMove(stateGroup, DowntimePrepare, DowntimeOptions{PrepareWithGroup: true})
		if group.HopeAfter != 3 {
			t.Fatalf("group prepare HopeAfter = %d, want 3", group.HopeAfter)
		}
	})

	t.Run("work on project has no state change", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Hope:        2,
			HopeMax:     6,
			Stress:      1,
			StressMax:   6,
			Armor:       1,
			ArmorMax:    2,
		})
		before := *state
		result := ApplyDowntimeMove(state, DowntimeWorkOnProject, DowntimeOptions{})
		if result.HopeAfter != before.Hope || result.StressAfter != before.Stress || result.ArmorAfter != before.Armor {
			t.Fatalf("unexpected downtime result: %+v", result)
		}
	})
}

func findAvoidDeathOutcome(t *testing.T, match func(DeathMoveOutcome) bool) (int64, DeathMoveOutcome) {
	t.Helper()
	for seed := int64(1); seed <= 5000; seed++ {
		outcome, err := ResolveDeathMove(DeathMoveInput{
			Move:      DeathMoveAvoidDeath,
			Level:     1,
			HP:        0,
			HPMax:     6,
			Hope:      4,
			HopeMax:   6,
			Stress:    2,
			StressMax: 6,
			Seed:      seed,
		})
		if err != nil {
			t.Fatalf("ResolveDeathMove avoid_death seed=%d: %v", seed, err)
		}
		if match(outcome) {
			return seed, outcome
		}
	}
	t.Fatalf("unable to find avoid_death outcome matching predicate")
	return 0, DeathMoveOutcome{}
}

func findRiskItAllOutcome(t *testing.T, input DeathMoveInput, match func(DeathMoveOutcome) bool) (int64, DeathMoveOutcome) {
	t.Helper()
	for seed := int64(1); seed <= 10000; seed++ {
		input.Seed = seed
		outcome, err := ResolveDeathMove(input)
		if err != nil {
			t.Fatalf("ResolveDeathMove risk_it_all seed=%d: %v", seed, err)
		}
		if match(outcome) {
			return seed, outcome
		}
	}
	t.Fatalf("unable to find risk_it_all outcome matching predicate")
	return 0, DeathMoveOutcome{}
}
