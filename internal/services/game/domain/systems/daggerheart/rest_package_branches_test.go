package daggerheart

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestResolveRestPackage_InterruptedShortRestRejectsCountdownAdvance(t *testing.T) {
	t.Parallel()

	_, err := ResolveRestPackage(RestPackageInput{
		RestType:    RestTypeShort,
		Interrupted: true,
		Participants: []RestParticipantInput{{
			CharacterID: "char-1",
			State:       testRestState("char-1", 5, 6, 1, 3, 0, 3, 0, 2),
		}},
		LongTermCountdown: &Countdown{ID: "cd-1", Name: "Travel", Kind: CountdownKindProgress, Current: 1, Max: 4, Direction: CountdownDirectionIncrease},
	})
	if err == nil || !strings.Contains(err.Error(), "interrupted short rests cannot advance a countdown") {
		t.Fatalf("ResolveRestPackage() error = %v, want interrupted countdown error", err)
	}
}

func TestNormalizeRestParticipants_RejectsMissingAndDuplicateIDs(t *testing.T) {
	t.Parallel()

	if _, _, err := normalizeRestParticipants([]RestParticipantInput{{CharacterID: ""}}); err == nil || !strings.Contains(err.Error(), "character_id is required") {
		t.Fatalf("normalizeRestParticipants missing id error = %v", err)
	}

	if _, _, err := normalizeRestParticipants([]RestParticipantInput{{CharacterID: "char-1"}, {CharacterID: "char-1"}}); err == nil || !strings.Contains(err.Error(), "is duplicated") {
		t.Fatalf("normalizeRestParticipants duplicate id error = %v", err)
	}
}

func TestCountPrepareGroups_IgnoresBlankAndDuplicateGroupsPerParticipant(t *testing.T) {
	t.Parallel()

	counts := countPrepareGroups([]RestParticipantInput{
		{
			CharacterID: "char-1",
			Moves: []DowntimeSelection{
				{Move: DowntimeMovePrepare, GroupID: "team"},
				{Move: DowntimeMovePrepare, GroupID: "team"},
				{Move: DowntimeMovePrepare, GroupID: " "},
				{Move: DowntimeMoveClearStress, GroupID: "team"},
			},
		},
		{
			CharacterID: "char-2",
			Moves: []DowntimeSelection{
				{Move: DowntimeMovePrepare, GroupID: "team"},
			},
		},
	})

	if got := counts["team"]; got != 2 {
		t.Fatalf("prepare group count = %d, want 2", got)
	}
}

func TestResolveDowntimeSelection_BranchesAndValidation(t *testing.T) {
	t.Parallel()

	participant := RestParticipantInput{
		CharacterID: "char-1",
		Level:       2,
		State:       testRestState("char-1", 4, 6, 1, 4, 2, 4, 0, 2),
	}
	states := map[ids.CharacterID]*CharacterState{
		"char-1": ptrState(testRestState("char-1", 4, 6, 1, 4, 2, 4, 0, 2)),
	}

	t.Run("missing move", func(t *testing.T) {
		_, _, err := resolveDowntimeSelection(RestTypeShort, participant, DowntimeSelection{}, states, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "downtime move is required") {
			t.Fatalf("resolveDowntimeSelection missing move error = %v", err)
		}
	})

	t.Run("rest type disallows move", func(t *testing.T) {
		_, _, err := resolveDowntimeSelection(RestTypeShort, participant, DowntimeSelection{Move: DowntimeMoveWorkOnProject}, states, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "is not allowed during a short rest") {
			t.Fatalf("resolveDowntimeSelection disallowed move error = %v", err)
		}
	})

	t.Run("missing rng branches", func(t *testing.T) {
		cases := []DowntimeSelection{
			{Move: DowntimeMoveTendToWounds, TargetCharacterID: "char-1"},
			{Move: DowntimeMoveClearStress},
			{Move: DowntimeMoveRepairArmor, TargetCharacterID: "char-1"},
		}
		for _, selection := range cases {
			_, _, err := resolveDowntimeSelection(RestTypeShort, participant, selection, states, nil, nil)
			if err == nil || !strings.Contains(err.Error(), "requires rng") {
				t.Fatalf("resolveDowntimeSelection(%s) error = %v, want rng error", selection.Move, err)
			}
		}
	})

	t.Run("prepare without actor state fails", func(t *testing.T) {
		_, _, err := resolveDowntimeSelection(RestTypeShort, participant, DowntimeSelection{Move: DowntimeMovePrepare}, map[ids.CharacterID]*CharacterState{}, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "state is missing") {
			t.Fatalf("resolveDowntimeSelection prepare state error = %v", err)
		}
	})

	t.Run("clear all stress without actor state fails", func(t *testing.T) {
		_, _, err := resolveDowntimeSelection(RestTypeLong, participant, DowntimeSelection{Move: DowntimeMoveClearAllStress}, map[ids.CharacterID]*CharacterState{}, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "state is missing") {
			t.Fatalf("resolveDowntimeSelection clear_all_stress state error = %v", err)
		}
	})

	t.Run("work on project auto mode advances countdown", func(t *testing.T) {
		payload, update, err := resolveDowntimeSelection(
			RestTypeLong,
			participant,
			DowntimeSelection{Move: DowntimeMoveWorkOnProject, CountdownID: "cd-1"},
			states,
			nil,
			map[ids.CountdownID]Countdown{
				"cd-1": {ID: "cd-1", Name: "Track", Kind: CountdownKindProgress, Current: 1, Max: 4, Direction: CountdownDirectionIncrease},
			},
		)
		if err != nil {
			t.Fatalf("resolveDowntimeSelection auto project error = %v", err)
		}
		if payload.CountdownID != "cd-1" || update == nil || update.Delta != 1 || update.Reason != "work_on_project" {
			t.Fatalf("auto project payload/update = %+v / %+v", payload, update)
		}
	})
}

func TestNextCountdownMutationClampAndErrors(t *testing.T) {
	t.Parallel()

	if _, err := nextCountdownMutation(map[ids.CountdownID]Countdown{}, "missing", 1, nil, "tick"); err == nil || !strings.Contains(err.Error(), "is not available") {
		t.Fatalf("nextCountdownMutation missing countdown error = %v", err)
	}

	update, err := nextCountdownMutation(map[ids.CountdownID]Countdown{
		"cd-1": {ID: "cd-1", Name: "Track", Kind: CountdownKindProgress, Current: 1, Max: 4, Direction: CountdownDirectionIncrease},
	}, "cd-1", 1, nil, " tick ")
	if err != nil {
		t.Fatalf("nextCountdownMutation valid error = %v", err)
	}
	if update.After != 2 || update.Reason != "tick" {
		t.Fatalf("nextCountdownMutation update = %+v, want after=2 reason=tick", update)
	}
}

func TestClampGMFearAndRestTypeToPayloadString(t *testing.T) {
	t.Parallel()

	if got := clampGMFear(GMFearMin - 1); got != GMFearMin {
		t.Fatalf("clampGMFear(min-1) = %d, want %d", got, GMFearMin)
	}
	if got := clampGMFear(GMFearMax + 1); got != GMFearMax {
		t.Fatalf("clampGMFear(max+1) = %d, want %d", got, GMFearMax)
	}
	if got := clampGMFear(3); got != 3 {
		t.Fatalf("clampGMFear(3) = %d, want 3", got)
	}
	if got := restTypeToPayloadString(RestTypeLong); got != "long" {
		t.Fatalf("restTypeToPayloadString(long) = %q, want long", got)
	}
	if got := restTypeToPayloadString(RestTypeShort); got != "short" {
		t.Fatalf("restTypeToPayloadString(short) = %q, want short", got)
	}
}

func TestStateFactoryNewCharacterStateDefaultsAndNPCAdjustments(t *testing.T) {
	t.Parallel()

	factory := NewStateFactory()
	gotAny, err := factory.NewCharacterState("camp-1", "char-1", " ")
	if err != nil {
		t.Fatalf("NewCharacterState blank kind error = %v", err)
	}
	pc, ok := gotAny.(CharacterState)
	if !ok {
		t.Fatalf("NewCharacterState type = %T, want CharacterState", gotAny)
	}
	if pc.Kind != "pc" || pc.Hope != HopeDefault || pc.StressMax != StressMaxDefault {
		t.Fatalf("pc defaults = %+v", pc)
	}

	gotAny, err = factory.NewCharacterState("camp-1", "char-2", "NPC")
	if err != nil {
		t.Fatalf("NewCharacterState npc error = %v", err)
	}
	npc := gotAny.(CharacterState)
	if npc.Kind != "npc" || npc.Hope != 0 || npc.StressMax != 0 {
		t.Fatalf("npc defaults = %+v, want npc hope/stress override", npc)
	}
}
