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

func TestEndCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.EndCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_MissingCampaignId(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndCampaign_NotFound(t *testing.T) {
	svc := NewCampaignService(newTestDeps().withSession().build())
	_, err := svc.EndCampaign(context.Background(), &statev1.EndCampaignRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndCampaign_ActiveSessionBlocks(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	ts.Session.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	ts.Session.ActiveSession["c1"] = "s1"

	svc := NewCampaignService(ts.build())
	_, err := svc.EndCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_DraftStatusDisallowed(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusDraft)

	svc := NewCampaignService(ts.build())
	_, err := svc.EndCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestEndCampaign_AllowsManagerAccess(t *testing.T) {
	ts := newTestDeps().withSession()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
	}
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	resp, err := svc.EndCampaign(requestctx.WithParticipantID(context.Background(), "manager-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if resp.GetCampaign().GetStatus() != statev1.CampaignStatus_COMPLETED {
		t.Fatalf("campaign status = %v, want %v", resp.GetCampaign().GetStatus(), statev1.CampaignStatus_COMPLETED)
	}
}

func TestEndCampaign_RequiresDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewCampaignService(ts.build())
	_, err := svc.EndCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndCampaign_Success(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("campaign-123"))

	resp, err := svc.EndCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if resp.Campaign.Status != statev1.CampaignStatus_COMPLETED {
		t.Errorf("Campaign Status = %v, want %v", resp.Campaign.Status, statev1.CampaignStatus_COMPLETED)
	}
	if resp.Campaign.CompletedAt == nil {
		t.Error("Campaign CompletedAt is nil")
	}

	stored, _ := ts.Campaign.Get(context.Background(), "c1")
	if stored.Status != campaign.StatusCompleted {
		t.Errorf("Stored campaign Status = %v, want %v", stored.Status, campaign.StatusCompleted)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
}

func TestEndCampaign_UsesDomainEngine(t *testing.T) {
	ts := newTestDeps().withSession()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.TestCampaignRecordWithStatus(campaign.StatusActive)

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"status":"completed"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	_, err := svc.EndCampaign(requestctx.WithParticipantID(context.Background(), "owner-1"), &statev1.EndCampaignRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("EndCampaign returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.end") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.end")
	}
}
