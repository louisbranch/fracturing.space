package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

func TestSessionGroupActionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionGroupActionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeader(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeaderTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSupporters(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: "req-group-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
			workflowtransport.KeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			workflowtransport.KeyHopeFear:    false,
		},
	})
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-group-1",
		RollSeq:   1,
		Targets:   []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	now := testTimestamp
	svc.stores.Write.Executor = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-1",
				EntityType:  "roll",
				EntityID:    "req-group-1",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-1",
				EntityType:  "outcome",
				EntityID:    "req-group-1",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-group-1")
	resp, err := svc.SessionGroupActionFlow(ctx, &pb.SessionGroupActionFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		LeaderCharacterId: "char-1",
		LeaderTrait:       "agility",
		Difficulty:        10,
		Supporters: []*pb.GroupActionSupporter{
			{CharacterId: "char-2", Trait: "strength"},
		},
	})
	if err != nil {
		t.Fatalf("SessionGroupActionFlow returned error: %v", err)
	}
	if resp.LeaderRoll == nil {
		t.Fatal("expected leader roll in response")
	}
	if len(resp.SupporterRolls) == 0 {
		t.Fatal("expected supporter rolls in response")
	}
}

func TestSessionGroupActionFlow_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-group-action",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
			workflowtransport.KeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			workflowtransport.KeyHopeFear:    false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomePayload, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-group-action",
		RollSeq:   1,
		Targets:   []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-action",
				EntityType:  "roll",
				EntityID:    "req-group-action",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-action",
				EntityType:  "outcome",
				EntityID:    "req-group-action",
				PayloadJSON: outcomePayload,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-group-action")
	_, err = svc.SessionGroupActionFlow(ctx, &pb.SessionGroupActionFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		LeaderCharacterId: "char-1",
		LeaderTrait:       "agility",
		Difficulty:        10,
		Supporters: []*pb.GroupActionSupporter{
			{CharacterId: "char-2", Trait: "strength"},
		},
	})
	if err != nil {
		t.Fatalf("SessionGroupActionFlow returned error: %v", err)
	}
	if len(domain.commands) != 3 {
		t.Fatalf("expected 3 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("action.roll.resolve") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "action.roll.resolve")
	}
	if domain.commands[1].Type != command.Type("action.roll.resolve") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.roll.resolve")
	}
	if domain.commands[2].Type != command.Type("action.outcome.apply") {
		t.Fatalf("third command type = %s, want %s", domain.commands[2].Type, "action.outcome.apply")
	}
}
