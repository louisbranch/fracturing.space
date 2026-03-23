package daggerheart

import (
	"strings"
	"testing"

	daggerheartfolder "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/folder"
	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestIsCharacterStatePatchNoMutation_FieldMismatchBranches(t *testing.T) {
	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {
				CharacterID: "char-1",
				HP:          6,
				Hope:        2,
				HopeMax:     6,
				Stress:      1,
				Armor:       1,
				LifeState:   daggerheartstate.LifeStateAlive,
			},
		},
	}

	tests := []struct {
		name    string
		payload daggerheartpayload.CharacterStatePatchPayload
	}{
		{
			name: "hp after mismatch",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID: "char-1",
				HPAfter:     intPtr(5),
			},
		},
		{
			name: "hope after mismatch",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID: "char-1",
				HopeAfter:   intPtr(3),
			},
		},
		{
			name: "hope max after mismatch",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID:  "char-1",
				HopeMaxAfter: intPtr(5),
			},
		},
		{
			name: "stress after mismatch",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID: "char-1",
				StressAfter: intPtr(2),
			},
		},
		{
			name: "armor after mismatch",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID: "char-1",
				ArmorAfter:  intPtr(2),
			},
		},
		{
			name: "life state after mismatch",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID:    "char-1",
				LifeStateAfter: strPtr(mechanics.LifeStateDead),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := daggerheartdecider.IsCharacterStatePatchNoMutation(state, tc.payload); got {
				t.Fatalf("daggerheartdecider.IsCharacterStatePatchNoMutation() = true, want false for %s", tc.name)
			}
		})
	}
}

func TestHasMissingConditionRemovals_Branches(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		removed []string
		want    bool
	}{
		{
			name:    "invalid current conditions are ignored",
			current: []string{""},
			removed: []string{rules.ConditionHidden},
			want:    false,
		},
		{
			name:    "invalid removed conditions are ignored",
			current: []string{rules.ConditionHidden},
			removed: []string{""},
			want:    false,
		},
		{
			name:    "missing removal returns true",
			current: []string{rules.ConditionHidden},
			removed: []string{rules.ConditionVulnerable},
			want:    true,
		},
		{
			name:    "existing removal returns false",
			current: []string{rules.ConditionHidden, rules.ConditionVulnerable},
			removed: []string{rules.ConditionHidden},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := daggerheartdecider.HasMissingConditionRemovals(tc.current, tc.removed)
			if got != tc.want {
				t.Fatalf("daggerheartdecider.HasMissingConditionRemovals() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateConditionSetPayload_Branches(t *testing.T) {
	tests := []struct {
		name      string
		before    []string
		after     []string
		added     []string
		removed   []string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "conditions after required",
			wantErr:   true,
			errSubstr: "conditions must change",
		},
		{
			name:      "conditions before required when removed provided",
			after:     []string{rules.ConditionHidden},
			removed:   []string{rules.ConditionVulnerable},
			wantErr:   true,
			errSubstr: "added must match conditions_before and conditions_after diff",
		},
		{
			name:      "added mismatch with before",
			before:    []string{rules.ConditionHidden},
			after:     []string{rules.ConditionVulnerable},
			added:     []string{rules.ConditionHidden},
			removed:   []string{rules.ConditionHidden},
			wantErr:   true,
			errSubstr: "added must match conditions_before and conditions_after diff",
		},
		{
			name:      "added mismatch without before",
			after:     []string{rules.ConditionHidden},
			added:     []string{rules.ConditionVulnerable},
			wantErr:   true,
			errSubstr: "added must match conditions_before and conditions_after diff",
		},
		{
			name:      "removed mismatch with before",
			before:    []string{rules.ConditionHidden, rules.ConditionVulnerable},
			after:     []string{rules.ConditionHidden},
			removed:   []string{rules.ConditionHidden},
			wantErr:   true,
			errSubstr: "removed must match conditions_before and conditions_after diff",
		},
		{
			name:      "no mutation with before",
			before:    []string{rules.ConditionHidden},
			after:     []string{rules.ConditionHidden},
			wantErr:   true,
			errSubstr: "conditions must change",
		},
		{
			name:      "no mutation without before",
			after:     []string{},
			added:     []string{},
			removed:   []string{},
			wantErr:   true,
			errSubstr: "conditions must change",
		},
		{
			name:    "valid diff",
			before:  []string{rules.ConditionHidden},
			after:   []string{rules.ConditionVulnerable},
			added:   []string{rules.ConditionVulnerable},
			removed: []string{rules.ConditionHidden},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := daggerheartvalidator.ValidateConditionSetPayload(
				mustConditionStates(tc.before),
				mustConditionStates(tc.after),
				mustConditionStates(tc.added),
				mustConditionStates(tc.removed),
			)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tc.errSubstr != "" && !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateConditionSetPayload() error = %v, want nil", err)
			}
		})
	}
}

func mustConditionStates(codes []string) []rules.ConditionState {
	out := make([]rules.ConditionState, 0, len(codes))
	for _, code := range codes {
		out = append(out, mustConditionState(code))
	}
	return out
}

func TestFoldGMFearChangedAndCountdownUpdate_Branches(t *testing.T) {
	t.Run("gm fear rejects out of range", func(t *testing.T) {
		err := daggerheartfolder.FoldGMFearChanged(&daggerheartstate.SnapshotState{}, daggerheartpayload.GMFearChangedPayload{Value: daggerheartstate.GMFearMax + 1})
		if err == nil || !strings.Contains(err.Error(), "gm fear value must be in range") {
			t.Fatalf("foldGMFearChanged() error = %v, want range error", err)
		}
	})

	t.Run("countdown advance updates remaining value and status", func(t *testing.T) {
		state := &daggerheartstate.SnapshotState{
			SceneCountdownStates: map[dhids.CountdownID]daggerheartstate.SceneCountdownState{
				"cd-1": {CountdownID: "cd-1", StartingValue: 4, RemainingValue: 1, LoopBehavior: "reset", Status: "active"},
			},
		}
		if err := daggerheartfolder.FoldSceneCountdownAdvanced(state, daggerheartpayload.SceneCountdownAdvancedPayload{
			CountdownID:     "cd-1",
			BeforeRemaining: 1,
			AfterRemaining:  0,
			AdvancedBy:      1,
			StatusBefore:    "active",
			StatusAfter:     "trigger_pending",
		}); err != nil {
			t.Fatalf("foldCountdownUpdated() error = %v", err)
		}
		got := state.SceneCountdownStates["cd-1"]
		if got.RemainingValue != 0 {
			t.Fatalf("countdown remaining_value = %d, want 0", got.RemainingValue)
		}
		if got.Status != "trigger_pending" {
			t.Fatalf("countdown status = %q, want trigger_pending", got.Status)
		}
	})
}

func TestHasStringFieldChange_NilBranches(t *testing.T) {
	before := "before"
	after := "after"
	if daggerheartvalidator.HasStringFieldChange(&before, nil) {
		t.Fatal("expected nil after to be non-mutation")
	}
	if !daggerheartvalidator.HasStringFieldChange(nil, &after) {
		t.Fatal("expected nil before with non-nil after to be mutation")
	}
}
