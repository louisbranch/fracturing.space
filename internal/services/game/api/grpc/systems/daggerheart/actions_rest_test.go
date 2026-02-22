package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

// --- ApplyRest tests ---

func TestApplyRest_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRest_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRest_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingRest(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_UnspecifiedRestType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_ShortRest_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "short",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 0,
		ShortRestsAfter:  1,
		RefreshRest:      false,
		RefreshLongRest:  false,
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
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
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
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

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
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-rest-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
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
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 1,
		ShortRestsAfter:  0,
		RefreshRest:      false,
		RefreshLongRest:  false,
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
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize: 3,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

func TestApplyRest_LongRest_CountdownFailureDoesNotCommitRest(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 1,
		ShortRestsAfter:  0,
		RefreshRest:      false,
		RefreshLongRest:  false,
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
	svc.stores.Domain = domain

	ctx := contextWithSessionID("sess-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:            pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize:           3,
			LongTermCountdownId: "missing-countdown",
		},
	})
	assertStatusCode(t, err, codes.Internal)

	if len(eventStore.Events["camp-1"]) != 0 {
		t.Fatalf("expected no events committed on failed rest flow, got %d", len(eventStore.Events["camp-1"]))
	}
}

func TestApplyRest_LongRest_WithCountdown_UsesSingleDomainCommand(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-1",
		Name:        "Long Term",
		Kind:        "progress",
		Current:     2,
		Max:         6,
		Direction:   "increase",
		Looping:     false,
	}
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 1,
		ShortRestsAfter:  0,
		RefreshRest:      false,
		RefreshLongRest:  false,
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
	svc.stores.Domain = domain

	ctx := contextWithSessionID("sess-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:            pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize:           3,
			LongTermCountdownId: "cd-1",
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
	var got daggerheart.RestTakePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode rest command payload: %v", err)
	}
	if got.LongTermCountdown == nil {
		t.Fatal("expected long_term_countdown payload")
	}
	if got.LongTermCountdown.CountdownID != "cd-1" {
		t.Fatalf("countdown id = %s, want %s", got.LongTermCountdown.CountdownID, "cd-1")
	}
}
