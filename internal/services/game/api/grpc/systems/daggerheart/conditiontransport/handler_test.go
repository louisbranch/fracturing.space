package conditiontransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testCampaignStore struct {
	record storage.CampaignRecord
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	return s.record, nil
}

type testSessionGateStore struct{}

func (testSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	state     projectionstore.DaggerheartCharacterState
	adversary projectionstore.DaggerheartAdversary
}

func (s testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return s.state, nil
}

func (s testDaggerheartStore) GetDaggerheartAdversary(context.Context, string, string) (projectionstore.DaggerheartAdversary, error) {
	return s.adversary, nil
}

type testEventStore struct {
	event event.Event
}

func (s testEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	return s.event, nil
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
		deps.SessionGate = testSessionGateStore{}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				LifeState:   daggerheart.LifeStateAlive,
				Conditions:  []string{daggerheart.ConditionHidden},
			},
			adversary: projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				Conditions:  []string{daggerheart.ConditionHidden},
			},
		}
	}
	if deps.Event == nil {
		deps.Event = testEventStore{}
	}
	return NewHandler(deps)
}

func testContextWithSessionID(sessionID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.SessionIDHeader, sessionID))
}

func TestHandlerApplyConditionsSuccess(t *testing.T) {
	var commands []DomainCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-1"), "inv-1")
	resp, err := handler.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if resp.CharacterID != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterID)
	}
	if len(resp.Added) != 1 || resp.Added[0] != daggerheart.ConditionVulnerable {
		t.Fatalf("added = %v, want vulnerable", resp.Added)
	}
	if len(commands) != 1 || commands[0].CommandType != commandids.DaggerheartConditionChange {
		t.Fatalf("commands = %+v, want one condition change", commands)
	}
}

func TestHandlerApplyConditionsLifeStatePatchOnly(t *testing.T) {
	var commands []DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				LifeState:   daggerheart.LifeStateAlive,
			},
		},
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(commands) != 1 || commands[0].CommandType != commandids.DaggerheartCharacterStatePatch {
		t.Fatalf("commands = %+v, want one character state patch", commands)
	}
}

func TestHandlerApplyAdversaryConditionsSuccess(t *testing.T) {
	var commands []DomainCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commands = append(commands, in)
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				Conditions:  []string{daggerheart.ConditionHidden},
			}, nil
		},
	})

	ctx := grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-adv-1")
	resp, err := handler.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if resp.AdversaryID != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryID)
	}
	if len(commands) != 1 || commands[0].CommandType != commandids.DaggerheartAdversaryConditionChange {
		t.Fatalf("commands = %+v, want one adversary condition change", commands)
	}
}

func TestHandlerApplyConditionsRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerApplyAdversaryConditionsRejectsSessionMismatch(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Event: testEventStore{event: event.Event{SessionID: "other-session"}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				Conditions:  []string{daggerheart.ConditionHidden},
			}, nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	rollSeq := uint64(7)
	_, err := handler.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		RollSeq:     &rollSeq,
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.InvalidArgument, err)
	}
}
