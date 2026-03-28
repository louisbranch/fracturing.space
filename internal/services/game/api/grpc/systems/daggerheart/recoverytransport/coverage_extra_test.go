package recoverytransport

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryHelperLocalCopiesCoverGateAndSystemBranches(t *testing.T) {
	t.Parallel()

	if !campaignSupportsDaggerheart(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}) {
		t.Fatal("campaignSupportsDaggerheart(daggerheart) = false")
	}
	if campaignSupportsDaggerheart(storage.CampaignRecord{System: systembridge.SystemID("other")}) {
		t.Fatal("campaignSupportsDaggerheart(other) = true")
	}

	if err := requireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}, "unsupported"); err != nil {
		t.Fatalf("requireDaggerheartSystem() error = %v", err)
	}
	if err := requireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	if err := ensureNoOpenSessionGate(context.Background(), nil, "camp-1", "sess-1"); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
	if err := ensureNoOpenSessionGate(context.Background(), testGateStore{err: storage.ErrNotFound}, "", ""); err != nil {
		t.Fatalf("ensureNoOpenSessionGate(blank ids) error = %v", err)
	}
	if err := ensureNoOpenSessionGate(context.Background(), testGateStore{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	if err := ensureNoOpenSessionGate(context.Background(), testGateStore{err: errors.New("boom")}, "camp-1", "sess-1"); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestRecoveryDependencyGuardsReportMissingSeedAndCallbacks(t *testing.T) {
	t.Parallel()

	base := Dependencies{
		Campaign:                    testCampaignStore{record: storage.CampaignRecord{ID: "camp-1", System: systembridge.SystemIDDaggerheart, Status: campaign.StatusActive}},
		SessionGate:                 testGateStore{err: storage.ErrNotFound},
		Daggerheart:                 &testDaggerheartStore{},
		ExecuteSystemCommand:        func(context.Context, SystemCommandInput) error { return nil },
		ApplyStressConditionChange:  func(context.Context, StressConditionInput) error { return nil },
		AppendCharacterDeletedEvent: func(context.Context, CharacterDeleteInput) error { return nil },
		SeedGenerator:               func() (int64, error) { return 7, nil },
	}

	tests := []struct {
		name        string
		requireSeed bool
		mutate      func(*Dependencies)
		wantMessage string
	}{
		{
			name: "missing campaign store",
			mutate: func(deps *Dependencies) {
				deps.Campaign = nil
			},
			wantMessage: "campaign store is not configured",
		},
		{
			name: "missing session gate store",
			mutate: func(deps *Dependencies) {
				deps.SessionGate = nil
			},
			wantMessage: "session gate store is not configured",
		},
		{
			name: "missing daggerheart store",
			mutate: func(deps *Dependencies) {
				deps.Daggerheart = nil
			},
			wantMessage: "daggerheart store is not configured",
		},
		{
			name: "missing system command executor",
			mutate: func(deps *Dependencies) {
				deps.ExecuteSystemCommand = nil
			},
			wantMessage: "system command executor is not configured",
		},
		{
			name: "missing stress condition callback",
			mutate: func(deps *Dependencies) {
				deps.ApplyStressConditionChange = nil
			},
			wantMessage: "stress condition callback is not configured",
		},
		{
			name: "missing character deleted callback",
			mutate: func(deps *Dependencies) {
				deps.AppendCharacterDeletedEvent = nil
			},
			wantMessage: "character deleted callback is not configured",
		},
		{
			name:        "missing seed generator when required",
			requireSeed: true,
			mutate: func(deps *Dependencies) {
				deps.SeedGenerator = nil
			},
			wantMessage: "seed generator is not configured",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := base
			tc.mutate(&deps)
			err := NewHandler(deps).requireDependencies(tc.requireSeed)
			if status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
			if got := status.Convert(err).Message(); got != tc.wantMessage {
				t.Fatalf("message = %q, want %q", got, tc.wantMessage)
			}
		})
	}
}

func TestRecoveryMappingHelpersCoverDefaultsAndErrorBranches(t *testing.T) {
	t.Parallel()

	if err := handleDomainError(context.Background(), errors.New("boom")); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}

	countdown := countdownFromStorage(projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "count-1",
		Name:              "Clock",
		StartingValue:     6,
		RemainingValue:    3,
		StartingRollMin:   1,
		StartingRollMax:   6,
		StartingRollValue: 4,
	})
	if countdown.AdvancementPolicy != rules.CountdownAdvancementPolicyManual ||
		countdown.LoopBehavior != rules.CountdownLoopBehaviorNone ||
		countdown.Status != rules.CountdownStatusActive ||
		countdown.StartingRoll == nil ||
		countdown.StartingRoll.Value != 4 {
		t.Fatalf("countdownFromStorage() = %+v", countdown)
	}

	if got, err := restTypeFromProto(pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT); err != nil || got != daggerheart.RestTypeShort {
		t.Fatalf("restTypeFromProto(short) = %v, %v", got, err)
	}
	if _, err := restTypeFromProto(pb.DaggerheartRestType(99)); err == nil {
		t.Fatal("expected invalid rest type error")
	}

	if got := projectAdvanceModeFromProto(pb.DaggerheartProjectAdvanceMode_DAGGERHEART_PROJECT_ADVANCE_MODE_GM_SET_DELTA); got != daggerheart.ProjectAdvanceModeGMSetDelta {
		t.Fatalf("projectAdvanceModeFromProto(gm_set_delta) = %q", got)
	}
	if got := projectAdvanceModeFromProto(pb.DaggerheartProjectAdvanceMode_DAGGERHEART_PROJECT_ADVANCE_MODE_UNSPECIFIED); got != daggerheart.ProjectAdvanceModeAuto {
		t.Fatalf("projectAdvanceModeFromProto(default) = %q", got)
	}

	if got, err := deathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL); err != nil || got != daggerheart.DeathMoveRiskItAll {
		t.Fatalf("deathMoveFromProto(risk_it_all) = %q, %v", got, err)
	}
	if _, err := deathMoveFromProto(pb.DaggerheartDeathMove(99)); err == nil {
		t.Fatal("expected invalid death move error")
	}
	if got := DeathMoveToProto("other"); got != pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED {
		t.Fatalf("DeathMoveToProto(other) = %v", got)
	}
}

func TestRecoveryDowntimeSelectionCoversSeededAndProjectMoves(t *testing.T) {
	t.Parallel()

	resolve := func(rng *commonv1.RngRequest, seedFunc func() (int64, error), replayMode func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
		if !replayMode(commonv1.RollMode_REPLAY) {
			t.Fatal("expected replay mode to be allowed")
		}
		if rng == nil {
			t.Fatal("expected rng request")
		}
		return 11, "generated", commonv1.RollMode_LIVE, nil
	}
	seedFunc := func() (int64, error) { return 11, nil }

	got, err := downtimeSelectionFromProto(&pb.DaggerheartDowntimeSelection{
		Move: &pb.DaggerheartDowntimeSelection_TendToWounds{
			TendToWounds: &pb.DaggerheartTendToWoundsMove{
				TargetCharacterId: " char-1 ",
				Rng:               &commonv1.RngRequest{},
			},
		},
	}, resolve, seedFunc)
	if err != nil {
		t.Fatalf("downtimeSelectionFromProto(tend_to_wounds) error = %v", err)
	}
	if got.Move != daggerheart.DowntimeMoveTendToWounds || string(got.TargetCharacterID) != "char-1" || got.RollSeed == nil || *got.RollSeed != 11 {
		t.Fatalf("selection = %+v", got)
	}

	got, err = downtimeSelectionFromProto(&pb.DaggerheartDowntimeSelection{
		Move: &pb.DaggerheartDowntimeSelection_WorkOnProject{
			WorkOnProject: &pb.DaggerheartWorkOnProjectMove{
				ProjectCampaignCountdownId: " count-1 ",
				AdvanceMode:                pb.DaggerheartProjectAdvanceMode_DAGGERHEART_PROJECT_ADVANCE_MODE_GM_SET_DELTA,
				AdvanceDelta:               2,
				Reason:                     " push forward ",
			},
		},
	}, resolve, seedFunc)
	if err != nil {
		t.Fatalf("downtimeSelectionFromProto(work_on_project) error = %v", err)
	}
	if got.Move != daggerheart.DowntimeMoveWorkOnProject ||
		string(got.CountdownID) != "count-1" ||
		got.ProjectAdvanceMode != daggerheart.ProjectAdvanceModeGMSetDelta ||
		got.ProjectAdvanceDelta != 2 ||
		got.ProjectReason != "push forward" {
		t.Fatalf("project selection = %+v", got)
	}

	_, err = downtimeSelectionFromProto(&pb.DaggerheartDowntimeSelection{
		Move: &pb.DaggerheartDowntimeSelection_ClearStress{
			ClearStress: &pb.DaggerheartClearStressMove{Rng: &commonv1.RngRequest{}},
		},
	}, func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
		return 0, "", commonv1.RollMode_LIVE, errors.New("seed failure")
	}, seedFunc)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
	if grpcerror.HandleDomainError(err) == nil {
		t.Fatal("expected grpcerror-compatible internal error")
	}
}

func TestRecoveryDowntimeSelectionCoversRemainingMovesAndValidation(t *testing.T) {
	t.Parallel()

	resolve := func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
		return 23, "generated", commonv1.RollMode_LIVE, nil
	}
	seedFunc := func() (int64, error) { return 23, nil }

	tests := []struct {
		name    string
		input   *pb.DaggerheartDowntimeSelection
		want    string
		wantID  string
		wantErr bool
	}{
		{
			name:    "nil selection",
			input:   nil,
			wantErr: true,
		},
		{
			name: "clear stress",
			input: &pb.DaggerheartDowntimeSelection{
				Move: &pb.DaggerheartDowntimeSelection_ClearStress{
					ClearStress: &pb.DaggerheartClearStressMove{Rng: &commonv1.RngRequest{}},
				},
			},
			want: daggerheart.DowntimeMoveClearStress,
		},
		{
			name: "repair armor",
			input: &pb.DaggerheartDowntimeSelection{
				Move: &pb.DaggerheartDowntimeSelection_RepairArmor{
					RepairArmor: &pb.DaggerheartRepairArmorMove{
						TargetCharacterId: " char-7 ",
						Rng:               &commonv1.RngRequest{},
					},
				},
			},
			want:   daggerheart.DowntimeMoveRepairArmor,
			wantID: "char-7",
		},
		{
			name: "tend to all wounds",
			input: &pb.DaggerheartDowntimeSelection{
				Move: &pb.DaggerheartDowntimeSelection_TendToAllWounds{
					TendToAllWounds: &pb.DaggerheartTendToAllWoundsMove{TargetCharacterId: " char-8 "},
				},
			},
			want:   daggerheart.DowntimeMoveTendToAllWounds,
			wantID: "char-8",
		},
		{
			name: "clear all stress",
			input: &pb.DaggerheartDowntimeSelection{
				Move: &pb.DaggerheartDowntimeSelection_ClearAllStress{
					ClearAllStress: &pb.DaggerheartClearAllStressMove{},
				},
			},
			want: daggerheart.DowntimeMoveClearAllStress,
		},
		{
			name: "repair all armor",
			input: &pb.DaggerheartDowntimeSelection{
				Move: &pb.DaggerheartDowntimeSelection_RepairAllArmor{
					RepairAllArmor: &pb.DaggerheartRepairAllArmorMove{TargetCharacterId: " char-9 "},
				},
			},
			want:   daggerheart.DowntimeMoveRepairAllArmor,
			wantID: "char-9",
		},
		{
			name:    "missing move",
			input:   &pb.DaggerheartDowntimeSelection{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := downtimeSelectionFromProto(tc.input, resolve, seedFunc)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("downtimeSelectionFromProto() error = %v", err)
			}
			if got.Move != tc.want {
				t.Fatalf("move = %q, want %q", got.Move, tc.want)
			}
			if tc.wantID != "" && string(got.TargetCharacterID) != tc.wantID && string(got.CountdownID) != tc.wantID {
				t.Fatalf("selection = %+v, want trimmed id %q", got, tc.wantID)
			}
			if tc.want == daggerheart.DowntimeMoveClearStress || tc.want == daggerheart.DowntimeMoveRepairArmor {
				if got.RollSeed == nil || *got.RollSeed != 23 {
					t.Fatalf("roll seed = %v, want 23", got.RollSeed)
				}
			}
		})
	}
}

func TestRecoveryHandlersValidateRequestsAndPreconditions(t *testing.T) {
	t.Parallel()

	t.Run("resolve blaze of glory guards", func(t *testing.T) {
		handler := newTestHandler(Dependencies{})
		if _, err := handler.ResolveBlazeOfGlory(testContext(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}

		handler = newTestHandler(Dependencies{
			Daggerheart: &testDaggerheartStore{
				states: map[string]projectionstore.DaggerheartCharacterState{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", LifeState: mechanics.LifeStateDead},
				},
			},
		})
		if _, err := handler.ResolveBlazeOfGlory(testContext(), &pb.DaggerheartResolveBlazeOfGloryRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}

		handler = newTestHandler(Dependencies{
			Daggerheart: &testDaggerheartStore{
				states: map[string]projectionstore.DaggerheartCharacterState{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", LifeState: mechanics.LifeStateAlive},
				},
			},
		})
		if _, err := handler.ResolveBlazeOfGlory(testContext(), &pb.DaggerheartResolveBlazeOfGloryRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}
	})

	t.Run("temporary armor validates request shape", func(t *testing.T) {
		handler := newTestHandler(Dependencies{})
		if _, err := handler.ApplyTemporaryArmor(testContext(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if _, err := handler.ApplyTemporaryArmor(testContext(), &pb.DaggerheartApplyTemporaryArmorRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	t.Run("swap loadout validates request shape", func(t *testing.T) {
		handler := newTestHandler(Dependencies{})
		if _, err := handler.SwapLoadout(testContext(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if _, err := handler.SwapLoadout(testContext(), &pb.DaggerheartSwapLoadoutRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if _, err := handler.SwapLoadout(testContext(), &pb.DaggerheartSwapLoadoutRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			Swap: &pb.DaggerheartLoadoutSwapRequest{
				CardId:     "card-1",
				RecallCost: -1,
			},
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	t.Run("death move validates request and state guards", func(t *testing.T) {
		handler := newTestHandler(Dependencies{})
		if _, err := handler.ApplyDeathMove(testContext(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if _, err := handler.ApplyDeathMove(testContext(), &pb.DaggerheartApplyDeathMoveRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
			HpClear:     int32Ptr(1),
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}

		handler = newTestHandler(Dependencies{
			Daggerheart: &testDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Level: 1, HpMax: 5, StressMax: 3},
				},
				states: map[string]projectionstore.DaggerheartCharacterState{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 2, Hope: 1, HopeMax: 2, Stress: 1},
				},
			},
		})
		if _, err := handler.ApplyDeathMove(testContext(), &pb.DaggerheartApplyDeathMoveRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}

		handler = newTestHandler(Dependencies{
			Daggerheart: &testDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Level: 1, HpMax: 5, StressMax: 3},
				},
				states: map[string]projectionstore.DaggerheartCharacterState{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 0, Hope: 1, HopeMax: 2, Stress: 1, LifeState: mechanics.LifeStateDead},
				},
			},
		})
		if _, err := handler.ApplyDeathMove(testContext(), &pb.DaggerheartApplyDeathMoveRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}
	})
}

func int32Ptr(v int32) *int32 {
	return &v
}
