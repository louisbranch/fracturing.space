package campaigntransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestArchiveCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.ArchiveCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestArchiveCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestArchiveCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.ArchiveCampaign(context.Background(), &statev1.ArchiveCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestArchiveCampaign_ActiveSessionBlocks(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Session.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	ts.Session.ActiveSession["c1"] = "s1"

	svc := NewCampaignService(ts.build())
	_, err := svc.ArchiveCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestArchiveCampaign_RequiresDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.ArchiveCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestArchiveCampaign_Success(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusCompleted)
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"archived"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("campaign-123"))

	resp, err := svc.ArchiveCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ArchiveCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_ARCHIVED {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_ARCHIVED)
	}
	if resp.Campaign.ArchivedAt == nil {
		t.Error("Campaign ArchivedAt is nil")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestArchiveCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusCompleted)

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"archived"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	_, err := svc.ArchiveCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.ArchiveCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ArchiveCampaign returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.archive") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.archive")
	}
}
