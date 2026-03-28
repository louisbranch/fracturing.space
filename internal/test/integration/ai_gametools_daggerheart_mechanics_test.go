//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/gametools"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestAIDirectSessionDaggerheartMechanicsTools(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
	defer stopServer()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := gamev1.NewCampaignServiceClient(conn)
	characterClient := gamev1.NewCharacterServiceClient(conn)
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	sceneClient := gamev1.NewSceneServiceClient(conn)
	interactionClient := gamev1.NewInteractionServiceClient(conn)
	snapshotClient := gamev1.NewSnapshotServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "ai-gametools-mechanics")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:        "AI Mechanics Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "ai mechanics",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()

	characterID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Aria")
	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, characterID)
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, ownerParticipantID, characterID)

	_, err = snapshotClient.PatchCharacterState(ctxWithUser, &gamev1.PatchCharacterStateRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:     6,
				Hope:   2,
				Stress: 3,
				Armor:  1,
			},
		},
	})
	if err != nil {
		t.Fatalf("patch character state: %v", err)
	}

	startSession := startSessionWithDefaultControllers(t, ctxWithUser, sessionClient, characterClient, campaignID, "AI Mechanics Session")
	sessionID := startSession.GetSession().GetId()

	createSceneResp, err := sceneClient.CreateScene(ctxWithUser, &gamev1.CreateSceneRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		Name:         "Harbor Breach",
		Description:  "The tide batters the gate.",
		CharacterIds: []string{characterID},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createSceneResp.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene id")
	}

	directSession := gametools.NewDirectSession(gametools.Clients{
		Interaction: interactionClient,
		Scene:       sceneClient,
		Campaign:    campaignClient,
		Participant: participantClient,
		Character:   characterClient,
		Session:     sessionClient,
		Snapshot:    snapshotClient,
		Daggerheart: daggerheartClient,
	}, gametools.SessionContext{
		CampaignID:    campaignID,
		SessionID:     sessionID,
		ParticipantID: ownerParticipantID,
	})

	t.Run("character sheet read then action roll resolve", func(t *testing.T) {
		sheetResult, err := directSession.CallTool(ctxWithUser, "character_sheet_read", map[string]any{
			"character_id": characterID,
		})
		if err != nil {
			t.Fatalf("character_sheet_read: %v", err)
		}
		if sheetResult.IsError {
			t.Fatalf("character_sheet_read returned tool error: %s", sheetResult.Output)
		}
		var sheet struct {
			Daggerheart struct {
				DomainCards []struct {
					Name string `json:"name"`
				} `json:"domain_cards"`
			} `json:"daggerheart"`
		}
		if err := json.Unmarshal([]byte(sheetResult.Output), &sheet); err != nil {
			t.Fatalf("decode sheet result: %v", err)
		}
		if len(sheet.Daggerheart.DomainCards) == 0 {
			t.Fatalf("expected populated domain cards, got %s", sheetResult.Output)
		}

		difficulty := 8
		seed := findReplaySeedForCritical(t, difficulty)
		resolveResult, err := directSession.CallTool(ctxWithUser, "daggerheart_action_roll_resolve", map[string]any{
			"character_id":              characterID,
			"trait":                     "presence",
			"difficulty":                difficulty,
			"modifiers":                 []map[string]any{},
			"advantage":                 0,
			"disadvantage":              0,
			"underwater":                false,
			"breath_scene_countdown_id": "",
			"scene_id":                  sceneID,
			"replace_hope_with_armor":   false,
			"context":                   "",
			"targets":                   []string{},
			"swap_hope_fear":            false,
			"rng": map[string]any{
				"seed":      seed,
				"roll_mode": "REPLAY",
			},
		})
		if err != nil {
			t.Fatalf("daggerheart_action_roll_resolve: %v", err)
		}
		if resolveResult.IsError {
			t.Fatalf("daggerheart_action_roll_resolve returned tool error: %s", resolveResult.Output)
		}
		var resolved struct {
			ActionRoll struct {
				Crit    bool   `json:"crit"`
				Outcome string `json:"outcome"`
			} `json:"action_roll"`
			RollOutcome struct {
				Updated struct {
					CharacterStates []struct {
						CharacterID string `json:"character_id"`
						Hope        int    `json:"hope"`
						Stress      int    `json:"stress"`
					} `json:"character_states"`
				} `json:"updated"`
			} `json:"roll_outcome"`
		}
		if err := json.Unmarshal([]byte(resolveResult.Output), &resolved); err != nil {
			t.Fatalf("decode action resolve result: %v", err)
		}
		if !resolved.ActionRoll.Crit {
			t.Fatalf("expected crit, got %s", resolveResult.Output)
		}
		if resolved.ActionRoll.Outcome != daggerheartv1.Outcome_CRITICAL_SUCCESS.String() {
			t.Fatalf("outcome = %q, want %q", resolved.ActionRoll.Outcome, daggerheartv1.Outcome_CRITICAL_SUCCESS.String())
		}
		state := fetchCharacterState(t, ctxWithUser, snapshotClient, campaignID, characterID)
		if state.GetHope() != 3 {
			t.Fatalf("hope = %d, want 3", state.GetHope())
		}
		if state.GetStress() != 2 {
			t.Fatalf("stress = %d, want 2", state.GetStress())
		}
	})

	t.Run("gm move apply", func(t *testing.T) {
		_, err := snapshotClient.UpdateSnapshotState(ctxWithUser, &gamev1.UpdateSnapshotStateRequest{
			CampaignId: campaignID,
			SystemSnapshotUpdate: &gamev1.UpdateSnapshotStateRequest_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartSnapshot{
					GmFear:                3,
					ConsecutiveShortRests: 0,
				},
			},
		})
		if err != nil {
			t.Fatalf("update snapshot state: %v", err)
		}

		moveResult, err := directSession.CallTool(ctxWithUser, "daggerheart_gm_move_apply", map[string]any{
			"fear_spent": 2,
			"scene_id":   sceneID,
			"direct_move": map[string]any{
				"kind":         "ADDITIONAL_MOVE",
				"shape":        "SHIFT_ENVIRONMENT",
				"description":  "The floodwater bursts through the lower planks.",
				"adversary_id": "",
			},
			"adversary_feature":    map[string]any{},
			"environment_feature":  map[string]any{},
			"adversary_experience": map[string]any{},
		})
		if err != nil {
			t.Fatalf("daggerheart_gm_move_apply: %v", err)
		}
		if moveResult.IsError {
			t.Fatalf("daggerheart_gm_move_apply returned tool error: %s", moveResult.Output)
		}
		var move struct {
			GMFearBefore int `json:"gm_fear_before"`
			GMFearAfter  int `json:"gm_fear_after"`
		}
		if err := json.Unmarshal([]byte(moveResult.Output), &move); err != nil {
			t.Fatalf("decode gm move result: %v", err)
		}
		if move.GMFearBefore != 3 || move.GMFearAfter != 1 {
			t.Fatalf("gm fear = %d -> %d, want 3 -> 1", move.GMFearBefore, move.GMFearAfter)
		}
	})

	t.Run("adversary create", func(t *testing.T) {
		createResult, err := directSession.CallTool(ctxWithUser, "daggerheart_adversary_create", map[string]any{
			"scene_id":           sceneID,
			"adversary_entry_id": "adversary.integration-foe",
			"notes":              "Testing pressure on the breach.",
		})
		if err != nil {
			t.Fatalf("daggerheart_adversary_create: %v", err)
		}
		if createResult.IsError {
			t.Fatalf("daggerheart_adversary_create returned tool error: %s", createResult.Output)
		}
		var adversary struct {
			ID      string `json:"id"`
			SceneID string `json:"scene_id"`
			Notes   string `json:"notes"`
		}
		if err := json.Unmarshal([]byte(createResult.Output), &adversary); err != nil {
			t.Fatalf("decode adversary create result: %v", err)
		}
		if adversary.ID == "" {
			t.Fatalf("expected created adversary id, got %s", createResult.Output)
		}
		if adversary.SceneID != sceneID {
			t.Fatalf("scene_id = %q, want %q", adversary.SceneID, sceneID)
		}

		boardResult, err := directSession.CallTool(ctxWithUser, "daggerheart_combat_board_read", map[string]any{})
		if err != nil {
			t.Fatalf("daggerheart_combat_board_read: %v", err)
		}
		if boardResult.IsError {
			t.Fatalf("daggerheart_combat_board_read returned tool error: %s", boardResult.Output)
		}
		var board struct {
			Adversaries []struct {
				ID string `json:"id"`
			} `json:"adversaries"`
		}
		if err := json.Unmarshal([]byte(boardResult.Output), &board); err != nil {
			t.Fatalf("decode combat board result: %v", err)
		}
		found := false
		for _, entry := range board.Adversaries {
			if entry.ID == adversary.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("created adversary %q not found on combat board: %s", adversary.ID, boardResult.Output)
		}
	})

	t.Run("countdown create update and adversary update", func(t *testing.T) {
		createCountdownResult, err := directSession.CallTool(ctxWithUser, "daggerheart_scene_countdown_create", map[string]any{
			"scene_id":             sceneID,
			"name":                 "Breach Collapse",
			"tone":                 "CONSEQUENCE",
			"advancement_policy":   "MANUAL",
			"fixed_starting_value": 4,
			"loop_behavior":        "NONE",
		})
		if err != nil {
			t.Fatalf("daggerheart_scene_countdown_create: %v", err)
		}
		if createCountdownResult.IsError {
			t.Fatalf("daggerheart_scene_countdown_create returned tool error: %s", createCountdownResult.Output)
		}
		var countdown struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			Tone              string `json:"tone"`
			StartingValue     int    `json:"starting_value"`
			RemainingValue    int    `json:"remaining_value"`
			LoopBehavior      string `json:"loop_behavior"`
			AdvancementPolicy string `json:"advancement_policy"`
			Status            string `json:"status"`
		}
		if err := json.Unmarshal([]byte(createCountdownResult.Output), &countdown); err != nil {
			t.Fatalf("decode countdown create result: %v", err)
		}
		if countdown.ID == "" || countdown.Name != "Breach Collapse" {
			t.Fatalf("unexpected countdown create result: %s", createCountdownResult.Output)
		}
		if countdown.Tone != "CONSEQUENCE" || countdown.StartingValue != 4 || countdown.RemainingValue != 4 || countdown.LoopBehavior != "NONE" || countdown.AdvancementPolicy != "MANUAL" || countdown.Status != "ACTIVE" {
			t.Fatalf("unexpected countdown metadata: %s", createCountdownResult.Output)
		}

		updateCountdownResult, err := directSession.CallTool(ctxWithUser, "daggerheart_scene_countdown_advance", map[string]any{
			"scene_id":     sceneID,
			"countdown_id": countdown.ID,
			"amount":       1,
			"reason":       "breach buckles under pressure",
		})
		if err != nil {
			t.Fatalf("daggerheart_scene_countdown_advance: %v", err)
		}
		if updateCountdownResult.IsError {
			t.Fatalf("daggerheart_scene_countdown_advance returned tool error: %s", updateCountdownResult.Output)
		}
		var countdownUpdate struct {
			Countdown struct {
				ID             string `json:"id"`
				RemainingValue int    `json:"remaining_value"`
			} `json:"countdown"`
			Advance struct {
				BeforeRemaining int `json:"before_remaining"`
				AfterRemaining  int `json:"after_remaining"`
				AdvancedBy      int `json:"advanced_by"`
			} `json:"advance"`
		}
		if err := json.Unmarshal([]byte(updateCountdownResult.Output), &countdownUpdate); err != nil {
			t.Fatalf("decode countdown update result: %v", err)
		}
		if countdownUpdate.Countdown.ID != countdown.ID {
			t.Fatalf("updated countdown id = %q, want %q", countdownUpdate.Countdown.ID, countdown.ID)
		}
		if countdownUpdate.Advance.BeforeRemaining != 4 || countdownUpdate.Advance.AfterRemaining != 3 || countdownUpdate.Advance.AdvancedBy != 1 || countdownUpdate.Countdown.RemainingValue != 3 {
			t.Fatalf("unexpected countdown update result: %s", updateCountdownResult.Output)
		}

		triggerCountdownResult, err := directSession.CallTool(ctxWithUser, "daggerheart_scene_countdown_advance", map[string]any{
			"scene_id":     sceneID,
			"countdown_id": countdown.ID,
			"amount":       3,
			"reason":       "the sea gate gives way",
		})
		if err != nil {
			t.Fatalf("daggerheart_scene_countdown_advance to trigger: %v", err)
		}
		if triggerCountdownResult.IsError {
			t.Fatalf("daggerheart_scene_countdown_advance to trigger returned tool error: %s", triggerCountdownResult.Output)
		}
		var countdownTrigger struct {
			Countdown struct {
				ID             string `json:"id"`
				RemainingValue int    `json:"remaining_value"`
				Status         string `json:"status"`
			} `json:"countdown"`
			Advance struct {
				BeforeRemaining int    `json:"before_remaining"`
				AfterRemaining  int    `json:"after_remaining"`
				AdvancedBy      int    `json:"advanced_by"`
				StatusBefore    string `json:"status_before"`
				StatusAfter     string `json:"status_after"`
				Triggered       bool   `json:"triggered"`
			} `json:"advance"`
		}
		if err := json.Unmarshal([]byte(triggerCountdownResult.Output), &countdownTrigger); err != nil {
			t.Fatalf("decode countdown trigger result: %v", err)
		}
		if countdownTrigger.Countdown.ID != countdown.ID {
			t.Fatalf("triggered countdown id = %q, want %q", countdownTrigger.Countdown.ID, countdown.ID)
		}
		if countdownTrigger.Advance.BeforeRemaining != 3 || countdownTrigger.Advance.AfterRemaining != 0 || countdownTrigger.Advance.AdvancedBy != 3 || !countdownTrigger.Advance.Triggered {
			t.Fatalf("unexpected countdown trigger advance result: %s", triggerCountdownResult.Output)
		}
		if countdownTrigger.Advance.StatusAfter != "TRIGGER_PENDING" || countdownTrigger.Countdown.Status != "TRIGGER_PENDING" {
			t.Fatalf("unexpected countdown trigger status: %s", triggerCountdownResult.Output)
		}

		resolveCountdownResult, err := directSession.CallTool(ctxWithUser, "daggerheart_scene_countdown_resolve_trigger", map[string]any{
			"scene_id":     sceneID,
			"countdown_id": countdown.ID,
			"reason":       "the collapse lands and the immediate danger is spent",
		})
		if err != nil {
			t.Fatalf("daggerheart_scene_countdown_resolve_trigger: %v", err)
		}
		if resolveCountdownResult.IsError {
			t.Fatalf("daggerheart_scene_countdown_resolve_trigger returned tool error: %s", resolveCountdownResult.Output)
		}
		var countdownResolved struct {
			ID             string `json:"id"`
			RemainingValue int    `json:"remaining_value"`
			Status         string `json:"status"`
		}
		if err := json.Unmarshal([]byte(resolveCountdownResult.Output), &countdownResolved); err != nil {
			t.Fatalf("decode countdown resolve result: %v", err)
		}
		if countdownResolved.ID != countdown.ID || countdownResolved.Status != "ACTIVE" || countdownResolved.RemainingValue != 0 {
			t.Fatalf("unexpected countdown resolve result: %s", resolveCountdownResult.Output)
		}

		createAdversaryResult, err := directSession.CallTool(ctxWithUser, "daggerheart_adversary_create", map[string]any{
			"scene_id":           sceneID,
			"adversary_entry_id": "adversary.integration-foe",
			"notes":              "Initial breach pressure.",
		})
		if err != nil {
			t.Fatalf("daggerheart_adversary_create for update: %v", err)
		}
		if createAdversaryResult.IsError {
			t.Fatalf("daggerheart_adversary_create for update returned tool error: %s", createAdversaryResult.Output)
		}
		var createdAdversary struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(createAdversaryResult.Output), &createdAdversary); err != nil {
			t.Fatalf("decode adversary create result: %v", err)
		}
		if createdAdversary.ID == "" {
			t.Fatalf("expected adversary id, got %s", createAdversaryResult.Output)
		}

		updateAdversaryResult, err := directSession.CallTool(ctxWithUser, "daggerheart_adversary_update", map[string]any{
			"adversary_id": createdAdversary.ID,
			"scene_id":     sceneID,
			"notes":        "Holding the broken gate while the collapse countdown advances.",
		})
		if err != nil {
			t.Fatalf("daggerheart_adversary_update: %v", err)
		}
		if updateAdversaryResult.IsError {
			t.Fatalf("daggerheart_adversary_update returned tool error: %s", updateAdversaryResult.Output)
		}
		var updatedAdversary struct {
			ID    string `json:"id"`
			Notes string `json:"notes"`
		}
		if err := json.Unmarshal([]byte(updateAdversaryResult.Output), &updatedAdversary); err != nil {
			t.Fatalf("decode adversary update result: %v", err)
		}
		if updatedAdversary.ID != createdAdversary.ID {
			t.Fatalf("updated adversary id = %q, want %q", updatedAdversary.ID, createdAdversary.ID)
		}
		if updatedAdversary.Notes != "Holding the broken gate while the collapse countdown advances." {
			t.Fatalf("adversary notes = %q", updatedAdversary.Notes)
		}

		boardResult, err := directSession.CallTool(ctxWithUser, "daggerheart_combat_board_read", map[string]any{})
		if err != nil {
			t.Fatalf("daggerheart_combat_board_read after update: %v", err)
		}
		if boardResult.IsError {
			t.Fatalf("daggerheart_combat_board_read after update returned tool error: %s", boardResult.Output)
		}
		var board struct {
			SceneID    string `json:"scene_id"`
			Countdowns []struct {
				ID             string `json:"id"`
				RemainingValue int    `json:"remaining_value"`
				Status         string `json:"status"`
			} `json:"countdowns"`
			Adversaries []struct {
				ID    string `json:"id"`
				Notes string `json:"notes"`
			} `json:"adversaries"`
		}
		if err := json.Unmarshal([]byte(boardResult.Output), &board); err != nil {
			t.Fatalf("decode combat board after update: %v", err)
		}
		if board.SceneID != sceneID {
			t.Fatalf("scene_id = %q, want %q", board.SceneID, sceneID)
		}
		countdownFound := false
		for _, entry := range board.Countdowns {
			if entry.ID == countdown.ID {
				countdownFound = true
				if entry.RemainingValue != 0 {
					t.Fatalf("combat board countdown remaining_value = %d, want 0", entry.RemainingValue)
				}
				if entry.Status != "ACTIVE" {
					t.Fatalf("combat board countdown status = %q, want ACTIVE", entry.Status)
				}
				break
			}
		}
		if !countdownFound {
			t.Fatalf("updated countdown %q not found on combat board: %s", countdown.ID, boardResult.Output)
		}
		found := false
		for _, entry := range board.Adversaries {
			if entry.ID == createdAdversary.ID {
				found = true
				if entry.Notes != "Holding the broken gate while the collapse countdown advances." {
					t.Fatalf("combat board adversary notes = %q", entry.Notes)
				}
				break
			}
		}
		if !found {
			t.Fatalf("updated adversary %q not found on combat board: %s", createdAdversary.ID, boardResult.Output)
		}
	})
}
