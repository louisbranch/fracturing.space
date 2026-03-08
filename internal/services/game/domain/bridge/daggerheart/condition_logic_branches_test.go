package daggerheart

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestIsCharacterStatePatchNoMutation_FieldMismatchBranches(t *testing.T) {
	state := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]CharacterState{
			"char-1": {
				CharacterID: "char-1",
				HP:          6,
				Hope:        2,
				HopeMax:     6,
				Stress:      1,
				Armor:       1,
				LifeState:   LifeStateAlive,
			},
		},
	}

	tests := []struct {
		name    string
		payload CharacterStatePatchPayload
	}{
		{
			name: "hp after mismatch",
			payload: CharacterStatePatchPayload{
				CharacterID: "char-1",
				HPAfter:     intPtr(5),
			},
		},
		{
			name: "hope after mismatch",
			payload: CharacterStatePatchPayload{
				CharacterID: "char-1",
				HopeAfter:   intPtr(3),
			},
		},
		{
			name: "hope max after mismatch",
			payload: CharacterStatePatchPayload{
				CharacterID:  "char-1",
				HopeMaxAfter: intPtr(5),
			},
		},
		{
			name: "stress after mismatch",
			payload: CharacterStatePatchPayload{
				CharacterID: "char-1",
				StressAfter: intPtr(2),
			},
		},
		{
			name: "armor after mismatch",
			payload: CharacterStatePatchPayload{
				CharacterID: "char-1",
				ArmorAfter:  intPtr(2),
			},
		},
		{
			name: "life state after mismatch",
			payload: CharacterStatePatchPayload{
				CharacterID:    "char-1",
				LifeStateAfter: strPtr(LifeStateDead),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isCharacterStatePatchNoMutation(state, tc.payload); got {
				t.Fatalf("isCharacterStatePatchNoMutation() = true, want false for %s", tc.name)
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
			removed: []string{ConditionHidden},
			want:    false,
		},
		{
			name:    "invalid removed conditions are ignored",
			current: []string{ConditionHidden},
			removed: []string{""},
			want:    false,
		},
		{
			name:    "missing removal returns true",
			current: []string{ConditionHidden},
			removed: []string{ConditionVulnerable},
			want:    true,
		},
		{
			name:    "existing removal returns false",
			current: []string{ConditionHidden, ConditionVulnerable},
			removed: []string{ConditionHidden},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasMissingConditionRemovals(tc.current, tc.removed)
			if got != tc.want {
				t.Fatalf("hasMissingConditionRemovals() = %v, want %v", got, tc.want)
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
			errSubstr: "conditions_after is required",
		},
		{
			name:      "conditions before required when removed provided",
			after:     []string{ConditionHidden},
			removed:   []string{ConditionVulnerable},
			wantErr:   true,
			errSubstr: "conditions_before is required when removed are provided",
		},
		{
			name:      "added mismatch with before",
			before:    []string{ConditionHidden},
			after:     []string{ConditionVulnerable},
			added:     []string{ConditionHidden},
			removed:   []string{ConditionHidden},
			wantErr:   true,
			errSubstr: "added must match conditions_before and conditions_after diff",
		},
		{
			name:      "added mismatch without before",
			after:     []string{ConditionHidden},
			added:     []string{ConditionVulnerable},
			wantErr:   true,
			errSubstr: "added must match conditions_after when conditions_before is omitted",
		},
		{
			name:      "removed mismatch with before",
			before:    []string{ConditionHidden, ConditionVulnerable},
			after:     []string{ConditionHidden},
			removed:   []string{ConditionHidden},
			wantErr:   true,
			errSubstr: "removed must match conditions_before and conditions_after diff",
		},
		{
			name:      "no mutation with before",
			before:    []string{ConditionHidden},
			after:     []string{ConditionHidden},
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
			before:  []string{ConditionHidden},
			after:   []string{ConditionVulnerable},
			added:   []string{ConditionVulnerable},
			removed: []string{ConditionHidden},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConditionSetPayload(tc.before, tc.after, tc.added, tc.removed)
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

func TestFoldGMFearChangedAndCountdownUpdate_Branches(t *testing.T) {
	t.Run("gm fear rejects out of range", func(t *testing.T) {
		err := foldGMFearChanged(&SnapshotState{}, GMFearChangedPayload{Value: GMFearMax + 1})
		if err == nil || !strings.Contains(err.Error(), "gm fear value must be in range") {
			t.Fatalf("foldGMFearChanged() error = %v, want range error", err)
		}
	})

	t.Run("countdown update sets looping on looped payload", func(t *testing.T) {
		state := &SnapshotState{
			CountdownStates: map[ids.CountdownID]CountdownState{
				"cd-1": {CountdownID: "cd-1", Current: 1, Looping: false},
			},
		}
		if err := foldCountdownUpdated(state, CountdownUpdatedPayload{
			CountdownID: "cd-1",
			Value:       2,
			Looped:      true,
		}); err != nil {
			t.Fatalf("foldCountdownUpdated() error = %v", err)
		}
		got := state.CountdownStates["cd-1"]
		if got.Current != 2 {
			t.Fatalf("countdown current = %d, want 2", got.Current)
		}
		if !got.Looping {
			t.Fatal("countdown looping = false, want true")
		}
	})
}

func TestHasStringFieldChange_NilBranches(t *testing.T) {
	before := "before"
	after := "after"
	if hasStringFieldChange(&before, nil) {
		t.Fatal("expected nil after to be non-mutation")
	}
	if !hasStringFieldChange(nil, &after) {
		t.Fatal("expected nil before with non-nil after to be mutation")
	}
}
