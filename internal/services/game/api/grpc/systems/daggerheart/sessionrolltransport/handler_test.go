package sessionrolltransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type testCampaignStore struct {
	record storage.CampaignRecord
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	return s.record, nil
}

type testSessionStore struct {
	record storage.SessionRecord
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	return s.record, nil
}

type testSessionGateStore struct{}

func (testSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	state projectionstore.DaggerheartCharacterState
}

func (s testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return s.state, nil
}

type testEventStore struct {
	latestSeq uint64
}

func (s testEventStore) GetLatestEventSeq(context.Context, string) (uint64, error) {
	return s.latestSeq, nil
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Session == nil {
		deps.Session = testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testSessionGateStore{}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{state: projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Hope:        3,
		}}
	}
	if deps.Event == nil {
		deps.Event = testEventStore{latestSeq: 4}
	}
	if deps.SeedFunc == nil {
		deps.SeedFunc = func() (int64, error) { return 42, nil }
	}
	return NewHandler(deps)
}

func TestHandlerSessionActionRollSuccess(t *testing.T) {
	var spendCalls []HopeSpendInput
	var resolveInput RollResolveInput
	var countdownID string

	handler := newTestHandler(Dependencies{
		ExecuteHopeSpend: func(_ context.Context, in HopeSpendInput) error {
			spendCalls = append(spendCalls, in)
			return nil
		},
		ExecuteActionRollResolve: func(_ context.Context, in RollResolveInput) (uint64, error) {
			resolveInput = in
			return 9, nil
		},
		AdvanceBreathCountdown: func(_ context.Context, _, _, id string, _ bool) error {
			countdownID = id
			return nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(context.Background(), "req-1"), "inv-1")
	resp, err := handler.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		CharacterId:       "char-1",
		Trait:             "agility",
		Difficulty:        10,
		BreathCountdownId: "countdown-1",
		Modifiers: []*pb.ActionRollModifier{
			{Value: 2, Source: "experience"},
		},
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.GetRollSeq() != 9 {
		t.Fatalf("roll_seq = %d, want 9", resp.GetRollSeq())
	}
	if len(spendCalls) != 1 {
		t.Fatalf("hope spend calls = %d, want 1", len(spendCalls))
	}
	if spendCalls[0].HopeBefore != 3 || spendCalls[0].HopeAfter != 2 {
		t.Fatalf("hope spend before/after = %d/%d, want 3/2", spendCalls[0].HopeBefore, spendCalls[0].HopeAfter)
	}
	if countdownID != "countdown-1" {
		t.Fatalf("countdown id = %q, want countdown-1", countdownID)
	}
	if resolveInput.EntityType != "roll" || resolveInput.EntityID != "req-1" {
		t.Fatalf("resolve entity = %s/%s", resolveInput.EntityType, resolveInput.EntityID)
	}

	var payload map[string]any
	if err := json.Unmarshal(resolveInput.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got := payload["request_id"]; got != "req-1" {
		t.Fatalf("payload request_id = %v, want req-1", got)
	}
}

func TestHandlerSessionDamageRollSuccess(t *testing.T) {
	var resolveCalls int
	handler := newTestHandler(Dependencies{
		ExecuteDamageRollResolve: func(_ context.Context, in RollResolveInput) (uint64, error) {
			resolveCalls++
			if in.MissingEventMsg != "damage roll did not emit an event" {
				t.Fatalf("missing event msg = %q", in.MissingEventMsg)
			}
			return 8, nil
		},
	})

	resp, err := handler.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Dice:        []*pb.DiceSpec{{Sides: 6, Count: 2}},
		Modifier:    1,
	})
	if err != nil {
		t.Fatalf("SessionDamageRoll returned error: %v", err)
	}
	if resp.GetRollSeq() != 8 {
		t.Fatalf("roll_seq = %d, want 8", resp.GetRollSeq())
	}
	if resolveCalls != 1 {
		t.Fatalf("resolve calls = %d, want 1", resolveCalls)
	}
}

func TestHandlerSessionAdversaryAttackRollSuccess(t *testing.T) {
	var loaded bool
	handler := newTestHandler(Dependencies{
		ExecuteAdversaryRollResolve: func(_ context.Context, in RollResolveInput) (uint64, error) {
			if in.EntityType != "adversary" || in.EntityID != "adv-1" {
				t.Fatalf("resolve entity = %s/%s", in.EntityType, in.EntityID)
			}
			return 6, nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			loaded = true
			return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
		},
	})

	resp, err := handler.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId:     "camp-1",
		SessionId:      "sess-1",
		AdversaryId:    "adv-1",
		AttackModifier: 2,
		Advantage:      1,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if !loaded {
		t.Fatal("expected adversary loader to be called")
	}
	if resp.GetRollSeq() != 6 {
		t.Fatalf("roll_seq = %d, want 6", resp.GetRollSeq())
	}
}

func TestHandlerSessionAdversaryActionCheckAutoSuccess(t *testing.T) {
	handler := newTestHandler(Dependencies{
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
		},
	})

	resp, err := handler.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
		Modifier:    2,
		Dramatic:    false,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if !resp.GetAutoSuccess() {
		t.Fatal("expected auto success")
	}
	if resp.GetRollSeq() != 5 {
		t.Fatalf("roll_seq = %d, want 5", resp.GetRollSeq())
	}
	if resp.GetTotal() != 2 {
		t.Fatalf("total = %d, want 2", resp.GetTotal())
	}
}
