//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	sessionv1 "github.com/louisbranch/fracturing.space/api/gen/go/session/v1"
	dualitydomain "github.com/louisbranch/fracturing.space/internal/duality/domain"
	"github.com/louisbranch/fracturing.space/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type outcomeAppliedPayload struct {
	RollSeq              uint64   `json:"roll_seq"`
	Targets              []string `json:"targets"`
	RequiresComplication bool     `json:"requires_complication"`
}

type actionRollResolvedPayload struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
	Dice       struct {
		HopeDie int `json:"hope_die"`
		FearDie int `json:"fear_die"`
	} `json:"dice"`
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

	t.Run("apply roll outcome fear increments gm fear", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		campaignResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Outcome Campaign Fear",
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

		participantResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "participant_create",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"display_name": "Outcome GM",
				"role":         "GM",
			},
		})
		if err != nil {
			t.Fatalf("call participant_create: %v", err)
		}
		if participantResult == nil || participantResult.IsError {
			t.Fatalf("participant_create failed: %+v", participantResult)
		}
		participantOutput := decodeStructuredContent[domain.ParticipantCreateResult](t, participantResult.StructuredContent)

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

		setControlResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "character_control_set",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"character_id": characterOutput.ID,
				"controller":   "GM",
			},
		})
		if err != nil {
			t.Fatalf("call character_control_set: %v", err)
		}
		if setControlResult == nil || setControlResult.IsError {
			t.Fatalf("character_control_set failed: %+v", setControlResult)
		}

		sessionResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_start",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Outcome Session Fear",
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

		difficulty := 8
		seed := findReplaySeedForFear(t, difficulty)
		actionRollResult, err := suite.client.CallTool(ctx, &mcp.CallToolParams{
			Name: "session_action_roll",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"session_id":   sessionOutput.ID,
				"character_id": characterOutput.ID,
				"trait":        "bravery",
				"difficulty":   difficulty,
				"rng": map[string]any{
					"seed":      seed,
					"roll_mode": "REPLAY",
				},
			},
		})
		if err != nil {
			t.Fatalf("call session_action_roll: %v", err)
		}
		if actionRollResult == nil || actionRollResult.IsError {
			t.Fatalf("session_action_roll failed: %+v", actionRollResult)
		}
		actionRollOutput := decodeStructuredContent[domain.SessionActionRollResult](t, actionRollResult.StructuredContent)
		if actionRollOutput.Rng == nil {
			t.Fatal("expected rng metadata in session_action_roll output")
		}
		if actionRollOutput.Rng.SeedUsed != seed {
			t.Fatalf("expected seed_used %d, got %d", seed, actionRollOutput.Rng.SeedUsed)
		}
		if actionRollOutput.Rng.RollMode != "REPLAY" {
			t.Fatalf("expected roll_mode REPLAY, got %q", actionRollOutput.Rng.RollMode)
		}
		if actionRollOutput.Flavor != "FEAR" {
			t.Fatalf("expected fear flavor, got %q", actionRollOutput.Flavor)
		}
		if actionRollOutput.Crit {
			t.Fatal("expected non-critical fear roll")
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
		resolved, err := findActionRollResolved(ctx, grpcClient, sessionOutput.ID, rollSeq)
		if err != nil {
			t.Fatalf("find action roll resolved event: %v", err)
		}
		if resolved.SeedUsed != seed {
			t.Fatalf("expected resolved seed_used %d, got %d", seed, resolved.SeedUsed)
		}
		if resolved.RollMode != "REPLAY" {
			t.Fatalf("expected resolved roll_mode REPLAY, got %q", resolved.RollMode)
		}
		if resolved.RngAlgo == "" || resolved.SeedSource == "" {
			t.Fatal("expected rng metadata in resolved payload")
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
		if !applyOutput.RequiresComplication {
			t.Fatal("expected requires_complication true")
		}
		if applyOutput.Updated.GMFear == nil || *applyOutput.Updated.GMFear == 0 {
			t.Fatal("expected gm fear increment")
		}
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

func findActionRollResolved(ctx context.Context, client sessionv1.SessionServiceClient, sessionID string, rollSeq uint64) (actionRollResolvedPayload, error) {
	response, err := client.SessionEventsList(ctx, &sessionv1.SessionEventsListRequest{
		SessionId: sessionID,
		Limit:     50,
	})
	if err != nil {
		return actionRollResolvedPayload{}, err
	}
	if response == nil {
		return actionRollResolvedPayload{}, nil
	}

	for i := len(response.Events); i > 0; i-- {
		event := response.Events[i-1]
		if event.GetType() != sessionv1.SessionEventType_ACTION_ROLL_RESOLVED {
			continue
		}
		if event.GetSeq() != rollSeq {
			continue
		}
		var payload actionRollResolvedPayload
		if err := json.Unmarshal(event.GetPayloadJson(), &payload); err != nil {
			return actionRollResolvedPayload{}, err
		}
		return payload, nil
	}

	return actionRollResolvedPayload{}, fmt.Errorf("action roll resolved event not found")
}

func findReplaySeedForFear(t *testing.T, difficulty int) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 50000; seed++ {
		difficultyValue := difficulty
		result, err := dualitydomain.RollAction(dualitydomain.ActionRequest{
			Modifier:   0,
			Difficulty: &difficultyValue,
			Seed:       int64(seed),
		})
		if err != nil {
			continue
		}
		if result.Fear > result.Hope && !result.IsCrit {
			return seed
		}
	}
	t.Fatal("no replay seed found for fear roll")
	return 0
}
