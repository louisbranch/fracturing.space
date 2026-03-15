package recoverytransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	gmetadata "google.golang.org/grpc/metadata"
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
