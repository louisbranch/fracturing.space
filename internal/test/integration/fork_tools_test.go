//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// runForkToolsTests exercises campaign forking gRPC operations.
func runForkToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("fork campaign at current state", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:        "Original Campaign",
			System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode:      statev1.GmMode_HUMAN,
			ThemePrompt: "A test campaign for forking",
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		forkResp, err := suite.fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
			SourceCampaignId: campaignID,
			NewCampaignName:  "Forked Campaign",
			CopyParticipants: false,
		})
		if err != nil {
			t.Fatalf("fork campaign: %v", err)
		}
		forked := forkResp.GetCampaign()
		if forked.GetId() == "" {
			t.Fatal("forked campaign id is empty")
		}
		if forked.GetId() == campaignID {
			t.Fatalf("forked campaign ID should differ from source: %s", forked.GetId())
		}
		if forked.GetName() != "Forked Campaign" {
			t.Fatalf("expected name 'Forked Campaign', got %q", forked.GetName())
		}
		// ForkCampaignResponse includes lineage directly — no separate GetLineage
		// call needed (which would require participant membership in the forked campaign).
		if forkResp.GetLineage().GetParentCampaignId() != campaignID {
			t.Fatalf("expected parent_campaign_id %q, got %q", campaignID, forkResp.GetLineage().GetParentCampaignId())
		}
		if forked.GetStatus() != statev1.CampaignStatus_DRAFT {
			t.Fatalf("expected status DRAFT, got %v", forked.GetStatus())
		}
		if forked.GetCreatedAt() == nil {
			t.Fatal("forked campaign created_at is nil")
		}
	})

	t.Run("fork campaign with auto-generated name", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "My Adventure",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		forkResp, err := suite.fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
			SourceCampaignId: campaignID,
		})
		if err != nil {
			t.Fatalf("fork campaign: %v", err)
		}
		if forkResp.GetCampaign().GetName() != "My Adventure (Fork)" {
			t.Fatalf("expected auto-generated name 'My Adventure (Fork)', got %q", forkResp.GetCampaign().GetName())
		}
	})

	t.Run("get campaign lineage for original", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Original Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		lineageResp, err := suite.fork.GetLineage(ctx, &statev1.GetLineageRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("get lineage: %v", err)
		}
		lineage := lineageResp.GetLineage()
		if lineage.GetCampaignId() != campaignID {
			t.Fatalf("expected campaign_id %q, got %q", campaignID, lineage.GetCampaignId())
		}
		if lineage.GetParentCampaignId() != "" {
			t.Fatalf("expected empty parent_campaign_id for original, got %q", lineage.GetParentCampaignId())
		}
		if lineage.GetOriginCampaignId() != campaignID {
			t.Fatalf("expected origin_campaign_id to be self for original, got %q", lineage.GetOriginCampaignId())
		}
		if lineage.GetDepth() != 0 {
			t.Fatalf("expected depth 0 for original, got %d", lineage.GetDepth())
		}
		if lineage.GetDepth() != 0 || lineage.GetParentCampaignId() != "" {
			t.Fatal("expected original campaign to have depth 0 and no parent")
		}
	})

	t.Run("get campaign lineage for fork", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Original Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		forkResp, err := suite.fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
			SourceCampaignId: campaignID,
			NewCampaignName:  "First Fork",
			CopyParticipants: true,
		})
		if err != nil {
			t.Fatalf("fork campaign: %v", err)
		}
		forkedID := forkResp.GetCampaign().GetId()

		lineageResp, err := suite.fork.GetLineage(ctx, &statev1.GetLineageRequest{CampaignId: forkedID})
		if err != nil {
			t.Fatalf("get lineage: %v", err)
		}
		lineage := lineageResp.GetLineage()
		if lineage.GetCampaignId() != forkedID {
			t.Fatalf("expected campaign_id %q, got %q", forkedID, lineage.GetCampaignId())
		}
		if lineage.GetParentCampaignId() != campaignID {
			t.Fatalf("expected parent_campaign_id %q, got %q", campaignID, lineage.GetParentCampaignId())
		}
		if lineage.GetOriginCampaignId() != campaignID {
			t.Fatalf("expected origin_campaign_id %q, got %q", campaignID, lineage.GetOriginCampaignId())
		}
		if lineage.GetDepth() != 1 {
			t.Fatalf("expected depth 1 for first fork, got %d", lineage.GetDepth())
		}
		if lineage.GetDepth() == 0 {
			t.Fatal("expected depth > 0 for forked campaign")
		}
	})

	t.Run("fork a fork (nested forking)", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Root Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		originalID := campaignResp.GetCampaign().GetId()

		fork1Resp, err := suite.fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
			SourceCampaignId: originalID,
			NewCampaignName:  "First Fork",
			CopyParticipants: true,
		})
		if err != nil {
			t.Fatalf("fork campaign (first): %v", err)
		}
		fork1ID := fork1Resp.GetCampaign().GetId()

		fork2Resp, err := suite.fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
			SourceCampaignId: fork1ID,
			NewCampaignName:  "Second Fork",
			CopyParticipants: true,
		})
		if err != nil {
			t.Fatalf("fork campaign (second): %v", err)
		}
		fork2 := fork2Resp.GetCampaign()

		lineageResp, err := suite.fork.GetLineage(ctx, &statev1.GetLineageRequest{CampaignId: fork2.GetId()})
		if err != nil {
			t.Fatalf("get lineage: %v", err)
		}
		lineage := lineageResp.GetLineage()
		if lineage.GetParentCampaignId() != fork1ID {
			t.Fatalf("expected parent_campaign_id %q, got %q", fork1ID, lineage.GetParentCampaignId())
		}
		if lineage.GetOriginCampaignId() != originalID {
			t.Fatalf("expected origin_campaign_id to trace back to root, got %q", lineage.GetOriginCampaignId())
		}
		if lineage.GetDepth() != 2 {
			t.Fatalf("expected depth 2 for second fork, got %d", lineage.GetDepth())
		}
	})

	t.Run("fork non-existent campaign", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		_, err := suite.fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
			SourceCampaignId: "non-existent-id",
		})
		if err == nil {
			t.Fatal("expected error for non-existent campaign fork")
		}
	})

	t.Run("get lineage for non-existent campaign", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		_, err := suite.fork.GetLineage(ctx, &statev1.GetLineageRequest{CampaignId: "non-existent-id"})
		if err == nil {
			t.Fatal("expected error for non-existent campaign lineage")
		}
	})
}
