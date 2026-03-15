package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

func TestSessionTagTeamFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionTagTeamFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingFirst(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSecond(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First: &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_SameParticipant(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: "req-tagteam-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 18},
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
		RequestID: "req-tagteam-1",
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
				RequestID:   "req-tagteam-1",
				EntityType:  "roll",
				EntityID:    "req-tagteam-1",
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
				RequestID:   "req-tagteam-1",
				EntityType:  "outcome",
				EntityID:    "req-tagteam-1",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-tagteam-1")
	resp, err := svc.SessionTagTeamFlow(ctx, &pb.SessionTagTeamFlowRequest{
		CampaignId:          "camp-1",
		SessionId:           "sess-1",
		Difficulty:          10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-2", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SessionTagTeamFlow returned error: %v", err)
	}
	if resp.FirstRoll == nil {
		t.Fatal("expected first roll in response")
	}
	if resp.SecondRoll == nil {
		t.Fatal("expected second roll in response")
	}
}

func TestSessionTagTeamFlow_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-tag-team",
		RollSeq:   1,
		Results:   map[string]any{"d20": 18},
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
		RequestID: "req-tag-team",
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
				RequestID:   "req-tag-team",
				EntityType:  "roll",
				EntityID:    "req-tag-team",
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
				RequestID:   "req-tag-team",
				EntityType:  "outcome",
				EntityID:    "req-tag-team",
				PayloadJSON: outcomePayload,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-tag-team")
	_, err = svc.SessionTagTeamFlow(ctx, &pb.SessionTagTeamFlowRequest{
		CampaignId:          "camp-1",
		SessionId:           "sess-1",
		Difficulty:          10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-2", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SessionTagTeamFlow returned error: %v", err)
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
