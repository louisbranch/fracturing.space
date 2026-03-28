package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerDependencyGuardsReportMissingRuntimeHandlers(t *testing.T) {
	t.Parallel()

	baseAttack := Dependencies{
		SessionActionRoll: func(context.Context, *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			return nil, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return nil, nil
		},
		ApplyRollOutcome: func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			return nil, nil
		},
		ApplyAttackOutcome: func(context.Context, *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
			return nil, nil
		},
		ApplyDamage: func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return nil, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		LoadArmor: func(context.Context, string) (contentstore.DaggerheartArmor, error) {
			return contentstore.DaggerheartArmor{}, nil
		},
	}

	tests := []struct {
		name    string
		mutate  func(*Dependencies)
		wantMsg string
	}{
		{
			name: "attack flow missing subclass loader",
			mutate: func(deps *Dependencies) {
				deps.LoadSubclass = nil
			},
			wantMsg: "subclass loader is not configured",
		},
		{
			name: "attack flow missing armor loader",
			mutate: func(deps *Dependencies) {
				deps.LoadArmor = nil
			},
			wantMsg: "armor loader is not configured",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := baseAttack
			tc.mutate(&deps)
			err := NewHandler(deps).requireAttackFlowDeps()
			if status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
			if got := status.Convert(err).Message(); got != tc.wantMsg {
				t.Fatalf("message = %q, want %q", got, tc.wantMsg)
			}
		})
	}

	baseAdversary := baseAttack
	baseAdversary.SessionAdversaryAttackRoll = func(context.Context, *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
		return nil, nil
	}
	baseAdversary.ApplyAdversaryAttackOutcome = func(context.Context, *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
		return nil, nil
	}
	baseAdversary.LoadAdversary = func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
		return projectionstore.DaggerheartAdversary{}, nil
	}
	baseAdversary.LoadAdversaryEntry = func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
		return contentstore.DaggerheartAdversaryEntry{}, nil
	}

	t.Run("adversary flow missing adversary entry loader", func(t *testing.T) {
		t.Parallel()
		deps := baseAdversary
		deps.LoadAdversaryEntry = nil
		err := NewHandler(deps).requireAdversaryAttackFlowDeps()
		if status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
		}
		if got := status.Convert(err).Message(); got != "adversary entry loader is not configured" {
			t.Fatalf("message = %q", got)
		}
	})
}

func TestAdversaryFeatureStateHelpersCoverFocusedAndStatusCases(t *testing.T) {
	t.Parallel()

	adversary := projectionstore.DaggerheartAdversary{
		FeatureStates: []projectionstore.DaggerheartAdversaryFeatureState{
			{FeatureID: " feature-1 ", Status: " active ", FocusedTargetID: " target-1 "},
			{FeatureID: "feature-2", Status: "ready"},
		},
	}
	if !hasActiveAdversaryFeatureState(adversary, "feature-1") {
		t.Fatal("hasActiveAdversaryFeatureState() = false, want true")
	}
	if hasActiveAdversaryFeatureState(adversary, "feature-2") {
		t.Fatal("hasActiveAdversaryFeatureState(ready) = true")
	}
	if got := focusedTargetIDForFeature(adversary, "feature-1"); got != "target-1" {
		t.Fatalf("focusedTargetIDForFeature() = %q, want %q", got, "target-1")
	}
	if !hasReadyAdversaryFeatureState(adversary, "feature-2") {
		t.Fatal("hasReadyAdversaryFeatureState() = false, want true")
	}

	cleared := clearAdversaryFeatureState(adversary.FeatureStates, "feature-1")
	if len(cleared) != 1 || cleared[0].FeatureID != "feature-2" {
		t.Fatalf("clearAdversaryFeatureState() = %#v", cleared)
	}

	updated := setAdversaryFeatureStateStatus(adversary.FeatureStates, "feature-2", " spent ")
	if got := updated[1].Status; got != "spent" {
		t.Fatalf("updated status = %q, want %q", got, "spent")
	}
	if got := updated[0].Status; got != " active " {
		t.Fatalf("unrelated status = %q, want unchanged", got)
	}
}

func TestProjectionAndFeatureHelpersNormalizeRuntimeValues(t *testing.T) {
	t.Parallel()

	entry := contentstore.DaggerheartAdversaryEntry{
		Features: []contentstore.DaggerheartAdversaryFeature{
			{ID: "feature-1", Name: "Unknown"},
			{ID: "feature-2", Name: "Box In"},
		},
	}
	if _, ok := findAdversaryEntryFeature(entry, "feature-2"); !ok {
		t.Fatal("findAdversaryEntryFeature() did not find feature")
	}
	featureID, rule, ok := firstAdversaryFeatureRuleByKind(entry, rules.AdversaryFeatureRuleKindFocusTargetDisadvantage)
	if !ok || featureID != "feature-2" || rule == nil || rule.Kind != rules.AdversaryFeatureRuleKindFocusTargetDisadvantage {
		t.Fatalf("firstAdversaryFeatureRuleByKind() = (%q, %#v, %v)", featureID, rule, ok)
	}

	conditions := sessionCharacterConditions(projectionstore.DaggerheartCharacterState{
		Conditions: []projectionstore.DaggerheartConditionState{
			{Code: "hidden"},
			{Standard: "vulnerable"},
			{},
		},
	})
	if len(conditions) != 2 || !hasCondition(conditions, " hidden ") || !hasCondition(conditions, "vulnerable") {
		t.Fatalf("sessionCharacterConditions() = %#v", conditions)
	}

	ptr := classStatePtr(daggerheartstate.CharacterClassState{
		FocusTargetID:              " target-1 ",
		AttackBonusUntilRest:       -1,
		DifficultyPenaltyUntilRest: 2,
		ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{
			BeastformID: "wolf",
			AttackRange: " melee ",
			DamageDice: []daggerheartstate.CharacterDamageDie{
				{Count: 0, Sides: 0},
				{Count: 1, Sides: 8},
			},
		},
		RallyDice: []int{0, 6},
	})
	if ptr.FocusTargetID != "target-1" || ptr.AttackBonusUntilRest != 0 || ptr.DifficultyPenaltyUntilRest != 0 {
		t.Fatalf("classStatePtr() = %#v", ptr)
	}
	if ptr.ActiveBeastform == nil || ptr.ActiveBeastform.AttackRange != "melee" || len(ptr.ActiveBeastform.DamageDice) != 1 {
		t.Fatalf("classStatePtr().ActiveBeastform = %#v", ptr.ActiveBeastform)
	}

	beastform := activeBeastformFromProjection(&projectionstore.DaggerheartActiveBeastformState{
		BeastformID:     "wolf",
		AttackTrait:     "agility",
		AttackRange:     "far",
		DamageDice:      []projectionstore.DaggerheartDamageDie{{Count: 2, Sides: 10}},
		DropOnAnyHPMark: true,
	})
	if beastform == nil || beastform.AttackRange != "far" || len(beastform.DamageDice) != 1 || beastform.DamageDice[0].Sides != 10 || !beastform.DropOnAnyHPMark {
		t.Fatalf("activeBeastformFromProjection() = %#v", beastform)
	}
	if classStateFromProjection(projectionstore.DaggerheartClassState{FocusTargetID: " target-2 "}).FocusTargetID != "target-2" {
		t.Fatal("classStateFromProjection() did not normalize focus target")
	}
}

func TestResolveAttackProfileCoversBeastformAndValidationBranches(t *testing.T) {
	t.Parallel()

	_, _, _, _, _, err := resolveAttackProfile(&pb.SessionAttackFlowRequest{
		AttackProfile: &pb.SessionAttackFlowRequest_BeastformAttack{
			BeastformAttack: &pb.SessionBeastformAttackProfile{},
		},
	}, daggerheartstate.CharacterClassState{})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	trait, dice, bonus, attackRange, critical, err := resolveAttackProfile(&pb.SessionAttackFlowRequest{
		AttackProfile: &pb.SessionAttackFlowRequest_BeastformAttack{
			BeastformAttack: &pb.SessionBeastformAttackProfile{},
		},
	}, daggerheartstate.CharacterClassState{
		ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{
			AttackTrait: "instinct",
			AttackRange: "very far",
			DamageDice:  []daggerheartstate.CharacterDamageDie{{Count: 2, Sides: 8}},
			DamageBonus: 3,
		},
	})
	if err != nil {
		t.Fatalf("resolveAttackProfile(beastform) error = %v", err)
	}
	if trait != "instinct" || len(dice) != 1 || dice[0].GetSides() != 8 || bonus != 3 || attackRange != pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED || critical {
		t.Fatalf("resolveAttackProfile(beastform) = (%q, %+v, %d, %v, %v)", trait, dice, bonus, attackRange, critical)
	}

	_, _, _, _, _, err = resolveAttackProfile(&pb.SessionAttackFlowRequest{
		AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
			StandardAttack: &pb.SessionStandardAttackProfile{},
		},
	}, daggerheartstate.CharacterClassState{
		ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{AttackRange: "melee"},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	tests := []struct {
		name string
		req  *pb.SessionAttackFlowRequest
		want codes.Code
	}{
		{
			name: "missing standard profile",
			req:  &pb.SessionAttackFlowRequest{},
			want: codes.InvalidArgument,
		},
		{
			name: "missing trait",
			req: &pb.SessionAttackFlowRequest{
				AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
					StandardAttack: &pb.SessionStandardAttackProfile{
						AttackRange: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE,
						DamageDice:  []*pb.DiceSpec{{Sides: 6, Count: 1}},
					},
				},
			},
			want: codes.InvalidArgument,
		},
		{
			name: "missing attack range",
			req: &pb.SessionAttackFlowRequest{
				AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
					StandardAttack: &pb.SessionStandardAttackProfile{
						Trait:      "agility",
						DamageDice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
					},
				},
			},
			want: codes.InvalidArgument,
		},
		{
			name: "missing damage dice",
			req: &pb.SessionAttackFlowRequest{
				AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
					StandardAttack: &pb.SessionStandardAttackProfile{
						Trait:       "agility",
						AttackRange: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE,
					},
				},
			},
			want: codes.InvalidArgument,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, _, _, err := resolveAttackProfile(tc.req, daggerheartstate.CharacterClassState{})
			if status.Code(err) != tc.want {
				t.Fatalf("status code = %v, want %v", status.Code(err), tc.want)
			}
		})
	}

	if _, err := beastformAttackRangeToProto("underground"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	if !isMeleeAttackRange(" melee ") {
		t.Fatal("isMeleeAttackRange() = false, want true")
	}
	if isMeleeAttackRange("far") {
		t.Fatal("isMeleeAttackRange(far) = true")
	}
}
