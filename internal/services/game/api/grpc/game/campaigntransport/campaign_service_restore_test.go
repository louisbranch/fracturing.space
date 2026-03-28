package campaigntransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

func TestRestoreCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.RestoreCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRestoreCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRestoreCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().build())
	_, err := svc.RestoreCampaign(context.Background(), &statev1.RestoreCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRestoreCampaign_NotArchivedDisallowed(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.RestoreCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRestoreCampaign_RequiresDomainEngine(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.RestoreCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestRestoreCampaign_Success(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	ts.Campaign.Campaigns["c1"] = gametest.TestArchivedCampaignRecord(archivedAt)
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"draft"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("campaign-123"))

	resp, err := svc.RestoreCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("RestoreCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_DRAFT {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_DRAFT)
	}
	if resp.Campaign.ArchivedAt != nil {
		t.Error("Campaign ArchivedAt should be nil after restore")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestRestoreCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	archivedAt := now.Add(-24 * time.Hour)

	ts.Campaign.Campaigns["c1"] = gametest.TestArchivedCampaignRecord(archivedAt)

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"draft"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	_, err := svc.RestoreCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.RestoreCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("RestoreCampaign returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.restore") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.restore")
	}
}
