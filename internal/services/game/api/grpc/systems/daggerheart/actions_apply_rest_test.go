package daggerheart

import (
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
)

func TestApplyRest_ShortRest_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	payloadJSON, err := json.Marshal(daggerheartpayload.RestTakenPayload{
		RestType:    "short",
		Interrupted: false,
		GMFear:      0,
		ShortRests:  1,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

func TestApplyRest_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	restPayload := struct {
		RestType    string `json:"rest_type"`
		Interrupted bool   `json:"interrupted"`
	}{
		RestType:    "short",
		Interrupted: false,
	}
	payloadJSON, err := json.Marshal(restPayload)
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-rest-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-rest-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.rest.take") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.rest.take")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got map[string]any
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode rest command payload: %v", err)
	}
	if got["rest_type"] != "short" {
		t.Fatalf("command rest_type = %v, want %s", got["rest_type"], "short")
	}
}

func TestApplyRest_LongRest_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	payloadJSON, err := json.Marshal(daggerheartpayload.RestTakenPayload{
		RestType:    "long",
		Interrupted: false,
		GMFear:      0,
		ShortRests:  0,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

func TestApplyRest_LongRest_ResponseIncludesCountdownAdvances(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := testTimestamp

	dhStore.Countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "cd-1",
		Name:              "Long-Term Countdown",
		Tone:              "consequence",
		AdvancementPolicy: "long_rest",
		StartingValue:     4,
		RemainingValue:    3,
		LoopBehavior:      "none",
		Status:            "active",
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.RestTakePayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 0,
		ShortRestsAfter:  0,
		CampaignCountdownAdvances: []daggerheartpayload.CampaignCountdownAdvancePayload{{
			CountdownID:     "cd-1",
			BeforeRemaining: 3,
			AfterRemaining:  2,
			AdvancedBy:      1,
			StatusBefore:    "active",
			StatusAfter:     "active",
			Reason:          "long_rest",
		}},
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain
	ctx := contextWithSessionID("sess-1")

	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType:                    pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			LongRestCampaignCountdownId: "cd-1",
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if len(resp.GetCountdownAdvances()) != 1 {
		t.Fatalf("countdown advances = %d, want 1", len(resp.GetCountdownAdvances()))
	}
	advance := resp.GetCountdownAdvances()[0]
	if advance.GetCountdownId() != "cd-1" {
		t.Fatalf("countdown id = %q, want %q", advance.GetCountdownId(), "cd-1")
	}
	if advance.GetRemainingBefore() != 3 || advance.GetRemainingAfter() != 2 || advance.GetAdvancedBy() != 1 {
		t.Fatalf("unexpected countdown advance = %#v", advance)
	}
	if advance.GetAdvancementPolicy() != pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST {
		t.Fatalf("advancement policy = %s, want LONG_REST", advance.GetAdvancementPolicy())
	}
}

func TestApplyRest_LongRest_CountdownFailureDoesNotCommitRest(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	payloadJSON, err := json.Marshal(daggerheartpayload.RestTakenPayload{
		RestType:    "long",
		Interrupted: false,
		GMFear:      0,
		ShortRests:  0,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := contextWithSessionID("sess-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType:                    pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			LongRestCampaignCountdownId: "missing-countdown",
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	assertStatusCode(t, err, codes.NotFound)

	if len(eventStore.Events["camp-1"]) != 0 {
		t.Fatalf("expected no events committed on failed rest flow, got %d", len(eventStore.Events["camp-1"]))
	}
}

func TestApplyRest_LongRest_WithCountdown_UsesSingleDomainCommand(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "cd-1",
		Name:              "Long Term",
		Tone:              "progress",
		AdvancementPolicy: "long_rest",
		StartingValue:     6,
		RemainingValue:    4,
		LoopBehavior:      "none",
		Status:            "active",
	}
	now := testTimestamp

	payloadJSON, err := json.Marshal(daggerheartpayload.RestTakenPayload{
		RestType:    "long",
		Interrupted: false,
		GMFear:      0,
		ShortRests:  0,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := contextWithSessionID("sess-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType:                    pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			LongRestCampaignCountdownId: "cd-1",
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected one domain command, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.rest.take") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.rest.take")
	}
	var got daggerheartpayload.RestTakePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode rest command payload: %v", err)
	}
	if len(got.CampaignCountdownAdvances) != 1 {
		t.Fatalf("campaign countdown advances = %d, want 1", len(got.CampaignCountdownAdvances))
	}
	if got.CampaignCountdownAdvances[0].CountdownID != "cd-1" {
		t.Fatalf("countdown id = %s, want %s", got.CampaignCountdownAdvances[0].CountdownID, "cd-1")
	}
}

func TestApplyRest_RejectsSceneCountdownForCampaignCountdownFields(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:scene-cd-1"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "scene-cd-1",
		SessionID:         "sess-1",
		SceneID:           "scene-1",
		Name:              "Breach",
		Tone:              "consequence",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    3,
		LoopBehavior:      "none",
		Status:            "active",
	}

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType:                    pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			LongRestCampaignCountdownId: "scene-cd-1",
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)

	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			Participants: []*pb.DaggerheartRestParticipant{
				{
					CharacterId: "char-1",
					DowntimeMoves: []*pb.DaggerheartDowntimeSelection{
						{
							Move: &pb.DaggerheartDowntimeSelection_WorkOnProject{
								WorkOnProject: &pb.DaggerheartWorkOnProjectMove{
									ProjectCampaignCountdownId: "scene-cd-1",
								},
							},
						},
					},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
