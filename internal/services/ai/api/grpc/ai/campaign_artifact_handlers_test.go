package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
	"google.golang.org/grpc/metadata"
)

func TestCampaignArtifactHandlersRoundTrip(t *testing.T) {
	store := aifakes.NewCampaignArtifactStore()
	svc, err := NewCampaignArtifactHandlers(CampaignArtifactHandlersConfig{
		Manager: campaigncontext.NewManager(campaigncontext.ManagerConfig{
			Store: store,
			Clock: func() time.Time {
				return time.Date(2026, 3, 14, 1, 32, 0, 0, time.UTC)
			},
		}),
		CampaignAuthorizer: &fakeCampaignAuthorizer{allowed: true},
	})
	if err != nil {
		t.Fatalf("NewCampaignArtifactHandlers: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	ensureResp, err := svc.EnsureCampaignArtifacts(ctx, &aiv1.EnsureCampaignArtifactsRequest{
		CampaignId:        "campaign-1",
		StorySeedMarkdown: "Starter hook",
	})
	if err != nil {
		t.Fatalf("EnsureCampaignArtifacts() error = %v", err)
	}
	if len(ensureResp.GetArtifacts()) != 3 {
		t.Fatalf("EnsureCampaignArtifacts() artifact count = %d, want 3", len(ensureResp.GetArtifacts()))
	}

	listResp, err := svc.ListCampaignArtifacts(ctx, &aiv1.ListCampaignArtifactsRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListCampaignArtifacts() error = %v", err)
	}
	if len(listResp.GetArtifacts()) != 3 {
		t.Fatalf("ListCampaignArtifacts() artifact count = %d, want 3", len(listResp.GetArtifacts()))
	}

	getResp, err := svc.GetCampaignArtifact(ctx, &aiv1.GetCampaignArtifactRequest{
		CampaignId: "campaign-1",
		Path:       campaigncontext.StoryArtifactPath,
	})
	if err != nil {
		t.Fatalf("GetCampaignArtifact() error = %v", err)
	}
	if getResp.GetArtifact().GetContent() != "Starter hook" {
		t.Fatalf("story content = %q, want %q", getResp.GetArtifact().GetContent(), "Starter hook")
	}

	upsertResp, err := svc.UpsertCampaignArtifact(ctx, &aiv1.UpsertCampaignArtifactRequest{
		CampaignId: "campaign-1",
		Path:       campaigncontext.MemoryArtifactPath,
		Content:    "Remember the market debt.",
	})
	if err != nil {
		t.Fatalf("UpsertCampaignArtifact() error = %v", err)
	}
	if upsertResp.GetArtifact().GetContent() != "Remember the market debt." {
		t.Fatalf("memory content = %q", upsertResp.GetArtifact().GetContent())
	}
}
