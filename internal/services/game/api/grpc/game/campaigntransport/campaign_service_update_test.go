package campaigntransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSetCampaignCover_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.SetCampaignCover(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_NilRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.UpdateCampaign(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_InvalidLocaleRejected(t *testing.T) {
	svc := NewCampaignService(Deps{})
	_, err := svc.UpdateCampaign(context.Background(), &statev1.UpdateCampaignRequest{
		CampaignId: "c1",
		Locale:     commonv1.Locale(99),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_Success(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	storedCampaign := gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	storedCampaign.Name = "Old Name"
	storedCampaign.ThemePrompt = "Old theme"
	storedCampaign.Locale = "en-US"
	ts.Campaign.Campaigns["c1"] = storedCampaign

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"name":"New Name","theme_prompt":"New theme","locale":"pt-BR"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	resp, err := svc.UpdateCampaign(requestctx.WithParticipantID("owner-1"), &statev1.UpdateCampaignRequest{
		CampaignId:  "c1",
		Name:        wrapperspb.String("  New Name  "),
		ThemePrompt: wrapperspb.String("  New theme  "),
		Locale:      commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("UpdateCampaign returned error: %v", err)
	}
	if resp.GetCampaign().GetName() != "New Name" {
		t.Fatalf("campaign name = %q, want %q", resp.GetCampaign().GetName(), "New Name")
	}
	if resp.GetCampaign().GetThemePrompt() != "New theme" {
		t.Fatalf("campaign theme = %q, want %q", resp.GetCampaign().GetThemePrompt(), "New theme")
	}
	if resp.GetCampaign().GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("campaign locale = %v, want %v", resp.GetCampaign().GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.update") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.update")
	}

	var payload campaign.UpdatePayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["name"] != "New Name" {
		t.Fatalf("payload name = %q, want %q", payload.Fields["name"], "New Name")
	}
	if payload.Fields["theme_prompt"] != "New theme" {
		t.Fatalf("payload theme_prompt = %q, want %q", payload.Fields["theme_prompt"], "New theme")
	}
	if payload.Fields["locale"] != "pt-BR" {
		t.Fatalf("payload locale = %q, want %q", payload.Fields["locale"], "pt-BR")
	}
}

func TestUpdateCampaign_NoOpSkipsDomainCommand(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	storedCampaign := gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	storedCampaign.Name = "Existing Name"
	storedCampaign.ThemePrompt = "Existing theme"
	storedCampaign.Locale = "en-US"
	ts.Campaign.Campaigns["c1"] = storedCampaign

	domain := &fakeDomainEngine{store: ts.Event}
	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	resp, err := svc.UpdateCampaign(requestctx.WithParticipantID("owner-1"), &statev1.UpdateCampaignRequest{
		CampaignId:  "c1",
		Name:        wrapperspb.String("Existing Name"),
		ThemePrompt: wrapperspb.String("Existing theme"),
		Locale:      commonv1.Locale_LOCALE_EN_US,
	})
	if err != nil {
		t.Fatalf("UpdateCampaign returned error: %v", err)
	}
	if domain.calls != 0 {
		t.Fatalf("expected no domain command for no-op update, got %d", domain.calls)
	}
	if resp.GetCampaign().GetName() != "Existing Name" {
		t.Fatalf("campaign name = %q, want %q", resp.GetCampaign().GetName(), "Existing Name")
	}
	if resp.GetCampaign().GetThemePrompt() != "Existing theme" {
		t.Fatalf("campaign theme = %q, want %q", resp.GetCampaign().GetThemePrompt(), "Existing theme")
	}
	if resp.GetCampaign().GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("campaign locale = %v, want %v", resp.GetCampaign().GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestSetCampaignCover_Success(t *testing.T) {
	ts := newTestDeps()
	ts.Participant = ownerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	storedCampaign := gametest.TestCampaignRecordWithStatus(campaign.StatusActive)
	storedCampaign.CoverAssetID = "camp-cover-01"
	ts.Campaign.Campaigns["c1"] = storedCampaign

	domain := &fakeDomainEngine{store: ts.Event, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("campaign.updated"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "campaign",
			EntityID:    "c1",
			PayloadJSON: []byte(`{"fields":{"cover_asset_id":"camp-cover-04"}}`),
		}),
	}}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	resp, err := svc.SetCampaignCover(requestctx.WithParticipantID("owner-1"), &statev1.SetCampaignCoverRequest{
		CampaignId:   "c1",
		CoverAssetId: "camp-cover-04",
	})
	if err != nil {
		t.Fatalf("SetCampaignCover returned error: %v", err)
	}
	if resp.GetCampaign().GetCoverAssetId() != "camp-cover-04" {
		t.Fatalf("campaign cover asset id = %q, want %q", resp.GetCampaign().GetCoverAssetId(), "camp-cover-04")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("campaign.update") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "campaign.update")
	}

	var payload campaign.UpdatePayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["cover_asset_id"] != "camp-cover-04" {
		t.Fatalf("cover_asset_id command field = %q, want %q", payload.Fields["cover_asset_id"], "camp-cover-04")
	}
}
