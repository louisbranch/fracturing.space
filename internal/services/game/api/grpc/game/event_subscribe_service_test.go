package game

import (
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeSubscribeCampaignUpdatesRequestDefaults(t *testing.T) {
	normalized, err := normalizeSubscribeCampaignUpdatesRequest(&campaignv1.SubscribeCampaignUpdatesRequest{
		CampaignId: " camp-1 ",
		AfterSeq:   9,
	})
	if err != nil {
		t.Fatalf("normalize subscribe request: %v", err)
	}
	if normalized.campaignID != "camp-1" {
		t.Fatalf("campaign id = %q, want %q", normalized.campaignID, "camp-1")
	}
	if normalized.afterSeq != 9 {
		t.Fatalf("after seq = %d, want %d", normalized.afterSeq, 9)
	}
	if !normalized.includeEventCommitted {
		t.Fatalf("include event committed = false, want true")
	}
	if !normalized.includeProjection {
		t.Fatalf("include projection = false, want true")
	}
	if normalized.pollInterval != defaultCampaignUpdatePollInterval {
		t.Fatalf("poll interval = %v, want %v", normalized.pollInterval, defaultCampaignUpdatePollInterval)
	}
}

func TestNormalizeSubscribeCampaignUpdatesRequestInvalidKind(t *testing.T) {
	_, err := normalizeSubscribeCampaignUpdatesRequest(&campaignv1.SubscribeCampaignUpdatesRequest{
		CampaignId: "camp-1",
		Kinds:      []campaignv1.CampaignUpdateKind{campaignv1.CampaignUpdateKind(999)},
	})
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestNormalizeSubscribeCampaignUpdatesRequestRequiresAtLeastOneKind(t *testing.T) {
	_, err := normalizeSubscribeCampaignUpdatesRequest(&campaignv1.SubscribeCampaignUpdatesRequest{
		CampaignId: "camp-1",
		Kinds:      []campaignv1.CampaignUpdateKind{campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_UNSPECIFIED},
	})
	if err == nil {
		t.Fatal("expected missing kind error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestNormalizeSubscribeCampaignUpdatesRequestProjectionScopesAndIntervalClamp(t *testing.T) {
	normalized, err := normalizeSubscribeCampaignUpdatesRequest(&campaignv1.SubscribeCampaignUpdatesRequest{
		CampaignId:       "camp-1",
		Kinds:            []campaignv1.CampaignUpdateKind{campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
		ProjectionScopes: []string{" campaign_summary ", "", "campaign_sessions"},
		PollIntervalMs:   1,
	})
	if err != nil {
		t.Fatalf("normalize subscribe request: %v", err)
	}
	if normalized.includeEventCommitted {
		t.Fatalf("include event committed = true, want false")
	}
	if !normalized.includeProjection {
		t.Fatalf("include projection = false, want true")
	}
	if normalized.pollInterval != minCampaignUpdatePollInterval {
		t.Fatalf("poll interval = %v, want %v", normalized.pollInterval, minCampaignUpdatePollInterval)
	}
	if len(normalized.projectionScopes) != 2 {
		t.Fatalf("projection scopes len = %d, want %d", len(normalized.projectionScopes), 2)
	}
	if _, ok := normalized.projectionScopes["campaign_summary"]; !ok {
		t.Fatalf("projection scopes missing %q", "campaign_summary")
	}
	if _, ok := normalized.projectionScopes["campaign_sessions"]; !ok {
		t.Fatalf("projection scopes missing %q", "campaign_sessions")
	}
}
