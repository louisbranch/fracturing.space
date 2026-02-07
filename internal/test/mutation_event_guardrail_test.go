//go:build integration

package integration

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	"github.com/louisbranch/fracturing.space/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runMutationEventGuardrailTests(t *testing.T, suite *integrationSuite, grpcAddr string) {
	t.Helper()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	eventClient := statev1.NewEventServiceClient(conn)

	t.Run("campaign mutations emit events", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		campaignResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Guardrail Campaign",
				"system":       "DAGGERHEART",
				"gm_mode":      "HUMAN",
				"theme_prompt": "",
			},
		})
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)
		if campaignOutput.ID == "" {
			t.Fatal("campaign_create returned empty id")
		}

		lastSeq := requireLatestSeq(t, ctx, eventClient, campaignOutput.ID)
		if lastSeq == 0 {
			t.Fatal("expected campaign_create to emit at least one event")
		}

		participantResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "participant_create",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"display_name": "Guardrail Player",
				"role":         "PLAYER",
				"controller":   "HUMAN",
			},
		})
		if err != nil {
			t.Fatalf("call participant_create: %v", err)
		}
		if participantResult == nil || participantResult.IsError {
			t.Fatalf("participant_create failed: %+v", participantResult)
		}
		participantOutput := decodeStructuredContent[domain.ParticipantCreateResult](t, participantResult.StructuredContent)
		lastSeq = requireEventAppended(t, ctx, eventClient, campaignOutput.ID, "participant_create", lastSeq)

		characterResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "character_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Guardrail Hero",
				"kind":        "PC",
			},
		})
		if err != nil {
			t.Fatalf("call character_create: %v", err)
		}
		if characterResult == nil || characterResult.IsError {
			t.Fatalf("character_create failed: %+v", characterResult)
		}
		characterOutput := decodeStructuredContent[domain.CharacterCreateResult](t, characterResult.StructuredContent)
		lastSeq = requireEventAppended(t, ctx, eventClient, campaignOutput.ID, "character_create", lastSeq)

		controlResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "character_control_set",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"character_id": characterOutput.ID,
				"controller":   participantOutput.ID,
			},
		})
		if err != nil {
			t.Fatalf("call character_control_set: %v", err)
		}
		if controlResult == nil || controlResult.IsError {
			t.Fatalf("character_control_set failed: %+v", controlResult)
		}
		lastSeq = requireEventAppended(t, ctx, eventClient, campaignOutput.ID, "character_control_set", lastSeq)

		sessionResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_start",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Guardrail Session",
			},
		})
		if err != nil {
			t.Fatalf("call session_start: %v", err)
		}
		if sessionResult == nil || sessionResult.IsError {
			t.Fatalf("session_start failed: %+v", sessionResult)
		}
		sessionOutput := decodeStructuredContent[domain.SessionStartResult](t, sessionResult.StructuredContent)

		endSessionResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_end",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"session_id":  sessionOutput.ID,
			},
		})
		if err != nil {
			t.Fatalf("call session_end: %v", err)
		}
		if endSessionResult == nil || endSessionResult.IsError {
			t.Fatalf("session_end failed: %+v", endSessionResult)
		}

		endCampaignResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_end",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		})
		if err != nil {
			t.Fatalf("call campaign_end: %v", err)
		}
		if endCampaignResult == nil || endCampaignResult.IsError {
			t.Fatalf("campaign_end failed: %+v", endCampaignResult)
		}
		lastSeq = requireEventAppended(t, ctx, eventClient, campaignOutput.ID, "campaign_end", lastSeq)

		archiveResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_archive",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		})
		if err != nil {
			t.Fatalf("call campaign_archive: %v", err)
		}
		if archiveResult == nil || archiveResult.IsError {
			t.Fatalf("campaign_archive failed: %+v", archiveResult)
		}
		lastSeq = requireEventAppended(t, ctx, eventClient, campaignOutput.ID, "campaign_archive", lastSeq)

		restoreResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_restore",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		})
		if err != nil {
			t.Fatalf("call campaign_restore: %v", err)
		}
		if restoreResult == nil || restoreResult.IsError {
			t.Fatalf("campaign_restore failed: %+v", restoreResult)
		}
		_ = requireEventAppended(t, ctx, eventClient, campaignOutput.ID, "campaign_restore", lastSeq)
	})

	t.Run("campaign fork emits event", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		campaignResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Fork Guardrail Campaign",
				"system":       "DAGGERHEART",
				"gm_mode":      "HUMAN",
				"theme_prompt": "",
			},
		})
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)

		forkResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": campaignOutput.ID,
				"new_campaign_name":  "Guardrail Fork",
			},
		})
		if err != nil {
			t.Fatalf("call campaign_fork: %v", err)
		}
		if forkResult == nil || forkResult.IsError {
			t.Fatalf("campaign_fork failed: %+v", forkResult)
		}
		forkOutput := decodeStructuredContent[domain.CampaignForkResult](t, forkResult.StructuredContent)

		requireEventType(t, ctx, eventClient, forkOutput.CampaignID, "campaign.forked")
	})
}

func requireLatestSeq(t *testing.T, ctx context.Context, client statev1.EventServiceClient, campaignID string) uint64 {
	t.Helper()

	response, err := client.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if response == nil || len(response.Events) == 0 {
		return 0
	}
	return response.Events[0].Seq
}

func requireEventAppended(t *testing.T, ctx context.Context, client statev1.EventServiceClient, campaignID, label string, before uint64) uint64 {
	t.Helper()

	after := requireLatestSeq(t, ctx, client, campaignID)
	if after <= before {
		t.Fatalf("expected %s to append event: before=%d after=%d", label, before, after)
	}
	return after
}

func requireEventType(t *testing.T, ctx context.Context, client statev1.EventServiceClient, campaignID, eventType string) {
	t.Helper()

	response, err := client.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   50,
		OrderBy:    "seq desc",
		Filter:     "type = \"" + eventType + "\"",
	})
	if err != nil {
		t.Fatalf("list events for %s: %v", eventType, err)
	}
	if response == nil || len(response.Events) == 0 {
		t.Fatalf("expected event type %s in campaign %s", eventType, campaignID)
	}
}
