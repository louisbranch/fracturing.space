package recoverytransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	gmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

type testGateStore struct {
	gate storage.SessionGate
	err  error
}

func (s testGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	if s.err != nil {
		return storage.SessionGate{}, s.err
	}
	if s.gate.GateID != "" {
		return s.gate, nil
	}
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	snapshot    projectionstore.DaggerheartSnapshot
	snapshotErr error
	countdowns  map[string]projectionstore.DaggerheartCountdown
	profiles    map[string]projectionstore.DaggerheartCharacterProfile
	states      map[string]projectionstore.DaggerheartCharacterState
}

func (s *testDaggerheartStore) GetDaggerheartSnapshot(context.Context, string) (projectionstore.DaggerheartSnapshot, error) {
	if s.snapshotErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.snapshotErr
	}
	return s.snapshot, nil
}

func (s *testDaggerheartStore) GetDaggerheartCountdown(context.Context, string, string) (projectionstore.DaggerheartCountdown, error) {
	for _, countdown := range s.countdowns {
		return countdown, nil
	}
	return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
}

func (s *testDaggerheartStore) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	for _, profile := range s.profiles {
		return profile, nil
	}
	return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
}

func (s *testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	for _, state := range s.states {
		return state, nil
	}
	return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	ctx = grpcmeta.WithInvocationID(ctx, "inv-1")
	return gmetadata.NewIncomingContext(ctx, gmetadata.Pairs(grpcmeta.SessionIDHeader, "sess-1"))
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = &testDaggerheartStore{
			snapshot: projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 1},
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Level: 1, HpMax: 5, StressMax: 3},
			},
			states: map[string]projectionstore.DaggerheartCharacterState{
				"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 0, Hope: 1, HopeMax: 2, Stress: 2, Armor: 0, LifeState: daggerheart.LifeStateBlazeOfGlory},
			},
		}
	}
	if deps.ResolveSeed == nil {
		deps.ResolveSeed = func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
			return 7, "generated", commonv1.RollMode_LIVE, nil
		}
	}
	if deps.SeedGenerator == nil {
		deps.SeedGenerator = func() (int64, error) { return 7, nil }
	}
	if deps.ExecuteSystemCommand == nil {
		deps.ExecuteSystemCommand = func(context.Context, SystemCommandInput) error { return nil }
	}
	if deps.ApplyStressConditionChange == nil {
		deps.ApplyStressConditionChange = func(context.Context, StressConditionInput) error { return nil }
	}
	if deps.AppendCharacterDeletedEvent == nil {
		deps.AppendCharacterDeletedEvent = func(context.Context, CharacterDeleteInput) error { return nil }
	}
	return NewHandler(deps)
}

func TestHandlerRequireDependencies(t *testing.T) {
	tests := []struct {
		name        string
		deps        Dependencies
		requireSeed bool
	}{
		{name: "missing campaign", deps: Dependencies{}, requireSeed: false},
		{name: "missing gate", deps: Dependencies{Campaign: testCampaignStore{}}, requireSeed: false},
		{name: "missing store", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testGateStore{}}, requireSeed: false},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testGateStore{}, Daggerheart: &testDaggerheartStore{}}, requireSeed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireDependencies(tt.requireSeed); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}

func TestHandlerApplyRestSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		snapshot: projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 2, ConsecutiveShortRests: 1},
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 3, Hope: 2, HopeMax: 2, Stress: 0},
		},
	}
	var commandCount int
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			commandCount++
			return nil
		},
	})

	result, err := handler.ApplyRest(testContext(), &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if commandCount != 1 {
		t.Fatalf("command count = %d, want 1", commandCount)
	}
	if len(result.CharacterStates) != 1 {
		t.Fatalf("character states = %d, want 1", len(result.CharacterStates))
	}
}

func TestHandlerApplyDowntimeMoveInvokesStressCallback(t *testing.T) {
	store := &testDaggerheartStore{
		snapshot: projectionstore.DaggerheartSnapshot{CampaignID: "camp-1"},
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, ArmorMax: 2},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 1, HopeMax: 2, Stress: 2, Armor: 0},
		},
	}
	var stressInput StressConditionInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ApplyStressConditionChange: func(_ context.Context, in StressConditionInput) error {
			stressInput = in
			return nil
		},
	})

	result, err := handler.ApplyDowntimeMove(testContext(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if stressInput.CharacterID != "char-1" {
		t.Fatalf("stress callback character id = %q, want char-1", stressInput.CharacterID)
	}
}

func TestHandlerSwapLoadoutWithRecallCostExecutesStressSpend(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, ArmorMax: 2},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 1, HopeMax: 2, Stress: 2, Armor: 0},
		},
	}
	var commands []SystemCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})

	_, err := handler.SwapLoadout(testContext(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 1,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("command count = %d, want 2", len(commands))
	}
}

func TestHandlerApplyTemporaryArmorSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, ArmorMax: 2},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 1, HopeMax: 2, Stress: 0, Armor: 2},
		},
	}
	var command SystemCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			command = in
			return nil
		},
	})

	result, err := handler.ApplyTemporaryArmor(testContext(), &pb.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		SceneId:     "scene-1",
		Armor: &pb.DaggerheartTemporaryArmor{
			Source:   "ritual",
			Duration: "short_rest",
			Amount:   2,
			SourceId: "blessing:1",
		},
	})
	if err != nil {
		t.Fatalf("ApplyTemporaryArmor returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if command.CommandType != "sys.daggerheart.character_temporary_armor.apply" {
		t.Fatalf("command type = %q, want temporary armor apply", command.CommandType)
	}
	if command.SceneID != "scene-1" {
		t.Fatalf("scene id = %q, want scene-1", command.SceneID)
	}
}

func TestHandlerApplyDeathMoveRiskItAllDeathAppendsDeleteEvent(t *testing.T) {
	seed := findRiskItAllDeathSeed(t)
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, Level: 1},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 0, Hope: 2, HopeMax: 2, Stress: 1},
		},
	}
	var deleted CharacterDeleteInput
	var stressInput StressConditionInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ResolveSeed: func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
			return seed, "replay", commonv1.RollMode_REPLAY, nil
		},
		AppendCharacterDeletedEvent: func(_ context.Context, in CharacterDeleteInput) error {
			deleted = in
			return nil
		},
		ApplyStressConditionChange: func(_ context.Context, in StressConditionInput) error {
			stressInput = in
			return nil
		},
	})

	result, err := handler.ApplyDeathMove(testContext(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		SceneId:     "scene-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if result.Outcome.LifeState != daggerheart.LifeStateDead {
		t.Fatalf("life state = %q, want dead", result.Outcome.LifeState)
	}
	if deleted.CharacterID != "char-1" {
		t.Fatalf("deleted character id = %q, want char-1", deleted.CharacterID)
	}
	if stressInput.CharacterID != "char-1" {
		t.Fatalf("stress callback character id = %q, want char-1", stressInput.CharacterID)
	}
}

func TestHandlerResolveBlazeOfGlorySuccess(t *testing.T) {
	store := &testDaggerheartStore{
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 0, Hope: 0, HopeMax: 2, Stress: 0, LifeState: daggerheart.LifeStateBlazeOfGlory},
		},
	}
	var deleted CharacterDeleteInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		AppendCharacterDeletedEvent: func(_ context.Context, in CharacterDeleteInput) error {
			deleted = in
			return nil
		},
	})

	result, err := handler.ResolveBlazeOfGlory(testContext(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if deleted.CharacterID != "char-1" {
		t.Fatalf("deleted character id = %q, want char-1", deleted.CharacterID)
	}
}

func findRiskItAllDeathSeed(t *testing.T) int64 {
	t.Helper()
	for seed := int64(1); seed <= 256; seed++ {
		outcome, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
			Move:      daggerheart.DeathMoveRiskItAll,
			Level:     1,
			HP:        0,
			HPMax:     5,
			Hope:      2,
			HopeMax:   2,
			Stress:    1,
			StressMax: 3,
			Seed:      seed,
		})
		if err != nil {
			t.Fatalf("ResolveDeathMove seed=%d: %v", seed, err)
		}
		if outcome.LifeState == daggerheart.LifeStateDead {
			return seed
		}
	}
	t.Fatal("could not find deterministic risk_it_all death seed")
	return 0
}
