package daggerheart

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestTierForLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level int
		want  int
	}{
		{level: 1, want: 1},
		{level: 2, want: 2},
		{level: 4, want: 2},
		{level: 5, want: 3},
		{level: 7, want: 3},
		{level: 8, want: 4},
		{level: 10, want: 4},
	}

	for _, tt := range tests {
		if got := TierForLevel(tt.level); got != tt.want {
			t.Fatalf("TierForLevel(%d) = %d, want %d", tt.level, got, tt.want)
		}
	}
}

func TestResolveRestPackage_InterruptedShortRestRejectsDowntimeSelections(t *testing.T) {
	t.Parallel()

	_, err := ResolveRestPackage(RestPackageInput{
		RestType:    RestTypeShort,
		Interrupted: true,
		Participants: []RestParticipantInput{{
			CharacterID: "char-1",
			Level:       1,
			State:       testRestState("char-1", 4, 5, 1, 3, 1, 3, 0, 2),
			Moves: []DowntimeSelection{{
				Move:     DowntimeMoveClearStress,
				RollSeed: int64Ptr(9),
			}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "interrupted short rests cannot include downtime moves") {
		t.Fatalf("ResolveRestPackage() error = %v, want interrupted short-rest selection error", err)
	}
}

func TestResolveRestPackage_ShortRestMovesResolveTargetsAndGrouping(t *testing.T) {
	t.Parallel()

	healSeed := int64(11)
	repairSeed := int64(17)
	clearSeed := int64(23)
	healAmount, err := rollDowntimeAmount(healSeed, 1)
	if err != nil {
		t.Fatalf("rollDowntimeAmount(heal) error = %v", err)
	}
	repairAmount, err := rollDowntimeAmount(repairSeed, 2)
	if err != nil {
		t.Fatalf("rollDowntimeAmount(repair) error = %v", err)
	}
	clearAmount, err := rollDowntimeAmount(clearSeed, 5)
	if err != nil {
		t.Fatalf("rollDowntimeAmount(clear) error = %v", err)
	}

	result, err := ResolveRestPackage(RestPackageInput{
		RestType:              RestTypeShort,
		RestSeed:              7,
		CurrentGMFear:         0,
		ConsecutiveShortRests: 0,
		Participants: []RestParticipantInput{
			{
				CharacterID: "char-1",
				Level:       1,
				State:       testRestState("char-1", 4, 6, 1, 3, 1, 3, 0, 2),
				Moves: []DowntimeSelection{
					{Move: DowntimeMoveTendToWounds, TargetCharacterID: "char-2", RollSeed: &healSeed},
					{Move: DowntimeMovePrepare, GroupID: "team"},
				},
			},
			{
				CharacterID: "char-2",
				Level:       2,
				State:       testRestState("char-2", 2, 6, 2, 4, 1, 3, 0, 2),
				Moves: []DowntimeSelection{
					{Move: DowntimeMoveRepairArmor, TargetCharacterID: "char-1", RollSeed: &repairSeed},
					{Move: DowntimeMovePrepare, GroupID: "team"},
				},
			},
			{
				CharacterID: "char-3",
				Level:       5,
				State:       testRestState("char-3", 6, 6, 1, 4, 4, 6, 0, 1),
				Moves: []DowntimeSelection{
					{Move: DowntimeMoveClearStress, RollSeed: &clearSeed},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveRestPackage returned error: %v", err)
	}
	if got, want := len(result.Payload.DowntimeMoves), 5; got != want {
		t.Fatalf("downtime move count = %d, want %d", got, want)
	}

	heal := result.Payload.DowntimeMoves[0]
	if heal.Move != DowntimeMoveTendToWounds || heal.TargetCharacterID != "char-2" || intValue(heal.HP) != minInt(2+healAmount, 6) {
		t.Fatalf("heal move = %+v, want healed char-2", heal)
	}

	prepare1 := result.Payload.DowntimeMoves[1]
	if prepare1.Move != DowntimeMovePrepare || prepare1.GroupID != "team" || intValue(prepare1.Hope) != 3 {
		t.Fatalf("first prepare move = %+v, want grouped prepare with hope 3", prepare1)
	}

	repair := result.Payload.DowntimeMoves[2]
	if repair.Move != DowntimeMoveRepairArmor || repair.TargetCharacterID != "char-1" || intValue(repair.Armor) != minInt(repairAmount, 2) {
		t.Fatalf("repair move = %+v, want repaired char-1 armor", repair)
	}

	prepare2 := result.Payload.DowntimeMoves[3]
	if prepare2.Move != DowntimeMovePrepare || prepare2.GroupID != "team" || intValue(prepare2.Hope) != 4 {
		t.Fatalf("second prepare move = %+v, want grouped prepare with capped hope 4", prepare2)
	}

	clearStress := result.Payload.DowntimeMoves[4]
	if clearStress.Move != DowntimeMoveClearStress || clearStress.TargetCharacterID != "char-3" || intValue(clearStress.Stress) != maxInt(4-clearAmount, 0) {
		t.Fatalf("clear stress move = %+v, want cleared stress on char-3", clearStress)
	}
}

func TestResolveRestPackage_LongRestMovesAndProjectAdvance(t *testing.T) {
	t.Parallel()

	result, err := ResolveRestPackage(RestPackageInput{
		RestType:              RestTypeLong,
		RestSeed:              7,
		CurrentGMFear:         0,
		ConsecutiveShortRests: 2,
		Participants: []RestParticipantInput{
			{
				CharacterID: "char-1",
				Level:       1,
				State:       testRestState("char-1", 4, 6, 1, 3, 2, 4, 0, 1),
				Moves: []DowntimeSelection{
					{Move: DowntimeMoveTendToAllWounds, TargetCharacterID: "char-2"},
					{
						Move:                DowntimeMoveWorkOnProject,
						CountdownID:         "cd-1",
						ProjectAdvanceMode:  ProjectAdvanceModeGMSetDelta,
						ProjectAdvanceDelta: 2,
						ProjectReason:       "breakthrough",
					},
				},
			},
			{
				CharacterID: "char-2",
				Level:       2,
				State:       testRestState("char-2", 2, 6, 1, 3, 3, 5, 0, 1),
				Moves: []DowntimeSelection{
					{Move: DowntimeMoveClearAllStress},
				},
			},
			{
				CharacterID: "char-3",
				Level:       3,
				State:       testRestState("char-3", 5, 6, 1, 3, 0, 3, 0, 3),
				Moves: []DowntimeSelection{
					{Move: DowntimeMoveRepairAllArmor},
				},
			},
		},
		AvailableCountdowns: map[ids.CountdownID]Countdown{
			"cd-1": {
				ID:        "cd-1",
				Name:      "Map the Hidden Pass",
				Kind:      CountdownKindProgress,
				Current:   0,
				Max:       4,
				Direction: CountdownDirectionIncrease,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveRestPackage returned error: %v", err)
	}
	if got, want := len(result.Payload.DowntimeMoves), 4; got != want {
		t.Fatalf("downtime move count = %d, want %d", got, want)
	}
	if got, want := len(result.Payload.CountdownUpdates), 1; got != want {
		t.Fatalf("countdown update count = %d, want %d", got, want)
	}

	tendAll := result.Payload.DowntimeMoves[0]
	if tendAll.Move != DowntimeMoveTendToAllWounds || tendAll.TargetCharacterID != "char-2" || intValue(tendAll.HP) != 6 {
		t.Fatalf("tend all wounds move = %+v, want char-2 hp restored to 6", tendAll)
	}

	project := result.Payload.DowntimeMoves[1]
	if project.Move != DowntimeMoveWorkOnProject || project.CountdownID != "cd-1" {
		t.Fatalf("project move = %+v, want countdown cd-1", project)
	}

	clearAll := result.Payload.DowntimeMoves[2]
	if clearAll.Move != DowntimeMoveClearAllStress || clearAll.TargetCharacterID != "char-2" || intValue(clearAll.Stress) != 0 {
		t.Fatalf("clear all stress move = %+v, want char-2 stress 0", clearAll)
	}

	repairAll := result.Payload.DowntimeMoves[3]
	if repairAll.Move != DowntimeMoveRepairAllArmor || repairAll.TargetCharacterID != "char-3" || intValue(repairAll.Armor) != 3 {
		t.Fatalf("repair all armor move = %+v, want char-3 armor 3", repairAll)
	}

	update := result.Payload.CountdownUpdates[0]
	if update.CountdownID != "cd-1" || update.Delta != 2 || update.After != 2 || update.Reason != "breakthrough" {
		t.Fatalf("countdown update = %+v, want gm_set_delta applied", update)
	}
}

func TestResolveRestPackage_WorkOnProjectValidation(t *testing.T) {
	t.Parallel()

	countdowns := map[ids.CountdownID]Countdown{
		"cd-1": {
			ID:        "cd-1",
			Name:      "Map the Hidden Pass",
			Kind:      CountdownKindProgress,
			Current:   0,
			Max:       4,
			Direction: CountdownDirectionIncrease,
		},
	}

	tests := []struct {
		name      string
		selection DowntimeSelection
		wantErr   string
	}{
		{
			name:      "missing countdown",
			selection: DowntimeSelection{Move: DowntimeMoveWorkOnProject},
			wantErr:   "requires countdown_id",
		},
		{
			name: "missing delta",
			selection: DowntimeSelection{
				Move:               DowntimeMoveWorkOnProject,
				CountdownID:        "cd-1",
				ProjectAdvanceMode: ProjectAdvanceModeGMSetDelta,
				ProjectReason:      "breakthrough",
			},
			wantErr: "requires non-zero advance_delta",
		},
		{
			name: "missing reason",
			selection: DowntimeSelection{
				Move:                DowntimeMoveWorkOnProject,
				CountdownID:         "cd-1",
				ProjectAdvanceMode:  ProjectAdvanceModeGMSetDelta,
				ProjectAdvanceDelta: 2,
			},
			wantErr: "requires reason",
		},
		{
			name: "invalid mode",
			selection: DowntimeSelection{
				Move:               DowntimeMoveWorkOnProject,
				CountdownID:        "cd-1",
				ProjectAdvanceMode: "mystery",
			},
			wantErr: "advance mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveRestPackage(RestPackageInput{
				RestType: RestTypeLong,
				Participants: []RestParticipantInput{{
					CharacterID: "char-1",
					Level:       1,
					State:       testRestState("char-1", 5, 6, 1, 3, 0, 3, 0, 2),
					Moves:       []DowntimeSelection{tt.selection},
				}},
				AvailableCountdowns: countdowns,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ResolveRestPackage() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestResolveRestTarget_DefaultsToActorAndRejectsMissingTarget(t *testing.T) {
	t.Parallel()

	states := map[ids.CharacterID]*CharacterState{
		"char-1": ptrState(testRestState("char-1", 5, 6, 1, 3, 0, 3, 0, 2)),
	}

	targetID, _, err := resolveRestTarget("char-1", "", states)
	if err != nil {
		t.Fatalf("resolveRestTarget default target error = %v", err)
	}
	if targetID != "char-1" {
		t.Fatalf("default target id = %q, want char-1", targetID)
	}

	if _, _, err := resolveRestTarget("char-1", "char-2", states); err == nil || !strings.Contains(err.Error(), "not participating") {
		t.Fatalf("resolveRestTarget missing target error = %v, want participation error", err)
	}
}

func TestRestHelpers(t *testing.T) {
	t.Parallel()

	if got := hasAnyDowntimeSelections([]RestParticipantInput{{CharacterID: "char-1"}, {CharacterID: "char-2", Moves: []DowntimeSelection{{Move: DowntimeMovePrepare}}}}); !got {
		t.Fatal("hasAnyDowntimeSelections() = false, want true")
	}
	if got := hasAnyDowntimeSelections([]RestParticipantInput{{CharacterID: "char-1"}}); got {
		t.Fatal("hasAnyDowntimeSelections() = true, want false")
	}

	if got := restTypeAllowsMove(RestTypeShort, DowntimeMovePrepare); !got {
		t.Fatal("restTypeAllowsMove(short, prepare) = false, want true")
	}
	if got := restTypeAllowsMove(RestTypeShort, DowntimeMoveWorkOnProject); got {
		t.Fatal("restTypeAllowsMove(short, work_on_project) = true, want false")
	}
	if got := restTypeToPayloadString(RestTypeLong); got != "long" {
		t.Fatalf("restTypeToPayloadString(long) = %q, want long", got)
	}
}

func TestRollDowntimeAmountAndClampGMFear(t *testing.T) {
	t.Parallel()

	got, err := rollDowntimeAmount(99, 5)
	if err != nil {
		t.Fatalf("rollDowntimeAmount() error = %v", err)
	}
	if got < 4 || got > 7 {
		t.Fatalf("rollDowntimeAmount(99, 5) = %d, want 1d4 + tier(5)=3", got)
	}

	if got := clampGMFear(-2); got != GMFearMin {
		t.Fatalf("clampGMFear(-2) = %d, want %d", got, GMFearMin)
	}
	if got := clampGMFear(5); got != 5 {
		t.Fatalf("clampGMFear(5) = %d, want 5", got)
	}
	if got := clampGMFear(GMFearMax + 10); got != GMFearMax {
		t.Fatalf("clampGMFear(max+10) = %d, want %d", got, GMFearMax)
	}
}

func testRestState(characterID ids.CharacterID, hp, hpMax, hope, hopeMax, stress, stressMax, armor, armorMax int) CharacterState {
	return CharacterState{
		CharacterID: characterID.String(),
		HP:          hp,
		HPMax:       hpMax,
		Hope:        hope,
		HopeMax:     hopeMax,
		Stress:      stress,
		StressMax:   stressMax,
		Armor:       armor,
		ArmorMax:    armorMax,
		LifeState:   LifeStateAlive,
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func intValue(v *int) int {
	if v == nil {
		return -1
	}
	return *v
}

func ptrState(v CharacterState) *CharacterState {
	return &v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
