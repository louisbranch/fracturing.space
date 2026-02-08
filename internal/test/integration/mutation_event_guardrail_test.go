//go:build integration

package integration

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
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
		participants := readParticipantList(t, suite.client, campaignOutput.ID)
		if len(participants.Participants) == 0 {
			t.Fatal("expected owner participant")
		}
		setContext(t, suite.client, campaignOutput.ID, participants.Participants[0].ID)

		lastSeq := requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, 0, "campaign.created")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "participant.joined")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "character.created", "character.profile_updated", "snapshot.character_state_changed")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "character.controller_assigned")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "session.started")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "session.ended")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "campaign.status_changed")

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
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "campaign.status_changed")

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
		_ = requireEventTypesAfterSeq(t, ctx, eventClient, campaignOutput.ID, lastSeq, "campaign.status_changed")
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

		requireEventTypesAfterSeq(t, ctx, eventClient, forkOutput.CampaignID, 0, "campaign.created", "campaign.forked")
	})
}
