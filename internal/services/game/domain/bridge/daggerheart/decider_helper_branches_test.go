package daggerheart

import "testing"

func TestIsCharacterStatePatchNoMutation_Branches(t *testing.T) {
	state := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {
				CharacterID: "char-1",
				HP:          0,
				Hope:        2,
				HopeMax:     6,
				Stress:      1,
				Armor:       1,
				LifeState:   LifeStateAlive,
			},
		},
	}
	zero := 0
	one := 1
	two := 2
	six := 6

	tests := []struct {
		name    string
		payload CharacterStatePatchPayload
		want    bool
	}{
		{
			name: "missing character is never no-mutation",
			payload: CharacterStatePatchPayload{
				CharacterID: "missing",
				HopeAfter:   &two,
			},
			want: false,
		},
		{
			name: "unchanged fields is no-mutation",
			payload: CharacterStatePatchPayload{
				CharacterID:  "char-1",
				HPAfter:      &zero,
				HopeAfter:    &two,
				HopeMaxAfter: &six,
				StressAfter:  &one,
				ArmorAfter:   &one,
			},
			want: true,
		},
		{
			name: "hp before mismatch branch when current hp is zero",
			payload: CharacterStatePatchPayload{
				CharacterID: "char-1",
				HPBefore:    &one,
			},
			want: false,
		},
		{
			name: "life state change is mutation",
			payload: CharacterStatePatchPayload{
				CharacterID:    "char-1",
				LifeStateAfter: strPtr(LifeStateDead),
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isCharacterStatePatchNoMutation(state, tc.payload)
			if got != tc.want {
				t.Fatalf("isCharacterStatePatchNoMutation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsConditionChangeNoMutation_NormalizationErrorBranches(t *testing.T) {
	stateInvalidCurrent := SnapshotState{
		CharacterStates: map[string]CharacterState{
			"char-1": {CharacterID: "char-1", Conditions: []string{""}},
		},
	}
	if got := isConditionChangeNoMutation(stateInvalidCurrent, ConditionChangePayload{
		CharacterID:     "char-1",
		ConditionsAfter: []string{"hidden"},
	}); got {
		t.Fatal("expected false when current conditions fail normalization")
	}

	stateValid := SnapshotState{
		CharacterStates: map[string]CharacterState{
			"char-1": {CharacterID: "char-1", Conditions: []string{"hidden"}},
		},
	}
	if got := isConditionChangeNoMutation(stateValid, ConditionChangePayload{
		CharacterID:     "char-1",
		ConditionsAfter: []string{""},
	}); got {
		t.Fatal("expected false when payload conditions fail normalization")
	}
}

func TestSnapshotCharacterState_DefaultsLifeStateAndCampaignID(t *testing.T) {
	snapshot := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {CharacterID: "char-1", HP: 5},
		},
	}

	character, ok := snapshotCharacterState(snapshot, " char-1 ")
	if !ok {
		t.Fatal("expected snapshotCharacterState to resolve character")
	}
	if character.CampaignID != "camp-1" {
		t.Fatalf("CampaignID = %s, want camp-1", character.CampaignID)
	}
	if character.LifeState != LifeStateAlive {
		t.Fatalf("LifeState = %s, want %s", character.LifeState, LifeStateAlive)
	}
}

func TestIsCountdownUpdateNoMutation_LoopedBranch(t *testing.T) {
	snapshot := SnapshotState{
		CountdownStates: map[string]CountdownState{
			"cd-1": {CountdownID: "cd-1", Current: 3, Looping: false},
		},
	}
	if got := isCountdownUpdateNoMutation(snapshot, CountdownUpdatePayload{
		CountdownID: "cd-1",
		After:       3,
		Looped:      true,
	}); got {
		t.Fatal("expected looped=true with non-looping countdown to be mutation")
	}
}

func TestSnapshotCountdownState_BlankIDReturnsFalse(t *testing.T) {
	if _, ok := snapshotCountdownState(SnapshotState{}, "  "); ok {
		t.Fatal("expected blank countdown id to return false")
	}
}

func strPtr(v string) *string {
	return &v
}
