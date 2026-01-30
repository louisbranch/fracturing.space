//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

type outcomeAppliedPayload struct {
	RollSeq              uint64   `json:"roll_seq"`
	Targets              []string `json:"targets"`
	RequiresComplication bool     `json:"requires_complication"`
}

// runSessionOutcomeTests exercises applying session roll outcomes.
func runSessionOutcomeTests(t *testing.T, suite *integrationSuite, grpcAddr string) {
	t.Helper()

	t.Run("apply roll outcome", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		campaignResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Outcome Campaign",
				"gm_mode":      "HUMAN",
				"theme_prompt": "roll outcomes",
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

		characterResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "character_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Outcome Hero",
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
		if characterOutput.ID == "" {
			t.Fatal("character_create returned empty id")
		}

		participantResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "participant_create",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"display_name": "Outcome GM",
				"role":         "GM",
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
		if participantOutput.ID == "" {
			t.Fatal("participant_create returned empty id")
		}

		sessionResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_start",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Outcome Session",
			},
		})
		if err != nil {
			t.Fatalf("call session_start: %v", err)
		}
		if sessionResult == nil || sessionResult.IsError {
			t.Fatalf("session_start failed: %+v", sessionResult)
		}
		sessionOutput := decodeStructuredContent[domain.SessionStartResult](t, sessionResult.StructuredContent)
		if sessionOutput.ID == "" {
			t.Fatal("session_start returned empty id")
		}

		contextResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "set_context",
			Arguments: map[string]any{
				"campaign_id":    campaignOutput.ID,
				"session_id":     sessionOutput.ID,
				"participant_id": participantOutput.ID,
			},
		})
		if err != nil {
			t.Fatalf("call set_context: %v", err)
		}
		if contextResult == nil || contextResult.IsError {
			t.Fatalf("set_context failed: %+v", contextResult)
		}

		actionRollResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_action_roll",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"session_id":   sessionOutput.ID,
				"character_id": characterOutput.ID,
				"trait":        "bravery",
				"difficulty":   8,
			},
		})
		if err != nil {
			t.Fatalf("call session_action_roll: %v", err)
		}
		if actionRollResult == nil || actionRollResult.IsError {
			t.Fatalf("session_action_roll failed: %+v", actionRollResult)
		}
		actionRollOutput := decodeStructuredContent[domain.SessionActionRollResult](t, actionRollResult.StructuredContent)
		requiresComplication := false
		if actionRollOutput.Crit {
			requiresComplication = false
		} else if actionRollOutput.Flavor == "FEAR" {
			requiresComplication = true
		}

		conn, err := grpc.NewClient(
			grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		)
		if err != nil {
			t.Fatalf("dial gRPC: %v", err)
		}
		defer conn.Close()
		grpcClient := sessionv1.NewSessionServiceClient(conn)

		rollSeq := actionRollOutput.RollSeq
		if rollSeq == 0 {
			t.Fatal("expected roll seq")
		}

		applyResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_roll_outcome_apply",
			Arguments: map[string]any{
				"session_id": sessionOutput.ID,
				"roll_seq":   rollSeq,
			},
		})
		if err != nil {
			t.Fatalf("call session_roll_outcome_apply: %v", err)
		}
		if applyResult == nil || applyResult.IsError {
			t.Fatalf("session_roll_outcome_apply failed: %+v", applyResult)
		}
		applyOutput := decodeStructuredContent[domain.SessionRollOutcomeApplyResult](t, applyResult.StructuredContent)
		if applyOutput.RollSeq != rollSeq {
			t.Fatalf("expected roll seq %d, got %d", rollSeq, applyOutput.RollSeq)
		}
		if applyOutput.RequiresComplication != requiresComplication {
			t.Fatalf("expected requires_complication %v, got %v", requiresComplication, applyOutput.RequiresComplication)
		}
		if len(applyOutput.Updated.CharacterStates) != 1 {
			t.Fatalf("expected updated character state, got %d", len(applyOutput.Updated.CharacterStates))
		}
		if applyOutput.Updated.CharacterStates[0].CharacterID != characterOutput.ID {
			t.Fatalf("expected character id %q, got %q", characterOutput.ID, applyOutput.Updated.CharacterStates[0].CharacterID)
		}

		applied, err := findOutcomeApplied(ctx, grpcClient, sessionOutput.ID, rollSeq)
		if err != nil {
			t.Fatalf("find outcome applied event: %v", err)
		}
		if applied.RollSeq != rollSeq {
			t.Fatalf("expected outcome roll seq %d, got %d", rollSeq, applied.RollSeq)
		}
		if applied.RequiresComplication != requiresComplication {
			t.Fatalf("expected outcome requires_complication %v, got %v", requiresComplication, applied.RequiresComplication)
		}
		if len(applied.Targets) != 1 || applied.Targets[0] != characterOutput.ID {
			t.Fatalf("expected targets [%s], got %v", characterOutput.ID, applied.Targets)
		}
	})

	// TODO: Enable once session_action_roll supports an optional seed.
	t.Run("apply roll outcome fear increments gm fear", func(t *testing.T) {
		t.Skip("requires deterministic roll seed to force FEAR")
	})
}

func findOutcomeApplied(ctx context.Context, client sessionv1.SessionServiceClient, sessionID string, rollSeq uint64) (outcomeAppliedPayload, error) {
	response, err := client.SessionEventsList(ctx, &sessionv1.SessionEventsListRequest{
		SessionId: sessionID,
		Limit:     50,
	})
	if err != nil {
		return outcomeAppliedPayload{}, err
	}
	if response == nil {
		return outcomeAppliedPayload{}, nil
	}

	for i := len(response.Events); i > 0; i-- {
		event := response.Events[i-1]
		if event.GetType() != sessionv1.SessionEventType_OUTCOME_APPLIED {
			continue
		}
		var payload outcomeAppliedPayload
		if err := json.Unmarshal(event.GetPayloadJson(), &payload); err != nil {
			return outcomeAppliedPayload{}, err
		}
		if payload.RollSeq != rollSeq {
			continue
		}
		return payload, nil
	}

	return outcomeAppliedPayload{}, fmt.Errorf("outcome applied event not found")
}
