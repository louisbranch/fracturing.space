package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestIsCharacterStatePatchNoMutation_Branches(t *testing.T) {
	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {
				CharacterID: "char-1",
				HP:          0,
				Hope:        2,
				HopeMax:     6,
				Stress:      1,
				Armor:       1,
				LifeState:   daggerheartstate.LifeStateAlive,
			},
		},
	}
	zero := 0
	one := 1
	two := 2
	six := 6

	tests := []struct {
		name    string
		payload daggerheartpayload.CharacterStatePatchPayload
		want    bool
	}{
		{
			name: "missing character is never no-mutation",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID: "missing",
				HopeAfter:   &two,
			},
			want: false,
		},
		{
			name: "unchanged fields is no-mutation",
			payload: daggerheartpayload.CharacterStatePatchPayload{
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
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID: "char-1",
				HPBefore:    &one,
			},
			want: false,
		},
		{
			name: "life state change is mutation",
			payload: daggerheartpayload.CharacterStatePatchPayload{
				CharacterID:    "char-1",
				LifeStateAfter: strPtr(mechanics.LifeStateDead),
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := daggerheartdecider.IsCharacterStatePatchNoMutation(state, tc.payload)
			if got != tc.want {
				t.Fatalf("daggerheartdecider.IsCharacterStatePatchNoMutation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsConditionChangeNoMutation_NormalizationErrorBranches(t *testing.T) {
	stateInvalidCurrent := daggerheartstate.SnapshotState{
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CharacterID: "char-1", Conditions: []string{""}},
		},
	}
	if got := daggerheartdecider.IsConditionChangeNoMutation(stateInvalidCurrent, daggerheartpayload.ConditionChangePayload{
		CharacterID:     "char-1",
		ConditionsAfter: []rules.ConditionState{mustConditionState("hidden")},
	}); got {
		t.Fatal("expected false when current conditions fail normalization")
	}

	stateValid := daggerheartstate.SnapshotState{
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CharacterID: "char-1", Conditions: []string{"hidden"}},
		},
	}
	if got := daggerheartdecider.IsConditionChangeNoMutation(stateValid, daggerheartpayload.ConditionChangePayload{
		CharacterID:     "char-1",
		ConditionsAfter: []rules.ConditionState{{Code: ""}},
	}); got {
		t.Fatal("expected false when payload conditions fail normalization")
	}
}

func TestSnapshotCharacterState_DefaultsLifeStateAndCampaignID(t *testing.T) {
	snapshot := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CharacterID: "char-1", HP: 5},
		},
	}

	character, ok := daggerheartdecider.SnapshotCharacterState(snapshot, ids.CharacterID(" char-1 "))
	if !ok {
		t.Fatal("expected daggerheartdecider.SnapshotCharacterState to resolve character")
	}
	if character.CampaignID != "camp-1" {
		t.Fatalf("CampaignID = %s, want camp-1", character.CampaignID)
	}
	if character.LifeState != daggerheartstate.LifeStateAlive {
		t.Fatalf("LifeState = %s, want %s", character.LifeState, daggerheartstate.LifeStateAlive)
	}
}

func TestIsCountdownUpdateNoMutation_LoopedBranch(t *testing.T) {
	snapshot := daggerheartstate.SnapshotState{
		CountdownStates: map[dhids.CountdownID]daggerheartstate.CountdownState{
			"cd-1": {CountdownID: "cd-1", Current: 3, Looping: false},
		},
	}
	if got := daggerheartdecider.IsCountdownUpdateNoMutation(snapshot, daggerheartpayload.CountdownUpdatePayload{
		CountdownID: "cd-1",
		After:       3,
		Looped:      true,
	}); got {
		t.Fatal("expected looped=true with non-looping countdown to be mutation")
	}
}

func TestSnapshotCountdownState_BlankIDReturnsFalse(t *testing.T) {
	if _, ok := daggerheartdecider.SnapshotCountdownState(daggerheartstate.SnapshotState{}, dhids.CountdownID("  ")); ok {
		t.Fatal("expected blank countdown id to return false")
	}
}

func strPtr(v string) *string {
	return &v
}

func TestApplyLevelUpToCharacterProfile_AllAdvancementBranches(t *testing.T) {
	profile := &daggerheartstate.CharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		Proficiency:     1,
		MajorThreshold:  3,
		SevereThreshold: 6,
		DomainCardIDs:   []string{"card-existing"},
	}

	applyLevelUpToCharacterProfile(profile, daggerheartpayload.LevelUpAppliedPayload{
		Level:          2,
		ThresholdDelta: 1,
		Advancements: []daggerheartpayload.LevelUpAdvancementPayload{
			{Type: "trait_increase", Trait: "agility"},
			{Type: "add_hp_slots"},
			{Type: "add_stress_slots"},
			{Type: "increase_evasion"},
			{Type: "increase_proficiency"},
			{Type: "increase_experience"},
			{Type: "domain_card", DomainCardID: "card-2"},
			{Type: "upgraded_subclass"},
		},
		Rewards: []daggerheartpayload.LevelUpRewardPayload{{Type: "domain_card", DomainCardID: "card-3", DomainCardLevel: 2}},
	})

	if profile.Level != 2 || profile.HpMax != 7 || profile.StressMax != 7 || profile.Evasion != 11 || profile.Proficiency != 2 {
		t.Fatalf("profile core fields = %+v, want leveled-up fields applied", profile)
	}
	if profile.MajorThreshold != 4 || profile.SevereThreshold != 8 {
		t.Fatalf("thresholds = (%d, %d), want (4, 8)", profile.MajorThreshold, profile.SevereThreshold)
	}
	if profile.Agility != 1 {
		t.Fatalf("agility = %d, want 1", profile.Agility)
	}
	if len(profile.DomainCardIDs) != 3 {
		t.Fatalf("domain card count = %d, want 3", len(profile.DomainCardIDs))
	}

	applyLevelUpToCharacterProfile(nil, daggerheartpayload.LevelUpAppliedPayload{Level: 3})
}

func TestDecideRestTake_EmitsDowntimeMoveEventsAndTrimsFields(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	payloadJSON, err := json.Marshal(daggerheartpayload.RestTakePayload{
		RestType:         " short ",
		GMFearBefore:     1,
		GMFearAfter:      2,
		ShortRestsBefore: 0,
		ShortRestsAfter:  1,
		Participants:     []ids.CharacterID{"char-1"},
		DowntimeMoves: []daggerheartpayload.DowntimeMoveAppliedPayload{{
			ActorCharacterID:  " char-1 ",
			TargetCharacterID: " char-2 ",
			Move:              " prepare ",
			GroupID:           " campfire ",
			RestType:          " short ",
			Hope:              intPtr(3),
		}},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	decision := daggerheartdecider.DecideRestTake(daggerheartstate.SnapshotState{}, command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "",
		PayloadJSON:   payloadJSON,
	}, func() time.Time { return now })

	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 2 {
		t.Fatalf("event count = %d, want 2", len(decision.Events))
	}

	move := decision.Events[1]
	if move.Type != event.Type(daggerheartpayload.EventTypeDowntimeMoveApplied) {
		t.Fatalf("downtime event type = %s, want %s", move.Type, daggerheartpayload.EventTypeDowntimeMoveApplied)
	}
	if move.EntityType != "character" || move.EntityID != "char-1" {
		t.Fatalf("downtime event entity = (%s, %s), want (character, char-1)", move.EntityType, move.EntityID)
	}

	var payload daggerheartpayload.DowntimeMoveAppliedPayload
	if err := json.Unmarshal(move.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal move payload: %v", err)
	}
	if payload.ActorCharacterID != "char-1" || payload.TargetCharacterID != "char-2" || payload.GroupID != "campfire" || payload.Move != "prepare" || payload.RestType != "short" {
		t.Fatalf("move payload = %+v, want trimmed values", payload)
	}
}
