package snapshottransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
)

func TestUpdateSnapshotState_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.UpdateSnapshotState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "nonexistent",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestUpdateSnapshotState_RequiresManageSessionsPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})

	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 2},
		},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestUpdateSnapshotState_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestUpdateSnapshotState_NegativeGmFear(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: -1},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})

	_, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 7},
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateSnapshotState_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	payloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 7})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 7},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 7 {
		t.Errorf("Response GmFear = %d, want %d", dh.GetGmFear(), 7)
	}

	stored, err := dhStore.GetDaggerheartSnapshot(context.Background(), "c1")
	if err != nil {
		t.Fatalf("DaggerheartSnapshot not persisted: %v", err)
	}
	if stored.GMFear != 7 {
		t.Errorf("Stored GMFear = %d, want %d", stored.GMFear, 7)
	}

	if len(eventStore.Events["c1"]) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.Events["c1"]))
	}
	if eventStore.Events["c1"][0].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, "sys.daggerheart.gm_fear_changed")
	}
}

func TestUpdateSnapshotState_UpdateExisting(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 3}

	payloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 10})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 10},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 10 {
		t.Errorf("Response GmFear = %d, want %d", dh.GetGmFear(), 10)
	}

	stored, _ := dhStore.GetDaggerheartSnapshot(context.Background(), "c1")
	if stored.GMFear != 10 {
		t.Errorf("Stored GMFear = %d, want %d", stored.GMFear, 10)
	}
}

func TestUpdateSnapshotState_SetToZero(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	payloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 0})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 0},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 0 {
		t.Errorf("Response GmFear = %d, want 0", dh.GetGmFear())
	}
}

func TestUpdateSnapshotState_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	payloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 5})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	_, err = svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_fear.set")
	}
}
