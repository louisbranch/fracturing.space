package damagetransport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireAdversaryDamageDependenciesCoversAllMissingBranches(t *testing.T) {
	emptyContent := testContentStore{
		adversaryEntries: make(map[string]contentstore.DaggerheartAdversaryEntry),
		armors:           make(map[string]contentstore.DaggerheartArmor),
	}

	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing gate", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}}},
		{name: "missing content", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}}},
		{name: "missing event", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Content: emptyContent}},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Content: emptyContent, Event: testEventStore{}}},
		{name: "missing loader", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Content: emptyContent, Event: testEventStore{}, ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil }}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireAdversaryDamageDependencies(); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}

func TestDamageStateHelpersCoverNormalizationAndBeastformAutoDrop(t *testing.T) {
	active := activeBeastformFromProjection(&projectionstore.DaggerheartActiveBeastformState{
		BeastformID:     "bear",
		BaseTrait:       "agility",
		AttackTrait:     "strength",
		DamageDice:      []projectionstore.DaggerheartDamageDie{{Count: 1, Sides: 8}},
		DamageType:      "physical",
		DropOnAnyHPMark: true,
	})
	if active == nil || active.BeastformID != "bear" || len(active.DamageDice) != 1 {
		t.Fatalf("activeBeastformFromProjection = %#v", active)
	}
	if activeBeastformFromProjection(nil) != nil {
		t.Fatal("expected nil active beastform for nil projection")
	}

	ptr := classStatePtr(daggerheartstate.CharacterClassState{
		AttackBonusUntilRest:       -1,
		DifficultyPenaltyUntilRest: 2,
		FocusTargetID:              " foe ",
		ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{
			BeastformID: " bear ",
			TraitBonus:  -1,
			DamageDice: []daggerheartstate.CharacterDamageDie{
				{Count: 0, Sides: 6},
				{Count: 1, Sides: 8},
			},
		},
		RallyDice: []int{0, 6},
	})
	if ptr.AttackBonusUntilRest != 0 || ptr.DifficultyPenaltyUntilRest != 0 || ptr.FocusTargetID != "foe" {
		t.Fatalf("classStatePtr normalized state = %#v", ptr)
	}
	if ptr.ActiveBeastform == nil || ptr.ActiveBeastform.TraitBonus != 0 || len(ptr.ActiveBeastform.DamageDice) != 1 {
		t.Fatalf("classStatePtr beastform = %#v", ptr.ActiveBeastform)
	}

	var commands []SystemCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})
	prev := projectionstore.DaggerheartCharacterState{
		ClassState: projectionstore.DaggerheartClassState{
			ActiveBeastform: &projectionstore.DaggerheartActiveBeastformState{
				BeastformID:     "bear",
				DropOnAnyHPMark: true,
			},
		},
		Hp: 3,
	}

	if err := handler.autoDropBeastform(context.Background(), "camp-1", "sess-1", "scene-1", "char-1", prev, projectionstore.DaggerheartCharacterState{Hp: 0}); err != nil {
		t.Fatalf("autoDropBeastform(hp_zero) returned error: %v", err)
	}
	if err := handler.autoDropBeastform(context.Background(), "camp-1", "sess-1", "scene-1", "char-1", prev, projectionstore.DaggerheartCharacterState{Hp: 2}); err != nil {
		t.Fatalf("autoDropBeastform(fragile) returned error: %v", err)
	}
	if err := handler.autoDropBeastform(context.Background(), "camp-1", "sess-1", "scene-1", "char-1", projectionstore.DaggerheartCharacterState{}, projectionstore.DaggerheartCharacterState{Hp: 5}); err != nil {
		t.Fatalf("autoDropBeastform(no_beastform) returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("command count = %d, want 2", len(commands))
	}

	var hpZeroPayload daggerheartpayload.BeastformDropPayload
	if err := json.Unmarshal(commands[0].PayloadJSON, &hpZeroPayload); err != nil {
		t.Fatalf("decode hp_zero payload: %v", err)
	}
	if commands[0].CommandType != commandids.DaggerheartBeastformDrop || hpZeroPayload.Source != "beastform.auto_drop.hp_zero" {
		t.Fatalf("hp_zero command = %+v payload=%+v", commands[0], hpZeroPayload)
	}

	var fragilePayload daggerheartpayload.BeastformDropPayload
	if err := json.Unmarshal(commands[1].PayloadJSON, &fragilePayload); err != nil {
		t.Fatalf("decode fragile payload: %v", err)
	}
	if fragilePayload.Source != "beastform.auto_drop.fragile" {
		t.Fatalf("fragile source = %q, want beastform.auto_drop.fragile", fragilePayload.Source)
	}
	if fragilePayload.ClassStateAfter == nil || fragilePayload.ClassStateAfter.ActiveBeastform != nil {
		t.Fatalf("fragile class state after = %#v, want dropped beastform", fragilePayload.ClassStateAfter)
	}
}

func TestApplyMinionSpilloverCoversDeleteAndErrorBranches(t *testing.T) {
	t.Run("deletes same-scene minions only", func(t *testing.T) {
		var deleted []string
		handler := newTestHandler(Dependencies{
			Daggerheart: testDaggerheartStore{
				listAdversaries: []projectionstore.DaggerheartAdversary{
					{AdversaryID: "adv-2", AdversaryEntryID: "entry-minion", SceneID: "scene-1"},
					{AdversaryID: "adv-3", AdversaryEntryID: "entry-bruiser", SceneID: "scene-1"},
					{AdversaryID: "adv-4", AdversaryEntryID: "entry-minion", SceneID: "scene-1"},
					{AdversaryID: "adv-5", AdversaryEntryID: "entry-minion", SceneID: "scene-2"},
				},
			},
			Content: testContentStore{
				adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
					"entry-primary": {ID: "entry-primary", MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3}},
					"entry-minion":  {ID: "entry-minion", MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3}},
					"entry-bruiser": {ID: "entry-bruiser"},
				},
			},
			ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
				if in.CommandType == commandids.DaggerheartAdversaryDelete {
					deleted = append(deleted, in.EntityID)
				}
				return nil
			},
		})
		primary := projectionstore.DaggerheartAdversary{
			AdversaryID:      "adv-1",
			AdversaryEntryID: "entry-primary",
			SceneID:          "scene-1",
		}

		if err := handler.applyMinionSpillover(context.Background(), "camp-1", "sess-1", "scene-1", primary, 7); err != nil {
			t.Fatalf("applyMinionSpillover returned error: %v", err)
		}
		if len(deleted) != 2 || deleted[0] != "adv-2" || deleted[1] != "adv-4" {
			t.Fatalf("deleted adversaries = %v, want adv-2 and adv-4", deleted)
		}
	})

	t.Run("list error", func(t *testing.T) {
		handler := newTestHandler(Dependencies{
			Daggerheart: testDaggerheartStore{listErr: errors.New("boom")},
			Content: testContentStore{
				adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
					"entry-primary": {ID: "entry-primary", MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3}},
				},
			},
			ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
		})
		err := handler.applyMinionSpillover(context.Background(), "camp-1", "sess-1", "scene-1", projectionstore.DaggerheartAdversary{
			AdversaryID:      "adv-1",
			AdversaryEntryID: "entry-primary",
			SceneID:          "scene-1",
		}, 6)
		if status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
		}
	})

	t.Run("candidate entry missing", func(t *testing.T) {
		handler := newTestHandler(Dependencies{
			Daggerheart: testDaggerheartStore{
				listAdversaries: []projectionstore.DaggerheartAdversary{
					{AdversaryID: "adv-2", AdversaryEntryID: "entry-missing", SceneID: "scene-1"},
				},
			},
			Content: testContentStore{
				adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
					"entry-primary": {ID: "entry-primary", MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3}},
				},
			},
			ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
		})
		err := handler.applyMinionSpillover(context.Background(), "camp-1", "sess-1", "scene-1", projectionstore.DaggerheartAdversary{
			AdversaryID:      "adv-1",
			AdversaryEntryID: "entry-primary",
			SceneID:          "scene-1",
		}, 6)
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}
	})
}
